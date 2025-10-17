import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import { createServer } from './server.js';

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

  // Create and start server
  const app = createServer(eventBus, stateManager);
  app.listen(PORT, () => {
    console.log(`Agate server listening on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
  });
}

start().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});
