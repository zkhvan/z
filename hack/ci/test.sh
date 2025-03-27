#!/usr/bin/env bash

set -o errexit  # exit on any error
set -o nounset  # treat unset variables as errors
set -o pipefail # fail a pipeline if any command fails
set -o xtrace   # print each command after expansion

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -func=coverage.out
__total=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')

echo "Coverage summary: ${__total}"

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Store artifacts
mkdir -p artifacts
cp coverage.* artifacts/
