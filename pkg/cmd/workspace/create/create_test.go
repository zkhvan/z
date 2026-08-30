package create_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zkhvan/z/pkg/assert"
	"github.com/zkhvan/z/pkg/cmd/workspace/internal/wstest"
	"github.com/zkhvan/z/pkg/workspace"
)

func TestCreate_single_member(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("login-feature", "--member", "org/repo@feature/login")
	assert.NoError(t, err)

	h.Workspace("login-feature").
		HasVersion(1).
		MemberCount(1).
		HasMember("org/repo", "feature/login", "")

	h.OutputContains(`Created workspace "login-feature"`)
}

func TestCreate_multiple_members_with_base_ref(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("auth",
		"--member", "org/repo1@feature/auth",
		"--member", "org/repo2@feature/auth:main",
	)
	assert.NoError(t, err)

	h.Workspace("auth").
		MemberCount(2).
		HasMember("org/repo1", "feature/auth", "").
		HasMember("org/repo2", "feature/auth", "main")
}

func TestCreate_missing_name_arg(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "accepts 1 arg(s)")
}

func TestCreate_no_members_outside_a_terminal_fails_loudly(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("empty")

	wstest.AssertErrorContains(t, err, "pass --member or --from, or run interactively")
	h.NoWorkspace("empty")
}

func TestCreate_malformed_member(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("bad", "--member", "org/repo") // missing @branch

	wstest.AssertErrorContains(t, err, "invalid member")
	h.NoWorkspace("bad")
}

func TestCreate_invalid_name(t *testing.T) {
	h := newCommandTest(t)

	err := h.run(".hidden", "--member", "org/repo@main")

	if !errors.Is(err, workspace.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
	h.NoWorkspace(".hidden")
}

func TestCreate_duplicate_member_base_name(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("conflict", "--member", "org1/repo@main", "--member", "org2/repo@main")

	wstest.AssertErrorContains(t, err, "worktree directory name")
	h.NoWorkspace("conflict")
}

func TestCreate_existing_empty_dir_is_rejected(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDir("dupe")

	err := h.run("dupe", "--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "already exists")

	// Nothing written into the pre-existing directory.
	h.NoWorkspace("dupe")
}

func TestCreate_existing_instance_is_rejected(t *testing.T) {
	h := newCommandTest(t)
	h.SeedManifest("dupe", "")

	err := h.run("dupe", "--member", "org/repo@main")

	wstest.AssertErrorContains(t, err, "already exists")

	// The seeded manifest must be untouched (still empty, no members).
	h.Workspace("dupe").MemberCount(0)
}

func TestCreate_from_definition_seeds_members(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}",
		workspace.Member{Repo: "acme/api", BaseRef: "develop"},
		workspace.Member{Repo: "acme/ui"},
	)

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Workspace("login").
		HasVersion(1).
		HasDefinition("api-feature").
		MemberCount(2).
		HasMember("acme/api", "feat/login", "develop").
		HasMember("acme/ui", "feat/login", "")
}

func TestCreate_from_definition_substitutes_repo_base_name(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}-{repo}",
		workspace.Member{Repo: "acme/api"},
		workspace.Member{Repo: "acme/ui"},
	)

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Workspace("login").
		HasMember("acme/api", "feat/login-api", "").
		HasMember("acme/ui", "feat/login-ui", "")
}

func TestCreate_from_definition_without_a_pattern_uses_the_default(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "", workspace.Member{Repo: "acme/api"})

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Workspace("login").HasMember("acme/api", "login", "")
}

// The headline promise: the same definition instantiated twice yields branches
// that do not collide, with no extra machinery.
func TestCreate_parallel_instances_get_non_colliding_branches(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}",
		workspace.Member{Repo: "acme/api"},
		workspace.Member{Repo: "acme/ui"},
	)

	if err := h.run("attempt-a", "--from", "api-feature"); err != nil {
		t.Fatalf("create attempt-a: %v", err)
	}
	if err := h.run("attempt-b", "--from", "api-feature"); err != nil {
		t.Fatalf("create attempt-b: %v", err)
	}

	for _, repo := range []string{"acme/api", "acme/ui"} {
		a := h.Workspace("attempt-a").BranchFor(repo)
		b := h.Workspace("attempt-b").BranchFor(repo)
		if a == b {
			t.Fatalf("member %q got the same branch %q in both instances", repo, a)
		}
	}

	h.Workspace("attempt-a").HasMember("acme/api", "feat/attempt-a", "")
	h.Workspace("attempt-b").HasMember("acme/api", "feat/attempt-b", "")
}

func TestCreate_branch_override_by_base_name(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}",
		workspace.Member{Repo: "acme/api", BaseRef: "develop"},
		workspace.Member{Repo: "acme/ui"},
	)

	err := h.run("login", "--from", "api-feature", "--branch", "api=hotfix/urgent")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// The override replaces one field; base_ref still comes from the definition.
	h.Workspace("login").
		HasMember("acme/api", "hotfix/urgent", "develop").
		HasMember("acme/ui", "feat/login", "")
}

