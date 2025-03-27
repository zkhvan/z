#!/usr/bin/env bash

set -o errexit  # exit on any error
set -o nounset  # treat unset variables as errors
set -o pipefail # fail a pipeline if any command fails
set -o xtrace   # print each command after expansion

# Build the binary
CGO_ENABLED=0 go build -o=./bin/z ./cmd/z

# Verify go.mod is tidy
go mod tidy
git diff --exit-code -- go.mod go.sum || {
  echo "go.mod/go.sum not tidy - run 'go mod tidy'"
  exit 1
}
