import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, rmSync, existsSync, writeFileSync, readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { StateManager } from '../manager.js';
import type { AppState, PersistedSession } from '@agate/shared';

describe('StateManager', () => {
  const testHome = join(homedir(), '.agate-test-manager-' + Date.now());
  const originalHome = process.env.HOME;
  let manager: StateManager;

  beforeEach(() => {
    // Override HOME for testing
    process.env.HOME = testHome;
    // Clean up any existing test directory
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
    mkdirSync(testHome, { recursive: true });
  });

  afterEach(() => {
    // Restore original HOME
    process.env.HOME = originalHome;
    // Clean up test directory
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
  });

  describe('Initialization', () => {
    it('should create default state when state file does not exist', async () => {
      manager = await StateManager.create();

      const state = manager.getState();
      expect(state.version).toBe(1);
      expect(state.ui.showWelcomeDialog).toBe(true);
      expect(state.workspace.repositories).toEqual({});
      expect(state.workspace.repoSelections).toEqual({});
      expect(state.workspace.lastRepo).toBe('');
      expect(state.sessions.sessionMappings).toEqual({});
      expect(state.sessions.activeSession).toBe('');
      expect(state.sessions.defaultAgent).toBe('claude');
    });

    it('should load existing state from disk', async () => {
      const agateDir = join(testHome, '.agate');
      mkdirSync(agateDir, { recursive: true });

      const existingState: AppState = {
        version: 1,
        ui: { showWelcomeDialog: false },
        workspace: {
          repositories: { 'test-repo': '/path/to/repo' },
          repoSelections: {},
          lastRepo: 'test-repo'
        },
        sessions: {
          sessionMappings: {},
          activeSession: '',
          defaultAgent: 'gemini'
        }
      };

      writeFileSync(
        join(agateDir, 'state.json'),
        JSON.stringify(existingState, null, 2)
      );

      manager = await StateManager.create();
      const state = manager.getState();

      expect(state.ui.showWelcomeDialog).toBe(false);
      expect(state.workspace.lastRepo).toBe('test-repo');
      expect(state.sessions.defaultAgent).toBe('gemini');
    });

    it('should return default state for corrupted JSON file', async () => {
      const agateDir = join(testHome, '.agate');
      mkdirSync(agateDir, { recursive: true });

      writeFileSync(join(agateDir, 'state.json'), '{ invalid json }');

      manager = await StateManager.create();
      const state = manager.getState();

      // Should get default state instead of throwing
      expect(state.version).toBe(1);
      expect(state.ui.showWelcomeDialog).toBe(true);
    });

    it('should normalize state after loading', async () => {
      const agateDir = join(testHome, '.agate');
      mkdirSync(agateDir, { recursive: true });

      // Write partial state missing some fields
      const partialState = {
        version: 1,
        workspace: {
          repositories: { 'repo1': '/path' }
        }
      };

      writeFileSync(
        join(agateDir, 'state.json'),
        JSON.stringify(partialState, null, 2)
      );

      manager = await StateManager.create();
      const state = manager.getState();

      // Should have all required fields after normalization
      expect(state.ui).toBeDefined();
      expect(state.sessions).toBeDefined();
      expect(state.workspace.repoSelections).toBeDefined();
    });
  });

  describe('State Persistence', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should write state to disk atomically', async () => {
      manager.updateWorkspace({
        repositories: { 'repo1': '/path/to/repo1' },
        repoSelections: {},
        lastRepo: 'repo1'
      });

      await manager.save();

      const agateDir = join(testHome, '.agate');
      const stateFile = join(agateDir, 'state.json');
      expect(existsSync(stateFile)).toBe(true);

      const savedState = JSON.parse(readFileSync(stateFile, 'utf-8'));
      expect(savedState.workspace.repositories['repo1']).toBe('/path/to/repo1');
    });

    it('should handle concurrent saves properly', async () => {
      // This tests that atomically library queues writes correctly
      const promises = Array.from({ length: 10 }, (_, i) =>
        (async () => {
          manager.updateWorkspace({
            repositories: { [`repo${i}`]: `/path/to/repo${i}` },
            repoSelections: {},
            lastRepo: `repo${i}`
          });
          await manager.save();
        })()
      );

      await Promise.all(promises);

      const agateDir = join(testHome, '.agate');
      const stateFile = join(agateDir, 'state.json');
      const savedState = JSON.parse(readFileSync(stateFile, 'utf-8'));

      // Should have valid JSON (not corrupted by concurrent writes)
      expect(savedState.version).toBe(1);
      expect(Object.keys(savedState.workspace.repositories).length).toBeGreaterThan(0);
    });
  });

  describe('Session Management', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should save session mapping', () => {
      const session: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'repo1:/path/to/worktree:main',
        tmuxName: 'agate_session_1',
        agentName: 'claude',
        worktreePath: '/path/to/worktree',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString()
      };

      manager.saveSessionMapping('repo1:/path/to/worktree:main', session);

      const mappings = manager.getSessionMappings();
      expect(mappings['repo1:/path/to/worktree:main']).toEqual(session);
    });

    it('should remove session mapping', () => {
      const session: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'repo1:/path/to/worktree:main',
        tmuxName: 'agate_session_1',
        agentName: 'claude',
        worktreePath: '/path/to/worktree',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString()
      };

      manager.saveSessionMapping('repo1:/path/to/worktree:main', session);
      manager.removeSessionMapping('repo1:/path/to/worktree:main');

      const mappings = manager.getSessionMappings();
      expect(mappings['repo1:/path/to/worktree:main']).toBeUndefined();
    });

    it('should set and get active session', () => {
      manager.setActiveSession('session-key-1');
      expect(manager.getActiveSession()).toBe('session-key-1');
    });

    it('should set and get default agent', () => {
      manager.setDefaultAgent('gemini');
      expect(manager.getDefaultAgent()).toBe('gemini');
    });

    it('should get all session mappings', () => {
      const session1: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'key1',
        tmuxName: 'tmux1',
        agentName: 'claude',
        worktreePath: '/path1',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString()
      };

      const session2: PersistedSession = {
        id: 'session-2',
        worktreeKey: 'key2',
        tmuxName: 'tmux2',
        agentName: 'gemini',
        worktreePath: '/path2',
        branch: 'dev',
        repoName: 'repo2',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString()
      };

      manager.saveSessionMapping('key1', session1);
      manager.saveSessionMapping('key2', session2);

      const mappings = manager.getSessionMappings();
      expect(Object.keys(mappings).length).toBe(2);
      expect(mappings['key1']).toEqual(session1);
      expect(mappings['key2']).toEqual(session2);
    });
  });

  describe('Workspace Management', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should add repository', () => {
      manager.addRepository('repo1', '/path/to/repo1');

      const repos = manager.getRepositories();
      expect(repos['repo1']).toBe('/path/to/repo1');
    });

    it('should remove repository', () => {
      manager.addRepository('repo1', '/path/to/repo1');
      manager.removeRepository('repo1');

      const repos = manager.getRepositories();
      expect(repos['repo1']).toBeUndefined();
    });

    it('should get all repositories', () => {
      manager.addRepository('repo1', '/path/to/repo1');
      manager.addRepository('repo2', '/path/to/repo2');

      const repos = manager.getRepositories();
      expect(Object.keys(repos).length).toBe(2);
      expect(repos['repo1']).toBe('/path/to/repo1');
      expect(repos['repo2']).toBe('/path/to/repo2');
    });

    it('should set and get last worktree for repo', () => {
      manager.setLastWorktreeForRepo('repo1', {
        path: '/path/to/worktree',
        branch: 'main'
      });

      const worktree = manager.getLastWorktreeForRepo('repo1');
      expect(worktree?.path).toBe('/path/to/worktree');
      expect(worktree?.branch).toBe('main');
    });

    it('should return undefined for repo with no last worktree', () => {
      const worktree = manager.getLastWorktreeForRepo('nonexistent');
      expect(worktree).toBeUndefined();
    });

    it('should get last active repo', () => {
      manager.updateWorkspace({
        repositories: {},
        repoSelections: {},
        lastRepo: 'repo1'
      });

      expect(manager.getLastActiveRepo()).toBe('repo1');
    });

    it('should persist workspace state (repositories and selections) across restarts', async () => {
      // Add repositories
      manager.addRepository('test-repo-1', '/path/to/repo1');
      manager.addRepository('test-repo-2', '/path/to/repo2');

      // Set worktree selections
      manager.setLastWorktreeForRepo('test-repo-1', {
        path: '/path/to/worktree1',
        branch: 'main',
      });

      // Update workspace state
      manager.updateWorkspace({
        repositories: manager.getRepositories(),
        repoSelections: manager.readWorkspace().repoSelections,
        lastRepo: 'test-repo-1',
      });

      // Save state
      await manager.save();

      // Simulate restart
      const restoredState = await StateManager.create();

      // Verify repositories were restored
      const repos = restoredState.getRepositories();
      expect(repos['test-repo-1']).toBe('/path/to/repo1');
      expect(repos['test-repo-2']).toBe('/path/to/repo2');

      // Verify worktree selection was restored
      const worktree = restoredState.getLastWorktreeForRepo('test-repo-1');
      expect(worktree?.path).toBe('/path/to/worktree1');
      expect(worktree?.branch).toBe('main');

      // Verify last repo was restored
      expect(restoredState.getLastActiveRepo()).toBe('test-repo-1');
    });
  });

  describe('UI State Management', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should update UI state', () => {
      manager.updateUI({ showWelcomeDialog: false });

      const ui = manager.readUI();
      expect(ui.showWelcomeDialog).toBe(false);
    });

    it('should read UI state', () => {
      const ui = manager.readUI();
      expect(ui.showWelcomeDialog).toBe(true); // default
    });
  });

  describe('State Immutability', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should return deep copy on getState to prevent external mutations', () => {
      const state1 = manager.getState();
      state1.workspace.repositories['hacked'] = '/hacked';

      const state2 = manager.getState();
      expect(state2.workspace.repositories['hacked']).toBeUndefined();
    });

    it('should return deep copy on getSessionMappings', () => {
      const session: PersistedSession = {
        id: 'session-1',
        worktreeKey: 'key1',
        tmuxName: 'tmux1',
        agentName: 'claude',
        worktreePath: '/path1',
        branch: 'main',
        repoName: 'repo1',
        createdAt: new Date().toISOString(),
        lastAccessed: new Date().toISOString()
      };

      manager.saveSessionMapping('key1', session);

      const mappings1 = manager.getSessionMappings();
      mappings1['hacked'] = {} as PersistedSession;

      const mappings2 = manager.getSessionMappings();
      expect(mappings2['hacked']).toBeUndefined();
    });

    it('should return deep copy on getRepositories', () => {
      manager.addRepository('repo1', '/path1');

      const repos1 = manager.getRepositories();
      repos1['hacked'] = '/hacked';

      const repos2 = manager.getRepositories();
      expect(repos2['hacked']).toBeUndefined();
    });
  });

  describe('Update Methods', () => {
    beforeEach(async () => {
      manager = await StateManager.create();
    });

    it('should update sessions state', () => {
      manager.updateSessions({
        sessionMappings: {},
        activeSession: 'new-session',
        defaultAgent: 'codex'
      });

      const sessions = manager.readSessions();
      expect(sessions.activeSession).toBe('new-session');
      expect(sessions.defaultAgent).toBe('codex');
    });

    it('should update workspace state', () => {
      manager.updateWorkspace({
        repositories: { 'repo1': '/path1' },
        repoSelections: {},
        lastRepo: 'repo1'
      });

      const workspace = manager.readWorkspace();
      expect(workspace.lastRepo).toBe('repo1');
      expect(workspace.repositories['repo1']).toBe('/path1');
    });
  });
});