func TestCreate_branch_override_by_remote_id(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	if err := h.run("login", "--from", "api-feature", "--branch", "acme/api=hotfix/urgent"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Workspace("login").HasMember("acme/api", "hotfix/urgent", "")
}

func TestCreate_branch_override_for_unknown_member(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--branch", "db=x")

	wstest.AssertErrorContains(t, err, "no member \"db\"")
	wstest.AssertErrorContains(t, err, "acme/api")
	h.NoWorkspace("login")
}

func TestCreate_duplicate_branch_override_key(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--branch", "api=x", "--branch", "api=y")

	wstest.AssertErrorContains(t, err, "already has an override")
	h.NoWorkspace("login")
}

func TestCreate_two_branch_override_keys_for_one_member(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--branch", "api=x", "--branch", "acme/api=y")

	wstest.AssertErrorContains(t, err, "both target member")
	h.NoWorkspace("login")
}

func TestCreate_malformed_branch_override(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--branch", "api")

	wstest.AssertErrorContains(t, err, "expected <repo>=<branch>")
	h.NoWorkspace("login")
}

func TestCreate_branch_override_with_invalid_branch(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--branch", "api=bad branch")

	wstest.AssertErrorContains(t, err, "invalid branch")
	h.NoWorkspace("login")
}

func TestCreate_branch_without_from_is_refused(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("login", "--member", "acme/api@main", "--branch", "api=x")

	wstest.AssertErrorContains(t, err, "--branch requires --from")
	h.NoWorkspace("login")
}

func TestCreate_from_and_member_are_mutually_exclusive(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "api-feature", "--member", "acme/db@main")

	wstest.AssertErrorContains(t, err, "mutually exclusive")
	h.NoWorkspace("login")
}

func TestCreate_from_missing_definition(t *testing.T) {
	h := newCommandTest(t)

	err := h.run("login", "--from", "nope")

	wstest.AssertErrorContains(t, err, "not found")
	h.NoWorkspace("login")
}

func TestCreate_from_definition_with_no_members(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("scaffold", "feat/{instance}")

	err := h.run("login", "--from", "scaffold")

	wstest.AssertErrorContains(t, err, "has no members")
	h.NoWorkspace("login")
}

func TestCreate_from_broken_definition(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("typo", "feat/{user}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "typo")

	wstest.AssertErrorContains(t, err, "not usable")
	wstest.AssertErrorContains(t, err, "{user}")
	h.NoWorkspace("login")
}

// The instance name is a legal dirname but not a legal branch: reported, never
// slugified into something the user did not write.
func TestCreate_from_definition_with_branch_unsafe_instance_name(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("my feature", "--from", "api-feature")

	wstest.AssertErrorContains(t, err, "feat/my feature")
	h.NoWorkspace("my feature")
}

func TestCreate_from_takes_a_name_not_a_path(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})

	err := h.run("login", "--from", "./api-feature")

	if !errors.Is(err, workspace.ErrInvalidName) {
		t.Fatalf("error %v does not wrap ErrInvalidName", err)
	}
	h.NoWorkspace("login")
}

func TestCreate_from_definition_copies_definition_files(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})
	h.SeedDefinitionFile("api-feature", "CLAUDE.md", "workspace instructions\n")
	h.SeedDefinitionExecFile("api-feature", "bin/setup", "#!/bin/sh\n")

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.FileContains(filepath.Join("login", "CLAUDE.md"), "workspace instructions\n")
	h.FileIsExecutable(filepath.Join("login", "bin", "setup"))
	h.SyncState("login").
		HasVersion(1).
		FileCount(2).
		Records("CLAUDE.md").
		RecordsExecutable("bin/setup")
}

func TestCreate_with_explicit_members_records_no_sync_state(t *testing.T) {
	h := newCommandTest(t)

	if err := h.run("login", "--member", "owner/repo@feature/login"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.NoSyncState("login")
}

func TestCreate_runs_pre_and_post_create_hooks(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})
	// pre-create runs before the instance exists, with cwd at the workspaces
	// root, so it records itself there.
	h.SeedDefinitionHook("api-feature", workspace.HookPreCreate,
		"#!/bin/sh\nprintf '%s' \"$Z_HOOK_PHASE\" > \"$Z_INSTANCE_NAME.pre\"\n")
	// post-create runs with cwd at the instance directory and full env.
	h.SeedDefinitionHook("api-feature", workspace.HookPostCreate,
		"#!/bin/sh\nprintf '%s\\n%s\\n%s\\n' \"$Z_HOOK_PHASE\" \"$Z_INSTANCE_NAME\" \"$Z_DEFINITION_PATH\" > env.out\n")

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.FileContains("login.pre", "pre-create")
	h.FileContains(filepath.Join("login", "env.out"), "post-create\nlogin\n"+h.DefinitionDir("api-feature")+"\n")
}

func TestCreate_pre_create_hook_failure_aborts(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})
	h.SeedDefinitionHook("api-feature", workspace.HookPreCreate, "#!/bin/sh\nexit 1\n")

	err := h.run("login", "--from", "api-feature")

	wstest.AssertErrorContains(t, err, "pre-create hook failed")
	h.NoWorkspace("login")
}

func TestCreate_non_executable_hook_is_an_error(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})
	h.SeedDefinitionFile("api-feature", filepath.Join("hooks", "pre-create"), "#!/bin/sh\n")

	err := h.run("login", "--from", "api-feature")

	wstest.AssertErrorContains(t, err, "not executable")
	h.NoWorkspace("login")
}

func TestCreate_hooks_are_not_synced_into_the_instance(t *testing.T) {
	h := newCommandTest(t)
	h.SeedDefinition("api-feature", "feat/{instance}", workspace.Member{Repo: "acme/api"})
	h.SeedDefinitionHook("api-feature", workspace.HookPostCreate, "#!/bin/sh\n")

	if err := h.run("login", "--from", "api-feature"); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.PathMissing(filepath.Join("login", "hooks"))
	h.PathMissing(filepath.Join("login", "hooks", "post-create"))
}
