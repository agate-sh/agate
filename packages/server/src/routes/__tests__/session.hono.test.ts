import { describe, it, expect, beforeAll, afterAll, beforeEach } from 'vitest';
import { EventBus } from '../../event-bus.js';
import { StateManager } from '../../state/manager.js';
import { createHonoServer } from '../../server.hono.js';
import type { PersistedSession, ListSessionsResponse } from '@agate/shared';

describe('Session Routes', () => {
  let app: ReturnType<typeof createHonoServer>;
  let eventBus: EventBus;
  let stateManager: StateManager;

  beforeAll(async () => {
    eventBus = new EventBus();
    stateManager = await StateManager.create();
    app = createHonoServer(eventBus, stateManager);
  });

  afterAll(() => {
    eventBus?.destroy();
  });

  beforeEach(() => {
    stateManager.updateSessions({
      sessionMappings: {},
      activeSession: '',
      defaultAgent: 'claude',
    });
  });

  it('should list persisted sessions with active session and default agent', async () => {
    const now = new Date().toISOString();
    const session: PersistedSession = {
      id: 'test-session-id',
      worktreeKey: 'repo:/tmp/worktree:main',
      tmuxName: 'tmux-test',
      agentName: 'claude',
      worktreePath: '/tmp/worktree',
      branch: 'main',
      repoName: 'test-repo',
      createdAt: now,
      lastAccessed: now,
    };

    stateManager.saveSessionMapping(session.worktreeKey, session);
    stateManager.setActiveSession(session.worktreeKey);
    stateManager.setDefaultAgent('claude');

    const response = await app.request('/sessions');

    expect(response.status).toBe(200);

    const data = await response.json() as ListSessionsResponse;

    expect(data.sessions).toHaveLength(1);
    expect(data.sessions[0]).toMatchObject(session);
    expect(data.activeSession).toBe(session.worktreeKey);
    expect(data.defaultAgent).toBe('claude');
  });
});
