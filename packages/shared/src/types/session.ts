/**
 * Persisted session data
 */
export interface PersistedSession {
  /** Unique session identifier */
  id: string;
  /** Unique key identifying the worktree */
  worktreeKey: string;
  /** Tmux session name */
  tmuxName: string;
  /** Agent used for this session */
  agentName: string;
  /** Path to the worktree */
  worktreePath: string;
  /** Git branch name */
  branch: string;
  /** Repository name */
  repoName: string;
  /** Session creation timestamp */
  createdAt: string;
  /** Last access timestamp */
  lastAccessed: string;
}

/**
 * Session state management
 */
export interface SessionState {
  /** Map of worktree keys to persisted sessions */
  sessionMappings: Record<string, PersistedSession>;
  /** Currently active session key */
  activeSession: string;
  /** Default agent for new sessions */
  defaultAgent: string;
}
