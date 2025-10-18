import { Hono } from 'hono';
import { describeRoute, resolver, validator } from 'hono-openapi';
import { randomUUID } from 'crypto';
import z from 'zod';
import { TmuxSessionManager } from '../tmux/session.js';
import { EventBus } from '../event-bus.js';
import type {
  CreateSessionResponse,
  GetSessionResponse,
  ResizeSessionResponse,
  DeleteSessionResponse,
  AgentName,
} from '@agate/shared';
import type { SessionCreatedEvent, SessionDeletedEvent } from '@agate/shared';

// In-memory session registry
export const sessions = new Map<string, TmuxSessionManager>();

// Define Zod schemas for validation
const CreateSessionSchema = z.object({
  worktreePath: z.string(),
  branch: z.string(),
  agentName: z.string(),
});

const ResizeSessionSchema = z.object({
  cols: z.number(),
  rows: z.number(),
});

const SessionIdParam = z.object({
  id: z.string(),
});

// Response schemas
const CreateSessionResponseSchema = z.object({
  sessionId: z.string(),
});

const GetSessionResponseSchema = z.object({
  id: z.string(),
  name: z.string(),
  agent: z.string(),
  cwd: z.string(),
  cols: z.number(),
  rows: z.number(),
  pid: z.number().optional(),
  isAlive: z.boolean(),
});

const SuccessResponseSchema = z.object({
  success: z.boolean(),
});

// Error responses
const ERRORS = {
  400: {
    description: 'Bad request',
    content: {
      'application/json': {
        schema: resolver(z.object({ error: z.string() })),
      },
    },
  },
  404: {
    description: 'Not found',
    content: {
      'application/json': {
        schema: resolver(z.object({ error: z.string() })),
      },
    },
  },
  500: {
    description: 'Internal server error',
    content: {
      'application/json': {
        schema: resolver(z.object({ error: z.string() })),
      },
    },
  },
} as const;

type Env = {
  Variables: {
    eventBus: EventBus;
  };
};

export const sessionRouter = new Hono<Env>()
  .post(
    '/',
    describeRoute({
      description: 'Create a new tmux session',
      operationId: 'session.create',
      responses: {
        201: {
          description: 'Session created successfully',
          content: {
            'application/json': {
              schema: resolver(CreateSessionResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', CreateSessionSchema),
    async (c) => {
      try {
        const body = c.req.valid('json');
        const { worktreePath, branch, agentName } = body;

        // Generate session ID
        const sessionId = randomUUID();

        // Get eventBus from context
        const eventBus = c.get('eventBus');

        // Create TmuxSessionManager
        const sessionManager = new TmuxSessionManager(eventBus, sessionId);

        // Create tmux session
        const sessionName = `agate_${branch}_${agentName}_${sessionId.substring(0, 8)}`;
        await sessionManager.createSession({
          name: sessionName,
          agent: agentName as AgentName,
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
        return c.json(response, 201);
      } catch (error) {
        console.error('Error creating session:', error);
        return c.json({ error: 'Failed to create session' }, 500);
      }
    }
  )
  .get(
    '/:id',
    describeRoute({
      description: 'Get session information',
      operationId: 'session.get',
      responses: {
        200: {
          description: 'Session information',
          content: {
            'application/json': {
              schema: resolver(GetSessionResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('param', SessionIdParam),
    (c) => {
      try {
        const { id } = c.req.valid('param');

        const sessionManager = sessions.get(id);
        if (!sessionManager) {
          return c.json({ error: 'Session not found' }, 404);
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

        return c.json(response);
      } catch (error) {
        console.error('Error getting session info:', error);
        return c.json({ error: 'Failed to get session info' }, 500);
      }
    }
  )
  .post(
    '/:id/resize',
    describeRoute({
      description: 'Resize session terminal',
      operationId: 'session.resize',
      responses: {
        200: {
          description: 'Terminal resized successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('param', SessionIdParam),
    validator('json', ResizeSessionSchema),
    (c) => {
      try {
        const { id } = c.req.valid('param');
        const { cols, rows } = c.req.valid('json');

        const sessionManager = sessions.get(id);
        if (!sessionManager) {
          return c.json({ error: 'Session not found' }, 404);
        }

        sessionManager.resize(cols, rows);

        const response: ResizeSessionResponse = {
          success: true,
        };
        return c.json(response);
      } catch (error) {
        console.error('Error resizing session:', error);
        return c.json({ error: 'Failed to resize session' }, 500);
      }
    }
  )
  .delete(
    '/:id',
    describeRoute({
      description: 'Kill and delete session',
      operationId: 'session.delete',
      responses: {
        200: {
          description: 'Session deleted successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('param', SessionIdParam),
    async (c) => {
      try {
        const { id } = c.req.valid('param');

        const sessionManager = sessions.get(id);
        if (!sessionManager) {
          return c.json({ error: 'Session not found' }, 404);
        }

        // Kill the session
        await sessionManager.kill();

        // Remove from registry
        sessions.delete(id);

        // Get eventBus from context
        const eventBus = c.get('eventBus');

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
        return c.json(response);
      } catch (error) {
        console.error('Error deleting session:', error);
        return c.json({ error: 'Failed to delete session' }, 500);
      }
    }
  );
