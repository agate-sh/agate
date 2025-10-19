import { serve } from '@hono/node-server';
import { createNodeWebSocket } from '@hono/node-ws';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import { createHonoServer } from './server.hono.js';
import { createWebSocketHandler } from './websocket.hono.js';
import { sessions } from './routes/session.hono.js';
import { TmuxSessionManager } from './tmux/session.js';
import { logger } from './logger.js';
import type { AgentName } from '@agate/shared';

const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : 3000;

/**
 * Restore sessions from persisted state by attaching to existing tmux sessions
 */
async function restoreSessions(eventBus: EventBus, stateManager: StateManager): Promise<void> {
  const state = stateManager.getState();
  const persistedSessions = Object.values(state.sessions.sessionMappings);

  logger.info({ count: persistedSessions.length }, 'Restoring sessions from state');

  for (const persistedSession of persistedSessions) {
    try {
      // Use the persisted session ID (which includes worktree_key + agent)
      const sessionId = persistedSession.id;
      const tmuxName = persistedSession.tmux_name;
      const agent = persistedSession.agent_name as AgentName;
      const cwd = persistedSession.worktree_path;

      logger.debug({
        sessionId,
        tmuxName,
        agent,
        cwd
      }, 'Attempting to restore session');

      // Check if tmux session still exists
      const { exec } = await import('node:child_process');
      const { promisify } = await import('node:util');
      const execAsync = promisify(exec);

      try {
        await execAsync(`tmux has-session -t ${tmuxName}`);
        logger.debug({ tmuxName }, 'Tmux session exists, attaching');

        // Create session manager and attach to existing tmux session
        const sessionManager = new TmuxSessionManager(eventBus, sessionId);
        await sessionManager.attachToSession(tmuxName, agent, cwd);

        // Add to sessions map
        sessions.set(sessionId, sessionManager);

        logger.info({
          sessionId,
          tmuxName,
          agent
        }, 'Session restored successfully');
      } catch (err) {
        logger.warn({
          tmuxName,
          err: err instanceof Error ? err.message : String(err)
        }, 'Tmux session no longer exists, skipping restore');
      }
    } catch (error) {
      logger.error({
        sessionId: persistedSession.id,
        err: error
      }, 'Failed to restore session');
    }
  }

  logger.info({ restored: sessions.size }, 'Session restoration complete');
}

// Initialize state and start server
async function start() {
  // Initialize dependencies
  const eventBus = new EventBus();
  const stateManager = await StateManager.create();
  logger.info('StateManager initialized');

  // Log loaded state summary
  const state = stateManager.getState();
  const sessionCount = Object.keys(state.sessions.sessionMappings).length;
  const repoCount = Object.keys(state.workspace.repositories).length;
  logger.info({ sessionCount, repoCount }, 'Loaded state');

  // Restore existing sessions from state
  await restoreSessions(eventBus, stateManager);

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
      logger.info({ port: info.port }, 'Agate server listening');
      logger.info(`Health check: http://localhost:${info.port}/health`);
      logger.info(`WebSocket endpoint: ws://localhost:${info.port}/ws`);
      logger.info(`OpenAPI docs: http://localhost:${info.port}/doc`);
    }
  );

  // Inject WebSocket upgrade handler into the Node.js server
  injectWebSocket(server);
}

start().catch((error) => {
  logger.fatal({ err: error }, 'Failed to start server');
  process.exit(1);
});
