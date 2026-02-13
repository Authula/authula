# Variables
APP_NAME=go-better-auth
BINARY_PATH=./tmp/$(APP_NAME)

.PHONY: help build build-exe run dev test clean install setup
.PHONY: test-coverage
.PHONY: lint fmt vet deps-update all check quick-check ci

# Help command
help: # Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Build commands`
build: # Build the package (library)
	@echo "Building $(APP_NAME) package..."
	@go build ./...
	@echo "Build complete!"

build-exe: # Build the binary executable
	@echo "Building $(APP_NAME) binary..."
	@mkdir -p ./tmp
	@rm -rf ./tmp/$(APP_NAME)
	@go build -o $(BINARY_PATH) ./cmd/main.go
	@echo "Binary built: $(BINARY_PATH)"

run: # Run the application
	@rm -f ./tmp/$(APP_NAME)
	@CGO_ENABLED=1 go run ./cmd/main.go

dev: # Run the application with live reloading using air
	@rm -f ./tmp/$(APP_NAME)
	@CGO_ENABLED=1 ./bin/air --build.cmd "go build -o ./tmp/$(APP_NAME) ./cmd/main.go" --build.entrypoint "./tmp/$(APP_NAME)"

# Test commands
test: # Run all tests
	@echo "Running tests..."
	@CGO_ENABLED=1 go test -v ./...

test-coverage: # Run tests with coverage report
	@echo "Running tests with coverage..."
	@CGO_ENABLED=1 go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Dependency management
install: # Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

deps-update: # Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Library mode development
library-test: test # Run library mode tests

# Development setup
setup: install # Setup development environment
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/cosmtrek/air@latest
	@echo "Development environment setup complete!"

# Clean commands
clean: # Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

# Code quality
lint: # Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: # Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: # Run go vet
	@echo "Running go vet..."
	@go vet ./...

# All-in-one commands
all: clean install build check # Clean, install deps, build, and run all checks

check: fmt vet lint test # Run all checks (format, vet, lint, test)

quick-check: fmt vet test # Run quick checks (format, vet, fast tests)

ci: clean install check # CI pipeline (clean, install, check)

# Integration testing
integration-test: docker-down docker-up docker-test # Run integration tests with Docker

# Default target
.DEFAULT_GOAL := help
