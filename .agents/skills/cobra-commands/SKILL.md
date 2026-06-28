---
name: cobra-commands
description: Conventions for adding or editing cobra commands in this repo (pkg/cmd/**). Use when creating a new command/subcommand, wiring a command group, or touching anything under pkg/cmd. Covers the Options + NewCmd + Complete/Run structure, factory-provided dependencies, registration, and testability seams.
---

# Working with Cobra Commands

Applies to `pkg/cmd/**/*.go`. Follow this structure exactly so new commands
mirror the existing ones (`pkg/cmd/project`, `pkg/cmd/tmux`).

## Command structure

Every command is three parts: an `Options` struct, a `NewCmdXXX` constructor,
and `Complete`/`Run` methods.

```go
type Options struct {
    io     *iolib.IOStreams // from factory, NOT an ad-hoc interface
    config cmdutil.Config   // from factory

    // command-specific flags/args
}

func NewCmdXXX(f *cmdutil.Factory) *cobra.Command {
    opts := &Options{
        io:     f.IOStreams,
        config: f.Config,
    }

    cmd := &cobra.Command{
        Use:   "xxx",
        Short: "Short description",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := opts.Complete(cmd, args); err != nil {
                return err
            }
            return opts.Run(cmd.Context())
        },
    }

    // cmd.Flags().StringVar(&opts.Foo, "foo", "", "...")
    return cmd
}

// Complete finalizes opts from cmd/args (positional args, flag normalization).
// Omit it only when there is nothing to complete.
func (opts *Options) Complete(cmd *cobra.Command, args []string) error {
    opts.Name = args[0]
    return nil
}

// Run holds the command logic.
func (opts *Options) Run(ctx context.Context) error {
    // ...
}
```

## Rules

- Use `RunE`, never `Run`. Propagate errors up the chain.
- Always thread `context.Context` from `cmd.Context()` into `Run`.
- Get dependencies from `cmdutil.Factory` — `f.IOStreams`, `f.Config`. Do not
  construct them or invent ad-hoc interfaces.
- Split arg/flag handling into `Complete`; keep `Run` focused on logic.
- Write to `opts.io.Out` / `opts.io.ErrOut`, never `os.Stdout` directly.

## Factory

Commands receive a `*cmdutil.Factory` providing common dependencies. Pull what
you need from it in `NewCmdXXX`:

```go
opts := &Options{
    io:     f.IOStreams,
    config: f.Config,
}
```

## Registration

Parent command groups register subcommands with `AddCommand`:

```go
func NewCmdSession(f *cmdutil.Factory) *cobra.Command {
    cmd := &cobra.Command{Use: "session", Short: "Manage tmux sessions"}
    cmd.AddCommand(listCmd.NewCmdList(f))
    cmd.AddCommand(killCmd.NewCmdKill(f))
    return cmd
}
```

A new top-level group must also be added in `pkg/cmd/root.go`.

## Organization

One package per command; subcommands live in their own subdirectories.

```
pkg/cmd/
├── tmux/
│   ├── session/
│   │   ├── list/
│   │   ├── kill/
│   │   ├── new/
│   │   └── use/
│   └── tmux.go
```

## Testability seam

When a command drives a domain service that shells out, inject the executor at
the `Options` level so command tests can drive a `FakeExec` through the whole
stack:

```go
type Options struct {
    io       *iolib.IOStreams
    config   cmdutil.Config
    executor exec.Interface
}

func NewCmdXXX(f *cmdutil.Factory) *cobra.Command {
    opts := &Options{io: f.IOStreams, config: f.Config, executor: exec.New()}
    // ...
}

func (opts *Options) Run(ctx context.Context) error {
    svc, err := domain.NewService(opts.config, domain.WithExecutor(opts.executor))
    // ...
}
```

Command tests are black-box (`package xxx_test`) and drive the real command via
`NewCmdXXX(f)` + `Execute()` using a test factory. See the `golang-harness-testing`
skill and `pkg/cmd/workspace/create/`. Because black-box tests reach only the
exported surface, any fake (e.g. `FakeExec`) must be injectable through the
factory or `Options` — that is the point of this seam.

## Checklist

```
[ ] Options struct with io/config from the factory
[ ] NewCmdXXX(f *cmdutil.Factory) constructor
[ ] RunE calls Complete (if needed) then Run(cmd.Context())
[ ] context.Context threaded into Run
[ ] output via opts.io, not os.Stdout
[ ] registered with AddCommand (and root.go for a new group)
[ ] executor injected at Options level if the command shells out
```
