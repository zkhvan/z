package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/workspace"
)

func setupLocateService(t *testing.T) (*workspace.Service, string) {
	t.Helper()

	td := setupServiceTestDir(t)
	cfg := setupServiceConfig(t, td, `
workspaces:
  root: $WORKSPACESDIR
`)
	svc, err := workspace.NewService(cfg)
	assert.NoError(t, err)

	seedLocateInstance(t, td.workspaces, "login", "zkhvan/api", "zkhvan/web")

	return svc, td.workspaces
}

func seedLocateInstance(t *testing.T, root, name string, repos ...string) {
	t.Helper()

	contents := "version: 1\nmembers:\n"
	for _, repo := range repos {
		contents += "  - repo: " + repo + "\n    branch: feature/login\n"
		assert.NoError(t, os.MkdirAll(filepath.Join(root, name, filepath.Base(repo)), 0o700))
	}

	assert.NoError(t, os.MkdirAll(filepath.Join(root, name, ".z"), 0o700))
	assert.NoError(t, os.WriteFile(
		filepath.Join(root, name, ".z", "instance.yaml"), []byte(contents), 0o600))
}

func TestResolveDir_instance_directory(t *testing.T) {
	svc, root := setupLocateService(t)

	loc, ok, err := svc.ResolveDir(filepath.Join(root, "login"))

	assert.NoError(t, err)
	if !ok {
		t.Fatal("expected the instance directory to resolve")
	}
	assert.EqualString(t, loc.Name, "login")
	assert.EqualString(t, loc.Member, "")
	assert.EqualString(t, loc.Dir, filepath.Join(root, "login"))
}

func TestResolveDir_member_worktree(t *testing.T) {
	svc, root := setupLocateService(t)

	loc, ok, err := svc.ResolveDir(filepath.Join(root, "login", "web"))

	assert.NoError(t, err)
	if !ok {
		t.Fatal("expected the member worktree to resolve")
	}
	assert.EqualString(t, loc.Name, "login")
	assert.EqualString(t, loc.Member, "web")
}

func TestResolveDir_nested_inside_member_worktree(t *testing.T) {
	svc, root := setupLocateService(t)

	loc, _, err := svc.ResolveDir(filepath.Join(root, "login", "api", "pkg", "deep"))

	assert.NoError(t, err)
	assert.EqualString(t, loc.Name, "login")
	assert.EqualString(t, loc.Member, "api")
}

// A directory that is not a declared member must not pose as one, or checkout
// would infer a repo the manifest has never heard of.
func TestResolveDir_undeclared_directory_is_not_a_member(t *testing.T) {
	svc, root := setupLocateService(t)

	for _, dir := range []string{".z", "notes"} {
		loc, ok, err := svc.ResolveDir(filepath.Join(root, "login", dir))

		assert.NoError(t, err)
		if !ok {
			t.Fatalf("expected %q to resolve to the instance", dir)
		}
		assert.EqualString(t, loc.Name, "login")
		assert.EqualString(t, loc.Member, "")
	}
}

// An instance is a directory under the root, manifest or not; the caller
// reports the missing manifest with its own error.
func TestResolveDir_directory_without_a_manifest(t *testing.T) {
	svc, root := setupLocateService(t)
	assert.NoError(t, os.MkdirAll(filepath.Join(root, "scratch"), 0o700))

	loc, ok, err := svc.ResolveDir(filepath.Join(root, "scratch"))

	assert.NoError(t, err)
	if !ok {
		t.Fatal("expected a manifest-less directory to resolve by name")
	}
	assert.EqualString(t, loc.Name, "scratch")
}

func TestResolveDir_root_itself(t *testing.T) {
	svc, root := setupLocateService(t)

	_, ok, err := svc.ResolveDir(root)

	assert.NoError(t, err)
	if ok {
		t.Fatal("expected the workspaces root not to resolve to an instance")
	}
}

func TestResolveDir_outside_the_root(t *testing.T) {
	svc, root := setupLocateService(t)

	_, ok, err := svc.ResolveDir(filepath.Dir(root))

	assert.NoError(t, err)
	if ok {
		t.Fatal("expected a directory outside the root not to resolve")
	}
}

func TestResolveDir_symlinked_path(t *testing.T) {
	svc, root := setupLocateService(t)

	link := filepath.Join(t.TempDir(), "current")
	assert.NoError(t, os.Symlink(filepath.Join(root, "login", "api"), link))

	loc, ok, err := svc.ResolveDir(link)

	assert.NoError(t, err)
	if !ok {
		t.Fatal("expected a symlink into the root to resolve")
	}
	assert.EqualString(t, loc.Name, "login")
	assert.EqualString(t, loc.Member, "api")
}

func TestResolveDir_missing_directory(t *testing.T) {
	svc, root := setupLocateService(t)

	loc, ok, err := svc.ResolveDir(filepath.Join(root, "login", "api", "not-created-yet"))

	assert.NoError(t, err)
	if !ok {
		t.Fatal("expected a path under the root to resolve without existing")
	}
	assert.EqualString(t, loc.Name, "login")
	assert.EqualString(t, loc.Member, "api")
}
