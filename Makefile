.PHONY: help setup all test test-verbose test-coverage test-coverage-html bench lint vet fmt fmt-check tidy download verify clean ci dev-setup watch-test examples security mod-graph updates docs workspace-check

# Default target
.DEFAULT_GOAL := help

# Variables
GO := go
GOFLAGS := -v
TIMEOUT := 5m
COVERAGE := coverage.out

# Help: display all available targets
help: ## Display this help message
	@echo "Traverse - OData Client Development Tasks"
	@echo ""
	@echo "Setup & Development:"
	@echo "  make setup         Install development tools and setup git hooks"
	@echo "  make dev-setup     Install development dependencies only"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt           Format all Go files (gofmt + goimports)"
	@echo "  make fmt-check     Check if code is properly formatted"
	@echo "  make lint          Run golangci-lint static analysis"
	@echo "  make vet           Run go vet"
	@echo "  make security      Run security checks with gosec"
	@echo ""
	@echo "Testing:"
	@echo "  make test          Run all tests with race detector"
	@echo "  make test-verbose  Run tests with verbose output"
	@echo "  make test-coverage Generate coverage report (terminal)"
	@echo "  make test-coverage-html Generate coverage report (HTML)"
	@echo ""
	@echo "Benchmarking:"
	@echo "  make bench         Run benchmarks"
	@echo "  make bench-save    Run benchmarks and save results"
	@echo ""
	@echo "Maintenance:"
	@echo "  make tidy          Tidy go.mod files"
	@echo "  make download      Download dependencies"
	@echo "  make verify        Verify dependencies"
	@echo "  make workspace-check Verify workspace setup (go.work)"
	@echo "  make mod-graph     Display module dependency graph"
	@echo "  make updates       Check for dependency updates"
	@echo "  make clean         Clean build artifacts and cache"
	@echo ""
	@echo "Development:"
	@echo "  make watch-test    Watch for changes and run tests"
	@echo "  make examples      Build all examples"
	@echo "  make docs          Generate API documentation"
	@echo ""
	@echo "Workflows:"
	@echo "  make all           Run: clean fmt lint vet test bench"
	@echo "  make ci            Run: lint vet test (CI workflow)"
	@echo ""

# Setup: install development tools and setup git hooks
setup: ## Setup development environment with tools and git hooks
	@echo "Installing development tools..."
	go install github.com/evilmartians/lefthook@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Setting up git hooks..."
	lefthook install
	@echo "✓ Development environment ready"

# Code formatting
fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .
	@echo "✓ Code formatted"

fmt-check: ## Check if code is formatted correctly
	@echo "Checking code format..."
	@if [ -n "$$(gofmt -l .)" ]; then echo "✗ Code not formatted with gofmt"; exit 1; fi
	@if [ -n "$$(goimports -l .)" ]; then echo "✗ Code not formatted with goimports"; exit 1; fi
	@echo "✓ Code format is correct"

# Linting and analysis
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run --timeout=$(TIMEOUT) ./...
	@echo "✓ Linting complete"

vet: ## Run go vet
	@echo "Running go vet (excluding cmd)..."
	$(GO) vet ./internal/... ./ext/... ./testutil/...
	@echo "✓ Vet complete"

security: ## Run security checks with gosec
	@echo "Running security checks..."
	gosec ./...
	@echo "✓ Security check complete"

# Testing
test: ## Run all tests with race detector
	@echo "Running all tests including CLI tools..."
	$(GO) test -race -count=1 -timeout=$(TIMEOUT) . ./internal/... ./ext/... ./testutil/... ./cmd/...
	@echo "✓ Tests passed"

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	$(GO) test -v -race -count=1 -timeout=$(TIMEOUT) ./internal/... ./ext/... ./testutil/... ./cmd/...
	@echo "✓ Tests passed"

test-coverage: ## Generate coverage report (terminal)
	@echo "Running tests with coverage..."
	$(GO) test -race -count=1 -coverprofile=$(COVERAGE) -covermode=atomic -timeout=$(TIMEOUT) ./internal/... ./ext/... ./testutil/... ./cmd/...
	@echo ""
	@echo "Coverage Summary:"
	@$(GO) tool cover -func=$(COVERAGE) | tail -1
	@echo "✓ Coverage report available: $(COVERAGE)"

test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=$(COVERAGE) -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem -benchtime=3s -run=^$$ ./internal/... ./ext/... ./testutil/... 2>/dev/null || $(GO) test -bench=. -benchmem -benchtime=3s -run=^$$ ./...
	@echo "✓ Benchmarks complete"

bench-save: ## Run benchmarks and save results
	@echo "Running benchmarks and saving results..."
	@mkdir -p benchmarks
	$(GO) test -bench=. -benchmem -benchtime=3s -run=^$$ ./internal/... ./ext/... ./testutil/... 2>/dev/null || $(GO) test -bench=. -benchmem -benchtime=3s -run=^$$ ./... | tee benchmarks/benchmark_results.txt
	@echo "✓ Benchmark results saved to benchmarks/benchmark_results.txt"

# Dependency management
tidy: ## Tidy go.mod file
	@echo "Tidying go.mod..."
	$(GO) mod tidy
	@echo "✓ go.mod tidied"

download: ## Download all dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "✓ Dependencies downloaded"

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GO) mod verify
	@echo "✓ Dependencies verified"

mod-graph: ## Display module dependency graph
	@echo "Module dependency graph:"
	$(GO) mod graph

updates: ## Check for available dependency updates
	@echo "Checking for dependency updates..."
	go list -u -m all

# Build and examples
build: ## Build all packages
	@echo "Building..."
	$(GO) build $(GOFLAGS) ./...
	@echo "✓ Build complete"

examples: ## Build all examples
	@echo "Building examples..."
	$(GO) build ./examples/...
	@echo "✓ Examples built"

# Cleanup
clean: ## Clean build artifacts and test cache
	@echo "Cleaning up..."
	rm -f $(COVERAGE) coverage.html
	$(GO) clean -testcache
	$(GO) clean -cache
	@echo "✓ Cleanup complete"

# Documentation
docs: ## Generate API documentation
	@echo "Generating documentation..."
	@mkdir -p docs
	$(GO) doc ./... > docs/api.txt
	@echo "✓ Documentation generated: docs/api.txt"

# Development helpers
dev-setup: ## Install development tools only (without git hooks)
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "✓ Development tools installed"

watch-test: ## Watch for changes and run tests (TDD workflow)
	@echo "Starting test watcher (Ctrl+C to exit)..."
	@while true; do \
		clear; \
		echo "Running tests..."; \
		$(GO) test -race -count=1 -timeout=$(TIMEOUT) ./internal/... ./ext/... ./testutil/... || true; \
		echo ""; \
		echo "Waiting for changes (checking every 2s)..."; \
		sleep 2; \
	done

# Compound workflows
all: clean fmt lint vet test bench ## Run all checks: clean, fmt, lint, vet, test, bench

ci: lint vet test ## Run CI checks: lint, vet, test (excluding cmd)

# Workspace verification
workspace-check: ## Verify go.work workspace setup
	@bash scripts/verify-workspace.sh
