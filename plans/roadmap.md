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

### Implementation Plan
1. Align server contracts and Go SDK
   - Audit `packages/server/src` routes to ensure worktree CRUD and session lifecycle endpoints expose everything the Go client needs (list/create/delete worktree, list/create/attach/detach session, session metadata).
   - Add or extend OpenAPI annotations where gaps exist, then run `pnpm --filter @agate/sdk-go generate` to refresh the Go SDK and commit the generated client (Stage 2 unblocks without worrying about backward compatibility).
   - Document any server changes required for Stage 2 directly in `packages/shared` types to keep both TS and Go sides in sync.
2. Build Go-side API surface
   - Add thin wrappers in `go/internal/remote` (or an equivalent new package) that call the generated SDK and return strongly typed responses tailored for the TUI.
   - Replace direct filesystem/git inspection in `go/pkg/session` and `go/pkg/state` with calls into the remote package; remove unused legacy helpers rather than shimming them.
   - Introduce context-aware helpers (with timeouts and cancellation) so the UI can abort long-running operations cleanly.
3. Rewire worktree picker and session dialogs
   - Update the data sources in `go/pkg/gui/panes/agents.go` (and any related presenters) to pull worktrees and sessions from the new remote layer.
   - Rebuild the creation/selection flows to use server mutations, mirroring the dialogs that exist today but sourcing data from the API responses.
   - Ensure selections persist via the shared state manager by storing server-issued identifiers instead of local tmux names.
4. Handle errors and logging
   - Surface structured errors with actionable messages, threading them through the existing notification/toast system.
   - Add targeted debug logging (mirroring the pino structure) to trace API requests, retries, and reconciliations.
5. Validate end-to-end
   - Create Go integration tests that spin up a mock or real server (using the TS implementation) and exercise the Stage 2 flows via the new SDK bindings.
   - Add manual test scripts that dogfood the rebuilt UI against the TypeScript server, covering happy paths and failure cases (network loss, stale sessions).
   - Capture follow-up tasks for any missing server capabilities identified during testing so Stage 3 can build on a stable foundation.

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
