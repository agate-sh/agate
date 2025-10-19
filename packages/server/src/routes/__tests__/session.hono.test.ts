import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { Hono } from 'hono';
import { join } from 'path';
import { mkdirSync, rmSync, existsSync, readFileSync } from 'fs';
import { homedir } from 'os';
import { EventBus } from '../../event-bus.js';
import { StateManager } from '../../state/manager.js';
import { sessionRouter } from '../session.hono.js';
import type { CreateSessionResponse, AppState } from '@agate/shared';

/**
 * Route-level integration tests for session persistence
 *
 * Following nodejs-integration-tests-best-practices:
 * - Real StateManager (not mocked)
 * - Real HTTP requests
 * - Isolated test environment
 * - Verify actual state.json persistence
 */
describe('Session Routes - Persistence Integration', () => {
  let testHome: string;
  let originalHome: string | undefined;
  let stateManager: StateManager;
  let eventBus: EventBus;
  let app: Hono;

  beforeEach(async () => {
    // Setup isolated test environment
    originalHome = process.env.HOME;
    testHome = join(homedir(), '.agate-test-routes-' + Date.now());

    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
    mkdirSync(testHome, { recursive: true });
    process.env.HOME = testHome;

    // Initialize real dependencies
    stateManager = await StateManager.create();
    eventBus = new EventBus();

    // Create Hono app with middleware
    app = new Hono();
    app.use('*', async (c, next) => {
      c.set('eventBus', eventBus);
      c.set('stateManager', stateManager);
      await next();
    });
    app.route('/session', sessionRouter);
  });

  afterEach(() => {
    // Cleanup
    eventBus?.destroy();

    if (testHome && existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }

    if (originalHome) {
      process.env.HOME = originalHome;
    } else {
      delete process.env.HOME;
    }
  });

  it('should persist session to state.json when created via POST /session', async () => {
    // Arrange
    const worktreePath = '/Users/test/repos/my-project';
    const branch = 'feature/test-branch';
    const agentName = 'claude';
    const repoName = 'my-project'; // Expected to be extracted from path

    // Act - Create session via HTTP
    const res = await app.request('/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        worktreePath,
        branch,
        agentName,
      }),
    });

    // Assert - HTTP response
    expect(res.status).toBe(201);
    const body = await res.json() as CreateSessionResponse;
    expect(body.sessionId).toBeDefined();
    expect(typeof body.sessionId).toBe('string');

    // Assert - State persisted to disk
    const stateFilePath = join(testHome, '.agate', 'state.json');
    expect(existsSync(stateFilePath)).toBe(true);

    const stateContent = readFileSync(stateFilePath, 'utf-8');
    const state: AppState = JSON.parse(stateContent);

    // Verify session exists in state
    const worktreeKey = `${repoName}:${worktreePath}`;
    expect(state.sessions.sessionMappings[worktreeKey]).toBeDefined();

    const persistedSession = state.sessions.sessionMappings[worktreeKey];

    // Verify all PersistedSession fields
    expect(persistedSession.id).toBe(body.sessionId);
    expect(persistedSession.worktreeKey).toBe(worktreeKey);
    expect(persistedSession.tmuxName).toContain('agate_');
    expect(persistedSession.tmuxName).toContain(branch);
    expect(persistedSession.tmuxName).toContain(agentName);
    expect(persistedSession.agentName).toBe(agentName);
    expect(persistedSession.worktreePath).toBe(worktreePath);
    expect(persistedSession.branch).toBe(branch);
    expect(persistedSession.repoName).toBe(repoName);
    expect(persistedSession.createdAt).toBeDefined();
    expect(persistedSession.lastAccessed).toBeDefined();

    // Verify timestamps are valid ISO strings
    expect(() => new Date(persistedSession.createdAt)).not.toThrow();
    expect(() => new Date(persistedSession.lastAccessed)).not.toThrow();
  });

  it('should persist session deletion to state.json when deleting via DELETE /session/:id', async () => {
    // Arrange - First create a session
    const worktreePath = '/Users/test/repos/test-project';
    const branch = 'main';
    const agentName = 'claude';

    const createRes = await app.request('/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        worktreePath,
        branch,
        agentName,
      }),
    });

    const createBody = await createRes.json() as CreateSessionResponse;
    const sessionId = createBody.sessionId;

    // Verify session was persisted
    const stateFilePath = join(testHome, '.agate', 'state.json');
    let stateContent = readFileSync(stateFilePath, 'utf-8');
    let state: AppState = JSON.parse(stateContent);

    const worktreeKey = `test-project:${worktreePath}`;
    expect(state.sessions.sessionMappings[worktreeKey]).toBeDefined();

    // Act - Delete session via HTTP
    const deleteRes = await app.request(`/session/${sessionId}`, {
      method: 'DELETE',
    });

    // Assert - HTTP response
    expect(deleteRes.status).toBe(200);

    // Assert - Session removed from state.json
    stateContent = readFileSync(stateFilePath, 'utf-8');
    state = JSON.parse(stateContent);

    expect(state.sessions.sessionMappings[worktreeKey]).toBeUndefined();
  });

  it('should handle multiple sessions for different worktrees', async () => {
    // Arrange
    const sessions = [
      { worktreePath: '/Users/test/repos/project-1', branch: 'main', agentName: 'claude' },
      { worktreePath: '/Users/test/repos/project-2', branch: 'feature/test', agentName: 'gemini' },
      { worktreePath: '/Users/test/repos/project-1/.agate/worktrees/feature-other', branch: 'feature/other', agentName: 'claude' }, // Different worktree path, same repo
    ];

    // Act - Create multiple sessions
    const sessionIds: string[] = [];
    for (const session of sessions) {
      const res = await app.request('/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(session),
      });

      const body = await res.json() as CreateSessionResponse;
      sessionIds.push(body.sessionId);
    }

    // Assert - All sessions persisted
    const stateFilePath = join(testHome, '.agate', 'state.json');
    const stateContent = readFileSync(stateFilePath, 'utf-8');
    const state: AppState = JSON.parse(stateContent);

    // Verify all 3 sessions exist (different worktree keys)
    const mappingKeys = Object.keys(state.sessions.sessionMappings);
    expect(mappingKeys.length).toBe(3);

    // Verify each session has correct data
    sessions.forEach((session, i) => {
      // Extract repo name from path
      // For paths like /Users/test/repos/project-1, extract 'project-1'
      // For paths like /Users/test/repos/project-1/.agate/worktrees/feature-other, extract 'feature-other'
      const pathParts = session.worktreePath.split('/').filter(Boolean);
      const repoName = pathParts[pathParts.length - 1];
      const worktreeKey = `${repoName}:${session.worktreePath}`;

      const persistedSession = state.sessions.sessionMappings[worktreeKey];
      expect(persistedSession).toBeDefined();
      expect(persistedSession.worktreePath).toBe(session.worktreePath);
      expect(persistedSession.branch).toBe(session.branch);
      expect(persistedSession.agentName).toBe(session.agentName);
    });
  });
});
