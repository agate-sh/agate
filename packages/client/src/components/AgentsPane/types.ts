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
  isMain: boolean; // Whether this is the main worktree
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
      type: 'section_header';
      sectionTitle: 'Main worktree' | 'Linked worktrees';
      repoName: string;
    }
  | {
      type: 'main_session';
      worktree: WorktreeInfo;
      isPinned: boolean;
      repoName: string;
    }
  | {
      type: 'linked_session';
      worktree: WorktreeInfo;
      isPinned: boolean;
      repoName: string;
    }
  | {
      type: 'empty_message';
      sectionTitle: 'main' | 'linked';
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
  session: PersistedSession,
  isMain: boolean = false
): WorktreeInfo {
  return {
    id: session.id,
    worktreePath: session.worktreePath,
    branch: session.branch,
    agentName: session.agentName,
    repoName: session.repoName,
    tmuxName: session.tmuxName,
    isMain,
  };
}
