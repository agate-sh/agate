import type { PersistedSession } from '@agate/shared';
import type { AgentListItem, WorktreeInfo } from './types';
import { sessionToWorktreeInfo } from './types';

/**
 * Group sessions by repository name
 */
function groupSessionsByRepo(
  sessions: PersistedSession[]
): Map<string, PersistedSession[]> {
  const grouped = new Map<string, PersistedSession[]>();

  for (const session of sessions) {
    const existing = grouped.get(session.repoName) || [];
    existing.push(session);
    grouped.set(session.repoName, existing);
  }

  return grouped;
}

/**
 * Check if a worktree is the currently pinned worktree
 */
function isPinnedWorktree(
  worktree: WorktreeInfo,
  pinnedWorktree: WorktreeInfo | null
): boolean {
  if (!pinnedWorktree) return false;
  return (
    worktree.id === pinnedWorktree.id ||
    (worktree.worktreePath === pinnedWorktree.worktreePath &&
      worktree.branch === pinnedWorktree.branch)
  );
}

/**
 * Build flat list of items for rendering
 */
export function buildItemList(
  sessions: PersistedSession[],
  expandedRepos: Set<string>,
  pinnedWorktree: WorktreeInfo | null,
  currentRepo: string | null = null
): AgentListItem[] {
  const items: AgentListItem[] = [];
  const groupedSessions = groupSessionsByRepo(sessions);

  // Get sorted repository names (current repo first if set)
  let repoNames = Array.from(groupedSessions.keys());

  // Always include current repo if set, even if no sessions
  if (currentRepo && !repoNames.includes(currentRepo)) {
    repoNames.push(currentRepo);
  }

  // Sort: current repo first, then alphabetically
  repoNames.sort((a, b) => {
    if (a === currentRepo) return -1;
    if (b === currentRepo) return 1;
    return a.localeCompare(b);
  });

  for (let idx = 0; idx < repoNames.length; idx++) {
    const repoName = repoNames[idx];
    if (!repoName) continue; // Skip if undefined

    const isLastRepo = idx === repoNames.length - 1;
    const repoSessions = groupedSessions.get(repoName) || [];

    // Add repo header
    items.push({
      type: 'repo_header',
      repoName,
      repoPath: repoName, // TODO: Get actual repo path
    });

    // Only show sessions if repo is expanded
    if (expandedRepos.has(repoName)) {
      // Sort sessions by branch name
      const sortedSessions = [...repoSessions].sort((a, b) =>
        a.branch.localeCompare(b.branch)
      );

      // Add all sessions
      for (const session of sortedSessions) {
        const worktree = sessionToWorktreeInfo(session);
        items.push({
          type: 'session',
          worktree,
          isPinned: isPinnedWorktree(worktree, pinnedWorktree),
          repoName,
        });
      }

      // Gap before next repo
      if (!isLastRepo) {
        items.push({
          type: 'gap',
          repoName,
        });
      }
    }
  }

  return items;
}
