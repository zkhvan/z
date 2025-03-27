# Manage development tools using a local bin directory.

# ==========================================================================
# INITIALIZATION
# ==========================================================================

HACK_DIR       ?= $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MOD_FILE := $(HACK_DIR)/tools/go.mod
BIN_DIR        ?= $(HACK_DIR)/bin

# Detect OS and architecture
OS   := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# ==========================================================================
# TOOL VERSIONS
# ==========================================================================

GOLANGCI_LINT_VERSION ?= $(shell grep github.com/golangci/golangci-lint $(TOOLS_MOD_FILE) | awk '{print $$NF}')

# ==========================================================================
# TOOL TARGETS
# ==========================================================================

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint-$(OS)-$(ARCH)-$(GOLANGCI_LINT_VERSION)

$(GOLANGCI_LINT):
	$(call install-golangci-lint,$@,$(GOLANGCI_LINT_VERSION))

# ==========================================================================
# SYMLINK TARGETS
# ==========================================================================

GOLANGCI_LINT_LINK := $(BIN_DIR)/golangci-lint

.PHONY: $(GOLANGCI_LINT_LINK)
$(GOLANGCI_LINT_LINK): $(GOLANGCI_LINT)
	$(call create-symlink,$(GOLANGCI_LINT),$(GOLANGCI_LINT_LINK))

# ==========================================================================
# ALIAS TARGETS
# ==========================================================================

TOOLS := install-golangci-lint

.PHONY: install-tools
install-tools: $(TOOLS)

.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) $(GOLANGCI_LINT_LINK)

# ==========================================================================
# CLEAN-UP TOOLS
# ==========================================================================

# Clean up all installed tools and symlinks
.PHONY: clean-tools
clean-tools:
	rm -rf $(BIN_DIR)/*
	rm -rf $(INCLUDE_DIR)/*

# Update all tools
.PHONY: update-tools
update-tools: clean-tools install-tools

# ==========================================================================
# HELPER FUNCTIONS
# ==========================================================================

# install-go-tool installs a Go tool.
#
# $(1) binary path
# $(2) repo URL
# $(3) version
define install-go-tool
	@$(HACK_DIR)/tools/install-go-tool "$(1)" "$(2)" "$(3)"
endef

# install-golangci-lint installs golangci-lint.
#
# $(1) binary path
# $(2) version
define install-golangci-lint
	@$(HACK_DIR)/tools/install-golangci-lint "$(1)" "$(2)"
endef

# create-symlink creates a relative symlink to the platform-specific binary.
#
# $(1) platform-specific binary path
# $(2) symlink path
define create-symlink
	@if [ ! -e $(2) ] || [ "$$(readlink $(2))" != "$$(basename $(1))" ]; then \
		echo "Creating symlink: $(2) -> $$(basename $(1))"; \
		ln -sf $$(basename $(1)) $(2); \
	fi
endef
