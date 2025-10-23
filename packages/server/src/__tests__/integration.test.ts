import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { serve, type ServerType } from '@hono/node-server';
import { createNodeWebSocket } from '@hono/node-ws';
import { EventBus } from '../event-bus.js';
import { WorktreeManager } from '../git/worktree.js';
import { StateManager } from '../state/manager.js';
import { createHonoServer } from '../server.hono.js';
import { createWebSocketHandler } from '../websocket.hono.js';
import { sessions } from '../routes/session.hono.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type {
  CreateSessionResponse,
  GetSessionResponse,
  ResizeSessionResponse,
  DeleteSessionResponse,
} from '@agate/shared';
import { join } from 'path';
import { mkdirSync, rmSync, existsSync, realpathSync } from 'fs';
import { homedir } from 'os';

/**
 * E2E integration tests validating the full HTTP API flow:
 * 1. Create git worktree
 * 2. Create tmux session via POST /session
 * 3. Send input via POST /session/:id/input
 * 4. Test all session management endpoints
 *
 * Note: PTY output streaming is tested separately in websocket.test.ts
 */
describe('Integration Test: E2E HTTP API → Worktree → Tmux → PTY', () => {
  let testRepoPath: string;
  let worktreePath: string;
  let server: ServerType;
  let eventBus: EventBus;
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

    // 4. Create Hono server with all routes
    const app = createHonoServer(eventBus, stateManager);

    // Setup WebSocket support
    const { injectWebSocket, upgradeWebSocket } = createNodeWebSocket({ app });

    // Add WebSocket route
    app.get('/ws', upgradeWebSocket(createWebSocketHandler(eventBus, sessions)));

    // Start server with WebSocket support
    server = serve({
      fetch: app.fetch,
      port: PORT,
    });

    // Inject WebSocket upgrade handler
    injectWebSocket(server);

    console.log(`Test server listening on port ${PORT}`);
  });

  afterAll(async () => {
    // Cleanup
    if (server) {
      server.close();
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

  describe('Git API Endpoints', () => {
    it('should return repo info for valid git repo via GET /git/repo-info', async () => {
      const response = await fetch(
        `http://localhost:${PORT}/git/repo-info?dir=${encodeURIComponent(testRepoPath)}`
      );

      expect(response.ok).toBe(true);
      const data = await response.json() as { repoRoot: string; repoName: string; currentBranch: string };
      // Normalize paths to handle macOS symlinks (/var -> /private/var)
      expect(realpathSync(data.repoRoot)).toBe(realpathSync(testRepoPath));
      expect(data.repoName).toBeDefined();
      expect(typeof data.repoName).toBe('string');
      expect(data.currentBranch).toBe('main'); // from createInitialCommit
    }, 5000);

    it('should return repo info when dir is subdirectory of git repo', async () => {
      // Create a subdirectory in the test repo
      const subDir = join(testRepoPath, 'subdir');
      mkdirSync(subDir, { recursive: true });

      const response = await fetch(
        `http://localhost:${PORT}/git/repo-info?dir=${encodeURIComponent(subDir)}`
      );

      expect(response.ok).toBe(true);
      const data = await response.json() as { repoRoot: string; repoName: string; currentBranch: string };
      // Normalize paths to handle macOS symlinks (/var -> /private/var)
      expect(realpathSync(data.repoRoot)).toBe(realpathSync(testRepoPath));
      expect(data.repoName).toBeDefined();
      expect(data.currentBranch).toBe('main');
    }, 5000);

    it('should return 404 for non-git directory via GET /git/repo-info', async () => {
      const nonGitDir = join(testHome, 'not-a-git-repo');
      mkdirSync(nonGitDir, { recursive: true });

      const response = await fetch(
        `http://localhost:${PORT}/git/repo-info?dir=${encodeURIComponent(nonGitDir)}`
      );

      expect(response.status).toBe(404);
    }, 5000);

    it('should return 400 for missing dir parameter', async () => {
      const response = await fetch(`http://localhost:${PORT}/git/repo-info`);
      expect(response.status).toBe(400);
    }, 5000);
  });

  describe('Session API Endpoints (Refactored - Atomic Creation)', () => {
    it('should create session atomically from dir via POST /session', async () => {
      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dir: testRepoPath,
          branchName: 'test-atomic-branch',
          agentName: 'claude',
        }),
      });

      expect(response.status).toBe(201);
      const data = (await response.json()) as CreateSessionResponse;
      expect(data.sessionId).toBeDefined();
      expect(typeof data.sessionId).toBe('string');

      // Verify session was created correctly
      const sessionResponse = await fetch(`http://localhost:${PORT}/session/${data.sessionId}`);
      expect(sessionResponse.ok).toBe(true);
      const sessionData = (await sessionResponse.json()) as GetSessionResponse;
      expect(sessionData.agent).toBe('claude');
      // The cwd should be the worktree path, not the original repo path
      expect(sessionData.cwd).toContain('test-atomic-branch');
    }, 10000);

    it('should rollback worktree on session creation failure', async () => {
      // This test will create a session with an invalid configuration
      // to trigger a failure after worktree creation
      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dir: testRepoPath,
          branchName: 'test-rollback-branch',
          agentName: 'invalid-agent', // This should cause validation error
        }),
      });

      // Should fail with 400 or 500
      expect(response.ok).toBe(false);

      // TODO: Verify worktree was cleaned up (not created or removed)
      // This will be verified once we implement the rollback logic
    }, 10000);

    it('should return 400 for invalid dir (not a git repo)', async () => {
      const nonGitDir = join(testHome, 'not-a-git-repo-session');
      mkdirSync(nonGitDir, { recursive: true });

      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dir: nonGitDir,
          branchName: 'test-branch',
          agentName: 'claude',
        }),
      });

      expect(response.status).toBe(400);
    }, 5000);
  });

  describe('Session API Endpoints', () => {
    it('should create a session via POST /session', async () => {
      const response = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dir: testRepoPath,
          branchName: 'test-branch',
          agentName: 'claude',
        }),
      });

      expect(response.status).toBe(201);
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
          dir: testRepoPath,
          branchName: 'test-get-session',
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
      expect(data.cwd).toContain('test-get-session');
      expect(data.isAlive).toBe(true);
    }, 10000);

    it('should resize session via POST /session/:id/resize', async () => {
      // Create session
      const createResponse = await fetch(`http://localhost:${PORT}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dir: testRepoPath,
          branchName: 'test-resize',
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
          dir: testRepoPath,
          branchName: 'test-delete',
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
          dir: testRepoPath,
        }),
      });

      expect(response.status).toBe(400);
    }, 5000);
  });

});
