import express, { type Request, type Response, type NextFunction } from 'express';
import cors from 'cors';
import { randomUUID } from 'crypto';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import gitRouter from './routes/git.js';

const app = express();
const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : 3000;
const eventBus = new EventBus();
let stateManager: StateManager;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static('public'));

// Health check endpoint
app.get('/health', (_req: Request, res: Response) => {
  res.status(200).json({ status: 'ok', timestamp: new Date().toISOString() });
});

// SSE endpoint
app.get('/events', (req: Request, res: Response) => {
  const clientId = (req.query.clientId as string) || randomUUID();
  eventBus.subscribe(clientId, res);
});

// API routes
app.use('/git', gitRouter);

// Error handling middleware
app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
  console.error('Error:', err);
  res.status(500).json({
    error: {
      message: err.message || 'Internal server error',
      ...(process.env.NODE_ENV === 'development' && { stack: err.stack }),
    },
  });
});

// Initialize state and start server
async function start() {
  // Initialize state manager
  stateManager = await StateManager.create();
  console.log('StateManager initialized');

  // Log loaded state summary
  const state = stateManager.getState();
  const sessionCount = Object.keys(state.sessions.sessionMappings).length;
  const repoCount = Object.keys(state.workspace.repositories).length;
  console.log(`Loaded state: ${sessionCount} sessions, ${repoCount} repositories`);

  // Start server
  app.listen(PORT, () => {
    console.log(`Agate server listening on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
  });
}

start().catch((error) => {
  console.error('Failed to start server:', error);
  process.exit(1);
});

// Export stateManager for use in routes
export { stateManager };
