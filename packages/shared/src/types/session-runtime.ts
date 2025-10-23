import type { AgentConfig } from './agent.js';
import type { Worktree } from './git.js';

/**
 * Extended worktree information for sessions
 */
export interface SessionWorktree extends Worktree {
  /** Display name */
  name: string;
  /** Repository name */
  repoName: string;
}

/**
 * Runtime session object (not persisted, includes active connections)
 */
export interface Session {
  /** Unique session identifier */
  id: string;
  /** Internal session name */
  name: string;
  /** Stable worktree key (repo:path format) */
  worktreeKey: string;
  /** Worktree information */
  worktree: SessionWorktree;
  /** Agent configuration */
  agent: AgentConfig;
  /** Session creation timestamp */
  createdAt: Date;
  /** Last access timestamp */
  lastAccessed: Date;
  /** Whether this session is currently active */
  isActive: boolean;
}

/**
 * Helper to generate a stable worktree key from worktree info
 */
export function generateWorktreeKey(worktree: SessionWorktree): string {
  return `${worktree.repoName}:${worktree.path}`;
}

/**
 * Helper to determine whether a worktree path refers to a linked worktree.
 * Linked worktrees live under ~/.agate/worktrees/<repo>/...
 */
export function isLinkedWorktree(path: string): boolean {
  return path.includes('/.agate/worktrees/');
}
