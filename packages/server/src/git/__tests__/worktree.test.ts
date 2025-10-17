import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from './test-helpers';
import { WorktreeManager } from '../worktree';
import { SimpleGit } from 'simple-git';
import { realpathSync } from 'fs';

function normalizePath(path: string): string {
  return realpathSync(path);
}

describe('Worktree Operations', () => {
  let repoPath: string;
  let git: SimpleGit;
  let worktreeManager: WorktreeManager;

  beforeEach(async () => {
    const testRepo = await createTestRepo();
    repoPath = testRepo.path;
    git = testRepo.git;
    await createInitialCommit(git);
    worktreeManager = new WorktreeManager(repoPath);
  });

  afterEach(async () => {
    await cleanupTestRepo(repoPath);
  });

  describe('list', () => {
    it('should list existing worktrees', async () => {
      const worktrees = await worktreeManager.list();

      expect(worktrees).toHaveLength(1);
      expect(normalizePath(worktrees[0]?.path ?? '')).toBe(normalizePath(repoPath));
      expect(worktrees[0]?.branch).toBe('main');
    });
  });

  describe('create', () => {
    it('should create a new worktree', async () => {
      const newPath = `${repoPath}-feature`;
      await worktreeManager.create(newPath, 'feature-branch');

      const worktrees = await worktreeManager.list();
      expect(worktrees).toHaveLength(2);
      const newWorktree = worktrees.find(w => w.branch === 'feature-branch');
      expect(newWorktree).toBeDefined();
      expect(normalizePath(newWorktree?.path ?? '')).toBe(normalizePath(newPath));
    });

    it('should create a worktree from existing branch', async () => {
      // Create a branch first
      await git.checkoutLocalBranch('existing-branch');
      await git.checkout('main');

      const newPath = `${repoPath}-existing`;
      await worktreeManager.create(newPath, 'existing-branch', { existingBranch: true });

      const worktrees = await worktreeManager.list();
      const newWorktree = worktrees.find(w => w.branch === 'existing-branch');
      expect(newWorktree).toBeDefined();
    });
  });

  describe('remove', () => {
    it('should remove a worktree', async () => {
      const newPath = `${repoPath}-temp`;
      await worktreeManager.create(newPath, 'temp-branch');

      await worktreeManager.remove(newPath);

      const worktrees = await worktreeManager.list();
      expect(worktrees.every(w => w.path !== newPath)).toBe(true);
    });
  });

  describe('prune', () => {
    it('should prune deleted worktrees', async () => {
      const newPath = `${repoPath}-prune`;
      await worktreeManager.create(newPath, 'prune-branch');

      // Manually delete the directory
      const { rm } = await import('fs/promises');
      await rm(newPath, { recursive: true, force: true });

      await worktreeManager.prune();

      const worktrees = await worktreeManager.list();
      expect(worktrees.every(w => w.path !== newPath)).toBe(true);
    });
  });
});
