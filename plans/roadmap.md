# Agate TUI Migration Roadmap

This roadmap replaces the previous plan and focuses on rebuilding the Go TUI on top of the new TypeScript backend and generated Go SDK.

## Stage 1 — Agents Pane (complete)
- **Done**
  - Added thin wrappers in `packages/tui/internal/api` that call the generated SDK for repo, worktree, and session data.
  - Rewired the Bubble Tea model in `packages/tui/internal/ui` to consume live server payloads, including main/linked worktree grouping and session mutations.
  - Recreated the Agents pane container in Go TUI with proper full-screen layout, fixed width (25%), disabled pagination keybindings, and alt-screen handling to eliminate scrollback.
  - Ported the legacy list delegate with repo expansion toggles, session-row hint rendering, delete/attach command emission, and persisted selection state tied to SDK data.
  - Fleshed out data plumbing so `PaneState` builds from real session/worktree payloads, including repo headers, empty states, and linked session grouping.
  - Integrated command hooks with the surrounding program (`enter` attach/select, `n` create, `d` delete) and wired Bubble Tea messages to app model.
  - Mirrored the original visual affordances (icons, agent-colored highlights, shortcut hints).

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

### Overview
Connect the Go TUI to live PTY streams delivered by the TypeScript server via WebSocket. This stage implements bidirectional streaming for terminal I/O without requiring local tmux attachment (the server manages tmux sessions via node-pty).

### Architecture Reference (Legacy Go Implementation)
The legacy implementation in `go/pkg/gui/panes/agentTmux.go` and `go/pkg/tmux/session.go` provides the pattern:
- **Two modes**: Preview mode (captures static content via `tmux capture-pane`) and Attached mode (streams I/O through a local PTY with `tmux attach-session`)
- **Attach flow**: Creates a PTY, spawns goroutines to copy I/O, monitors for Ctrl+Q to detach
- **Layout**: Three-column layout with agents (25%), tmux (50%), and right column (25% for git/shell)

The **new WebSocket approach** simplifies this by eliminating local PTY management and tmux attach/detach—everything streams through the WebSocket connection to the server. For Stage 3, we implement a **two-column layout** (agents pane + tmux pane taking remaining space). The third column (git/shell) will be added in Stage 4.

### Implementation Plan

#### Phase 1: Tmux Pane Structure and Layout
**Goal**: Wire up a basic tmux pane that occupies the remaining space after the agents pane (two-column layout for now).

1. Create `packages/tui/internal/ui/panes/tmux.go`:
   - Implement `Pane` interface from `base.go`
   - Add `TmuxPane` struct with fields: `sessionID`, `content` (buffered output), `isSubscribed`
   - Implement `View()` to render content or show "No session selected" placeholder
   - Implement `GetTitleStyle()` to return badge-style title with agent color (similar to legacy `agentTmux.go:56-75`)
   - Add `SetSession(sessionID string)` method to track active session
   - Add `AppendContent(data string)` to buffer incoming PTY output

2. Update `packages/tui/internal/ui/app.go`:
   - Add `tmuxPane` field to `Model` struct
   - Initialize tmux pane in `NewModel()` with index 1
   - Add to `panes` slice
   - Wire pane focus handling (tab/number keys to switch panes)

3. Update layout logic in `resizePanes()`:
   - Calculate two-column layout: fixed width for agents pane (25% or similar), remaining space for tmux pane
   - Implement layout math similar to `go/pkg/gui/layout/layout.go:89-165`:
     - Reserve chrome height (title, footer, padding)
     - Calculate usable width with horizontal gaps
     - Agents pane: calculate content width (e.g., 25% of available content width)
     - Tmux pane: remaining width = total available width - agents pane full width - horizontal gap
     - Subtract frame sizes and padding
     - Set tmux pane content dimensions
   - Call `tmuxPane.SetSize(contentWidth, contentHeight)`

4. Update `View()` rendering:
   - Render agents pane on left (existing)
   - Render tmux pane on right taking all remaining space
   - Use `lipgloss.JoinHorizontal` to place panes side-by-side with gap
   - Follow border/padding pattern from `go/pkg/gui/layout/layout.go:168-277`

**Acceptance criteria**:
- Tmux pane visible on right side, occupying all space after agents pane
- Pane shows placeholder text when no session is selected
- Pane can receive focus via keybindings

