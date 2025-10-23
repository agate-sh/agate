import { useState, useMemo, useCallback, useEffect } from 'react';
import type { PersistedSession } from '@agate/shared';
import type { AgentListItem, WorktreeInfo } from '../components/AgentsPane/types';
import { buildItemList } from '../components/AgentsPane/buildItemList';

export interface AgentsState {
  sessions: PersistedSession[];
  items: AgentListItem[];
  expandedRepos: Set<string>;
  pinnedWorktree: WorktreeInfo | null;
  selectedIndex: number;
  currentRepo: string | null;
}

export interface AgentsStateActions {
  setSessions: (
    sessions: PersistedSession[] | ((prev: PersistedSession[]) => PersistedSession[])
  ) => void;
  toggleRepo: (repoName: string) => void;
  setPinnedWorktree: (worktree: WorktreeInfo | null) => void;
  setSelectedIndex: (index: number) => void;
  moveSelectionUp: () => void;
  moveSelectionDown: () => void;
}

/**
 * State management hook for AgentsPane
 * Manages sessions, expanded repos, pinned worktree, and selection
 */
export function useAgentsState(): AgentsState & AgentsStateActions {
  const [sessions, setSessions] = useState<PersistedSession[]>([]);
  const [expandedRepos, setExpandedRepos] = useState<Set<string>>(new Set());
  const [pinnedWorktree, setPinnedWorktree] = useState<WorktreeInfo | null>(
    null
  );
  const [selectedIndex, setSelectedIndex] = useState(0);
  // Build flat item list whenever dependencies change
  const items = useMemo(
    () => buildItemList(sessions, expandedRepos, pinnedWorktree),
    [sessions, expandedRepos, pinnedWorktree]
  );

  // Ensure selection index stays within bounds when item list changes
  useEffect(() => {
    if (items.length === 0) {
      setSelectedIndex(0);
      return;
    }

    setSelectedIndex((prev) => Math.min(prev, items.length - 1));
  }, [items.length]);

  // Derive the current repository from the selected list item
  const currentRepo = useMemo(() => {
    if (items.length === 0) {
      return null;
    }

    const clampedIndex = Math.min(selectedIndex, items.length - 1);
    const selectedItem = items[clampedIndex];

    if (!selectedItem) {
      return null;
    }

    if (selectedItem.type === 'session') {
      return selectedItem.worktree.repoName;
    }

    return selectedItem.repoName;
  }, [items, selectedIndex]);

  // Auto-expand first repo if none are expanded
  useEffect(() => {
    if (sessions.length === 0) {
      return;
    }

    const firstRepo = sessions[0]?.repoName;
    if (!firstRepo) {
      return;
    }

    setExpandedRepos((prev) => {
      if (prev.size > 0 || prev.has(firstRepo)) {
        return prev;
      }

      const next = new Set(prev);
      next.add(firstRepo);
      return next;
    });
  }, [sessions]);

  // Toggle repo expansion
  const toggleRepo = useCallback((repoName: string) => {
    setExpandedRepos((prev) => {
      const next = new Set(prev);
      if (next.has(repoName)) {
        next.delete(repoName);
      } else {
        next.add(repoName);
      }
      return next;
    });
  }, []);

  // Navigate selection up
  const moveSelectionUp = useCallback(() => {
    setSelectedIndex((prev) => Math.max(0, prev - 1));
  }, []);

  // Navigate selection down
  const moveSelectionDown = useCallback(() => {
    setSelectedIndex((prev) => Math.min(items.length - 1, prev + 1));
  }, [items.length]);

  return {
    // State
    sessions,
    items,
    expandedRepos,
    pinnedWorktree,
    selectedIndex,
    currentRepo,

    // Actions
    setSessions,
    toggleRepo,
    setPinnedWorktree,
    setSelectedIndex,
    moveSelectionUp,
    moveSelectionDown,
  };
}
