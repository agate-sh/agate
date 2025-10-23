import { simpleGit, SimpleGit } from 'simple-git';
import { mkdtemp, rm } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';

/**
 * Creates a temporary Git repository for testing.
 * Returns the path and a simple-git instance.
 */
export async function createTestRepo(): Promise<{ path: string; git: SimpleGit }> {
  const path = await mkdtemp(join(tmpdir(), 'agate-test-'));
  const git = simpleGit(path);

  await git.init(['-b', 'main']); // Use 'main' as the default branch
  await git.addConfig('user.name', 'Test User');
  await git.addConfig('user.email', 'test@example.com');

  return { path, git };
}

/**
 * Cleans up a test repository.
 */
export async function cleanupTestRepo(path: string): Promise<void> {
  await rm(path, { recursive: true, force: true });
}

/**
 * Creates a file with content in the test repo.
 */
export async function createFile(git: SimpleGit, filename: string, content: string): Promise<void> {
  const fs = await import('fs/promises');
  const path = await git.revparse(['--show-toplevel']);
  await fs.writeFile(join(path.trim(), filename), content);
}

/**
 * Creates an initial commit in a test repo.
 */
export async function createInitialCommit(git: SimpleGit): Promise<void> {
  await createFile(git, 'README.md', '# Test Repo\n');
  await git.add('README.md');
  await git.commit('Initial commit');
}