---

#### Phase 2: WebSocket Connection and Subscription
**Goal**: Establish WebSocket connection to the server and subscribe to session PTY output.

1. Add WebSocket client dependency:
   - Add `github.com/gorilla/websocket` to `packages/tui/go.mod`
   - Run `go get github.com/gorilla/websocket`

2. Create `packages/tui/internal/ws/client.go`:
   - Implement `Client` struct with fields: `conn *websocket.Conn`, `url string`, `eventChan chan Event`
   - Add `Connect(url string) error` method
   - Add `Subscribe(sessionID string) error` to send `{"type": "subscribe", "sessionId": "<id>"}`
   - Add `Unsubscribe() error` to send `{"type": "unsubscribe"}`
   - Add `SendInput(sessionID, data string) error` to send `{"type": "pty:input", "sessionId": "<id>", "data": "<input>"}`
   - Add goroutine in `Connect()` to continuously read messages and publish to `eventChan`
   - Handle `pty:output` messages by emitting events with `{Type: "pty:output", SessionID: string, Data: string}`
   - Handle connection errors and reconnection logic with exponential backoff

3. Create `packages/tui/internal/ws/events.go`:
   - Define `Event` struct with fields: `Type string`, `SessionID string`, `Data string`
   - Define `EventType` constants: `EventPtyOutput`, `EventSessionCreated`, `EventSessionDeleted`

4. Update `Model` in `app.go`:
   - Add `wsClient *ws.Client` field
   - Initialize in `NewModel()` and connect to `ws://localhost:24283/ws` (use `AGATE_SERVER_PORT` from `@agate/shared`)
   - Add `wsEventChan` field to receive events from WebSocket
   - Add `wsSubscribedSessionID` field to track current subscription

5. Add Bubble Tea message types:
   - `wsConnectedMsg struct{}`
   - `wsErrorMsg struct{ err error }`
   - `wsPtyOutputMsg struct{ sessionID string; data string }`

6. Wire subscription on session selection:
   - When user selects a session in agents pane (existing behavior), detect session change
   - Unsubscribe from previous session if any
   - Subscribe to new session via `wsClient.Subscribe(sessionID)`
   - Update `tmuxPane.SetSession(sessionID)`

7. Handle WebSocket events in `Update()`:
   - Add case for `wsPtyOutputMsg`: append data to `tmuxPane` buffer and trigger re-render
   - Add error handling for `wsErrorMsg`

**Acceptance criteria**:
- WebSocket connects to server on startup
- Subscribing to a session streams initial tmux buffer content to pane
- PTY output updates appear in tmux pane in real-time

---

#### Phase 3: Input Forwarding
**Goal**: Forward keyboard input to the active session when tmux pane has focus.

1. Update `TmuxPane.HandleKey()` in `tmux.go`:
   - When pane is active and has a session, forward all keypresses to WebSocket
   - Call `wsClient.SendInput(sessionID, key)` for each key event
   - Handle special keys (arrow keys, enter, backspace, ctrl sequences)
   - Map Bubble Tea key events to PTY input format (matching what tmux expects)

2. Add input mode tracking:
   - Add `inputMode bool` field to `TmuxPane` to indicate if input is being captured
   - Set `inputMode = true` when pane receives focus and has active session
   - Set `inputMode = false` when pane loses focus

3. Update `Model.Update()`:
   - When tmux pane is active (`activePaneIndex == 1`), route all key messages to pane
   - Allow Ctrl+C and tab/pane-switching keys to bypass input forwarding

**Acceptance criteria**:
- Typing into active tmux pane forwards input to server
- Commands execute in the remote tmux session
- Output appears in pane in real-time
- Pane switching still works (tab, number keys)

---

#### Phase 4: Resize Handling
**Goal**: Notify server when tmux pane dimensions change so the PTY resizes correctly.

1. Update `TmuxPane.SetSize()`:
   - When dimensions change, send resize event to server via HTTP API
   - Use existing SDK method: `client.SessionsSessionIdResizePost(ctx, sessionID, ResizeRequest{Cols: width, Rows: height})`
   - Debounce resize events to avoid flooding the server (100ms delay)

2. Add resize tracking:
   - Store last known dimensions in `TmuxPane`
   - Only send resize request if dimensions actually changed

