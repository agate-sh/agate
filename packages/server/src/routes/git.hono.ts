import { Hono } from 'hono';
import { describeRoute, resolver, validator } from 'hono-openapi';
import z from 'zod';
import { createGitClient } from '../git/client.js';
import { getFileStatus, getCurrentBranch } from '../git/status.js';
import { stageFile, stageAll, unstageFile, commitAll, discardChanges } from '../git/operations.js';
import { WorktreeManager } from '../git/worktree.js';
import { GitStatusSchema, WorktreesResponseSchema } from '@agate/shared';

// Query schemas
const RepoPathQuery = z.object({
  repoPath: z.string(),
});

// Request body schemas
const WorktreeCreateSchema = z.object({
  repoPath: z.string(),
  path: z.string(),
  branch: z.string(),
  existingBranch: z.string().optional(),
  useCOW: z.boolean().optional(),
});

const WorktreeDeleteSchema = z.object({
  repoPath: z.string(),
  path: z.string(),
  force: z.boolean().optional(),
});

const StageFileSchema = z.object({
  repoPath: z.string(),
  file: z.string(),
});

const StageAllSchema = z.object({
  repoPath: z.string(),
});

const UnstageFileSchema = z.object({
  repoPath: z.string(),
  file: z.string(),
});

const CommitSchema = z.object({
  repoPath: z.string(),
  message: z.string(),
});

const DiscardChangesSchema = z.object({
  repoPath: z.string(),
  file: z.string(),
});

// Response schemas
const SuccessResponseSchema = z.object({
  success: z.boolean(),
});

const WorktreeCreateResponseSchema = z.object({
  success: z.boolean(),
  path: z.string(),
  branch: z.string(),
});

const CommitResponseSchema = z.object({
  success: z.boolean(),
  commit: z.string(),
});

