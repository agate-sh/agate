# Stage 1 — Agents Panel Rewrite

Goal: Ship a minimal, server-backed agents pane that can be launched via the CLI while other panes remain stubbed.

## Deliverables
- `packages/client` Go module that compiles independently of the legacy `go/` tree
- Agents pane rendering data fetched from the TypeScript backend (via generated Go SDK or lightweight client)
- Ability to list agents, select repos/worktrees, and trigger session creation through API calls
- Temporary stubs or placeholders for tmux/git/shell panes so the application still renders

## Work Breakdown
1. **Go module scaffolding**
   - Create `packages/client/go.mod` with a standalone module name
   - Wire module tooling (Makefile, package.json scripts) to build/test the Go client
   - Add minimal logging/config helpers as needed
2. **API client plumbing**
   - Generate or draft the Go SDK from the OpenAPI spec exposed by the TypeScript server
   - Establish configuration (server URL, auth) and helper functions to fetch agent state
   - Add mocks/fakes for offline development and tests
3. **State & message architecture**
   - Introduce a new Bubble Tea model focused on agents data
   - Replace usage of legacy managers (`session.Manager`, `state.Manager`, etc.) with API calls
   - Define update messages for list refresh, selection, and mutations
4. **UI implementation**
   - Rebuild the agents list component for the new data contracts
   - Implement creation/delete flows by calling the server and updating local state
   - Add loading/error states and toasts based on API responses
5. **Cleanup & validation**
   - Remove unused legacy code paths while keeping minimal stubs for other panes
   - Add unit tests for state transitions and API integration (mocked)
   - Document usage in `packages/client/README.md` (to do)

We will iterate on these tasks and keep the plan updated as work progresses.

## Status
- Generated initial Go SDK under `packages/sdk/go` using OpenAPI Generator (CLI 7.16.0).
- TypeScript CLI uses the regenerated `@agate/sdk` package in `packages/sdk/typescript`.
- CLI now builds and launches the Go TUI binary directly (`pnpm agate`), removing the temporary workspace package shim.
