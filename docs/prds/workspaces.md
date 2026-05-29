# PRD: Workspaces

## Problem Statement

`z project` treats every unit of work as a single git repository. In practice, real work often spans multiple repositories at once — a feature change that touches an API repo and a UI repo, a bugfix that needs a coordinated branch across two services, an exploration that wants the same agent context shared across a handful of related repos. The current model has no way to express "these repos belong together, for this work, with this branch state, with this agent configuration."

A second pressure has emerged: AI coding agents increasingly run inside containers or VMs that need a well-defined view of the code they're allowed to touch. Today, pointing an agent at a project means pointing it at one repo. There's no first-class concept for "give the agent these N repos, with this `AGENTS.md`, with these skills, mounted into a sandbox" — and trying to bolt it onto `z project` confuses the existing fast-navigation flow.

Finally, work isn't always one-shot. The same multi-repo feature may need parallel attempts — three agents trying three different implementations of the same change, each in its own isolated worktree, each reading from the same shared configuration. There is no primitive for "instantiate this kind of work multiple times in parallel."

## Solution

Introduce a new top-level concept, **workspace**, layered on top of (not replacing) `z project`.

A workspace groups one or more repositories under a single human-named handle. It carries its own agent configuration (`AGENTS.md` / `CLAUDE.md`, skills, MCP servers, settings), its own lifecycle hooks, and its own runtime adapter (devcontainer, docker, podman, VM, etc.) so an agent or developer can be dropped into a coherent multi-repo view with a single command.

Workspaces split into two artifacts:

- A **definition** is the reusable template — which repos, which default branches, which agent configuration, which hooks, which runtime. Definitions live as plain directories under a configured root and can be version-controlled however the user prefers.
- An **instance** is a materialization of a definition: a real directory on disk containing git worktrees of the member repos, alongside the synced agent configuration and hooks. Multiple instances of the same definition can exist in parallel, each on its own branches, each independently usable.

A workspace's directory is the unit a container or VM mounts. The agent inside sees a clean `/workspace/<repo>/` layout, regardless of where the canonical clones live on the host.

The existing `z project` flow is preserved unchanged for single-repo navigation. Workspaces are reserved for the heavier multi-repo + agent-environment case, and consume `z project` internally for repo discovery and cloning rather than duplicating that machinery.

## User Stories

