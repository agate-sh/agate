import { simpleGit, SimpleGit, SimpleGitOptions } from 'simple-git';

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
