import type { SessionState } from './session.js';

/**
 * Worktree reference for tracking repository locations
 */
export interface WorktreeRef {
  /** Absolute path to the worktree */
  path: string;
  /** Git branch name */
  branch: string;
}

/**
 * Repository selection state - tracks last selected worktree per repo
 */
export interface RepoSelection {
  /** Last selected worktree reference for this repo */
  worktreeRef: WorktreeRef;
}

/**
 * Workspace state - manages repositories and selections
 */
export interface WorkspaceState {
  /** Map of repository names to their paths */
  repositories: Record<string, string>;
  /** Map of repository names to their last selected worktree */
  repoSelections: Record<string, RepoSelection>;
  /** Name of the last active repository */
  lastRepo: string;
}

/**
 * UI state - welcome dialog and other UI preferences
 */
export interface UIState {
  /** Whether to show the welcome dialog */
  showWelcomeDialog: boolean;
}

/**
 * Root application state
 */
export interface AppState {
  /** State version for migrations */
  version: number;
  /** UI preferences */
  ui: UIState;
  /** Workspace and repository state */
  workspace: WorkspaceState;
  /** Session state */
  sessions: SessionState;
}
