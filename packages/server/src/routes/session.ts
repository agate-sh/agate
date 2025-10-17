import { Router, Request, Response } from 'express';
import type { Router as ExpressRouter } from 'express';
import { randomUUID } from 'crypto';
import { TmuxSessionManager } from '../tmux/session.js';
import { EventBus } from '../event-bus.js';
import type {
  CreateSessionRequest,
  CreateSessionResponse,
  GetSessionResponse,
  SendSessionInputRequest,
  SendSessionInputResponse,
  ResizeSessionRequest,
  ResizeSessionResponse,
  DeleteSessionResponse,
} from '@agate/shared';
import type { SessionCreatedEvent, SessionDeletedEvent } from '@agate/shared';

const router: ExpressRouter = Router();

// In-memory session registry
const sessions = new Map<string, TmuxSessionManager>();

/**
 * POST /session
 * Create a new tmux session
 */
router.post('/', async (req: Request, res: Response) => {
  try {
    const body = req.body as CreateSessionRequest;
    const { worktreePath, branch, agentName } = body;

    // Validate request
    if (!worktreePath || !branch || !agentName) {
      res.status(400).json({
        error: 'worktreePath, branch, and agentName are required',
      });
      return;
    }

    // Generate session ID
    const sessionId = randomUUID();

    // Get eventBus from app.locals
    const eventBus = req.app.locals.eventBus as EventBus;

    // Create TmuxSessionManager
    const sessionManager = new TmuxSessionManager(eventBus, sessionId);

    // Create tmux session
    const sessionName = `agate_${branch}_${agentName}_${sessionId.substring(0, 8)}`;
    await sessionManager.createSession({
      name: sessionName,
      agent: agentName,
      cwd: worktreePath,
    });

    // Store in registry
    sessions.set(sessionId, sessionManager);

    // Emit session created event
    const event: SessionCreatedEvent = {
      type: 'session.created',
      sessionId,
      timestamp: new Date().toISOString(),
    };
    eventBus.publish(event);

    // Return response
    const response: CreateSessionResponse = {
      sessionId,
    };
    res.status(201).json(response);
  } catch (error) {
    console.error('Error creating session:', error);
    res.status(500).json({ error: 'Failed to create session' });
  }
});

/**
 * GET /session/:id
 * Get session information
 */
router.get('/:id', (req: Request, res: Response) => {
  try {
    const id = req.params.id;
    if (!id) {
      res.status(400).json({ error: 'Session ID is required' });
      return;
    }

    const sessionManager = sessions.get(id);
    if (!sessionManager) {
      res.status(404).json({ error: 'Session not found' });
      return;
    }

    const info = sessionManager.getInfo();
    const response: GetSessionResponse = {
      id: info.id,
      name: info.name,
      agent: info.agent,
      cwd: info.cwd,
      cols: info.cols,
      rows: info.rows,
      pid: info.pid,
      isAlive: info.isAlive,
    };

    res.json(response);
  } catch (error) {
    console.error('Error getting session info:', error);
    res.status(500).json({ error: 'Failed to get session info' });
  }
});

/**
 * POST /session/:id/input
 * Send input to session
 */
router.post('/:id/input', (req: Request, res: Response) => {
  try {
    const id = req.params.id;
    if (!id) {
      res.status(400).json({ error: 'Session ID is required' });
      return;
    }

    const { data } = req.body as SendSessionInputRequest;

    // Validate request
    if (!data) {
      res.status(400).json({ error: 'data is required' });
      return;
    }

    const sessionManager = sessions.get(id);
    if (!sessionManager) {
      res.status(404).json({ error: 'Session not found' });
      return;
    }

    sessionManager.write(data);

    const response: SendSessionInputResponse = {
      success: true,
    };
    res.json(response);
  } catch (error) {
    console.error('Error sending input:', error);
    res.status(500).json({ error: 'Failed to send input' });
  }
});

/**
 * POST /session/:id/resize
 * Resize session terminal
 */
router.post('/:id/resize', (req: Request, res: Response) => {
  try {
    const id = req.params.id;
    if (!id) {
      res.status(400).json({ error: 'Session ID is required' });
      return;
    }

    const { cols, rows } = req.body as ResizeSessionRequest;

    // Validate request
    if (typeof cols !== 'number' || typeof rows !== 'number') {
      res.status(400).json({ error: 'cols and rows must be numbers' });
      return;
    }

    const sessionManager = sessions.get(id);
    if (!sessionManager) {
      res.status(404).json({ error: 'Session not found' });
      return;
    }

    sessionManager.resize(cols, rows);

    const response: ResizeSessionResponse = {
      success: true,
    };
    res.json(response);
  } catch (error) {
    console.error('Error resizing session:', error);
    res.status(500).json({ error: 'Failed to resize session' });
  }
});

/**
 * DELETE /session/:id
 * Kill and delete session
 */
router.delete('/:id', async (req: Request, res: Response) => {
  try {
    const id = req.params.id;
    if (!id) {
      res.status(400).json({ error: 'Session ID is required' });
      return;
    }

    const sessionManager = sessions.get(id);
    if (!sessionManager) {
      res.status(404).json({ error: 'Session not found' });
      return;
    }

    // Kill the session
    await sessionManager.kill();

    // Remove from registry
    sessions.delete(id);

    // Get eventBus from app.locals
    const eventBus = req.app.locals.eventBus as EventBus;

    // Emit session deleted event
    const event: SessionDeletedEvent = {
      type: 'session.deleted',
      sessionId: id,
      timestamp: new Date().toISOString(),
    };
    eventBus.publish(event);

    const response: DeleteSessionResponse = {
      success: true,
    };
    res.json(response);
  } catch (error) {
    console.error('Error deleting session:', error);
    res.status(500).json({ error: 'Failed to delete session' });
  }
});

export default router;
