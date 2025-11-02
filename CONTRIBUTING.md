# Contributing to Agate

Thank you for your interest in contributing to Agate! This guide will help you get up and running quickly.

## Quick Start

### Prerequisites

- Go 1.21 or higher

### Setup

```bash
# Clone and build
git clone https://github.com/agate-sh/agate
cd agate
make build

# Run it
./agate claude
```

That's it! For development, you may want to install additional tools:

```bash
make install-tools  # Installs golangci-lint and goimports
```

## Development Builds

```bash
# Standard build
make build

# Debug build (adds debug panel and logging)
go build -tags debug
```

## Code Quality

```bash
make fmt       # Format code
make lint      # Run linters
make lint-fix  # Auto-fix linting issues
make check     # Run all checks
make test      # Run tests
```

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires API keys)
export ANTHROPIC_API_KEY="sk-ant-..."
go test -v -tags=integration ./pkg/git
```

## Debugging

Debug logs are written to `~/.agate/debug.log`:

```bash
tail -f ~/.agate/debug.log
```

The `~/.agate` directory also contains:

- `state.json` - Persisted application state
- `worktrees/` - Git worktree data for agent sessions

## Understanding the Codebase

This repository is indexed by [DeepWiki](https://deepwiki.com/agate-sh/agate) at **deepwiki.com/agate-sh/agate**.

**What is DeepWiki?** It's an AI-powered documentation service that allows AI assistants to search and query GitHub repositories. Think of it as having the entire codebase indexed and searchable by your AI assistant.

**To use DeepWiki with Claude Code:**

```bash
claude mcp add @cognition-labs/deepwiki-mcp
```

Then ask questions like:

```
Use deepwiki to understand how session management works in agate-sh/agate
```

Your assistant can now answer detailed questions about Agate's implementation, architecture, and code patterns.

## Development Workflow

```bash
# 1. Create a feature branch
git checkout -b feature/your-feature-name

# 2. Make changes and test
make check && make test

# 3. Commit and push
git add .
git commit -m "Brief description"
git push origin feature/your-feature-name
```

Then open a pull request!

## Architecture

Agate tackles three core challenges:

1. **Custom UI Rendering** - Information-dense terminal interface using Bubble Tea + Lipgloss
2. **Full Terminal Emulation** - Proper handling of ANSI escape sequences for agent output
3. **Process Multiplexing** - Managing multiple PTY sessions with proper lifecycle management

**For detailed technical information**, use DeepWiki (see above) to query the codebase directly.

## Need Help?

- Check [README.md](README.md) for usage
- Visit [agate.sh](https://agate.sh) for more info
- Ask questions in GitHub issues
- Use DeepWiki to understand the code
