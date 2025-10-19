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
 * Get main session for a repository (session with worktreePath matching repo name)
 */
function getMainSession(
  sessions: PersistedSession[]
): PersistedSession | null {
  // Main session is the one where worktreePath doesn't contain '.git/worktrees'
  const mainSession = sessions.find(
    (s) => !s.worktreePath.includes('.git/worktrees')
  );
  return mainSession || null;
}

/**
 * Get linked sessions for a repository (sessions with worktreePath containing '.git/worktrees')
 */
function getLinkedSessions(sessions: PersistedSession[]): PersistedSession[] {
  return sessions
    .filter((s) => s.worktreePath.includes('.git/worktrees'))
    .sort((a, b) => a.branch.localeCompare(b.branch));
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
 * Mirrors the Go implementation's buildItemList function
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
      const mainSession = getMainSession(repoSessions);
      const linkedSessions = getLinkedSessions(repoSessions);

      // Main worktree section
      items.push({
        type: 'section_header',
        sectionTitle: 'Main worktree',
        repoName,
      });

      if (mainSession) {
        const worktree = sessionToWorktreeInfo(mainSession, true);
        items.push({
          type: 'main_session',
          worktree,
          isPinned: isPinnedWorktree(worktree, pinnedWorktree),
          repoName,
        });
      } else {
        items.push({
          type: 'empty_message',
          sectionTitle: 'main',
          repoName,
        });
      }

      // Gap between sections
      items.push({
        type: 'gap',
        repoName,
      });

      // Linked worktrees section
      items.push({
        type: 'section_header',
        sectionTitle: 'Linked worktrees',
        repoName,
      });

      if (linkedSessions.length > 0) {
        for (const session of linkedSessions) {
          const worktree = sessionToWorktreeInfo(session, false);
          items.push({
            type: 'linked_session',
            worktree,
            isPinned: isPinnedWorktree(worktree, pinnedWorktree),
            repoName,
          });
        }
      } else {
        items.push({
          type: 'empty_message',
          sectionTitle: 'linked',
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
