.PHONY: help build clean fmt lint lint-fix vet test install-tools

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the agate binary"
	@echo "  clean        - Remove build artifacts"
	@echo "  fmt          - Format Go code with gofmt"
	@echo "  lint         - Run golangci-lint"
	@echo "  lint-fix     - Run golangci-lint with auto-fix"
	@echo "  vet          - Run go vet"
	@echo "  test         - Run tests"
	@echo "  check        - Run fmt, vet, and lint"
	@echo "  fix          - Run fmt and lint-fix"
	@echo "  install-tools - Install required development tools"

# Build the binary
build:
	go build -o agate .

# Clean build artifacts
clean:
	rm -f agate agate-test

# Format code
fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

# Run linter
lint:
	golangci-lint run

# Run linter with auto-fix
lint-fix:
	golangci-lint run --fix

# Run go vet
vet:
	go vet ./...

# Run tests
test:
	go test ./...

# Run all checks (format, vet, lint)
check: fmt vet lint
	@echo "All checks completed successfully!"

# Auto-fix formatting and linting issues
fix: fmt lint-fix
	@echo "Auto-fix completed!"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; brew install golangci-lint; }
	@command -v goimports >/dev/null 2>&1 || { echo "Installing goimports..."; go install golang.org/x/tools/cmd/goimports@latest; }
	@echo "Development tools installed!"