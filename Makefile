include $(CURDIR)/hack/tools.mk

MAIN_PACKAGE_PATH := ./cmd/z
BINARY_NAME       := ./bin/z
VERSION           ?= DEV
VERSION_PACKAGE   ?= github.com/zkhvan/z/internal/build

# ==========================================================================
# HELPERS
# ==========================================================================

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==========================================================================
# QUALITY CONTROL
# ==========================================================================

# --------------------------------------------------------------------------
# HELPERS
# --------------------------------------------------------------------------

## tidy: tidy the code
.PHONY: tidy
tidy: tidy-go

## tidy-go: format code and tidy modfile
.PHONY: tidy-go
tidy-go: format-go
	go mod tidy -v

## tidy-go-verify: verify go.mod is tidy
.PHONY: tidy-go-verify
tidy-go-verify: tidy-go
	git diff --exit-code -- go.mod go.sum || { \
	echo "go.mod/go.sum not tidy - run 'go mod tidy'"; \
	exit 1; \
}; \

# --------------------------------------------------------------------------
# LINTERS
# --------------------------------------------------------------------------

## lint: lint the code
.PHONY: lint
lint: lint-go

## lint-go: lint the go code
.PHONY: lint-go
lint-go: install-golangci-lint
	$(GOLANGCI_LINT) run

## lint-go-fix: lint the go code, auto-fix if possible
.PHONY: lint-go-fix
lint-go-fix:
	$(GOLANGCI_LINT) run --fix

# --------------------------------------------------------------------------
# FORMATTERS
# --------------------------------------------------------------------------

.PHONY: format-go
format-go:
	go fmt ./...

# ==========================================================================
# DEVELOPMENT
# ==========================================================================

## test: run all tests
.PHONY: test
test:
	go test \
		-v \
		-timeout=300s \
		-coverprofile=coverage.out \
		-covermode=atomic \
		-race \
		./...

## test-report: generate a test report
test-report: test
	go tool cover -func coverage.out
	go tool cover -html coverage.out -o coverage.html


## build: build the application
.PHONY: build
build:
	CGO_ENABLED=0 go build \
		-ldflags "-w -X $(VERSION_PACKAGE).Version=$(VERSION) -X $(VERSION_PACKAGE).Date=$$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-o=${BINARY_NAME} \
		${MAIN_PACKAGE_PATH}

# ==========================================================================
# CI
#
# These targets are used to run the tests and build the application in CI.
# ==========================================================================

## ci: run automated CI checks
.PHONY: ci
ci: tidy-go-verify test-report build

## ci-deps: install ci dependencies
.PHONY: ci-deps
ci-deps:
	sudo apt-get update
	sudo apt-get install -y fd-find

	# Optionally, create a symlink so you can call it with 'fd'
	mkdir -p hack/bin
	ln -s $$(which fdfind) hack/bin/fd || true
	echo $$(pwd)/hack/bin >> "${GITHUB_PATH}"
