import { describe, it, expect, beforeEach } from 'vitest';
import type {
  SessionWorktree,
  PersistedSession,
  SessionState,
  AgentConfig,
} from '@agate/shared';
import { AGENTS } from '@agate/shared';
import { SessionManager } from '../manager.js';
import { StateManager } from '../../state/manager.js';

// Mock StateManager for testing
class MockStateManager {
  private sessionMappings: Record<string, PersistedSession> = {};
  private activeSession = '';

  saveSessionMapping(key: string, session: PersistedSession): void {
    this.sessionMappings[key] = session;
  }

  removeSessionMapping(key: string): void {
    delete this.sessionMappings[key];
  }

  getSessionMappings(): Record<string, PersistedSession> {
    return { ...this.sessionMappings };
  }

  setActiveSession(key: string): void {
    this.activeSession = key;
  }

  getActiveSession(): string {
    return this.activeSession;
  }

  updateSessions(sessions: SessionState): void {
    this.sessionMappings = sessions.sessionMappings;
    this.activeSession = sessions.activeSession;
  }

  readSessions(): SessionState {
    return {
      sessionMappings: this.sessionMappings,
      activeSession: this.activeSession,
      defaultAgent: 'claude',
    };
  }
}

describe('SessionManager', () => {
  let sessionManager: SessionManager;
  let stateManager: MockStateManager;

  beforeEach(() => {
    stateManager = new MockStateManager();
    sessionManager = new SessionManager(stateManager as unknown as StateManager);
  });

  const createMockWorktree = (
    repoName: string,
    path: string,
    branch: string
  ): SessionWorktree => ({
    name: branch,
    path,
    branch,
    repoName,
    commit: 'abc123',
    isMain: !path.includes('/.agate/worktrees/'),
    bare: false,
    detached: false,
  });

  const createMockAgent = (name: string = 'claude'): AgentConfig => {
    return AGENTS[name as keyof typeof AGENTS] || AGENTS.claude;
  };

  describe('Initialization', () => {
    it('should initialize with empty session registry', () => {
      const sessions = sessionManager.listSessions();
      expect(sessions).toHaveLength(0);
    });

    it('should have no active session initially', () => {
      const activeSession = sessionManager.getActiveSession();
      expect(activeSession).toBeNull();
    });
  });

  describe('Session Creation', () => {
    it('should create a new session for a worktree', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent('claude');

      const session = sessionManager.createSession(worktree, agent);

      expect(session).toBeDefined();
      expect(session.worktree).toBe(worktree);
      expect(session.agent).toBe(agent);
      expect(session.worktreeKey).toBe('test-repo:/path/to/repo');
      expect(session.id).toContain('test-repo');
      expect(session.isActive).toBe(false);
    });

    it('should throw error when worktree is null', () => {
      const agent = createMockAgent();
      expect(() => {
        sessionManager.createSession(null as any, agent);
      }).toThrow('worktree cannot be null');
    });

    it('should throw error when creating session with same worktree key', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent('claude');

      sessionManager.createSession(worktree, agent);

      expect(() => {
        sessionManager.createSession(worktree, agent);
      }).toThrow('Session already exists');
    });

    it('should generate unique session IDs', () => {
      const worktree1 = createMockWorktree('repo1', '/path/to/repo1', 'main');
      const worktree2 = createMockWorktree('repo2', '/path/to/repo2', 'dev');
      const agent = createMockAgent();

      const session1 = sessionManager.createSession(worktree1, agent);
      const session2 = sessionManager.createSession(worktree2, agent);

      expect(session1.id).not.toBe(session2.id);
    });
  });

  describe('Get or Create Session', () => {
    it('should create new session if none exists', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.getOrCreateSession(worktree, agent);

      expect(session).toBeDefined();
      expect(session.worktreeKey).toBe('test-repo:/path/to/repo');
    });

    it('should return existing session if one exists', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session1 = sessionManager.getOrCreateSession(worktree, agent);
      const session2 = sessionManager.getOrCreateSession(worktree, agent);

      expect(session1).toBe(session2);
      expect(session1.id).toBe(session2.id);
    });

    it('should update lastAccessed when reusing session', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session1 = sessionManager.getOrCreateSession(worktree, agent);
      const firstAccess = session1.lastAccessed;

      // Wait a bit
      const start = Date.now();
      while (Date.now() - start < 10) {
        // Small delay
      }

      const session2 = sessionManager.getOrCreateSession(worktree, agent);

      expect(session2.lastAccessed.getTime()).toBeGreaterThanOrEqual(
        firstAccess.getTime()
      );
    });
  });

  describe('Active Session Management', () => {
    it('should get active session', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      sessionManager.switchToSession(session.worktreeKey);

      const activeSession = sessionManager.getActiveSession();
      expect(activeSession).toBe(session);
      expect(activeSession?.isActive).toBe(true);
    });

    it('should switch to a different session', () => {
      const worktree1 = createMockWorktree('repo1', '/path/to/repo1', 'main');
      const worktree2 = createMockWorktree('repo2', '/path/to/repo2', 'dev');
      const agent = createMockAgent();

      const session1 = sessionManager.createSession(worktree1, agent);
      const session2 = sessionManager.createSession(worktree2, agent);

      sessionManager.switchToSession(session1.worktreeKey);
      expect(sessionManager.getActiveSession()).toBe(session1);
      expect(session1.isActive).toBe(true);

      sessionManager.switchToSession(session2.worktreeKey);
      expect(sessionManager.getActiveSession()).toBe(session2);
      expect(session2.isActive).toBe(true);
      expect(session1.isActive).toBe(false);
    });

    it('should throw error when switching to non-existent session', () => {
      expect(() => {
        sessionManager.switchToSession('nonexistent:key');
      }).toThrow('Session not found');
    });

    it('should deactivate previous session when switching', () => {
      const worktree1 = createMockWorktree('repo1', '/path/to/repo1', 'main');
      const worktree2 = createMockWorktree('repo2', '/path/to/repo2', 'dev');
      const agent = createMockAgent();

      const session1 = sessionManager.createSession(worktree1, agent);
      const session2 = sessionManager.createSession(worktree2, agent);

      sessionManager.switchToSession(session1.worktreeKey);
      sessionManager.switchToSession(session2.worktreeKey);

      expect(session1.isActive).toBe(false);
      expect(session2.isActive).toBe(true);
    });

    it('should update lastAccessed when switching sessions', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      const beforeSwitch = session.lastAccessed;

      // Small delay
      const start = Date.now();
      while (Date.now() - start < 10) {
        // wait
      }

      sessionManager.switchToSession(session.worktreeKey);

      expect(session.lastAccessed.getTime()).toBeGreaterThanOrEqual(
        beforeSwitch.getTime()
      );
    });
  });

  describe('Session Registry & Lookup', () => {
    it('should list all sessions', () => {
      const worktree1 = createMockWorktree('repo1', '/path/to/repo1', 'main');
      const worktree2 = createMockWorktree('repo2', '/path/to/repo2', 'dev');
      const agent = createMockAgent();

      sessionManager.createSession(worktree1, agent);
      sessionManager.createSession(worktree2, agent);

      const sessions = sessionManager.listSessions();
      expect(sessions).toHaveLength(2);
    });

    it('should get session for specific worktree', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      const found = sessionManager.getSessionForWorktree(worktree);

      expect(found).toBe(session);
    });

    it('should return null when getting session for unknown worktree', () => {
      const worktree = createMockWorktree('unknown', '/unknown/path', 'main');
      const found = sessionManager.getSessionForWorktree(worktree);

      expect(found).toBeNull();
    });
  });

  describe('Session Deletion', () => {
    it('should delete a session', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      sessionManager.deleteSession(session.worktreeKey);

      const sessions = sessionManager.listSessions();
      expect(sessions).toHaveLength(0);
    });

    it('should throw error when deleting non-existent session', () => {
      expect(() => {
        sessionManager.deleteSession('nonexistent:key');
      }).toThrow('Session not found');
    });

    it('should clear active session when deleting it', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      sessionManager.switchToSession(session.worktreeKey);

      expect(sessionManager.getActiveSession()).toBe(session);

      sessionManager.deleteSession(session.worktreeKey);

      expect(sessionManager.getActiveSession()).toBeNull();
    });

    it('should not affect active session when deleting different session', () => {
      const worktree1 = createMockWorktree('repo1', '/path/to/repo1', 'main');
      const worktree2 = createMockWorktree('repo2', '/path/to/repo2', 'dev');
      const agent = createMockAgent();

      const session1 = sessionManager.createSession(worktree1, agent);
      const session2 = sessionManager.createSession(worktree2, agent);

      sessionManager.switchToSession(session1.worktreeKey);
      sessionManager.deleteSession(session2.worktreeKey);

      expect(sessionManager.getActiveSession()).toBe(session1);
    });
  });

  describe('State Persistence Integration', () => {
    it('should persist sessions after creation', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent('claude');

      sessionManager.createSession(worktree, agent);
      sessionManager.persistSessions();

      const persisted = stateManager.getSessionMappings();
      expect(Object.keys(persisted)).toHaveLength(1);
      expect(persisted['test-repo:/path/to/repo']).toBeDefined();
    });

    it('should persist active session', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      sessionManager.switchToSession(session.worktreeKey);
      sessionManager.persistSessions();

      const activeKey = stateManager.getActiveSession();
      expect(activeKey).toBe(session.worktreeKey);
    });

    it('should load sessions from state', () => {
      const persistedSession: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'repo1:/path/to/repo1',
        tmuxName: 'agate_session_1',
        agentName: 'claude',
        worktreePath: '/path/to/repo1',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString(),
      };

      stateManager.saveSessionMapping(persistedSession.worktreeKey, persistedSession);
      stateManager.setActiveSession(persistedSession.worktreeKey);

      sessionManager.loadSessions();

      const sessions = sessionManager.listSessions();
      expect(sessions).toHaveLength(1);
      expect(sessions[0]?.worktreeKey).toBe('repo1:/path/to/repo1');
    });

    it('should restore active session when loading', () => {
      const persistedSession: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'repo1:/path/to/repo1',
        tmuxName: 'agate_session_1',
        agentName: 'claude',
        worktreePath: '/path/to/repo1',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString(),
      };

      stateManager.saveSessionMapping(persistedSession.worktreeKey, persistedSession);
      stateManager.setActiveSession(persistedSession.worktreeKey);

      sessionManager.loadSessions();

      const activeSession = sessionManager.getActiveSession();
      expect(activeSession).toBeDefined();
      expect(activeSession?.worktreeKey).toBe('repo1:/path/to/repo1');
    });

    it('should remove session from persistence when deleted', () => {
      const worktree = createMockWorktree('test-repo', '/path/to/repo', 'main');
      const agent = createMockAgent();

      const session = sessionManager.createSession(worktree, agent);
      sessionManager.persistSessions();

      sessionManager.deleteSession(session.worktreeKey);
      sessionManager.persistSessions();

      const persisted = stateManager.getSessionMappings();
      expect(Object.keys(persisted)).toHaveLength(0);
    });
  });

  describe('Full Persistence E2E with Real StateManager', () => {
    it('should persist session state and restore it across restarts', async () => {
      const { mkdirSync, rmSync, existsSync } = await import('fs');
      const { homedir } = await import('os');
      const { join } = await import('path');

      const testHome = join(homedir(), '.agate-test-session-persist-' + Date.now());
      const originalHome = process.env.HOME;

      try {
        // Setup isolated test environment
        process.env.HOME = testHome;
        if (existsSync(testHome)) {
          rmSync(testHome, { recursive: true, force: true });
        }
        mkdirSync(testHome, { recursive: true });

        // Create real StateManager and SessionManager
        const realStateManager = await StateManager.create();
        const realSessionManager = new SessionManager(realStateManager);

        // Create worktree metadata for session
        const worktree: SessionWorktree = {
          name: 'test-branch',
          path: '/test/worktree/path',
          branch: 'test-branch',
          repoName: 'test-repo',
          commit: 'abc123',
          isMain: true,
          bare: false,
          detached: false,
        };

        const agent = AGENTS.claude;

        // Create session via SessionManager
        const session = realSessionManager.createSession(worktree, agent);
        expect(session).toBeDefined();
        expect(session.worktreeKey).toBe(`test-repo:/test/worktree/path`);

        // Switch to session to make it active
        realSessionManager.switchToSession(session.worktreeKey);
        expect(realSessionManager.getActiveSession()).toBe(session);

        // Persist sessions via SessionManager
        realSessionManager.persistSessions();
        await realStateManager.save();

        // Verify state was persisted via StateManager
        const savedMappings = realStateManager.getSessionMappings();
        expect(savedMappings[session.worktreeKey]).toBeDefined();
        expect(realStateManager.getActiveSession()).toBe(session.worktreeKey);

        // Simulate restart - create new StateManager and SessionManager instances
        const restoredState = await StateManager.create();
        const restoredSessManager = new SessionManager(restoredState);

        // Load sessions from disk
        restoredSessManager.loadSessions();

        // Verify session was restored
        const restoredSessions = restoredSessManager.listSessions();
        expect(restoredSessions).toHaveLength(1);
        expect(restoredSessions[0]?.worktreeKey).toBe(session.worktreeKey);

        // Verify active session was restored
        const restoredActive = restoredSessManager.getActiveSession();
        expect(restoredActive).toBeDefined();
        expect(restoredActive?.worktreeKey).toBe(session.worktreeKey);
        expect(restoredActive?.isActive).toBe(true);
      } finally {
        // Cleanup
        process.env.HOME = originalHome;
        if (existsSync(testHome)) {
          rmSync(testHome, { recursive: true, force: true });
        }
      }
    }, 10000);
  });
});
