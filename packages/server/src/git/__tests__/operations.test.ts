import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createTestRepo, cleanupTestRepo, createFile, createInitialCommit } from './test-helpers';
import { stageFile, stageAll, unstageFile, commitAll, discardChanges } from '../operations';
import { SimpleGit } from 'simple-git';

describe('Git Operations', () => {
  let repoPath: string;
  let git: SimpleGit;

  beforeEach(async () => {
    const testRepo = await createTestRepo();
    repoPath = testRepo.path;
    git = testRepo.git;
    await createInitialCommit(git);
  });

  afterEach(async () => {
    await cleanupTestRepo(repoPath);
  });

  describe('stageFile', () => {
    it('should stage a single file', async () => {
      await createFile(git, 'test.txt', 'content\n');

      await stageFile(git, 'test.txt');

      const status = await git.status();
      const stagedFile = status.files.find(f => f.path === 'test.txt');
      expect(stagedFile?.index).toBe('A');
    });
  });

  describe('stageAll', () => {
    it('should stage all files', async () => {
      await createFile(git, 'file1.txt', 'content1\n');
      await createFile(git, 'file2.txt', 'content2\n');

      await stageAll(git);

      const status = await git.status();
      expect(status.staged.length).toBe(2);
    });
  });

  describe('unstageFile', () => {
    it('should unstage a staged file', async () => {
      await createFile(git, 'test.txt', 'content\n');
      await git.add('test.txt');

      await unstageFile(git, 'test.txt');

      const status = await git.status();
      const file = status.files.find(f => f.path === 'test.txt');
      expect(file?.index).toBe('?');
    });
  });

  describe('commitAll', () => {
    it('should commit all staged changes', async () => {
      await createFile(git, 'test.txt', 'content\n');
      await git.add('test.txt');

      const result = await commitAll(git, 'Test commit');

      expect(result.commit).toBeDefined();
      const status = await git.status();
      expect(status.files.length).toBe(0);
    });

    it('should fail with empty commit message', async () => {
      await createFile(git, 'test.txt', 'content\n');
      await git.add('test.txt');

      await expect(commitAll(git, '')).rejects.toThrow();
    });

    it('should fail when no changes are staged', async () => {
      await expect(commitAll(git, 'Test commit')).rejects.toThrow();
    });
  });

  describe('discardChanges', () => {
    it('should discard unstaged changes to a file', async () => {
      await createFile(git, 'README.md', 'Modified content\n');

      await discardChanges(git, 'README.md');

      const status = await git.status();
      expect(status.files.length).toBe(0);
    });

    it('should handle untracked files by removing them', async () => {
      await createFile(git, 'untracked.txt', 'content\n');

      await discardChanges(git, 'untracked.txt');

      const status = await git.status();
      expect(status.not_added).not.toContain('untracked.txt');
    });
  });
});
