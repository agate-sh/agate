# Agate Migration Overview

We are rebuilding the Go Bubble Tea TUI so it runs entirely on top of the new TypeScript backend. The goal is to keep the rich terminal UI while delegating persistence, git metadata, tmux orchestration, and agent lifecycle to the Node.js services exposed via OpenAPI.

## Guiding Principles
- **API-first**: Every piece of client state should be derived from the backend through the generated Go SDK (or direct REST calls during the transition).
- **Incremental UI rewrite**: Rebuild one pane at a time, keeping the application runnable with stubs for unfinished areas.
- **No legacy managers**: Replace direct usage of `state`, `session`, `git`, and `tmux` Go packages with API requests as each feature migrates.
- **Fast feedback**: Keep the Go client compiling and add targeted tests/mocks to validate each new pane.

## Current Focus
Stage 1 targets the **Agents Panel**. We will:
- Scaffold a standalone Go module in `packages/client`
- Generate or author the Go SDK from the OpenAPI schema
- Render agents/repos/worktrees using live API data
- Wire agent/session mutations through the server

Subsequent stages bring in worktree/session management, PTY streaming, git panes, and polish. See `plans/roadmap.md` for the staged breakdown.

## Supporting Packages
- `packages/sdk/typescript`: Generated TypeScript client (`@agate/sdk`) consumed by the Node CLI.
- `packages/sdk/go`: Generated Go client (OpenAPI Generator 7.16) that wraps the TypeScript server API for consumption by the Bubble Tea TUI.
