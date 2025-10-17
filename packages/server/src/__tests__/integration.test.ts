import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { Server } from 'http';
import { EventBus } from '../event-bus.js';
import { TmuxSessionManager } from '../tmux/session.js';
import { WorktreeManager } from '../git/worktree.js';
import { StateManager } from '../state/manager.js';
import { SessionManager } from '../session/manager.js';
import { createServer } from '../server.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type {
  PtyOutputEvent,
  SessionWorktree,
  CreateSessionResponse,
  GetSessionResponse,
  SendSessionInputResponse,
  ResizeSessionResponse,
  DeleteSessionResponse,
} from '@agate/shared';
import { AGENTS } from '@agate/shared';
import { randomUUID } from 'crypto';
import { join } from 'path';
import { mkdirSync, rmSync, existsSync } from 'fs';
import { homedir } from 'os';

/**
 * Integration test validating the full flow:
 * 1. Create git worktree
 * 2. Spawn tmux session in worktree
 * 3. Run subprocess (simulated with echo)
 * 4. Verify PTY output streams via SSE
 */
describe('Integration Test: Worktree → Tmux → PTY → SSE', () => {
  let testRepoPath: string;
  let worktreePath: string;
  let server: Server;
  let eventBus: EventBus;
  let sessionManager: TmuxSessionManager;
  let stateManager: StateManager;
  const PORT = 3001; // Use different port to avoid conflicts
  const testHome = join(homedir(), '.agate-test-integration-' + Date.now());
  const originalHome = process.env.HOME;

  beforeAll(async () => {
    // 0. Setup test environment with isolated state directory
    process.env.HOME = testHome;
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
    mkdirSync(testHome, { recursive: true });

    // 1. Create test git repository
    const { path, git } = await createTestRepo();
    testRepoPath = path;
    await createInitialCommit(git);

    // 2. Create worktree
    const worktreeManager = new WorktreeManager(testRepoPath);
    const branchName = `test-branch-${Date.now()}`;
    worktreePath = join(testRepoPath, '..', `worktree-${branchName}`);
    await worktreeManager.create(worktreePath, branchName);

    // 3. Initialize dependencies
    stateManager = await StateManager.create();
    eventBus = new EventBus();

    // 4. Create real Express server with all routes
    const app = createServer(eventBus, stateManager);

    // Start server
    await new Promise<void>((resolve) => {
      server = app.listen(PORT, () => {
        console.log(`Test server listening on port ${PORT}`);
        resolve();
      });
    });
  });

  afterAll(async () => {
    // Cleanup
    if (sessionManager?.isAlive()) {
      await sessionManager.kill();
    }

    if (server) {
      await new Promise<void>((resolve) => {
        server.close(() => resolve());
      });
    }

    eventBus?.destroy();

    if (testRepoPath) {
      await cleanupTestRepo(testRepoPath);
    }

    if (worktreePath) {
      await cleanupTestRepo(worktreePath);
    }

    // Restore original HOME and cleanup test directory
    process.env.HOME = originalHome;
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
  });

  it('should create worktree, spawn tmux session with subprocess, and stream PTY output via SSE', async () => {
    // Verify worktree was created
    const worktreeManager = new WorktreeManager(testRepoPath);
    const worktrees = await worktreeManager.list();
    expect(worktrees.length).toBeGreaterThan(1); // main + new worktree

    // Track events received
    const receivedEvents: PtyOutputEvent[] = [];
    let sseConnected = false;

    // Create SSE client manually (using fetch with streaming)
    const clientId = randomUUID();
    const controller = new AbortController();

    // Start listening to SSE endpoint
    const ssePromise = (async () => {
      try {
        const response = await fetch(`http://localhost:${PORT}/events?clientId=${clientId}`, {
          signal: controller.signal,
          headers: {
            'Accept': 'text/event-stream',
          },
        });

        if (!response.body) {
          throw new Error('No response body');
        }

        sseConnected = true;
        const reader = response.body.getReader();
        const decoder = new TextDecoder();

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value);
          const lines = chunk.split('\n');

          for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            if (line?.startsWith('event: pty.output')) {
              // Next line should be the data
              const dataLine = lines[i + 1];
              if (dataLine?.startsWith('data: ')) {
                const jsonData = dataLine.substring('data: '.length);
                const event = JSON.parse(jsonData) as PtyOutputEvent;
                receivedEvents.push(event);
              }
            }
          }
        }
      } catch (error: unknown) {
        const err = error as Error;
        if (err.name !== 'AbortError') {
          console.error('SSE error:', err);
        }
      }
    })();

    // Wait for SSE connection
    await new Promise((resolve) => {
      const checkInterval = setInterval(() => {
        if (sseConnected) {
          clearInterval(checkInterval);
          resolve(undefined);
        }
      }, 50);
    });

    // Create tmux session in worktree
    const sessionId = randomUUID();
    sessionManager = new TmuxSessionManager(eventBus, sessionId);

    const sessionName = `agate_test_${Date.now()}`;
    await sessionManager.createSession({
      name: sessionName,
      agent: 'claude',
      cwd: worktreePath,
      cols: 80,
      rows: 24,
    });

    expect(sessionManager.isAlive()).toBe(true);

    // Spawn a subprocess that generates output (simulate claude with echo)
    // We'll use a simple command that produces clear output
    sessionManager.write('echo "AGATE_TEST_OUTPUT_MARKER"\n');

    // Wait for output to be captured and streamed
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Stop SSE client
    controller.abort();
    await ssePromise.catch(() => {}); // Ignore abort errors

    // Verify we received PTY output events
    expect(receivedEvents.length).toBeGreaterThan(0);

    // Verify event structure
    const firstEvent = receivedEvents[0];
    expect(firstEvent).toBeDefined();
    expect(firstEvent?.type).toBe('pty.output');
    expect(firstEvent?.sessionId).toBe(sessionId);
    expect(firstEvent?.data).toBeDefined();
    expect(typeof firstEvent?.data).toBe('string');

    // Verify the output contains our marker (may be across multiple events)
    const allOutput = receivedEvents.map(e => e.data).join('');
    expect(allOutput).toContain('AGATE_TEST_OUTPUT_MARKER');

    console.log(`✅ Integration test passed!`);
    console.log(`   - Created worktree at: ${worktreePath}`);
    console.log(`   - Spawned tmux session: ${sessionName}`);
    console.log(`   - Received ${receivedEvents.length} PTY output events`);
    console.log(`   - Total output size: ${allOutput.length} bytes`);
  }, 15000); // Increase timeout for this integration test

  it('should persist session state and restore it across restarts', async () => {
    // Create a tmux session
    const sessionId = randomUUID();
    const sessionName = `agate_persist_test_${Date.now()}`;
    const tmuxManager = new TmuxSessionManager(eventBus, sessionId);

    await tmuxManager.createSession({
      name: sessionName,
      agent: 'claude',
      cwd: worktreePath,
      cols: 80,
      rows: 24,
    });

    expect(tmuxManager.isAlive()).toBe(true);

    // Use SessionManager to manage the session
    const sessManager = new SessionManager(stateManager);

    // Create worktree metadata for session
    const worktree: SessionWorktree = {
      name: 'test-branch',
      path: worktreePath,
      branch: 'test-branch',
      repoName: 'test-repo',
      isMain: false,
      commit: 'abc123',
    };

    const agent = AGENTS.claude;

    // Create session via SessionManager
    const session = sessManager.createSession(worktree, agent);
    expect(session).toBeDefined();
    expect(session.worktreeKey).toBe(`test-repo:${worktreePath}`);

    // Switch to session to make it active
    sessManager.switchToSession(session.worktreeKey);
    expect(sessManager.getActiveSession()).toBe(session);

    // Persist sessions via SessionManager
    sessManager.persistSessions();
    await stateManager.save();

    // Verify state was persisted via StateManager
    const savedMappings = stateManager.getSessionMappings();
    expect(savedMappings[session.worktreeKey]).toBeDefined();
    expect(stateManager.getActiveSession()).toBe(session.worktreeKey);

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

    console.log(`✅ Session persistence test passed!`);
    console.log(`   - Saved session: ${sessionName}`);
    console.log(`   - Restored session from disk via SessionManager`);
    console.log(`   - Active session restored correctly`);

    // Cleanup
    await tmuxManager.kill();
  }, 10000);

  it('should persist workspace state (repositories and selections)', async () => {
    // Add repositories
    stateManager.addRepository('test-repo-1', '/path/to/repo1');
    stateManager.addRepository('test-repo-2', '/path/to/repo2');

    // Set worktree selections
    stateManager.setLastWorktreeForRepo('test-repo-1', {
      path: '/path/to/worktree1',
      branch: 'main',
    });

    // Update workspace state
    stateManager.updateWorkspace({
      repositories: stateManager.getRepositories(),
      repoSelections: stateManager.readWorkspace().repoSelections,
      lastRepo: 'test-repo-1',
    });

    // Save state
    await stateManager.save();

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

    console.log(`✅ Workspace persistence test passed!`);
    console.log(`   - Saved 2 repositories`);
    console.log(`   - Restored workspace state successfully`);
  }, 5000);

  describe('Session API Endpoints', () => {
    it('should create a session via POST /session', async () => {
      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worktreePath,
          branch: 'test-branch',
          agentName: 'claude',
        }),
      });

      expect(response.ok).toBe(true);
      const data = (await response.json()) as CreateSessionResponse;
      expect(data.sessionId).toBeDefined();
      expect(typeof data.sessionId).toBe('string');
    }, 10000);

    it('should get session info via GET /session/:id', async () => {
      // First create a session
      const createResponse = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worktreePath,
          branch: 'test-branch',
          agentName: 'claude',
        }),
      });

      const { sessionId } = (await createResponse.json()) as CreateSessionResponse;

      // Now get session info
      const getResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}`);
      expect(getResponse.ok).toBe(true);

      const data = (await getResponse.json()) as GetSessionResponse;
      expect(data.id).toBe(sessionId);
      expect(data.name).toBeDefined();
      expect(data.agent).toBe('claude');
      expect(data.cwd).toBe(worktreePath);
      expect(data.isAlive).toBe(true);
    }, 10000);

    it('should send input to session via POST /session/:id/input', async () => {
      // Create session
      const createResponse = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worktreePath,
          branch: 'test-branch',
          agentName: 'claude',
        }),
      });

      const { sessionId } = (await createResponse.json()) as CreateSessionResponse;

      // Send input
      const inputResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}/input`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          data: 'echo "test input"\n',
        }),
      });

      expect(inputResponse.ok).toBe(true);
      const data = (await inputResponse.json()) as SendSessionInputResponse;
      expect(data.success).toBe(true);
    }, 10000);

    it('should resize session via POST /session/:id/resize', async () => {
      // Create session
      const createResponse = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worktreePath,
          branch: 'test-branch',
          agentName: 'claude',
        }),
      });

      const { sessionId } = (await createResponse.json()) as CreateSessionResponse;

      // Resize
      const resizeResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}/resize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cols: 120,
          rows: 40,
        }),
      });

      expect(resizeResponse.ok).toBe(true);
      const data = (await resizeResponse.json()) as ResizeSessionResponse;
      expect(data.success).toBe(true);
    }, 10000);

    it('should delete session via DELETE /session/:id', async () => {
      // Create session
      const createResponse = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worktreePath,
          branch: 'test-branch',
          agentName: 'claude',
        }),
      });

      const { sessionId } = (await createResponse.json()) as CreateSessionResponse;

      // Delete
      const deleteResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}`, {
        method: 'DELETE',
      });

      expect(deleteResponse.ok).toBe(true);
      const data = (await deleteResponse.json()) as DeleteSessionResponse;
      expect(data.success).toBe(true);

      // Verify session no longer exists
      const getResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}`);
      expect(getResponse.status).toBe(404);
    }, 10000);

    it('should return 404 for non-existent session', async () => {
      const response = await fetch(`http://localhost:${PORT}/session/non-existent-id`);
      expect(response.status).toBe(404);
    }, 5000);

    it('should return 400 for invalid create session request', async () => {
      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          // Missing required fields
          worktreePath,
        }),
      });

      expect(response.status).toBe(400);
    }, 5000);
  });

});