1. As a developer, I want to declare that a piece of work spans multiple repositories so that I can manage their branches together as one unit.
2. As a developer, I want to define a workspace template once and instantiate it multiple times so that I can run parallel attempts at the same feature without re-specifying which repos and branches are involved.
3. As a developer, I want my workspace instance to live in a real on-disk directory containing all its member repositories as subdirectories so that I can navigate, edit, search, and use any standard tool across the whole feature in one place.
4. As a developer, I want each workspace instance to have isolated working state (its own branches, its own checkouts) so that I can have two instances of the same workspace on two different branches simultaneously without conflicts.
5. As a developer, I want member repositories to live in their canonical location on disk and only appear in workspaces as worktrees so that I don't waste disk on duplicate clones and so my existing `z project` flow is unaffected.
6. As a developer, I want workspaces to track the branch of each member repository independently so that I can support stacked-PR workflows where one repo moves forward in a stack while another stays put.
7. As a developer, I want to materialize a workspace with a single command that auto-clones any member repos I don't have yet, so that creating a workspace doesn't require multiple prep steps.
8. As a developer, I want workspace creation to fail loudly if a branch I'm trying to materialize is already checked out somewhere so that I never end up with silently-renamed branches behind my back.
9. As a developer, I want to be able to create branches at materialize-time from a configurable base reference so that "start working on feature X" is a single primitive that produces a coordinated branch across all member repos.
10. As a developer, I want a workspace's default branch names to be derived from a definition-level pattern so that creating multiple instances doesn't require manually specifying branch names for every member repo every time.
11. As a developer, I want to override default branch names at creation time for individual members so that I can deviate from the pattern when I need to.
12. As a developer, I want to switch the active branch of a member repo within a workspace so that I can move along a PR stack without leaving the workspace.
13. As a developer, I want a workspace to carry a `CLAUDE.md` / `AGENTS.md` at its root, layered above the per-repo `CLAUDE.md` files inside member worktrees, so that I can express agent instructions that apply across the whole feature.
14. As a developer, I want skills and MCP server configurations to be definable at the workspace level so that agent behaviors specific to "this kind of work" travel with the workspace rather than living globally or per-repo.
15. As a developer, I want truly global skills to keep living in my global Claude configuration so that the workspace layer doesn't force me to copy ubiquitous skills into every workspace.
16. As a developer, I want a workspace to be containerizable via a single bind mount of its directory so that running an agent in a container is one mount, not N.
17. As a developer, I want the container/VM runtime for a workspace to be pluggable so that I'm not locked into devcontainer, docker, or any one tool.
18. As a developer, I want to bring a workspace's runtime environment up, run commands in it, and tear it down via `z` so that the launch UX is uniform regardless of what runtime is configured underneath.
19. As a developer, I want lifecycle hooks (create, materialize, archive, delete) so that I can wire in setup work like installing dependencies, copying `.envrc` files, or registering with external systems without re-implementing it per workspace.
20. As a developer, I want hooks to be discovered by file convention (one script per phase) so that I don't have to learn a hook-declaration syntax.
21. As a developer, I want hooks to run on the host with a documented environment (instance path, instance name, definition path) so that I can write them in plain shell without orchestration glue.
22. As a developer, I want a workspace's definition files (agent config, hooks, runtime scripts) to propagate into instances by default so that I edit them in one place and consumers stay current.
23. As a developer, I want to override any synced file in a single instance (without affecting the definition or other instances) so that I can experiment locally without forking the definition.
24. As a developer, I want `z` to detect when I've locally modified a synced file and refuse to overwrite it on the next sync so that my local edits aren't silently clobbered.
25. As a developer, I want a `sync` command that pulls the latest definition state into an instance so that I can refresh on demand after editing the definition.
26. As a developer, I want to archive a workspace instance (remove its worktrees but keep its metadata and agent config) so that I can free disk and branches without losing the workspace's identity.
27. As a developer, I want to re-materialize an archived workspace so that I can pick a feature back up without re-declaring its members.
28. As a developer, I want to list my workspaces (definitions and instances, separately) so that I can see what I've defined and what's currently materialized.
29. As a developer, I want workspaces to be discovered by scanning a known directory rather than via a central registry so that the filesystem is the source of truth and there are no registry/filesystem-drift bugs.
30. As a developer, I want a workspace name to be its directory name so that renaming is a `mv`, uniqueness is enforced by the filesystem, and there's nothing to learn.
31. As a developer, I want the workspaces directory and the definitions directory to be configurable so that I can place them where my dotfiles or backup setup expects them.
32. As a developer, I want my existing `z project` flow (`select`, `clone`, `list`, etc.) to work exactly as it does today so that the workspace feature is purely additive and I can adopt it gradually.
33. As a developer, I want workspaces to consume `z project`'s repo identity, discovery, and clone machinery so that I don't have two parallel notions of "where does repo X live."
34. As an AI agent author, I want to launch an agent inside a workspace's runtime with one command so that I can drop a model into a multi-repo context without writing mount and env plumbing per workspace.
35. As an AI agent runtime, I want to see a stable, clean `/workspace/<repo>/` layout inside the container so that prompts and tooling don't need to know about host-side paths or username-containing directories.
36. As a developer running parallel agent attempts, I want each instance to share live definition-level skills and `CLAUDE.md` by default so that improvements to the agent context propagate to in-flight experiments.

## Implementation Decisions

**Layering relative to existing concepts.** Workspaces are built on top of the existing project concept; they do not replace it. The project layer owns repo identity, discovery, canonical clone paths, the local/remote/synced source markers, and the `remote_patterns` rewriting. Workspace consumes that surface to resolve member repos to canonical clones and to auto-clone missing ones. The project layer has no knowledge of workspaces.

