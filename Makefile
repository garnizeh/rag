# Variables
BINARY_NAME=rag
MAIN_PATH=./cmd/server
DEV_CLIENT_PATH=./cmd/dev/ollama-client
GO_VERSION := $(shell go version)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS = -ldflags "-X main.version=${GIT_HASH} -X main.buildTime=${BUILD_TIME}"

.PHONY: help build run test clean fmt lint install-deps dev-client deps-update mod-tidy

# Default target
all: help

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the main application
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: ## Run the main application
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_PATH)

dev-client: ## Run the development Ollama client
	@echo "Running development Ollama client..."
	go run $(DEV_CLIENT_PATH)

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

lint: install-lint ## Run linter
	@echo "Running linter..."
	golangci-lint run

install-lint: ## Install golangci-lint if not present
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

install-deps: ## Install project dependencies
	@echo "Installing dependencies..."
	go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	go mod tidy

deps-vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	go mod vendor

# Development targets
dev-setup: install-deps mod-tidy ## Set up development environment
	@echo "Development environment setup complete!"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(BINARY_NAME):latest

# Database/migration targets (for future use)
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	# migrate -path migrations -database "postgres://..." up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	# migrate -path migrations -database "postgres://..." down

# Info targets
info: ## Show project information
	@echo "Project: $(BINARY_NAME)"
	@echo "Go version: $(GO_VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git hash: $(GIT_HASH)"

# Quick development workflow
quick: fmt test build ## Format, test, and build
