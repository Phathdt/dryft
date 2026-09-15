.PHONY: help build test test-short test-integration clean install fmt vet lint run

# Default target
help:
	@echo "dryft - Makefile commands:"
	@echo ""
	@echo "  make build            Build the dryft binary"
	@echo "  make install          Install dryft to GOPATH/bin"
	@echo "  make test             Run all tests"
	@echo "  make test-short       Run tests in short mode (skip integration)"
	@echo "  make test-integration Run integration tests only"
	@echo "  make fmt              Format code with gofmt"
	@echo "  make vet              Run go vet"
	@echo "  make lint             Run golangci-lint (if installed)"
	@echo "  make clean            Remove build artifacts"
	@echo "  make run              Build and run dryft"
	@echo ""

# Build the binary
build:
	@echo "Building dryft..."
	@mkdir -p bin
	@go build -o bin/dryft ./cmd/dryft
	@echo "✓ Built: ./bin/dryft"

# Install to GOPATH/bin
install:
	@echo "Installing dryft..."
	@go install ./cmd/dryft
	@echo "✓ Installed to $(shell go env GOPATH)/bin/dryft"

# Run all tests
test:
	@echo "Running all tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"
	@echo ""
	@echo "Coverage:"
	@go tool cover -func=coverage.out | tail -1

# Run tests in short mode (skip integration tests)
test-short:
	@echo "Running tests (short mode)..."
	@go test -v -short ./...
	@echo "✓ Tests complete (integration tests skipped)"

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@go test -v -run Integration ./...
	@echo "✓ Integration tests complete"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w -s $(shell find . -name "*.go" -not -path "./vendor/*")
	@echo "✓ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet checks passed"

# Run linter (requires golangci-lint)
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✓ Lint checks passed"; \
	else \
		echo "⚠ golangci-lint not installed. Skipping."; \
		echo "  Install: https://golangci-lint.run/usage/install/"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f dryft
	@rm -f coverage.out
	@rm -rf ./dryft/
	@echo "✓ Clean complete"

# Build and run
run: build
	@echo ""
	@./bin/dryft

# Development workflow
dev: fmt vet test-short build
	@echo ""
	@echo "✓ Development checks complete"

# CI workflow
ci: fmt vet test
	@echo ""
	@echo "✓ CI checks complete"
