import { SimpleGit } from 'simple-git';
import { unlink } from 'fs/promises';
import { join } from 'path';

/**
 * Stages a single file.
 */
export async function stageFile(git: SimpleGit, path: string): Promise<void> {
  await git.add(path);
}

/**
 * Stages all files (tracked and untracked).
 */
export async function stageAll(git: SimpleGit): Promise<void> {
  await git.add('.');
}

/**
 * Unstages a file.
 */
export async function unstageFile(git: SimpleGit, path: string): Promise<void> {
  await git.reset(['HEAD', '--', path]);
}

/**
 * Commits all staged changes.
 */
export async function commitAll(git: SimpleGit, message: string): Promise<{ commit: string }> {
  if (!message || message.trim().length === 0) {
    throw new Error('Commit message cannot be empty');
  }

  const status = await git.status();
  if (status.staged.length === 0) {
    throw new Error('No changes staged for commit');
  }

  const result = await git.commit(message);
  return { commit: result.commit };
}

/**
 * Discards changes to a file.
 * For tracked files, restores from HEAD.
 * For untracked files, deletes the file.
 */
export async function discardChanges(git: SimpleGit, path: string): Promise<void> {
  const status = await git.status();
  const file = status.files.find(f => f.path === path);

  if (!file) {
    throw new Error(`File not found in git status: ${path}`);
  }

  // If the file is untracked, delete it
  if (file.index === '?' && file.working_dir === '?') {
    const repoPath = await git.revparse(['--show-toplevel']);
    const filePath = join(repoPath.trim(), path);
    await unlink(filePath);
  } else {
    // For tracked files, restore from HEAD
    await git.checkout(['--', path]);
  }
}
