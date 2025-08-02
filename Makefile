# Makefile for Key-Value Store Concurrent
# Professional Go project Makefile

.PHONY: fill build test run lint ci format clean help deps

# Variables
BINARY_NAME=kvstore
MODULE_NAME=kvstore
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Key-Value Store Concurrent - Makefile Commands"
	@echo "============================================="
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

fill: deps ## Install dependencies and initialize project
	@echo "Installing Go dependencies..."
	@go mod tidy
	@go mod download
	@echo "Dependencies installed successfully"

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies verified"

build: ## Build the application
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kvstore
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run tests with coverage
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total:
	@echo "Tests completed"

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Verbose tests completed"

run: build ## Run the application
	@echo "Starting $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

lint: ## Run golangci-lint (install if needed)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...
	@echo "Linting completed"

format: ## Format code with gofmt
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)
	@echo "Code formatting completed"

ci: format lint test ## Run CI pipeline (format + lint + test)
	@echo "CI pipeline completed successfully"

clean: ## Clean build artifacts and caches
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)/
	@rm -f coverage.out
	@go clean -cache -testcache -modcache
	@echo "Clean completed"

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@go install $(LDFLAGS) ./cmd/kvstore
	@echo "Installation completed"

check: ## Check for common issues
	@echo "Running health checks..."
	@go mod verify
	@go vet ./...
	@echo "Health checks completed"

dev: ## Development mode with file watching (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...
	@echo "Benchmarks completed"