# ccAgents Makefile

.PHONY: build test lint fmt clean install dev-deps help build-all build-linux build-darwin build-windows archives checksums release-prep ci-lint ci-test ci-test-unit ci-test-integration ci-test-fast ci-build ci-release

# Build variables
BINARY_NAME=ccagents
MAIN_PACKAGE=./cmd/ccagents
BUILD_DIR=./bin
DIST_DIR=./dist
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -s -w"

# Test parallelization - dynamically detect CPU cores
CPU_CORES=$(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "4")
TEST_PARALLEL_FLAGS=-parallel $(CPU_CORES) -p $(CPU_CORES)

# Test timeout configuration
GO_TEST_TIMEOUT?=5m

# Platform variables
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
ARCHIVE_FORMATS=tar.gz zip

# Default target
help: ## Show this help message
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development dependencies
dev-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install mvdan.cc/gofumpt@latest

# Build targets
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

build-dev: ## Build with debug symbols
	@echo "Building $(BINARY_NAME) (development)..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="-N -l" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

install: ## Install the application
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PACKAGE)

# Code quality targets
fmt: ## Format Go code
	@echo "Formatting Go code..."
	gofumpt -w .
	goimports -w -local github.com/fumiya-kume/cca .

lint: ## Run linters
	@echo "Running linters..."
	golangci-lint run

# Testing targets
test: ## Run tests
	@echo "Running tests with $(CPU_CORES) parallel workers..."
	go test -race $(TEST_PARALLEL_FLAGS) -timeout=$(GO_TEST_TIMEOUT) -coverprofile=coverage.out ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose) with $(CPU_CORES) parallel workers..."
	go test -race -v $(TEST_PARALLEL_FLAGS) -timeout=$(GO_TEST_TIMEOUT) -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	@echo "Test coverage:"
	go tool cover -func=coverage.out

test-coverage-html: test ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Utility targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html
	go clean

mod-tidy: ## Tidy Go modules
	@echo "Tidying Go modules..."
	go mod tidy

# Development workflow
dev: fmt lint test ## Run full development workflow (format, lint, test)

# Cross-platform build targets
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for $$GOOS/$$GOARCH..."; \
		if [ "$$GOOS" = "windows" ]; then \
			GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH.exe $(MAIN_PACKAGE); \
		else \
			GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH $(MAIN_PACKAGE); \
		fi \
	done

build-linux: ## Build for Linux (amd64 and arm64)
	@echo "Building for Linux..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)

build-darwin: ## Build for macOS (amd64 and arm64)
	@echo "Building for macOS..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)

build-windows: ## Build for Windows (amd64 and arm64)
	@echo "Building for Windows..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)

# Archive creation
archives: build-all ## Create release archives
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		if [[ "$$binary" == *".exe" ]]; then \
			base=$$(basename $$binary .exe); \
			zip $$base.zip $$binary; \
		else \
			tar -czf $$binary.tar.gz $$binary; \
		fi \
	done
	@echo "Archives created in $(DIST_DIR)/"

# Checksums
checksums: archives ## Generate checksums for releases
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && \
	sha256sum *.tar.gz *.zip > checksums.txt
	@echo "Checksums generated in $(DIST_DIR)/checksums.txt"

# Release preparation
release-prep: clean build-all archives checksums ## Prepare complete release
	@echo "Release preparation complete!"
	@echo "Files in $(DIST_DIR):"
	@ls -la $(DIST_DIR)/

# CI targets (for GitHub Actions)
ci-lint: ## Run linting for CI
	golangci-lint run --timeout 3m

ci-test: ## Run tests for CI (combined, slower but comprehensive)
	go test -race $(TEST_PARALLEL_FLAGS) -timeout=$(GO_TEST_TIMEOUT) -coverprofile=coverage.out -covermode=atomic ./...

ci-test-unit: ## Run unit tests for CI (fast tests only)
	@echo "Running unit tests with optimized settings..."
	go test $(TEST_PARALLEL_FLAGS) -timeout=2m -short -coverprofile=coverage.out -covermode=atomic \
		-run="^Test[^I].*" ./pkg/... ./internal/... ./cmd/...

ci-test-integration: ## Run integration tests for CI
	@echo "Running integration tests..."
	go test -race $(TEST_PARALLEL_FLAGS) -timeout=3m -coverprofile=coverage-integration.out -covermode=atomic \
		-run="TestIntegration|TestE2E|Test.*Integration" ./... || \
	go test -race $(TEST_PARALLEL_FLAGS) -timeout=3m -coverprofile=coverage-integration.out -covermode=atomic \
		./test/...

ci-test-fast: ## Run fast tests for CI (unit tests without race detector)
	@echo "Running fast unit tests..."
	go test $(TEST_PARALLEL_FLAGS) -timeout=90s -short -count=1 ./pkg/... ./internal/... ./cmd/...

ci-build: ## Build for CI
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

ci-release: ## Build release artifacts for CI
	$(MAKE) release-prep