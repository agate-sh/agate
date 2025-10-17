import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { Server } from 'http';
import { EventBus } from '../event-bus.js';
import { WorktreeManager } from '../git/worktree.js';
import { StateManager } from '../state/manager.js';
import { createServer } from '../server.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type {
  CreateSessionResponse,
  GetSessionResponse,
  SendSessionInputResponse,
  ResizeSessionResponse,
  DeleteSessionResponse,
} from '@agate/shared';
import { join } from 'path';
import { mkdirSync, rmSync, existsSync } from 'fs';
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
  let server: Server;
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
