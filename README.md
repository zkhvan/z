# Zhenya's CLI - z

[![Tests](https://github.com/zkhvan/z/actions/workflows/test.yml/badge.svg)](https://github.com/zkhvan/z/actions/workflows/test.yml)

A CLI designed for productivity, written in Go.

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
