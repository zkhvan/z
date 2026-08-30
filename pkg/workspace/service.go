package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/exec"
	gitlib "github.com/zkhvan/z/pkg/git"
	"github.com/zkhvan/z/pkg/iolib"
	"github.com/zkhvan/z/pkg/project"
)

var defaultExecutor exec.Interface = exec.New()

// Service manages workspace instances on disk.
type Service struct {
	cfg      Config
	executor exec.Interface
	cacheDir string
	project  *project.Service
	git      *gitlib.Client
	// io is the sink lifecycle hooks stream their stdout/stderr to. It defaults
	// to the real process streams so a hook is visible even when no caller wired
	// one in.
	io *iolib.IOStreams
}

type ServiceOption func(*Service)

// WithExecutor drives the internal project/gh stack (and future git client).
func WithExecutor(executor exec.Interface) ServiceOption {
	return func(s *Service) {
		s.executor = executor
		if s.git != nil {
			s.git.SetExecutor(executor)
		}
	}
}

func WithCacheDir(dir string) ServiceOption {
	return func(s *Service) {
		s.cacheDir = dir
	}
}

// WithIOStreams routes lifecycle hook output to the given streams.
func WithIOStreams(io *iolib.IOStreams) ServiceOption {
	return func(s *Service) {
		s.io = io
	}
}

func NewService(cfg cmdutil.Config, opts ...ServiceOption) (*Service, error) {
	wsCfg, err := NewConfig(cfg)
	if err != nil {
		return nil, err
	}

	s := &Service{
		cfg:      wsCfg,
		executor: defaultExecutor,
		git:      gitlib.NewClient(),
		io:       iolib.System(),
	}

	for _, o := range opts {
		o(s)
	}

	projOpts := []project.ServiceOption{
		project.WithExecutor(s.executor),
	}
	if s.cacheDir != "" {
		projOpts = append(projOpts, project.WithCacheDir(s.cacheDir))
	}

	projSvc, err := project.NewService(cfg, projOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating project service: %w", err)
	}
	s.project = projSvc

	return s, nil
}

// CreateOptions carries every create-time input. One entry point keeps the
// --from/--member exclusion at the domain boundary, and gives later create-time
// concerns (hooks, sync state) a single place to land.
type CreateOptions struct {
	// Members declares the member set directly.
	Members []Member

	// From names a definition to seed the member set from. Mutually exclusive
	// with Members.
	From string

	// BranchOverrides replaces the pattern-derived branch for individual
	// members, keyed by base name or full remote ID. Requires From.
	BranchOverrides map[string]string
}

// Create validates input and writes the manifest atomically. Offline only.
func (s *Service) Create(_ context.Context, name string, opts CreateOptions) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if opts.From != "" && len(opts.Members) > 0 {
		return fmt.Errorf("cannot combine a definition with explicit members")
	}
	if opts.From == "" && len(opts.BranchOverrides) > 0 {
		return fmt.Errorf("branch overrides require a definition")
	}

	members, err := s.resolveCreateMembers(name, opts)
	if err != nil {
		return err
	}
	if err := ValidateMembers(members); err != nil {
		return err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	if _, err := os.Stat(instanceDir); err == nil {
		return fmt.Errorf("workspace %q already exists at %s", name, instanceDir)
	}

	// resolveCreateMembers has already proven a non-empty definition loads and is
	// usable, so this reload cannot fail; it only hands the hooks a Definition. An
	// empty From yields a zero Definition, i.e. no hooks.
	var def Definition
	if opts.From != "" {
		loaded, err := s.Definition(opts.From)
		if err != nil {
			return err
		}
		def = loaded
	}

	// pre-create runs before the instance directory exists, with cwd set to its
	// parent, which must therefore exist first.
	if err := os.MkdirAll(s.cfg.Root, 0o700); err != nil {
		return fmt.Errorf("creating workspaces root: %w", err)
	}
	if err := s.runHook(def, name, instanceDir, HookPreCreate); err != nil {
		return err
	}

	mf := manifest{
		Version:    manifestVersion,
		Definition: opts.From,
		Members:    make([]manifestMember, len(members)),
	}
	for i, m := range members {
		mf.Members[i] = manifestMemberFromMember(m)
	}

	if err := writeManifest(instanceDir, mf); err != nil {
		return err
	}

	if opts.From != "" {
		if err := s.populate(instanceDir, opts.From, members); err != nil {
			return err
		}
	}
	return s.runHook(def, name, instanceDir, HookPostCreate)
}

// populate performs the create-time copy. It is reconciliation against an absent
// ancestor, so there is one code path rather than two. A failure leaves a valid
// instance with accurate partial state that `z workspace sync` completes, which
// is why create needs no rollback.
func (s *Service) populate(instanceDir, from string, members []Member) error {
	def, err := s.Definition(from)
	if err != nil {
		return err
	}
	if _, err := s.sync(instanceDir, def, members, SyncOptions{}); err != nil {
		return err
	}
	return nil
}

