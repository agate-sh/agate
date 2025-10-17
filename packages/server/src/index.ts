import express, { type Request, type Response, type NextFunction } from 'express';
import cors from 'cors';
import { randomUUID } from 'crypto';
import { EventBus } from './event-bus.js';

const app = express();
const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : 3000;
const eventBus = new EventBus();

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

// Start server
app.listen(PORT, () => {
  console.log(`Agate server listening on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
});
