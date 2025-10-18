# @agate/sdk

Auto-generated TypeScript SDK for the Agate Server API.

## Installation

```bash
pnpm add @agate/sdk
```

## Usage

```typescript
import { client, sessionCreate, sessionGet, sessionList } from '@agate/sdk';

// Configure the client (default: http://localhost:3000)
client.setConfig({
  baseUrl: 'http://localhost:3000'
});

// Create a new session
const { data: session } = await sessionCreate({
  body: {
    name: 'my-session',
    tmuxSessionName: 'agate_my_repo_main_claude_abc123'
  }
});

// Get session info
const { data: sessionInfo } = await sessionGet({
  path: { id: session.id }
});

// List all sessions
const { data: sessions } = await sessionList();

// Check server health
const { data: health } = await healthCheck();
console.log(health.status); // 'ok'
```

## Development

This SDK is auto-generated from the OpenAPI specification. **Do not edit the generated code manually.**

### Regenerate the SDK

```bash
# Regenerate OpenAPI spec + SDK
pnpm generate

# Or individually:
pnpm generate:spec  # Generate OpenAPI spec from server routes
pnpm generate:sdk   # Generate SDK from OpenAPI spec
```

### Auto-Regeneration (Dev Mode)

The SDK automatically regenerates when server routes change:

```bash
# From the root directory:
pnpm dev

# This runs watchers that:
# 1. Watch server route files -> regenerate OpenAPI spec
# 2. Watch OpenAPI spec -> regenerate SDK
# 3. Watch SDK src -> rebuild SDK dist
```

## Architecture

- **Generated from**: `@agate/server` OpenAPI specification
- **Generator**: [@hey-api/openapi-ts](https://github.com/hey-api/openapi-ts)
- **HTTP Client**: `@hey-api/client-fetch`
- **Type Safety**: Full TypeScript types for all endpoints

## Generated Files

All files in `src/gen/` are auto-generated:

- `types.gen.ts` - TypeScript types for all schemas
- `sdk.gen.ts` - API methods (e.g., `sessionCreate`, `sessionGet`)
- `client.gen.ts` - HTTP client configuration
- `index.ts` - Barrel export

These files are **not committed to git** and are regenerated on build.
