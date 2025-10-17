import { Router, Request, Response } from 'express';
import type { Router as ExpressRouter } from 'express';
import { createGitClient } from '../git/client.js';
import { getFileStatus } from '../git/status.js';
import { stageFile, stageAll, unstageFile, commitAll, discardChanges } from '../git/operations.js';
import { WorktreeManager } from '../git/worktree.js';

const router: ExpressRouter = Router();

/**
 * GET /git/status
 * Get detailed file status with add/del counts
 */
router.get('/status', async (req: Request, res: Response) => {
  try {
    const { repoPath } = req.query;

    if (!repoPath || typeof repoPath !== 'string') {
      res.status(400).json({ error: 'repoPath query parameter is required' });
      return;
    }

    const git = createGitClient(repoPath);
    const status = await getFileStatus(git);

    res.json(status);
  } catch (error) {
    console.error('Error getting git status:', error);
    res.status(500).json({ error: 'Failed to get git status' });
  }
});

/**
 * GET /git/worktrees
 * List all worktrees for repository
 */
router.get('/worktrees', async (req: Request, res: Response) => {
  try {
    const { repoPath } = req.query;

    if (!repoPath || typeof repoPath !== 'string') {
      res.status(400).json({ error: 'repoPath query parameter is required' });
      return;
    }

    const manager = new WorktreeManager(repoPath);
    const worktrees = await manager.list();

    res.json({ worktrees });
  } catch (error) {
    console.error('Error listing worktrees:', error);
    res.status(500).json({ error: 'Failed to list worktrees' });
  }
});

/**
 * POST /git/worktree
 * Create worktree with optional COW
 */
router.post('/worktree', async (req: Request, res: Response) => {
  try {
    const { repoPath, path, branch, existingBranch, useCOW } = req.body;

    if (!repoPath || !path || !branch) {
      res.status(400).json({ error: 'repoPath, path, and branch are required' });
      return;
    }

    const manager = new WorktreeManager(repoPath);
    await manager.create(path, branch, { existingBranch, useCOW });

    res.json({ success: true, path, branch });
  } catch (error) {
    console.error('Error creating worktree:', error);
    res.status(500).json({ error: 'Failed to create worktree' });
  }
});

/**
 * DELETE /git/worktree
 * Delete worktree and branch
 */
router.delete('/worktree', async (req: Request, res: Response) => {
  try {
    const { repoPath, path, force } = req.body;

    if (!repoPath || !path) {
      res.status(400).json({ error: 'repoPath and path are required' });
      return;
    }

    const manager = new WorktreeManager(repoPath);
    await manager.remove(path, { force });

    res.json({ success: true });
  } catch (error) {
    console.error('Error removing worktree:', error);
    res.status(500).json({ error: 'Failed to remove worktree' });
  }
});

/**
 * POST /git/stage
 * Stage file(s)
 */
router.post('/stage', async (req: Request, res: Response) => {
  try {
    const { repoPath, file } = req.body;

    if (!repoPath || !file) {
      res.status(400).json({ error: 'repoPath and file are required' });
      return;
    }

    const git = createGitClient(repoPath);
    await stageFile(git, file);

    res.json({ success: true });
  } catch (error) {
    console.error('Error staging file:', error);
    res.status(500).json({ error: 'Failed to stage file' });
  }
});

/**
 * POST /git/stage-all
 * Stage all files
 */
router.post('/stage-all', async (req: Request, res: Response) => {
  try {
    const { repoPath } = req.body;

    if (!repoPath) {
      res.status(400).json({ error: 'repoPath is required' });
      return;
    }

    const git = createGitClient(repoPath);
    await stageAll(git);

    res.json({ success: true });
  } catch (error) {
    console.error('Error staging all files:', error);
    res.status(500).json({ error: 'Failed to stage all files' });
  }
});

/**
 * POST /git/unstage
 * Unstage file(s)
 */
router.post('/unstage', async (req: Request, res: Response) => {
  try {
    const { repoPath, file } = req.body;

    if (!repoPath || !file) {
      res.status(400).json({ error: 'repoPath and file are required' });
      return;
    }

    const git = createGitClient(repoPath);
    await unstageFile(git, file);

    res.json({ success: true });
  } catch (error) {
    console.error('Error unstaging file:', error);
    res.status(500).json({ error: 'Failed to unstage file' });
  }
});

/**
 * POST /git/commit
 * Commit all changes
 */
router.post('/commit', async (req: Request, res: Response) => {
  try {
    const { repoPath, message } = req.body;

    if (!repoPath || !message) {
      res.status(400).json({ error: 'repoPath and message are required' });
      return;
    }

    const git = createGitClient(repoPath);
    const result = await commitAll(git, message);

    res.json({ success: true, commit: result.commit });
  } catch (error) {
    console.error('Error committing:', error);
    res.status(500).json({ error: 'Failed to commit changes' });
  }
});

/**
 * POST /git/discard
 * Discard file changes
 */
router.post('/discard', async (req: Request, res: Response) => {
  try {
    const { repoPath, file } = req.body;

    if (!repoPath || !file) {
      res.status(400).json({ error: 'repoPath and file are required' });
      return;
    }

    const git = createGitClient(repoPath);
    await discardChanges(git, file);

    res.json({ success: true });
  } catch (error) {
    console.error('Error discarding changes:', error);
    res.status(500).json({ error: 'Failed to discard changes' });
  }
});

export default router;
