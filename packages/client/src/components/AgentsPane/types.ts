import type { PersistedSession } from '@agate/shared';

/**
 * Worktree information for UI display
 * Derived from PersistedSession
 */
export interface WorktreeInfo {
  id: string;
  worktreePath: string;
  branch: string;
  agentName: string;
  repoName: string;
  tmuxName: string;
}

/**
 * List item types for the agents pane
 * Represents different types of rows in the hierarchical list
 */
export type AgentListItem =
  | {
      type: 'repo_header';
      repoName: string;
      repoPath: string;
    }
  | {
      type: 'session';
      worktree: WorktreeInfo;
      isPinned: boolean;
      repoName: string;
    }
  | {
      type: 'gap';
      repoName: string;
    };

/**
 * Convert PersistedSession to WorktreeInfo
 */
export function sessionToWorktreeInfo(
  session: PersistedSession
): WorktreeInfo {
  return {
    id: session.id,
    worktreePath: session.worktreePath,
    branch: session.branch,
    agentName: session.agentName,
    repoName: session.repoName,
    tmuxName: session.tmuxName,
  };
}