**Definition vs instance.** A workspace has two artifacts: a definition (the reusable template) and an instance (the materialized workspace on disk). Definitions live in a configurable definitions root (default `~/.config/z/definitions/`). Instances live in a configurable workspaces root (default `~/Workspaces/`). Both are addressed by directory name; the filesystem is the registry.

**Soft-linked configuration via hash-tracked sync.** Configuration in an instance (agent config, hooks, runtime scripts, seeded files) is copied from the definition at create-time and tracked in a sync-state file recording the hash of each file as it was written. A `sync` operation compares each instance file's current hash against the recorded one: if equal, the definition's current version overwrites it; if different, the file is treated as user-modified and skipped (with `--force` available). Symlinks were rejected because they break inside containers when the symlink target lives outside the bind mount.

**Workspace directory as the container surface.** The instance directory contains the worktrees alongside the synced configuration. This directory is the single bind mount a runtime exposes to a container or VM, appearing inside as `/workspace`. There is no separate config directory to mount.

**Per-repo tracked branches with definition-level patterns.** The manifest stores an explicit branch and base ref per member, so different members may sit on differently-named branches (necessary for stacked-PR workflows). At create-time, a pattern declared in the definition (with placeholders like `{instance}` and `{repo}`) is resolved against the new instance to produce default branch names, which are written into the instance manifest as concrete strings. The pattern is a seed, not persistent state — after creation, branches are managed by editing the manifest directly or via dedicated commands.

**Materialize semantics.** Materialize is a separate phase from create. It ensures each member's canonical clone exists (auto-cloning if absent), then creates a git worktree for each member at the configured branch, creating the branch from the configured base ref if it does not yet exist. If a branch is already checked out elsewhere, materialize fails loudly with a clear pointer to the conflict — it never silently renames or detaches.

**Lifecycle phases.** Workspaces follow a `create → materialize → archive → delete` lifecycle. Create produces the instance directory and sync state. Materialize adds worktrees. Archive removes worktrees but preserves the instance directory and its synced configuration. Delete removes everything. Re-materializing an archived workspace is supported and idempotent.

