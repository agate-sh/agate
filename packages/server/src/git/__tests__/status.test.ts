import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createTestRepo, cleanupTestRepo, createFile, createInitialCommit } from './test-helpers';
import { getFileStatus } from '../status';
import { SimpleGit } from 'simple-git';

describe('Git Status Operations', () => {
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

  it('should detect unstaged modification', async () => {
    // Modify file without staging
    await createFile(git, 'README.md', '# Modified\n');

    const status = await getFileStatus(git);

    expect(status.files).toHaveLength(1);
    expect(status.files[0]).toMatchObject({
      path: 'README.md',
      working_dir: 'M',
      index: ' ',
    });
    expect(status.modified).toContain('README.md');
  });

  it('should detect staged modification', async () => {
    // Modify and stage file
    await createFile(git, 'README.md', '# Modified\n');
    await git.add('README.md');

    const status = await getFileStatus(git);

    expect(status.files).toHaveLength(1);
    expect(status.files[0]).toMatchObject({
      path: 'README.md',
      working_dir: ' ',
      index: 'M',
    });
    expect(status.staged).toContain('README.md');
  });

  it('should calculate additions and deletions', async () => {
    // Add lines to README
    await createFile(git, 'README.md', '# Test Repo\nLine 2\nLine 3\n');

    const status = await getFileStatus(git);

    expect(status.files[0]?.additions).toBeGreaterThan(0);
    expect(status.files[0]?.deletions).toBeGreaterThanOrEqual(0);
  });

  it('should detect untracked files', async () => {
    await createFile(git, 'new-file.txt', 'New content\n');

    const status = await getFileStatus(git);

    expect(status.not_added).toContain('new-file.txt');
  });
});
