import { simpleGit, SimpleGit } from 'simple-git';
import { platform } from 'os';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

export interface Worktree {
  path: string;
  branch: string;
  commit: string;
  bare: boolean;
  detached: boolean;
}

export interface CreateWorktreeOptions {
  existingBranch?: boolean;
  useCOW?: boolean;
}

/**
 * Manages Git worktrees for a repository.
 */
export class WorktreeManager {
  private git: SimpleGit;
  private repoPath: string;

  constructor(repoPath: string) {
    this.repoPath = repoPath;
    this.git = simpleGit(repoPath);
  }

  /**
   * Lists all worktrees in the repository.
   */
  async list(): Promise<Worktree[]> {
    const result = await this.git.raw(['worktree', 'list', '--porcelain']);
    return this.parseWorktreeList(result);
  }

  /**
   * Creates a new worktree.
   */
  async create(
    path: string,
    branch: string,
    options: CreateWorktreeOptions = {}
  ): Promise<void> {
    const args = ['worktree', 'add'];

    if (options.existingBranch) {
      args.push(path, branch);
    } else {
      args.push('-b', branch, path);
    }

    // Check if COW is available and requested
    if (options.useCOW && (await this.isCOWAvailable())) {
      // For APFS on macOS, we can use COW
      // This needs to be done before git worktree add
      if (platform() === 'darwin') {
        await execAsync(`cp -c -R "${this.repoPath}" "${path}"`);
        // Then convert it to a worktree
        await this.git.raw(['worktree', 'add', '--force', path, branch]);
        return;
      }
    }

    await this.git.raw(args);
  }

  /**
   * Removes a worktree.
   */
  async remove(path: string, options?: { force?: boolean }): Promise<void> {
    const args = ['worktree', 'remove', path];
    if (options?.force) {
      args.push('--force');
    }
    await this.git.raw(args);
  }

  /**
   * Prunes worktree information for deleted worktrees.
   */
  async prune(): Promise<void> {
    await this.git.raw(['worktree', 'prune']);
  }

  /**
   * Checks if Copy-on-Write is available on the current filesystem.
   * Currently only supports macOS APFS.
   */
  private async isCOWAvailable(): Promise<boolean> {
    if (platform() !== 'darwin') {
      return false;
    }

    try {
      // Check if the filesystem is APFS
      const { stdout } = await execAsync(`df -T apfs "${this.repoPath}"`);
      return stdout.includes('apfs');
    } catch {
      return false;
    }
  }

  /**
   * Parses the output of `git worktree list --porcelain`.
   */
  private parseWorktreeList(output: string): Worktree[] {
    const worktrees: Worktree[] = [];
    const lines = output.trim().split('\n');

    let current: Partial<Worktree> = {};

    for (const line of lines) {
      if (line.startsWith('worktree ')) {
        current.path = line.substring('worktree '.length);
      } else if (line.startsWith('HEAD ')) {
        current.commit = line.substring('HEAD '.length);
      } else if (line.startsWith('branch ')) {
        const branchRef = line.substring('branch '.length);
        current.branch = branchRef.replace('refs/heads/', '');
        current.detached = false;
      } else if (line === 'detached') {
        current.detached = true;
        current.branch = 'HEAD';
      } else if (line === 'bare') {
        current.bare = true;
      } else if (line === '') {
        // Empty line indicates end of worktree entry
        if (current.path && current.commit !== undefined) {
          worktrees.push({
            path: current.path,
            branch: current.branch ?? 'HEAD',
            commit: current.commit,
            bare: current.bare ?? false,
            detached: current.detached ?? false,
          });
        }
        current = {};
      }
    }

    // Handle last entry if file doesn't end with empty line
    if (current.path && current.commit !== undefined) {
      worktrees.push({
        path: current.path,
        branch: current.branch ?? 'HEAD',
        commit: current.commit,
        bare: current.bare ?? false,
        detached: current.detached ?? false,
      });
    }

    return worktrees;
  }
}
