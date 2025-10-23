# Agate TUI Migration Roadmap

This roadmap replaces the previous plan and focuses on rebuilding the Go TUI on top of the new TypeScript backend and generated Go SDK.

## Stage 1 — Agents Panel (in progress)
- Generate or hand-author minimal Go client data structures for listing agents, repos, and sessions from the API
- Replace legacy stateful managers with data fetched from the HTTP API / WebSocket streams
- Rebuild the Agents pane UI to render the new data model and notify the server for mutations
- Remove or stub unrelated panes while the agents experience is under construction

## Stage 2 — Worktree & Session Management
- Introduce Go bindings for worktree CRUD and session lifecycle APIs
- Recreate the worktree picker and session dialogs using server-backed operations
- Implement optimistic updates and error handling around long-running operations

## Stage 3 — PTY / tmux Interaction
- Connect tmux pane to live PTY streams delivered by the TypeScript server
- Implement input forwarding, resize events, and status reporting through WebSockets
- Reconcile loading/toast UI with server lifecycle events

## Stage 4 — Git & Shell Panes
- Replace legacy git polling with API-backed file status and commit flows
- Implement shell pane via streamed sessions from the server rather than local commands
- Restore overlays (commit, debug, welcome) grounded in the new data sources

## Stage 5 — Polish & Packaging
- Audit logging, error surfaces, and configuration persistence
- Produce platform-specific binaries and wire them into the CLI release pipeline
- Update docs, release scripts, and automated tests to reflect the new architecture

Stages may overlap where practical, but each stage should reach a usable checkpoint before moving on.
