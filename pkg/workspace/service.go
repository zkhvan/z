package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/exec"
	"github.com/zkhvan/z/pkg/project"
)

var defaultExecutor exec.Interface = exec.New()

// Service manages workspace instances on disk.
type Service struct {
	cfg      Config
	executor exec.Interface
	cacheDir string
	project  *project.Service
}

type ServiceOption func(*Service)

// WithExecutor drives the internal project/gh stack (and future git client).
func WithExecutor(executor exec.Interface) ServiceOption {
	return func(s *Service) {
		s.executor = executor
	}
}

func WithCacheDir(dir string) ServiceOption {
	return func(s *Service) {
		s.cacheDir = dir
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

// Create validates input and writes the manifest atomically. Offline only.
func (s *Service) Create(_ context.Context, name string, members []Member) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := ValidateMembers(members); err != nil {
		return err
	}

	instanceDir := filepath.Join(s.cfg.Root, name)
	if _, err := os.Stat(instanceDir); err == nil {
		return fmt.Errorf("workspace %q already exists at %s", name, instanceDir)
	}

	mf := manifest{
		Version: manifestVersion,
		Members: make([]manifestMember, len(members)),
	}
	for i, m := range members {
		mf.Members[i] = manifestMember(m)
	}

	return writeManifest(instanceDir, mf)
}

// List returns instances one level under the root, sorted by name.
// Directories without .z/instance.yaml are skipped silently.
func (s *Service) List(_ context.Context) ([]Instance, error) {
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

		members := make([]Member, len(mf.Members))
		for i, mm := range mf.Members {
			members[i] = Member(mm)
		}

		instances = append(instances, Instance{
			Name:    e.Name(),
			Dir:     dir,
			Members: members,
			Status:  InstanceStatusNotMaterialized,
		})
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})

	return instances, nil
}
