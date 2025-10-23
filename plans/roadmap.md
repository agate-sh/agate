# Agate TUI Migration Roadmap

This roadmap replaces the previous plan and focuses on rebuilding the Go TUI on top of the new TypeScript backend and generated Go SDK.

## Stage 1 — Agents Pane (in progress)
- **Done**
  - Added thin wrappers in `packages/client/internal/api` that call the generated SDK for repo, worktree, and session data.
  - Rewired the Bubble Tea model in `packages/client/internal/ui` to consume live server payloads, including main/linked worktree grouping and session mutations.
- **Next**
  - Port the legacy Agents pane view, keybindings, and selection logic from `go/pkg/gui/panes/agents.go` into the new client so the UI matches the original screenshot.
  - Stub the other panes for now; focus on restoring the Agents column end-to-end before reintroducing tmux/git/shell views.

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
   - Add thin wrappers in `packages/client/internal/api` that call the generated SDK directly (no hand-authored models) for worktree and session operations.
   - Replace legacy filesystem/git inspection inside the client with these remote calls; remove unused helpers rather than shimming them.
   - Introduce context-aware helpers (with timeouts and cancellation) so the UI can abort long-running operations cleanly.
3. Rewire worktree picker and session dialogs
   - Update the Agents pane implementation in `packages/client/internal/ui` to source worktrees and sessions from the server responses.
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
