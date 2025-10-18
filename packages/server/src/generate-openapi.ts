/**
 * Script to generate OpenAPI specification from Hono routes
 * Usage: tsx src/generate-openapi.ts
 */

import { writeFileSync } from 'fs';
import { join } from 'path';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import { createHonoServer, generateOpenAPISpec } from './server.hono.js';

async function main() {
  console.log('Generating OpenAPI specification...');

  // Create minimal dependencies just for spec generation
  const eventBus = new EventBus();
  const stateManager = await StateManager.create();

  // Create the app
  const app = createHonoServer(eventBus, stateManager);

  // Generate spec
  const spec = await generateOpenAPISpec(app);

  // Write to file
  const outputPath = join(process.cwd(), 'openapi.json');
  writeFileSync(outputPath, JSON.stringify(spec, null, 2));

  console.log(`✓ OpenAPI spec generated at: ${outputPath}`);
  console.log(`✓ Title: ${spec.info?.title}`);
  console.log(`✓ Version: ${spec.info?.version}`);
  console.log(`✓ Routes: ${Object.keys(spec.paths || {}).length}`);

  process.exit(0);
}

main().catch((error) => {
  console.error('Failed to generate OpenAPI spec:', error);
  process.exit(1);
});
