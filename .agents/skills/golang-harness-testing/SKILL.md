---
name: golang-harness-testing
description: Conventions for writing black-box harness tests in this repo, especially command tests under pkg/cmd/**. Use when testing a cobra command end-to-end, building a test harness, or when tests should read like specifications. Covers the harness + namespaced options + chainable assertions pattern. Pairs with cobra-commands and golang-code.
---

# Go Harness Testing

One pattern for the heavier tests in this repo, used when it pays off (see *When
to reach for a harness* below): a per-package **harness** arranges the world, a
**runner** drives the real public API, and **chainable assertions** verify
observable behavior. Reference implementation: `pkg/cmd/workspace/create/`
(`create_test.go` + `create_setup_test.go`).

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

4. **One harness value, multiple views.** `newCommandTest(t)` returns the runner,
   the harness, and a cleanup func, all backed by one struct, so setup and
   assertions stay coordinated but read as separate concerns.

5. **Comments need-to-know.** Scenario test names carry the meaning (see
   `golang-code`). Only comment the genuinely non-obvious.

## The shape

```go
cmd, harness, cleanup := newCommandTest(t)
defer cleanup()

harness.config(withWorkspaceRoot())                 // arrange
harness.seedWorkspaceDir(wsDir.withName("dupe"))

err := cmd.run(                                     // act
    cmd.withName("dupe"),
    cmd.withMember("org/repo@main"),
)

assertErrorContains(t, err, "already exists")       // assert
harness.noWorkspace("dupe")
```

## Namespaced functional options

Options belong to the thing they configure. This avoids name collisions and makes
each option's scope obvious at the call site.

- **Action options** are methods on the runner: `cmd.withName(...)`,
  `cmd.withMember(...)` return a `runOption` consumed by `cmd.run(opts...)`.
- **Setup options** live under a stateless namespace var: `wsDir.withName(...)`,
  `wsDir.withEmptyInstanceManifest()` return a `wsDirOption` consumed by
  `harness.seedWorkspaceDir(opts...)`.

```go
// cmd namespace (methods on *cmdRunner; receiver unused, just namespaces the name)
type runSpec struct {
    name    string
    nameSet bool
    members []string
}
type runOption func(*runSpec)

func (*cmdRunner) withName(name string) runOption {
    return func(s *runSpec) { s.name = name; s.nameSet = true }
}
func (*cmdRunner) withMember(m string) runOption {
    return func(s *runSpec) { s.members = append(s.members, m) }
}

// wsDir namespace (stateless var used purely for naming)
type wsDirSpec struct {
    name             string
    instanceManifest bool
}
type wsDirOption func(*wsDirSpec)
type wsDirNS struct{}
var wsDir wsDirNS

func (wsDirNS) withName(name string) wsDirOption {
    return func(s *wsDirSpec) { s.name = name }
}
func (wsDirNS) withEmptyInstanceManifest() wsDirOption {
    return func(s *wsDirSpec) { s.instanceManifest = true }
}
```

**Avoid** one shared builder (a free `withName(...)` used everywhere). It collides
across contexts and hides which options apply where. Prefer two small namespaced
sets over one clever shared one.

## Track "set" vs zero value

When an optional positional/flag must be *omittable*, record whether it was
provided — don't infer from the zero value. Passing `""` as an argument is not
the same as passing no argument.

```go
var args []string
if spec.nameSet {            // not: if spec.name != ""
    args = append(args, spec.name)
}
for _, m := range spec.members {
    args = append(args, "--member", m)
}
```

This is what lets a "missing required arg" test actually reach cobra's
`ExactArgs` validation instead of silently passing an empty name.

## Chainable, semantic assertions

Read artifacts back through a parser, not raw `strings.Contains`, and expose
fluent matchers that fail with helpful messages via `t.Helper()`.

```go
harness.workspace("auth").
    hasVersion(1).
    memberCount(2).
    hasMember("org/repo1", "feature/auth", "").
    hasMember("org/repo2", "feature/auth", "main")
```

Each matcher returns the receiver for chaining and calls `t.Helper()` so failures
point at the test line. Parsing the manifest (vs. substring matching) means an
empty `base_ref` is matched as an empty *field*, not an accidental substring.

Keep small free helpers for common cases, and assert sentinel errors through the
public sentinel:

```go
func assertErrorContains(t *testing.T, err error, want string) {
    t.Helper()
    if err == nil {
        t.Fatalf("expected error containing %q, got nil", want)
    }
    if !strings.Contains(err.Error(), want) {
        t.Fatalf("error %q does not contain %q", err.Error(), want)
    }
}

// sentinel:
if !errors.Is(err, workspace.ErrInvalidName) { ... }
```

## Build the real factory; inject fakes at a seam

The runner builds a real `cmdutil.Factory` but lets you capture output and swap
dependencies. Point `IOStreams.Out` at a buffer you own; discard the rest.

```go
f := &cmdutil.Factory{
    IOStreams: &iolib.IOStreams{In: strings.NewReader(""), Out: &h.out, ErrOut: io.Discard},
    Config:    h.cfg,
}
cmd := create.NewCmdCreate(f)
cmd.SilenceUsage = true
cmd.SilenceErrors = true
```

If a fake must be injected but the exported surface has no seam for it (e.g. a
`FakeExec` a black-box test can't reach), that's a finding: add the seam to the
factory or `Options` (see the testability seam in `cobra-commands`) rather than
dropping to a white-box test.

## Naming

- Test functions: `TestThing_<scenario>` with `<scenario>` in lower
  `snake_case`: `TestCreate_single_member`, `TestCreate_missing_name_arg`,
  `TestCreate_existing_instance_is_rejected`.
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

## File layout

- `xxx_test.go` — scenarios and assertions only; reads as a spec.
- `xxx_setup_test.go` — the harness: `newCommandTest`, config/seed builders, the
  runner, and assertion helpers. Reusable across the package's tests.

## Anti-patterns

```
[ ] white-box test reaching unexported fields just to inject a value → add a seam
[ ] running the command under test to set up a precondition → seed via the harness
[ ] one shared withX builder across unrelated contexts → namespace them
[ ] `if v != ""` to detect an omitted argument → track an explicit `set` bool
[ ] asserting on raw serialized bytes → parse and assert on fields
[ ] doc comments restating the test name → delete them
```