3. Handle `tea.WindowSizeMsg`:
   - Recalculate layout (already done in `resizePanes()`)
   - Trigger resize for tmux pane if active session exists

**Acceptance criteria**:
- Terminal resize triggers PTY resize on server
- PTY output reflowers to new width
- No visual glitches during resize

---

#### Phase 5: Loading States and Error Handling
**Goal**: Polish the UX with loading states, error recovery, and status indicators.

1. Add loading state to `TmuxPane`:
   - Show spinner while connecting to WebSocket
   - Show "Connecting to session..." while waiting for initial buffer
   - Mirror legacy loading view from `go/pkg/gui/panes/agentTmux.go:78-89`

2. Add error states:
   - Show "Failed to connect" if WebSocket connection fails
   - Show "Session not found" if subscription fails
   - Add retry button or auto-reconnect with backoff

3. Add status indicators:
   - Show "Connected" badge when actively streaming
   - Show "Disconnected" badge when WebSocket is down
   - Show agent color badge in title (already planned in Phase 1)

4. Handle server lifecycle events:
   - When `session.created` or `session.deleted` events arrive via WebSocket, refresh agents pane
   - Unsubscribe if current session is deleted

**Acceptance criteria**:
- Loading states appear during connection/subscription
- Errors display clearly with actionable messages
- WebSocket reconnects automatically on disconnect
- Session lifecycle events update UI correctly

---

### Testing Strategy
1. **Unit tests** (Go):
   - Test WebSocket client message parsing
   - Test pane layout calculations
   - Test input forwarding logic

2. **Integration tests** (manual):
   - Start server with `pnpm dev`
   - Launch TUI in tmux session: `tmux new-session -d -s agate-tui 'cd packages/tui && go run ./cmd/agate'`
   - Create session via agents pane
   - Verify PTY output streams to tmux pane
   - Type commands and verify execution
   - Resize terminal and verify reflow
   - Switch sessions and verify subscription changes
   - Kill server and verify reconnection

3. **End-to-end tests** (automated):
   - Use existing WebSocket tests in `packages/server/src/__tests__/websocket.test.ts` as reference
   - Add Go integration tests that:
     - Spin up TypeScript server
     - Connect Go WebSocket client
     - Subscribe to session
     - Send input
     - Verify output

---

### Open Questions
1. **Attach/Detach**: Do we need local tmux attach (Ctrl+Q detach) or is WebSocket streaming sufficient?
   - **Decision**: WebSocket streaming only. The server manages tmux via node-pty; the Go client just streams I/O. No need for local tmux attach since we're not running tmux locally.

2. **Mouse support**: Should we forward mouse events (clicks, scrolling)?
   - **Decision**: Defer to Stage 4. The server already supports PTY resize; mouse events can be added later if needed.

3. **Scrollback**: How do we handle tmux scrollback (copy mode)?
   - **Decision**: Defer to Stage 4. The initial implementation shows live output; scrollback can use `tmux capture-pane` with line offsets (similar to legacy `go/pkg/tmux/session.go:357-366`).

---

### Dependencies
- `github.com/gorilla/websocket` - WebSocket client
- `@agate/shared` types (for `AGATE_SERVER_PORT`)
- Existing SDK (`packages/sdk/go`) for session resize API

---

### Success Criteria
Stage 3 is complete when:
1. ✅ Tmux pane visible in 50% middle column
2. ✅ WebSocket connects to server on startup
3. ✅ Subscribing to a session streams PTY output in real-time
4. ✅ Keyboard input forwards to active session
5. ✅ Terminal resize triggers PTY resize
6. ✅ Loading/error states display correctly
7. ✅ Session switching updates subscription automatically

## Stage 4 — Git & Shell Panes
- Replace legacy git polling with API-backed file status and commit flows
- Implement shell pane via streamed sessions from the server rather than local commands
- Restore overlays (commit, debug, welcome) grounded in the new data sources

## Stage 5 — Polish & Packaging
- Audit logging, error surfaces, and configuration persistence
- Produce platform-specific binaries and wire them into the CLI release pipeline
- Update docs, release scripts, and automated tests to reflect the new architecture

Stages may overlap where practical, but each stage should reach a usable checkpoint before moving on.
