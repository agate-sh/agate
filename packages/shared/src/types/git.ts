/**
 * Git repository status information
 */
export interface GitStatus {
  /** Current branch name */
  branch: string;
  /** Whether there are uncommitted changes */
  isDirty: boolean;
  /** Number of files staged for commit */
  stagedFiles: number;
  /** Number of modified files */
  modifiedFiles: number;
  /** Number of untracked files */
  untrackedFiles: number;
  /** Commits ahead of remote */
  ahead: number;
  /** Commits behind remote */
  behind: number;
}

/**
 * Git worktree information
 */
export interface Worktree {
  /** Worktree path */
  path: string;
  /** Branch associated with worktree */
  branch: string;
  /** Commit hash */
  commit: string;
}
