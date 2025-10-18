import { serve } from '@hono/node-server';
import { createNodeWebSocket } from '@hono/node-ws';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import { createHonoServer } from './server.hono.js';
import { createWebSocketHandler } from './websocket.hono.js';
import { sessions } from './routes/session.hono.js';

const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : 3000;

// Initialize state and start server
async function start() {
  // Initialize dependencies
  const eventBus = new EventBus();
  const stateManager = await StateManager.create();
  console.log('StateManager initialized');

  // Log loaded state summary
  const state = stateManager.getState();
  const sessionCount = Object.keys(state.sessions.sessionMappings).length;
  const repoCount = Object.keys(state.workspace.repositories).length;
  console.log(`Loaded state: ${sessionCount} sessions, ${repoCount} repositories`);

  // Create Hono app
  const app = createHonoServer(eventBus, stateManager);

  // Setup WebSocket support
  const { injectWebSocket, upgradeWebSocket } = createNodeWebSocket({ app });

  // Add WebSocket route
  app.get('/ws', upgradeWebSocket(createWebSocketHandler(eventBus, sessions)));

  // Start server with WebSocket support
  const server = serve(
    {
      fetch: app.fetch,
      port: PORT,
    },
    (info) => {
      console.log(`Agate server listening on port ${info.port}`);
      console.log(`Health check: http://localhost:${info.port}/health`);
      console.log(`WebSocket endpoint: ws://localhost:${info.port}/ws`);
      console.log(`OpenAPI docs: http://localhost:${info.port}/doc`);
    }
  );

  // Inject WebSocket upgrade handler into the Node.js server
  injectWebSocket(server);
}

start().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
