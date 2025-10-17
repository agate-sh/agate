import express, { type Request, type Response, type NextFunction, type Express } from 'express';
import cors from 'cors';
import { EventBus } from './event-bus.js';
import { StateManager } from './state/manager.js';
import gitRouter from './routes/git.js';
import sessionRouter from './routes/session.js';

/**
 * Create and configure the Express application
 * This is extracted from index.ts to allow reuse in tests
 */
export function createServer(eventBus: EventBus, stateManager: StateManager): Express {
  const app = express();

  // Middleware
  app.use(cors());
  app.use(express.json());
  app.use(express.static('public'));

  // Make eventBus and stateManager available to routes
  app.locals.eventBus = eventBus;
  app.locals.stateManager = stateManager;

  // Health check endpoint
  app.get('/health', (_req: Request, res: Response) => {
    res.status(200).json({ status: 'ok', timestamp: new Date().toISOString() });
  });

  // API routes
  app.use('/git', gitRouter);
  app.use('/session', sessionRouter);

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

  return app;
}
