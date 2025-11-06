.PHONY: help build clean fmt lint lint-fix vet test install-tools list-tmux-sessions kill-agate-sessions clean-testing-worktrees rebuild tail-logs archive-logs

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
	@echo "  kill-agate-sessions - Kill all sessions in the agate tmux server"
	@echo "  clean-testing-worktrees - Delete all git worktrees with branch names starting with 'testing'"
	@echo "  rebuild - Clean up sessions, worktrees, build artifacts, and rebuild"
	@echo "  tail-logs - Tail the agate.log file"
	@echo "  archive-logs - Archive log file with timestamp to logs/ directory"

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

# Kill all sessions in the agate tmux server
kill-agate-sessions:
	@echo "Killing all sessions in agate tmux server..."
	@if tmux -L agate list-sessions >/dev/null 2>&1; then \
		sessions=$$(tmux -L agate list-sessions -F "#{session_name}" 2>/dev/null); \
		if [ -n "$$sessions" ]; then \
			echo "Found sessions: $$sessions"; \
			tmux -L agate kill-server; \
			echo "All agate sessions killed."; \
		else \
			echo "No sessions found in agate server."; \
		fi; \
	else \
		echo "Agate tmux server not running or no sessions exist."; \
	fi

# Delete all git worktrees with branch names starting with 'testing'
clean-testing-worktrees:
	@echo "Searching for git worktrees with branch names starting with 'testing'..."
	@worktree_base="$$HOME/.agate/worktrees"; \
	if [ ! -d "$$worktree_base" ]; then \
		echo "No agate worktrees directory found at $$worktree_base"; \
		exit 0; \
	fi; \
	removed_count=0; \
	for repo_dir in "$$worktree_base"/*; do \
		if [ ! -d "$$repo_dir" ]; then \
			continue; \
		fi; \
		repo_name=$$(basename "$$repo_dir"); \
		echo "Checking repository: $$repo_name"; \
		for branch_dir in "$$repo_dir"/*; do \
			if [ ! -d "$$branch_dir" ]; then \
				continue; \
			fi; \
			branch_name=$$(basename "$$branch_dir"); \
			if [ -z "$${branch_name##testing*}" ]; then \
				echo "  Found testing worktree: $$branch_name at $$branch_dir"; \
				if git -C "$$branch_dir" rev-parse --git-dir >/dev/null 2>&1; then \
					common_dir=$$(git -C "$$branch_dir" rev-parse --git-common-dir 2>/dev/null); \
					if [ -n "$$common_dir" ]; then \
						main_repo=$$(dirname "$$common_dir"); \
						if [ -d "$$main_repo/.git" ] || [ -f "$$main_repo/.git" ] || [ -d "$$main_repo" ]; then \
							echo "    Removing worktree from repo: $$main_repo"; \
							if git -C "$$main_repo" worktree remove -f "$$branch_dir" 2>/dev/null; then \
								echo "    ✓ Removed worktree: $$branch_name"; \
								removed_count=$$((removed_count + 1)); \
								git -C "$$main_repo" branch -D "$$branch_name" 2>/dev/null || true; \
							else \
								echo "    ✗ Failed to remove worktree via git (trying to remove directory directly)"; \
								if rm -rf "$$branch_dir" 2>/dev/null; then \
									echo "    ✓ Removed directory: $$branch_name"; \
									removed_count=$$((removed_count + 1)); \
									git -C "$$main_repo" branch -D "$$branch_name" 2>/dev/null || true; \
								fi; \
							fi; \
						else \
							echo "    ⚠ Could not find main repository at: $$main_repo"; \
						fi; \
					else \
						echo "    ⚠ Could not determine git common directory for worktree: $$branch_name"; \
					fi; \
				else \
					echo "    ⚠ Not a valid git worktree: $$branch_dir"; \
				fi; \
			fi; \
		done; \
	done; \
	if [ $$removed_count -eq 0 ]; then \
		echo "No testing worktrees found."; \
	else \
		echo "Removed $$removed_count testing worktree(s)."; \
	fi

# Rebuild: clean up sessions, worktrees, artifacts, and rebuild
rebuild: kill-agate-sessions
	@$(MAKE) clean-testing-worktrees
	@$(MAKE) clean
	@$(MAKE) build
	@echo "Rebuild completed successfully!"

# Tail the agate.log file
tail-logs:
	@log_file="agate.log"; \
	if [ ! -f "$$log_file" ]; then \
		echo "No log file found ($$log_file)"; \
		echo "Waiting for log file to be created..."; \
		while [ ! -f "$$log_file" ]; do \
			sleep 1; \
		done; \
	fi; \
	echo "Tailing agate.log (Press Ctrl+C to exit)..."; \
	tail -f "$$log_file"

# Archive log file with timestamp to logs/ directory
archive-logs:
	@log_file="agate.log"; \
	archive_dir="logs"; \
	timestamp=$$(date +"%Y%m%d_%H%M%S"); \
	\
	# Create logs directory if it doesn't exist \
	mkdir -p "$$archive_dir"; \
	\
	# Archive agate.log if it exists \
	if [ -f "$$log_file" ]; then \
		archived_name="agate_$$timestamp.log"; \
		mv "$$log_file" "$$archive_dir/$$archived_name"; \
		echo "Archived agate.log to $$archive_dir/$$archived_name"; \
	else \
		echo "No log file found ($$log_file) to archive"; \
	fi
