import { Hono } from 'hono';
import { describeRoute, resolver, validator } from 'hono-openapi';
import { randomUUID } from 'crypto';
import { join } from 'path';
import { homedir } from 'os';
import { existsSync } from 'fs';
import z from 'zod';
import { TmuxSessionManager } from '../tmux/session.js';
import { EventBus } from '../event-bus.js';
import { StateManager } from '../state/manager.js';
import { getRepoInfo } from '../git/client.js';
import { WorktreeManager } from '../git/worktree.js';
import type {
  CreateSessionResponse,
  GetSessionResponse,
  ResizeSessionResponse,
  DeleteSessionResponse,
  AgentName,
  PersistedSession,
} from '@agate/shared';
import type { SessionCreatedEvent, SessionDeletedEvent } from '@agate/shared';
import { generateWorktreeKey, AGENTS, generateBranchName } from '@agate/shared';
import { simpleGit } from 'simple-git';

async function resolveUniqueBranchName(
  repoRoot: string,
  worktreeManager: WorktreeManager,
  initialName: string
): Promise<string> {
  const git = simpleGit(repoRoot);
  const existingBranches = new Set((await git.branchLocal()).all);
  const existingWorktrees = new Set((await worktreeManager.list()).map((w) => w.branch));

  let candidate = initialName;
  const seen = new Set<string>();

  while (true) {
    if (!seen.has(candidate) && !existingBranches.has(candidate) && !existingWorktrees.has(candidate)) {
      return candidate;
    }

    seen.add(candidate);
    candidate = generateBranchName();
  }
}

// Session metadata for persistence
interface SessionMetadata {
  manager: TmuxSessionManager;
  worktreeKey: string;
  worktreePath: string;
  branch: string;
  repoName: string;
  agentName: string;
}

// In-memory session registry
export const sessions = new Map<string, SessionMetadata>();

// Define Zod schemas for validation
const validAgentNames = Object.keys(AGENTS) as [AgentName, ...AgentName[]];

const CreateSessionSchema = z.object({
  dir: z.string(),
  branchName: z.string(),
  agentName: z.enum(validAgentNames),
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
  tmuxName: z.string(),
  worktreePath: z.string(),
  branchName: z.string(),
  repoName: z.string(),
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
    stateManager: StateManager;
  };
};

export const sessionRouter = new Hono<Env>()
  .post(
    '/',
    describeRoute({
      description: 'Create a new tmux session with atomic worktree creation',
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
      const body = c.req.valid('json');
      const { dir, branchName, agentName } = body;

      let worktreePath: string | null = null;
      let resolvedBranchName = branchName;
      let sessionManager: TmuxSessionManager | null = null;

      try {
        // 1. Get repository information from dir
        const repoInfo = await getRepoInfo(dir);
        const { repoRoot, repoName } = repoInfo;

        // 2. Determine worktree path
        const worktreesDir = join(homedir(), '.agate', 'worktrees', repoName);

        const worktreeManager = new WorktreeManager(repoRoot);
        resolvedBranchName = await resolveUniqueBranchName(repoRoot, worktreeManager, branchName);

        worktreePath = join(worktreesDir, resolvedBranchName);

        // 3. Create worktree atomically
        await worktreeManager.create(worktreePath, resolvedBranchName);

        // 4. Generate session ID
        const sessionId = randomUUID();

        // 5. Get dependencies from context
        const eventBus = c.get('eventBus');
        const stateManager = c.get('stateManager');

        // 6. Create TmuxSessionManager
        sessionManager = new TmuxSessionManager(eventBus, sessionId);

        // 7. Create tmux session in the worktree
        const sessionName = `agate_${resolvedBranchName}_${agentName}_${sessionId.substring(0, 8)}`;
        await sessionManager.createSession({
          name: sessionName,
          agent: agentName as AgentName,
          cwd: worktreePath,
        });

        // 8. Generate worktree key for persistence
        const worktreeKey = generateWorktreeKey({
          repoName,
          path: worktreePath,
          branch: resolvedBranchName,
          commit: '',
          name: resolvedBranchName,
        });

        // 9. Store in registry with metadata
        sessions.set(sessionId, {
          manager: sessionManager,
          worktreeKey,
          worktreePath,
          branch: resolvedBranchName,
          repoName,
          agentName,
        });

        // 10. Persist to state.json
        const now = new Date().toISOString();
        const persistedSession: PersistedSession = {
          id: sessionId,
          worktreeKey,
          tmuxName: sessionName,
          agentName,
          worktreePath,
          branch: resolvedBranchName,
          repoName,
          createdAt: now,
          lastAccessed: now,
        };

        stateManager.saveSessionMapping(worktreeKey, persistedSession);
        await stateManager.save();

        // 11. Emit session created event
        const event: SessionCreatedEvent = {
          type: 'session.created',
          sessionId,
          timestamp: now,
        };
        eventBus.publish(event);

        // 12. Return response
        const response: CreateSessionResponse = {
          sessionId,
          tmuxName: sessionName,
          worktreePath,
          branchName: resolvedBranchName,
          repoName,
        };
        return c.json(response, 201);
      } catch (error) {
        // Rollback: Clean up worktree if it was created
        if (worktreePath && existsSync(worktreePath)) {
          try {
            const worktreeManager = new WorktreeManager(worktreePath);
            await worktreeManager.remove(worktreePath, { force: true });
          } catch (cleanupError) {
            console.error('Error rolling back worktree:', cleanupError);
          }
        }

        // Rollback: Kill session if it was created
        if (sessionManager) {
          try {
            await sessionManager.kill();
          } catch (cleanupError) {
            console.error('Error killing session during rollback:', cleanupError);
          }
        }

        // Handle specific error cases
        if (error instanceof Error && error.message === 'Not a git repository') {
          return c.json({ error: 'Not a git repository' }, 400);
        }

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

        const sessionMetadata = sessions.get(id);
        if (!sessionMetadata) {
          return c.json({ error: 'Session not found' }, 404);
        }

        const info = sessionMetadata.manager.getInfo();
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

        const sessionMetadata = sessions.get(id);
        if (!sessionMetadata) {
          return c.json({ error: 'Session not found' }, 404);
        }

        sessionMetadata.manager.resize(cols, rows);

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

        const sessionMetadata = sessions.get(id);
        if (!sessionMetadata) {
          return c.json({ error: 'Session not found' }, 404);
        }

        // Kill the session
        await sessionMetadata.manager.kill();

        // Get dependencies from context
        const eventBus = c.get('eventBus');
        const stateManager = c.get('stateManager');

        // Remove from state.json
        stateManager.removeSessionMapping(sessionMetadata.worktreeKey);
        await stateManager.save();

        // Remove from registry
        sessions.delete(id);

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
