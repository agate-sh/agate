# @agate/sdk

Auto-generated TypeScript SDK for the Agate Server API.

## Installation

```bash
pnpm add @agate/sdk
```

## Usage

```typescript
import { client, sessionCreate, sessionGet } from '@agate/sdk';

client.setConfig({
  baseUrl: 'http://localhost:24283',
});

const { data, error } = await sessionCreate({
  body: {
    dir: process.cwd(),
    branchName: 'my-feature-branch',
    agentName: 'claude',
  },
});

if (error) throw error;

console.log('Created session', data?.sessionId);
```

## Regenerating the SDK

```bash
pnpm --filter @agate/sdk generate
```

This command refreshes the OpenAPI specification and regenerates the TypeScript client under `src/gen/`.

Generated files are ignored from version control—never edit files in `src/gen/` manually.
