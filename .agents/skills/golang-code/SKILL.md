---
name: golang-code
description: Conventions for working with Golang code in this repo. Use when writing or editing any Go (.go) file. Covers comment style (need-to-know only).
---

# Working with Golang Code

## After editing

After making changes, verify things with the following make targets:

```
make tidy-go lint-go test build
```

## Comments

Keep comments need-to-know. A comment should explain *why*, a non-obvious
invariant, or a gotcha — never restate what the code or signature already says.

- Don't narrate obvious steps (`// write to a temp file then rename`).
- Don't add doc comments that just repeat the function/field name.
- Do keep comments that capture design invariants (e.g. "name and status are
  never stored; derived from disk") or non-obvious semantics (e.g. "empty
  BaseRef resolves the default branch at materialize time").
