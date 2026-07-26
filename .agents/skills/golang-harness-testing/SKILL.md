---
name: golang-harness-testing
description: Conventions for writing black-box harness tests in this repo, especially command tests under pkg/cmd/**. Use when testing a cobra command end-to-end, building a test harness, or when tests should read like specifications. Covers the shared world + thin per-package harness + chainable assertions pattern. Pairs with cobra-commands and golang-code.
---

# Go Harness Testing

One pattern for the heavier tests in this repo, used when it pays off (see *When
to reach for a harness* below): a shared **world** package arranges the
filesystem and config, a thin **per-package harness** binds the command under
test, and **chainable assertions** verify observable behavior. Reference
implementation: `pkg/cmd/workspace/internal/wstest/` plus any of
`pkg/cmd/workspace/{create,archive,delete,materialize}/`.

For command structure see the `cobra-commands` skill; for comment style see
`golang-code`.

## When to reach for a harness

This is a tool, not a mandate. The harness earns its weight when a test needs to
**arrange a world, drive a real entry point, and assert observable outcomes** —
typically command (`pkg/cmd/**`) and service-level tests with filesystem/config
setup and several scenarios sharing that setup.

Prefer a harness when most of these hold:

- multiple scenarios share non-trivial setup (config, seeded dirs/files)
- you're exercising a real entry point end-to-end (cobra command, service method)
- assertions inspect produced artifacts (manifests, output) across cases
- preconditions need to be seeded *without* running the code under test

Skip it — a plain table test or a few direct assertions is better — when:

- the unit is pure/standalone (parsing, validation, small helpers). e.g.
  `ParseMember`, `ValidateName`, and the `workspace.Service` domain tests use
  plain table/`t.TempDir` tests, no harness.
- there's one scenario with trivial setup
- a harness would be more code than the thing it tests

Rule of thumb: reach for the harness on the **second or third** scenario that
wants the same setup, not the first. Don't build infrastructure speculatively.
The user decides when it's the right tool — offer it, don't impose it.

## Core principles

1. **Black-box package.** Put command/service tests in `package xxx_test`, not
   the implementation package. You reach only exported API — the same surface
   real callers use. If a test seems to *need* unexported access, that's a signal
   the seam belongs in the public API or the factory, not that the test should
   go white-box.

2. **Drive the real entry point.** Build the actual cobra command with
   `NewCmdXXX(f)` and `Execute()` it. This exercises flag parsing, arg
   validation, and `Complete`/`Run` wiring for free.

3. **Arrange with the harness, act with the command.** Preconditions (existing
   directories, seeded files, config) are created by harness helpers, *never* by
   running the command under test a second time. Keep arrange/act/assert distinct.

4. **Comments need-to-know.** Scenario test names carry the meaning (see
   `golang-code`). Only comment the genuinely non-obvious.

## The shape

```go
h := newCommandTest(t)                                  // arrange
h.SeedInstance("login", workspace.Member{Repo: "owner/repo", Branch: "feature/login"})
h.SeedFile(filepath.Join("login", "notes.md"), "content\n")

err := h.run("login")                                   // act

wstest.AssertErrorContains(t, err, "files not managed by z")   // assert
h.FileExists(filepath.Join("login", "notes.md"))
```

## Shared world, thin per-package harness

Go cannot import identifiers from another package's `_test.go` files, so shared
test scaffolding lives in a normal `internal` package. Split it this way:

- **`internal/wstest`** owns the *world*: temp config, temp roots, the real
  `cmdutil.Factory` with a captured output buffer, `Run(newCmd, args...)`,
  seeding helpers, and filesystem/manifest assertions. Everything reusable
  across commands.
- **`xxx_setup_test.go`** is ~18 lines: embed the world, bind this package's
  constructor, and hold helpers only this package needs.

```go
type harness struct{ *wstest.Harness }

func newCommandTest(t *testing.T) *harness {
    t.Helper()
    return &harness{wstest.New(t)}
}

func (h *harness) run(args ...string) error {
    return h.Run(archive.NewCmdArchive, args...)
}
```

Embedding promotes every shared method, so `h.SeedInstance(...)`,
`h.PathMissing(...)`, `h.OutputContains(...)` work with no forwarding
boilerplate. `newCommandTest(t)` stays the entry point every test opens with, so
a reader doesn't need to know `wstest` exists to follow a test.

**Keep the world unconditional.** `wstest.New(t)` takes no options — every test
wants the same world, and a per-test config builder is ceremony that drifts
(one package silently configured a projects root that others didn't). If a test
ever needs a genuinely different world, add a named constructor then.

## Pass argv, not typed options

Tests pass the arguments a user would type. The per-package `run` is variadic
and forwards straight to cobra.

```go
err := h.run("login", "--force")
err := h.run("auth", "--member", "org/repo1@feature/auth", "--member", "org/repo2@feature/auth:main")
err := h.run()                       // omits the name; reaches cobra's ExactArgs
```

This removes the whole functional-options layer that would otherwise sit in
every command package, and it removes the need to track "was this argument
set?" — with variadic argv, absence is absence. An options struct can't tell
`""` from unset, which is the only reason a `nameSet` bool ever existed.

Trade-off, accepted deliberately: renaming a flag becomes a runtime failure
(`unknown flag`) in each test rather than one compile error. Loud enough.

Inputs that argv *cannot* express — stdin content, TTY/color capability —
belong on the harness (`h.WithStdin(...)`), not in a parallel options layer.
One place to configure a run.

## Chainable, semantic assertions

Read artifacts back through a parser, not raw `strings.Contains`, and expose
fluent matchers that fail with helpful messages via `t.Helper()`.

```go
h.Workspace("auth").
    HasVersion(1).
    MemberCount(2).
    HasMember("org/repo1", "feature/auth", "").
    HasMember("org/repo2", "feature/auth", "main")
```

Each matcher returns the receiver for chaining and calls `t.Helper()` so failures
point at the test line. Parsing the manifest (vs. substring matching) means an
empty `base_ref` is matched as an empty *field*, not an accidental substring.

**Keep the read path independent of the domain type.** `wstest`'s
`instanceManifest` declares its own YAML tags rather than reusing
`workspace.Member`, so a schema change breaks assertions instead of silently
following along. Seeding may use the domain type for convenience; reading must
not.

Keep small free helpers for common cases, and assert sentinel errors through the
public sentinel:

```go
wstest.AssertErrorContains(t, err, "already exists")

// sentinel:
if !errors.Is(err, workspace.ErrInvalidName) { ... }
```

## Build the real factory; inject fakes at a seam

The world builds a real `cmdutil.Factory` but captures output and lets you swap
dependencies. If a fake must be injected but the exported surface has no seam
for it (e.g. a `FakeExec` a black-box test can't reach), that's a finding: add
the seam to the factory or `Options` (see the testability seam in
`cobra-commands`) rather than dropping to a white-box test.

Prefer scenarios that don't need the seam at all. Guards that fail before the
command shells out (validation, missing manifests, unmanaged files) exercise
flag wiring end-to-end with no git fixture.

## Naming

- Test functions: `TestThing_<scenario>` with `<scenario>` in lower
  `snake_case`: `TestCreate_single_member`, `TestCreate_missing_name_arg`,
  `TestArchive_orphaned_member_is_refused`.
- The scenario *is* the documentation — drop redundant doc comments.

## Scenario coverage checklist

For each command/operation, cover:

```
[ ] happy path, minimal (single_member)
[ ] richer happy path with optional inputs (multiple_members_with_base_ref)
[ ] each invalid-args case (missing required arg, missing required flag, malformed value)
[ ] domain validation rejections (assert via sentinel where one exists)
[ ] precondition variants that look alike but differ (empty dir vs dir with the artifact)
[ ] non-destructive failure: after a rejection, nothing written / existing artifact untouched
```

## Refactoring the harness

A change that only moves scaffolding must prove it changed nothing:

```bash
go test -v ./pkg/cmd/... | grep '^--- ' | sort > before.txt
# refactor
diff before.txt after.txt    # empty
```

Preserve test names exactly (rename in a separate commit where the rename is the
visible change), keep per-package coverage from dropping, and keep non-test Go
files out of the diff.

## Anti-patterns

```
[ ] white-box test reaching unexported fields just to inject a value → add a seam
[ ] running the command under test to set up a precondition → seed via the harness
[ ] a functional-options layer wrapping what argv already expresses → pass argv
[ ] `if v != ""` to detect an omitted argument → omit it from argv
[ ] copying the world into each command package → embed the shared harness
[ ] reusing the domain struct to read artifacts back → declare an independent one
[ ] asserting on raw serialized bytes → parse and assert on fields
[ ] doc comments restating the test name → delete them
```