const CurrentBranchResponseSchema = z.object({
  branch: z.string(),
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
  500: {
    description: 'Internal server error',
    content: {
      'application/json': {
        schema: resolver(z.object({ error: z.string() })),
      },
    },
  },
} as const;

export const gitRouter = new Hono()
  .get(
    '/status',
    describeRoute({
      description: 'Get detailed file status with add/del counts',
      operationId: 'git.status',
      responses: {
        200: {
          description: 'Git file status',
          content: {
            'application/json': {
              schema: resolver(GitStatusSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('query', RepoPathQuery),
    async (c) => {
      try {
        const { repoPath } = c.req.valid('query');

        const git = createGitClient(repoPath);
        const status = await getFileStatus(git);

        return c.json(status);
      } catch (error) {
        console.error('Error getting git status:', error);
        return c.json({ error: 'Failed to get git status' }, 500);
      }
    }
  )
  .get(
    '/head',
    describeRoute({
      description: 'Get current branch (HEAD)',
      operationId: 'git.head',
      responses: {
        200: {
          description: 'Current branch name',
          content: {
            'application/json': {
              schema: resolver(CurrentBranchResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('query', RepoPathQuery),
    async (c) => {
      try {
        const { repoPath } = c.req.valid('query');

        const git = createGitClient(repoPath);
        const branch = await getCurrentBranch(git);

        if (!branch) {
          return c.json({ error: 'Could not determine current branch' }, 500);
        }

        return c.json({ branch });
      } catch (error) {
        console.error('Error getting current branch:', error);
        return c.json({ error: 'Failed to get current branch' }, 500);
      }
    }
  )
  .get(
    '/worktrees',
    describeRoute({
      description: 'List all worktrees for repository',
      operationId: 'git.worktrees.list',
      responses: {
        200: {
          description: 'List of worktrees',
          content: {
            'application/json': {
              schema: resolver(WorktreesResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('query', RepoPathQuery),
    async (c) => {
      try {
        const { repoPath } = c.req.valid('query');

        const manager = new WorktreeManager(repoPath);
        const worktrees = await manager.list();

        return c.json({ worktrees });
      } catch (error) {
        console.error('Error listing worktrees:', error);
        return c.json({ error: 'Failed to list worktrees' }, 500);
      }
    }
  )
  .post(
    '/worktree',
    describeRoute({
      description: 'Create worktree with optional COW',
      operationId: 'git.worktree.create',
      responses: {
        200: {
          description: 'Worktree created successfully',
          content: {
            'application/json': {
              schema: resolver(WorktreeCreateResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', WorktreeCreateSchema),
    async (c) => {
      try {
        const body = c.req.valid('json');
        const { repoPath, path, branch, existingBranch, useCOW } = body;

        const manager = new WorktreeManager(repoPath);
        await manager.create(path, branch, {
          existingBranch: existingBranch as boolean | undefined,
          useCOW: useCOW as boolean | undefined
        });

        return c.json({ success: true, path, branch });
      } catch (error) {
        console.error('Error creating worktree:', error);
        return c.json({ error: 'Failed to create worktree' }, 500);
      }
    }
  )
  .delete(
    '/worktree',
    describeRoute({
      description: 'Delete worktree and branch',
      operationId: 'git.worktree.delete',
      responses: {
        200: {
          description: 'Worktree deleted successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', WorktreeDeleteSchema),
    async (c) => {
      try {
        const { repoPath, path, force } = c.req.valid('json');

        const manager = new WorktreeManager(repoPath);
        await manager.remove(path, { force });

        return c.json({ success: true });
      } catch (error) {
        console.error('Error removing worktree:', error);
        return c.json({ error: 'Failed to remove worktree' }, 500);
      }
    }
  )
  .post(
    '/stage',
    describeRoute({
      description: 'Stage file(s)',
      operationId: 'git.stage',
      responses: {
        200: {
          description: 'File staged successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', StageFileSchema),
    async (c) => {
      try {
        const { repoPath, file } = c.req.valid('json');

        const git = createGitClient(repoPath);
        await stageFile(git, file);

        return c.json({ success: true });
      } catch (error) {
        console.error('Error staging file:', error);
        return c.json({ error: 'Failed to stage file' }, 500);
      }
    }
  )
  .post(
    '/stage-all',
    describeRoute({
      description: 'Stage all files',
      operationId: 'git.stageAll',
      responses: {
        200: {
          description: 'All files staged successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', StageAllSchema),
    async (c) => {
      try {
        const { repoPath } = c.req.valid('json');

        const git = createGitClient(repoPath);
        await stageAll(git);

        return c.json({ success: true });
      } catch (error) {
        console.error('Error staging all files:', error);
        return c.json({ error: 'Failed to stage all files' }, 500);
      }
    }
  )
  .post(
    '/unstage',
    describeRoute({
      description: 'Unstage file(s)',
      operationId: 'git.unstage',
      responses: {
        200: {
          description: 'File unstaged successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', UnstageFileSchema),
    async (c) => {
      try {
        const { repoPath, file } = c.req.valid('json');

        const git = createGitClient(repoPath);
        await unstageFile(git, file);

        return c.json({ success: true });
      } catch (error) {
        console.error('Error unstaging file:', error);
        return c.json({ error: 'Failed to unstage file' }, 500);
      }
    }
  )
  .post(
    '/commit',
    describeRoute({
      description: 'Commit all changes',
      operationId: 'git.commit',
      responses: {
        200: {
          description: 'Changes committed successfully',
          content: {
            'application/json': {
              schema: resolver(CommitResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', CommitSchema),
    async (c) => {
      try {
        const { repoPath, message } = c.req.valid('json');

        const git = createGitClient(repoPath);
        const result = await commitAll(git, message);

        return c.json({ success: true, commit: result.commit });
      } catch (error) {
        console.error('Error committing:', error);
        return c.json({ error: 'Failed to commit changes' }, 500);
      }
    }
  )
  .post(
    '/discard',
    describeRoute({
      description: 'Discard file changes',
      operationId: 'git.discard',
      responses: {
        200: {
          description: 'Changes discarded successfully',
          content: {
            'application/json': {
              schema: resolver(SuccessResponseSchema),
            },
          },
        },
        ...ERRORS,
      },
    }),
    validator('json', DiscardChangesSchema),
    async (c) => {
      try {
        const { repoPath, file } = c.req.valid('json');

        const git = createGitClient(repoPath);
        await discardChanges(git, file);

        return c.json({ success: true });
      } catch (error) {
        console.error('Error discarding changes:', error);
        return c.json({ error: 'Failed to discard changes' }, 500);
      }
    }
  );
