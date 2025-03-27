MAIN_PACKAGE_PATH := ./cmd/z
BINARY_NAME       := ./bin/z

# ============================================================================
# HELPERS
# ============================================================================

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ============================================================================
# QUALITY CONTROL
# ============================================================================

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## lint: lint the code
.PHONY: lint
lint:
	golangci-lint run

## lint/fix: lint the code, auto-fix if possible
.PHONY: lint/fix
lint/fix:
	golangci-lint run --fix

# ============================================================================
# DEVELOPMENT
# ============================================================================

## test: run all tests
.PHONY: test
test:
	go test -v -race ./...

## build: build the application
.PHONY: build
build:
	CGO_ENABLED=0 go build -o=${BINARY_NAME} ${MAIN_PACKAGE_PATH}

# ============================================================================
# CI
#
# These targets are used to run the tests and build the application in the CI.
# ============================================================================

## ci-test: run tests with coverage for CI
.PHONY: ci-test
ci-test:
	./hack/ci/test.sh

## ci-build: verify build for CI
.PHONY: ci-build
ci-build:
	./hack/ci/build.sh