**Hooks: typed phases, host-side, shell scripts by file convention.** Each lifecycle phase has a corresponding optional script discovered by filename in the definition's `hooks/` directory. Presence triggers execution at the matching phase; absence is a no-op. Hooks run on the host with cwd set to the instance directory and with a documented environment (`Z_INSTANCE_PATH`, `Z_INSTANCE_NAME`, `Z_DEFINITION_PATH`). Container-internal lifecycle stays the runtime's responsibility (e.g., devcontainer's `postCreateCommand`).

**Runtime contract: four shell verbs.** `z` does not own container or VM launching directly. Each definition ships a `runtime/` directory containing four executables — `up`, `exec`, `down`, `status` — invoked by `z` with the documented environment. The runtime is fully responsible for image selection, mount setup, networking, and process lifecycle inside the environment. `z` provides starter templates for common runtimes (devcontainer, plain docker, docker compose, podman, Apple Container) so users can scaffold a definition with a sensible default and edit from there.

**Agent configuration layering.** Three layers, all already supported by Claude Code's hierarchical config resolution: global (`~/.claude/`), workspace (`<instance>/CLAUDE.md`, `<instance>/.claude/`), and repo (`<instance>/<repo>/CLAUDE.md`, `<instance>/<repo>/.claude/`). Global is for ubiquitous skills; workspace is for "this kind of work" skills and MCP servers; repo is for repo-specific instructions. The workspace dir is the agent's cwd inside the runtime, so the existing upward-walk merge picks all three up naturally.

**Discovery model.** No registry. Instances are discovered by scanning the workspaces root for directories containing an instance manifest marker. Definitions are discovered by scanning the definitions root the same way. Names are dirnames. Renaming is `mv`.

**No source changes expected in the project layer.** Workspaces consume existing project APIs (resolving a remote ID to a canonical clone, cloning if absent). If a small read-only accessor is required to expose canonical paths cleanly, it can be added; the project layer's external behavior is otherwise unchanged.

## Testing Decisions

Good tests for this feature exercise externally observable behavior — the state of the filesystem, the manifest contents, the worktree configuration on disk, the exit codes and output of `z` commands — and avoid coupling to internal package structure.

The components most worth isolated testing are the pure or near-pure ones:

- The manifest layer: parsing, validation, default application, and round-trip read/write are pure data and produce high-confidence tests with low setup cost.
- The sync layer: hash tracking, the "modified-locally → skip" rule, and the "clean → overwrite" rule can be tested by setting up a small temp-dir fixture with a definition tree and an instance tree, running sync, and asserting which files moved and which were preserved.
- The branch resolver: pattern substitution against `{instance}` / `{repo}` placeholders and override-flag merging is pure string work and trivially testable.
- The discovery layer: scanning a temp-dir layout for instances and definitions is a small integration test against the filesystem.

The worktree layer is best covered by integration tests against a real temporary git repository, because the cases worth catching (branch already checked out elsewhere, missing canonical, missing base ref) are inherently git-stateful and a fake would replicate too much git behavior to be trustworthy.

The orchestration that composes the above into create / materialize / archive / delete flows, and the runtime and hook shell-out layers, are best covered by end-to-end tests at the CLI surface — invoking `z workspace ...` against a tempdir and asserting filesystem and exit-code outcomes — rather than by unit tests of internal interfaces. The existing tests in `pkg/project` (notably `list_test.go`) provide a precedent for the shape of these tests in this codebase.

## Out of Scope

- A stacking tool, or any awareness inside `z` of stack relationships between branches. Workspaces track the *currently checked-out* branch per member; the relationship between branches in a stack lives in whatever external stacking tool the user chooses.
- Multiple worktrees of the same repo inside a single workspace instance. A workspace member is one repo, one branch, one worktree. Users wanting parallel side-by-side checkouts of the same repo within one workspace fall back to plain `git worktree add`.
- A `z`-native diff or merge tool for syncing user-modified instance files against definition updates. The sync command skips modified files with a warning; reconciliation is left to the user with their editor or merge tool.
- Coordinated PR creation, review, or merge across member repos. Workspaces produce coordinated *branches*; opening PRs and shepherding them remains a separate concern (potentially a future feature, but not part of this PRD).
- Sharing workspace definitions across multiple users via any `z`-native mechanism. The definitions directory is a plain directory; users may version-control or syndicate it externally, but `z` is agnostic.
- A registry, name service, or central index of workspaces or definitions. Discovery is filesystem-only.
- Replacing or deprecating the existing `z project` surface. All current `z project` commands continue to work unchanged; workspaces are purely additive.
- An in-manifest declarative runtime spec (image, mounts, env). The runtime contract is shell-only; a higher-level DSL may be added later as a generator if patterns emerge.

## Further Notes

The design deliberately stays small at the seams: the runtime contract is four shell verbs, the hook surface is typed phase scripts, the configuration layering reuses Claude Code's existing upward-merge resolution rather than introducing a new precedence rule, and discovery is "the filesystem." Each of these is a place where the design resisted introducing a richer abstraction in favor of letting an existing primitive carry the weight.

The definition / instance split is the most consequential architectural decision; it is what makes parallel agentic attempts feasible without copy-paste, and it is what cleanly separates "the kind of work this workspace is for" from "this particular attempt at that work." Many downstream decisions (sync rather than symlink, soft propagation by default with override-as-escape-hatch, runtime scripts in the definition rather than the instance) follow from taking that split seriously.

The interaction with stacked-PR tooling is the area most likely to need revisiting after first use. The current design treats stack-awareness as out of scope and assumes the workspace only tracks the active branch per member, with stack navigation handled externally. If that proves clumsy in practice, a follow-up could add awareness of "a member is on a stack" to the manifest without changing any other part of the design.
