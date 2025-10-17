# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

NEVER MAINTAIN BACKWARDS COMPATIBILITY. This project is new and we don't need to care.

## Project Overview

Agate is a terminal multiplexer built for managing CLI agents (Claude, Gemini, Codex, etc.) with an information-dense, intuitive interface. The project is **currently being migrated from Go to TypeScript**.

**Current state:** The Go implementation is in `go/` (functional), and a new TypeScript implementation is being built in `packages/` using a pnpm monorepo.

## Architecture

The TypeScript implementation uses a **monorepo structure with 4 packages**:

- **`@agate/shared`** - Shared TypeScript types (agent configs, session state, git types, tmux types, API contracts)
- **`@agate/server`** - Express server for managing terminal sessions via node-pty
- **`@agate/sdk`** - Auto-generated TypeScript client SDK (from OpenAPI specs)
- **`@agate/client`** - OpenTUI-based React terminal UI (inspired by [sst/opencode](https://github.com/sst/opencode) `opentui` branch)

**Key inspiration:** This project draws heavily from `sst/opencode` (specifically the `opentui` branch), but uses **React instead of Solid**. A local clone exists at `/Users/patrickerichsen/Git/opencode`.

### Core Technologies

- **OpenTUI** (`@opentui/core`, `@opentui/react`) - Terminal UI framework (React-based)
- **node-pty** - PTY (pseudo-terminal) management for subprocess interaction
- **Express** - HTTP server with REST API and WebSocket streaming
- **WebSocket** (`ws`) - Real-time bidirectional communication for PTY I/O
- **TypeScript** - Strict mode with comprehensive type checking
- **pnpm** - Workspace management
- **Vitest** - Test framework with integration testing

### Server Architecture

The Express server (`@agate/server`) is fully implemented with:

- **Session Management API** - REST endpoints for creating/managing tmux sessions
  - `POST /session` - Create new session
  - `GET /session/:id` - Get session info
  - `POST /session/:id/input` - Send input to PTY
  - `POST /session/:id/resize` - Resize terminal
  - `DELETE /session/:id` - Kill session
- **WebSocket Streaming** - Real-time PTY I/O via WebSocket (`/ws`)
  - Bidirectional communication for input and output
  - Session subscription model
  - Event-based architecture with transport-agnostic EventBus
- **Git Operations API** - Repository and worktree management
- **State Persistence** - Atomic writes to `~/.agate/state.json`
- **Integration Tests** - Full end-to-end tests using real HTTP requests and WebSocket connections

## Building & Development

**All commands use pnpm:**

```bash
# Install dependencies
pnpm install

# Build all packages
pnpm build

# Run type checking across all packages
pnpm typecheck

# Development mode (watches all packages in parallel)
# This automatically rebuilds @agate/shared when types change
# and restarts the server on file changes
pnpm dev

# Clean build artifacts
pnpm clean
```

**Individual package commands:**

```bash
# Work in a specific package
cd packages/server
pnpm dev          # Watch mode
pnpm build        # Production build
pnpm typecheck    # Type check only
pnpm test         # Run tests
```

**IMPORTANT:** When developing, always use `pnpm dev` from the **root directory** to ensure `@agate/shared` types are automatically rebuilt when modified. Running individual package dev commands will cause type import errors.

## Testing

```bash
# Run all tests
pnpm --filter @agate/server test

# Run specific test file
pnpm --filter @agate/server test src/__tests__/integration.test.ts

# Run tests in watch mode
pnpm --filter @agate/server test --watch
```

The server has 131 passing tests including unit tests for all modules and integration tests that verify the full HTTP API.

## TypeScript Configuration

- **Strict mode enabled** with additional safety checks:
  - `noUncheckedIndexedAccess: true`
  - `noUnusedLocals: true`
  - `noUnusedParameters: true`
  - `noFallthroughCasesInSwitch: true`
- **ESNext modules** with `moduleResolution: "bundler"`
- **Composite project references** for incremental builds across packages

## Key Design Principles (from Go implementation)

### State Management

The Go implementation uses a **centralized, thread-safe StateManager** to prevent race conditions. The TypeScript version should follow similar patterns:

- **Single source of truth**: In-memory state protected by appropriate concurrency controls
- **Atomic updates**: All modifications happen in single operations
- **Atomic file writes**: State written to temp file, then atomically renamed
- State stored at `~/.agate/state.json`

### Session Naming

Session names must be:
1. Valid tmux names (alphanumeric, underscores, hyphens, dots)
2. Unique per worktree/branch
3. **Idempotent** - processing a sanitized name returns the same name

**Format:** `agate_<repo>_<branch>_<agent>_<8-hex-hash>`

**Critical:** Never call sanitization functions multiple times (causes double-prefixing bugs).

### Pane Layout System

The UI uses a **three-column layout**:
1. Repos/worktrees pane (25% width)
2. Tmux/agent interaction pane (50% width)
3. Right column split between Git status and Shell (remaining width)

**Key layout principles:**
- Compute content dimensions first, then add borders
- Use consistent border styles and padding
- Account for horizontal gutters between panes

## Debugging

Debug logs are written to `~/.agate/debug.log`:

```bash
# View recent logs
tail -f ~/.agate/debug.log

# Search for specific issues
grep "session restore" ~/.agate/debug.log
```

When debugging issues, add strategic debug logs and walk through actions to trigger them.

## Agent Configuration

Supported agents are defined in `packages/shared/src/types/agent.ts` with their:
- Display names
- Border colors (hex)
- Executable names (for process matching)
- Company names

Current agents: Claude, Amp, Gemini, Codex, OpenCode, Cursor, Copilot, Continue, Cline

## Migration Context

The TypeScript rewrite aims to:
1. Leverage the React ecosystem and OpenTUI for richer UI capabilities
2. Provide better developer ergonomics with TypeScript
3. Enable easier extensibility for plugins and custom panes
4. Maintain the core architecture patterns that work (state management, session naming)

**Reference the Go implementation** (`go/` folder) for understanding core logic, but don't maintain compatibility with it.
