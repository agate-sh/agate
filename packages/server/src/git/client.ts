import { simpleGit, SimpleGit, SimpleGitOptions } from 'simple-git';
import { basename } from 'path';

/**
 * Creates a configured simple-git instance for a repository.
 * @param baseDir - The repository path
 * @param options - Additional simple-git options
 */
export function createGitClient(baseDir: string, options?: Partial<SimpleGitOptions>): SimpleGit {
  return simpleGit(baseDir, {
    binary: 'git',
    maxConcurrentProcesses: 6,
    trimmed: true,
    ...options,
  });
}

export interface RepoInfo {
  repoRoot: string;
  repoName: string;
  currentBranch: string;
}

/**
 * Get repository information from a given directory path.
 * Works even if dir is a subdirectory of the repo.
 * @param dir - Any path within a git repository
 * @returns Repository info including root path, name, and current branch
 * @throws Error if not in a git repository
 */
export async function getRepoInfo(dir: string): Promise<RepoInfo> {
  const git = createGitClient(dir);

  // Check if we're in a git repository
  const isRepo = await git.checkIsRepo();
  if (!isRepo) {
    throw new Error('Not a git repository');
  }

  // Get the repository root (works from any subdirectory)
  const repoRoot = await git.revparse(['--show-toplevel']);

  // Get current branch
  const branch = await git.revparse(['--abbrev-ref', 'HEAD']);

  // Extract repo name from root path
  const repoName = basename(repoRoot);

  return {
    repoRoot,
    repoName,
    currentBranch: branch,
  };
}
