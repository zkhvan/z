# Zhenya's CLI - z

[![CI](https://github.com/zkhvan/z/actions/workflows/ci.yaml/badge.svg)](https://github.com/zkhvan/z/actions/workflows/ci.yaml)
[![License: MIT](https://img.shields.io/github/license/zkhvan/z)](https://github.com/zkhvan/z/blob/main/LICENSE)

A CLI designed for productivity, written in Go.

> [!NOTE]
> This project is only compatible with MacOS and zsh.

## Projects

### Getting started

`z project` works based off of [projects](#whats-a-project). To see all the
projects discovered by `z`:

```console
$ z project list
[S] my-personal-org/repo1
[L] my-personal-org/repo2
[R] oss/cli/cli
```

The example output above shows that:

- `[L]` means the project is a local repository (exists only locally).
- `[S]` means the project is a synced repository (exists locally and remotely).
- `[R]` means the project is a remote repository (exists only remotely).

This example is based on the example configuration in [Configuration](#configuration) and the commented out `remote_patterns` in the configuration file.

Once you have your projects setup, the `z project select` command lets you
quickly navigate to projects (leveraging `fzf` under the hood).

```console
$ # Add zsh integration for automatic `cd` integration
$ source $(z shell zsh)
$ # Now you can use `z project select` to navigate to a project
$ z project select
```

### What's a project?

A project basically a Git repository. It maps a GitHub repository to a local
directory, leveraging the owner/repo format to identify the remote repository
and a local directory path.

By default, projects live in the `~/Projects` directory. In order to
categorize a project, specify an alternate path in the configuration file. For
example:

```yaml
projects:
  remotePatterns:
    - my-personal-org/* -> ./personal
    - my-work-org/* -> ./work
```

In this case, all the repositories from `my-personal-org` will be mapped to
`~/Projects/personal` and all the repositories from `my-work-org` will be
mapped to `~/Projects/work`.

## Configuration

The configuration file is located at `~/.config/z/config.yaml` (or `$XDG_CONFIG_HOME/z/config.yaml` if `$XDG_CONFIG_HOME` is set; on macOS without XDG, the path is `~/Library/Application Support/z/config.yaml`).

The default configuration looks like this:

```yaml
projects:
  # The root directory for projects
  root: ~/Projects
  # The maximum depth to search for local Git repositories
  max_depth: 3
  # Cache remote projects for 1 day
  ttl: 86400
  # The remote repository patterns to search and cache (GitHub only, for now)
  # remote_patterns:
  #   - my-personal-org/*
  #   - cli/cli -> ./oss/
```
