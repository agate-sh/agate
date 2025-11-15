.PHONY: help build clean fmt lint lint-fix vet test install-tools list-tmux-sessions logs

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
	@echo "  list-tmux-sessions - List all tmux sessions grouped by server"
	@echo "  logs         - Tail debug logs at ~/.agate/debug.log"

# Build the binary
build:
	go build -o agate ./cmd/agate

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

# List all tmux sessions grouped by server
list-tmux-sessions:
	@echo "Tmux sessions by server:"
	@if [ -d "/tmp/tmux-$$(id -u)" ]; then \
		for socket in /tmp/tmux-$$(id -u)/*; do \
			if [ -S "$$socket" ]; then \
				server_name=$$(basename "$$socket"); \
				echo ""; \
				echo "Server: $$server_name"; \
				sessions=$$(tmux -L "$$server_name" list-sessions 2>/dev/null); \
				if [ -n "$$sessions" ]; then \
					echo "$$sessions" | sed 's/^/  /'; \
				else \
					echo "  (no sessions)"; \
				fi; \
			fi; \
		done; \
	else \
		echo "  No tmux servers found for current user"; \
	fi

# Tail debug logs
logs:
	tail -f ~/.agate/debug.log
