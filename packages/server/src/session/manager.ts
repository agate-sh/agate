import type {
  Session,
  SessionWorktree,
  AgentConfig,
  PersistedSession,
  SessionState,
} from '@agate/shared';
import { generateWorktreeKey, isLinkedWorktree } from '@agate/shared';
import type { StateManager } from '../state/manager.js';

/**
 * SessionManager - Manages active sessions and their lifecycle
 *
 * Sessions are indexed by worktree key (format: "repoName:path")
 * Provides session creation, switching, deletion, and persistence
 */
export class SessionManager {
  private sessions: Map<string, Session>;
  private activeSession: Session | null;
  private stateManager: StateManager;

  constructor(stateManager: StateManager) {
    this.sessions = new Map();
    this.activeSession = null;
    this.stateManager = stateManager;
  }

  /**
   * Create a new session for the given worktree and agent
   * Throws if session already exists for this worktree
   */
  createSession(worktree: SessionWorktree, agent: AgentConfig): Session {
    if (!worktree) {
      throw new Error('worktree cannot be null');
    }

    const worktreeKey = generateWorktreeKey(worktree);

    // Check if session already exists
    if (this.sessions.has(worktreeKey)) {
      throw new Error(`Session already exists for worktree key: ${worktreeKey}`);
    }

    // Create new session
    const now = new Date();
    const session: Session = {
      id: `${worktreeKey}_${agent.name}`,
      name: `agate_${worktree.repoName}_${worktree.branch}_${agent.name}`,
      worktreeKey,
      worktree,
      agent,
      createdAt: now,
      lastAccessed: now,
      isActive: false,
    };

    // Store session
    this.sessions.set(worktreeKey, session);

    return session;
  }

  /**
   * Get existing session or create a new one
   * Updates lastAccessed time if session exists
   */
  getOrCreateSession(worktree: SessionWorktree, agent: AgentConfig): Session {
    if (!worktree) {
      throw new Error('worktree cannot be null');
    }

    const worktreeKey = generateWorktreeKey(worktree);
    const existing = this.sessions.get(worktreeKey);

    if (existing) {
      // Update access time
      existing.lastAccessed = new Date();
      return existing;
    }

    // Create new session
    return this.createSession(worktree, agent);
  }

  /**
   * Switch to the specified session by worktree key
   * Deactivates previous session and activates target session
   */
  switchToSession(worktreeKey: string): Session {
    const session = this.sessions.get(worktreeKey);

    if (!session) {
      throw new Error(`Session not found for worktree key: ${worktreeKey}`);
    }

    // Deactivate current session
    if (this.activeSession) {
      this.activeSession.isActive = false;
    }

    // Activate new session
    session.isActive = true;
    session.lastAccessed = new Date();
    this.activeSession = session;

    return session;
  }

  /**
   * Get the currently active session
   */
  getActiveSession(): Session | null {
    return this.activeSession;
  }

  /**
   * Get session for a specific worktree
   */
  getSessionForWorktree(worktree: SessionWorktree): Session | null {
    if (!worktree) {
      return null;
    }

    const worktreeKey = generateWorktreeKey(worktree);
    return this.sessions.get(worktreeKey) || null;
  }

  /**
   * List all sessions
   */
  listSessions(): Session[] {
    return Array.from(this.sessions.values());
  }

  /**
   * Get the main worktree session for a repository
   * Returns null if no main session exists
   */
  getMainSession(repoName: string): Session | null {
    for (const session of this.sessions.values()) {
      if (
        session.worktree.repoName === repoName &&
        !isLinkedWorktree(session.worktree.path)
      ) {
        return session;
      }
    }
    return null;
  }

  /**
   * Get all linked worktree sessions for a repository
   */
  getLinkedSessions(repoName: string): Session[] {
    const linkedSessions: Session[] = [];

    for (const session of this.sessions.values()) {
      if (
        session.worktree.repoName === repoName &&
        isLinkedWorktree(session.worktree.path)
      ) {
        linkedSessions.push(session);
      }
    }

    return linkedSessions;
  }

  /**
   * Delete a session by worktree key
   * Clears active session if it's being deleted
   */
  deleteSession(worktreeKey: string): void {
    const session = this.sessions.get(worktreeKey);

    if (!session) {
      throw new Error(`Session not found for worktree key: ${worktreeKey}`);
    }

    // Clear active session if it's being deleted
    if (this.activeSession === session) {
      this.activeSession = null;
    }

    // Remove from registry
    this.sessions.delete(worktreeKey);
  }

  /**
   * Persist all sessions to state
   */
  persistSessions(): void {
    const sessionMappings: Record<string, PersistedSession> = {};

    // Convert all sessions to persisted format
    for (const [key, session] of this.sessions.entries()) {
      sessionMappings[key] = {
        id: session.id,
        worktreeKey: session.worktreeKey,
        tmuxName: session.name,
        agentName: session.agent.name,
        worktreePath: session.worktree.path,
        branch: session.worktree.branch,
        repoName: session.worktree.repoName,
        createdAt: session.createdAt.toISOString(),
        lastAccessed: session.lastAccessed.toISOString(),
      };
    }

    // Update state
    const sessionState: SessionState = {
      sessionMappings,
      activeSession: this.activeSession?.worktreeKey || '',
      defaultAgent: 'claude',
    };

    this.stateManager.updateSessions(sessionState);
  }

  /**
   * Load sessions from persisted state
   */
  loadSessions(): void {
    const sessionState = this.stateManager.readSessions();
    const { sessionMappings, activeSession } = sessionState;

    // Clear existing sessions
    this.sessions.clear();
    this.activeSession = null;

    // Restore sessions from persisted data
    for (const [key, persistedSession] of Object.entries(sessionMappings)) {
      const session: Session = {
        id: persistedSession.id,
        name: persistedSession.tmuxName,
        worktreeKey: persistedSession.worktreeKey,
        worktree: {
          name: persistedSession.branch,
          path: persistedSession.worktreePath,
          branch: persistedSession.branch,
          repoName: persistedSession.repoName,
          isMain: !isLinkedWorktree(persistedSession.worktreePath),
          commit: '', // Not persisted
        },
        agent: {
          name: persistedSession.agentName,
          borderColor: '#da7756', // Default, should be looked up
          executableName: persistedSession.agentName,
          companyName: persistedSession.agentName,
        },
        createdAt: new Date(persistedSession.createdAt),
        lastAccessed: new Date(persistedSession.lastAccessed),
        isActive: false, // Will be set when restoring active session
      };

      this.sessions.set(key, session);
    }

    // Restore active session
    if (activeSession && this.sessions.has(activeSession)) {
      const session = this.sessions.get(activeSession);
      if (session) {
        session.isActive = true;
        this.activeSession = session;
      }
    }
  }
}
