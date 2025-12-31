# Default shell for make
SHELL := /bin/bash

BIN_PATH ?= github-review-helper

GO ?= go
GOFLAGS ?= -v
# LDFLAGS: -s removes symbol table, -w removes DWARF debug info, resulting in smaller binary
LDFLAGS ?= -ldflags "-s -w"

.DEFAULT_GOAL := all
all: test build

# Help target - shows available targets
.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build: Build the binary
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_PATH) .

.PHONY: test
test: ## test: Run tests
	$(GO) test $(GOFLAGS) ./...

.PHONY: clean
clean: ## clean: Remove build artifacts
	rm -f $(BIN_PATH)

.PHONY: fmt
fmt: ## fmt: Format Go code
	$(GO) fmt ./...

.PHONY: vet
vet: ## vet: Run go vet
	$(GO) vet ./...

.PHONY: check
check: fmt vet test ## check: Run fmt, vet, and test
