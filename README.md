# Zhenya's CLI - z

[![CI](https://github.com/zkhvan/z/actions/workflows/ci.yaml/badge.svg)](https://github.com/zkhvan/z/actions/workflows/ci.yaml)
[![License: MIT](https://img.shields.io/github/license/zkhvan/z)](https://github.com/zkhvan/z/blob/main/LICENSE)

A CLI designed for productivity, written in Go.

## Compatibility

This project is only compatible with MacOS and zsh.

## Projects

A project is a structured representation of a Git repository. It maps a remote
GitHub repository to a local directory, leveraging the owner/repo format to
identify the remote repository and a local directory path.

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
