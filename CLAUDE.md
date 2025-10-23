# CLAUDE.md

NEVER MAINTAIN BACKWARDS COMPATIBILITY. This project is new and we don't need to care.

## Project Overview

Agate is a terminal multiplexer built for managing CLI agents (Claude, Gemini, Codex, etc.) with an information-dense, intuitive interface.

**Current state:** The Go implementation now lives in `packages/tui/` (functional), and a new TypeScript implementation is being built in `packages/` using a pnpm monorepo.

## Architecture

The TypeScript implementation uses a **monorepo structure with 4 packages**:

- **`@agate/shared`** - Shared TypeScript types (agent configs, session state, git types, tmux types, API contracts)
- **`@agate/server`** - Express server for managing terminal sessions via node-pty
- **`@agate/sdk`** - Auto-generated TypeScript client SDK (lives in `packages/sdk/typescript`)
- **`@agate/client`** - OpenTUI-based React terminal UI

The Go Bubble Tea client consumes an auto-generated Go SDK located at `packages/sdk/go`.

### Core Technologies

- **OpenTUI** (`@opentui/core`, `@opentui/react`) - Terminal UI framework (React-based)
- **node-pty** - PTY (pseudo-terminal) management for subprocess interaction
- **Express** - HTTP server with REST API and WebSocket streaming
- **WebSocket** (`ws`) - Real-time bidirectional communication for PTY I/O
- **TypeScript** - Strict mode with comprehensive type checking
- **pnpm** - Workspace management
- **Vitest** - Test framework with integration testing

### Server Architecture

The Hono server (`@agate/server`) is fully implemented with:

- **Default Port**: `24283` (defined in `@agate/shared` as `AGATE_SERVER_PORT`)

  - Can be overridden with `PORT` environment variable
  - WebSocket endpoint: `ws://localhost:24283/ws`
  - Health check: `http://localhost:24283/health`
  - OpenAPI docs: `http://localhost:24283/doc`

- **Session Management API** - REST endpoints for creating/managing tmux sessions
  - `POST /sessions` - Create new session
  - `GET /sessions/:id` - Get session info
  - `POST /sessions/:id/input` - Send input to PTY
  - `POST /sessions/:id/resize` - Resize terminal
  - `DELETE /sessions/:id` - Kill session
- **WebSocket Streaming** - Real-time PTY I/O via WebSocket (`/ws`)
  - Bidirectional communication for input and output
  - Session subscription model
  - Event-based architecture with transport-agnostic EventBus
- **Git Operations API** - Repository and worktree management
- **State Persistence** - Atomic writes to `~/.agate/state.json`
- **Session Restoration** - On server start, automatically reattaches to existing tmux sessions from persisted state
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

## OpenAPI Specification & SDK Generation

The server uses **Hono + hono-openapi** for automatic OpenAPI spec generation from route definitions. The TypeScript SDK (`@agate/sdk`) regenerates from this spec in `packages/sdk/typescript`, and the Go SDK lives in `packages/sdk/go`.

### Accessing the OpenAPI Spec

1. **Swagger UI (interactive docs)**: `GET /doc`
2. **Raw JSON spec**: `GET /openapi.json`
3. **Generate to file**: `pnpm --filter @agate/server generate:openapi`

### SDK Generation Workflow

The SDK is generated on demand via the workspace script:

```bash
# Generate fresh OpenAPI spec and TypeScript SDK
pnpm --filter @agate/sdk generate

# Generate fresh OpenAPI spec and Go SDK
pnpm --filter @agate/sdk-go generate
```

These scripts regenerate `packages/server/openapi.json`, refresh the TypeScript client under `packages/sdk/typescript/src/gen/`, and rewrite the Go client under `packages/sdk/go/gen/`, running `go mod tidy` to keep dependencies current.

### What's NOT Committed to Git

- `packages/server/openapi.json` - Regenerated from routes
- `packages/sdk/typescript/src/gen/` - Regenerated from OpenAPI spec (ignored in git)
- `packages/sdk/go/gen/` - Regenerated from OpenAPI spec

Both are always generated from source code, so there's no risk of staleness.

### Key Files

- `packages/server/src/server.hono.ts` - Route definitions with `describeRoute()` decorators
- `packages/server/src/generate-openapi.ts` - Script to generate spec file
- `packages/sdk/typescript/src/index.ts` - Barrel export that re-exports generated helpers
- `packages/sdk/typescript/tsup.config.ts` - Bundles the SDK for consumption
- `packages/sdk/go/scripts/generate.sh` - Orchestrates Go SDK regeneration
- Route files use `hono-openapi`'s `describeRoute()` and `resolver()` for type-safe API definitions

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

## Logging

The project uses **pino** for structured logging with the following setup:

### Log Locations

- **Development mode**: Both server and client write to `~/.agate/dev.log` (unified)
- **Production mode**: Separate logs at `~/.agate/server.log` and `~/.agate/client.log`

### Viewing Logs

```bash
# In development - unified log stream
pnpm dev:logs

# Or manually tail the dev log
tail -f ~/.agate/dev.log

# Search for specific issues
grep "session restore" ~/.agate/dev.log
```

### Using the Logger

**IMPORTANT**: Always use the `logger` object, NEVER use `console.log`, `console.error`, etc.

The logger is available in all packages:

```typescript
// Server and shared packages
import { logger } from "./logger.js";

// Client package
import { logger } from "../logger.js";

// Structured logging examples
logger.info({ sessionId, userId }, "Session created");
logger.error({ err }, "Failed to connect"); // Use { err } not { error }
logger.debug({ state }, "Current state");
logger.warn({ sessionId, availableSessions }, "Session not found");
```

**Log levels**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`

**Best practices**:

- First parameter is always an object with context (structured data)
- Second parameter is a human-readable message string
- Use `{ err: error }` not `{ error }` when logging errors (pino convention)
- Include relevant IDs and context in the structured data
- Pretty-printed output in dev mode (with colors and timestamps)
- Structured JSON in production
- Automatic component tagging (`[server]` or `[client]`)
- File persistence for debugging

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

**Reference the Go implementation** (`packages/tui/` folder) for understanding core logic, but don't maintain compatibility with it.