// resolveCreateMembers produces the final member set before anything is
// written, so a failed resolution leaves no partial instance behind.
func (s *Service) resolveCreateMembers(name string, opts CreateOptions) ([]Member, error) {
	if opts.From == "" {
		return opts.Members, nil
	}

	def, err := s.Definition(opts.From)
	if err != nil {
		return nil, err
	}
	if def.Broken() {
		return nil, fmt.Errorf("definition %q is not usable: %w", def.Name, def.Err)
	}
	if len(def.Members) == 0 {
		return nil, fmt.Errorf("definition %q has no members: add them to %s",
			def.Name, filepath.Join(def.Dir, definitionManifestRelPath))
	}

	members := make([]Member, len(def.Members))
	for i, dm := range def.Members {
		branch, err := ResolveBranch(def.BranchPattern, name, dm.Repo)
		if err != nil {
			return nil, err
		}
		members[i] = Member{Repo: dm.Repo, Branch: branch, BaseRef: dm.BaseRef}
	}

	if err := applyBranchOverrides(members, opts.BranchOverrides); err != nil {
		return nil, err
	}

	return members, nil
}

// applyBranchOverrides matches each key against a member's base name or full
// remote ID. Base names cannot contain "/" and are unique within a workspace,
// so neither form can match two members.
func applyBranchOverrides(members []Member, overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}

	matched := make(map[int]string)
	for _, key := range sortedKeys(overrides) {
		idx := -1
		for i, m := range members {
			if m.Repo == key || m.BaseName() == key {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no member %q; definition members are %s",
				key, strings.Join(memberRepos(members), ", "))
		}
		if prev, ok := matched[idx]; ok {
			return fmt.Errorf("branch overrides %q and %q both target member %q",
				prev, key, members[idx].Repo)
		}
		matched[idx] = key
		members[idx].Branch = overrides[key]
	}

	return nil
}

func memberRepos(members []Member) []string {
	repos := make([]string, len(members))
	for i, m := range members {
		repos[i] = m.Repo
	}
	return repos
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// InitDefinition scaffolds a definition directory. The result is intentionally
// not yet instantiable: it has no members until the user adds them.
func (s *Service) InitDefinition(_ context.Context, name string) (string, error) {
	if err := ValidateDefinitionName(name); err != nil {
		return "", err
	}

	dir := filepath.Join(s.cfg.DefinitionsRoot, name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("definition %q already exists at %s", name, dir)
	}

	dotZ := filepath.Join(dir, ".z")
	if err := os.MkdirAll(dotZ, 0o700); err != nil {
		return "", fmt.Errorf("creating definition directory: %w", err)
	}

	path := filepath.Join(dotZ, "definition.yaml")
	contents := fmt.Sprintf(definitionTemplate, definitionVersion)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return "", fmt.Errorf("writing definition manifest: %w", err)
	}

	if err := writeExampleHooks(dir); err != nil {
		return "", err
	}

	return dir, nil
}

// Definition loads one definition by name from the configured root.
func (s *Service) Definition(name string) (Definition, error) {
	if err := ValidateDefinitionName(name); err != nil {
		return Definition{}, err
	}

	def, err := readDefinition(s.cfg.DefinitionsRoot, name)
	if err != nil {
		if os.IsNotExist(err) {
			return Definition{}, fmt.Errorf("definition %q not found: missing %s",
				name, filepath.Join(s.cfg.DefinitionsRoot, name, definitionManifestRelPath))
		}
		return Definition{}, err
	}

	return def, nil
}

// ListDefinitions returns definitions one level under the root, sorted by name.
// Directories without a definition manifest are skipped; directories with one
// that cannot be used are reported with Err set, because the marker file says
// a definition was intended.
func (s *Service) ListDefinitions(_ context.Context) ([]Definition, error) {
	entries, err := os.ReadDir(s.cfg.DefinitionsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading definitions root %s: %w", s.cfg.DefinitionsRoot, err)
	}

	var defs []Definition
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		def, err := readDefinition(s.cfg.DefinitionsRoot, e.Name())
		if err != nil {
			continue // not a definition directory
		}
		defs = append(defs, def)
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})

	return defs, nil
}

// List returns instances one level under the root, sorted by name.
// Directories without .z/instance.yaml are skipped silently.
func (s *Service) List(ctx context.Context) ([]Instance, error) {
	entries, err := os.ReadDir(s.cfg.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading workspaces root %s: %w", s.cfg.Root, err)
	}

	var instances []Instance
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		dir := filepath.Join(s.cfg.Root, e.Name())
		mf, err := readManifest(dir)
		if err != nil {
			continue // not an instance directory
		}

		members := make([]InstanceMember, len(mf.Members))
		for i, mm := range mf.Members {
			members[i] = memberFromManifest(mm)
			members[i].State = MemberStateUnknown
		}
		if err := ValidateMembers(members); err != nil {
			continue
		}
		status := s.deriveInstanceStatus(ctx, dir, members)

		instances = append(instances, Instance{
			Name:    e.Name(),
			Dir:     dir,
			Members: members,
			Status:  status,
		})
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})

	return instances, nil
}
