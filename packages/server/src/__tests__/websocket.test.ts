import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { Server, createServer as createHttpServer } from 'http';
import { EventBus } from '../event-bus.js';
import { WorktreeManager } from '../git/worktree.js';
import { StateManager } from '../state/manager.js';
import { createServer } from '../server.js';
import { setupWebSocket } from '../websocket.js';
import { sessions } from '../routes/session.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type { CreateSessionResponse } from '@agate/shared';
import { randomUUID } from 'crypto';
import { join } from 'path';
import { mkdirSync, rmSync, existsSync } from 'fs';
import { homedir } from 'os';
import WebSocket from 'ws';

/**
 * WebSocket integration tests validating real-time PTY I/O streaming
 */
describe('WebSocket Integration Tests', () => {
  let testRepoPath: string;
  let worktreePath: string;
  let server: Server;
  let eventBus: EventBus;
  let stateManager: StateManager;
  const PORT = 3002; // Use different port to avoid conflicts with other tests
  const testHome = join(homedir(), '.agate-test-websocket-' + Date.now());
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
    server = createHttpServer(app);

    // 5. Setup WebSocket server
    setupWebSocket(server, eventBus, sessions);

    // Start server
    await new Promise<void>((resolve) => {
      server.listen(PORT, () => {
        console.log(`WebSocket test server listening on port ${PORT}`);
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

  it('should stream PTY output via WebSocket', async () => {
    // 1. Create session via HTTP (existing)
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

    // 2. Connect WebSocket
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    // 3. Subscribe to session
    ws.send(JSON.stringify({
      type: 'subscribe',
      sessionId
    }));

    // 4. Collect output
    const output: string[] = [];
    ws.on('message', (data) => {
      const msg = JSON.parse(data.toString());
      if (msg.type === 'pty:output') {
        output.push(msg.data);
      }
    });

    // 5. Send input via WebSocket
    ws.send(JSON.stringify({
      type: 'pty:input',
      sessionId,
      data: 'echo "WS_TEST_MARKER"\n'
    }));

    // 6. Wait for output
    await new Promise(resolve => setTimeout(resolve, 1000));

    // 7. Verify
    const allOutput = output.join('');
    expect(allOutput).toContain('WS_TEST_MARKER');

    ws.close();
  }, 10000);

  it('should handle multiple concurrent WebSocket clients', async () => {
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

    // Connect two clients
    const ws1 = new WebSocket(`ws://localhost:${PORT}/ws`);
    const ws2 = new WebSocket(`ws://localhost:${PORT}/ws`);

    await Promise.all([
      new Promise(resolve => ws1.once('open', resolve)),
      new Promise(resolve => ws2.once('open', resolve))
    ]);

    // Both subscribe
    ws1.send(JSON.stringify({ type: 'subscribe', sessionId }));
    ws2.send(JSON.stringify({ type: 'subscribe', sessionId }));

    // Collect output from both
    const output1: string[] = [];
    const output2: string[] = [];

    ws1.on('message', (data) => {
      const msg = JSON.parse(data.toString());
      if (msg.type === 'pty:output') output1.push(msg.data);
    });

    ws2.on('message', (data) => {
      const msg = JSON.parse(data.toString());
      if (msg.type === 'pty:output') output2.push(msg.data);
    });

    // Send input from first client
    ws1.send(JSON.stringify({
      type: 'pty:input',
      sessionId,
      data: 'echo "MULTI_CLIENT"\n'
    }));

    await new Promise(resolve => setTimeout(resolve, 1000));

    // Both clients should receive the output
    expect(output1.join('')).toContain('MULTI_CLIENT');
    expect(output2.join('')).toContain('MULTI_CLIENT');

    ws1.close();
    ws2.close();
  }, 10000);

  it('should handle graceful unsubscribe', async () => {
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

    // Connect and subscribe
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    ws.send(JSON.stringify({ type: 'subscribe', sessionId }));

    // Collect initial output
    const output: string[] = [];
    ws.on('message', (data) => {
      const msg = JSON.parse(data.toString());
      if (msg.type === 'pty:output') output.push(msg.data);
    });

    // Send input
    ws.send(JSON.stringify({
      type: 'pty:input',
      sessionId,
      data: 'echo "BEFORE_UNSUB"\n'
    }));

    await new Promise(resolve => setTimeout(resolve, 500));

    // Unsubscribe
    ws.send(JSON.stringify({ type: 'unsubscribe' }));

    // Send more input (should not receive output)
    ws.send(JSON.stringify({
      type: 'pty:input',
      sessionId,
      data: 'echo "AFTER_UNSUB"\n'
    }));

    await new Promise(resolve => setTimeout(resolve, 500));

    // Verify only received output before unsubscribe
    const allOutput = output.join('');
    expect(allOutput).toContain('BEFORE_UNSUB');
    expect(allOutput).not.toContain('AFTER_UNSUB');

    ws.close();
  }, 10000);

  it('should handle invalid message types gracefully', async () => {
    // Connect WebSocket
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    // Send invalid message type
    ws.send(JSON.stringify({
      type: 'invalid:type',
      data: 'test'
    }));

    // Should not crash - wait a bit
    await new Promise(resolve => setTimeout(resolve, 100));

    // Should still be connected
    expect(ws.readyState).toBe(WebSocket.OPEN);

    ws.close();
  }, 5000);

  it('should cleanup event listeners on WebSocket close', async () => {
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

    // Connect and subscribe
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    ws.send(JSON.stringify({ type: 'subscribe', sessionId }));

    // Wait for subscription
    await new Promise(resolve => setTimeout(resolve, 100));

    // Close connection
    ws.close();
    await new Promise(resolve => ws.once('close', resolve));

    // Send input to session via HTTP (should not crash server)
    const inputResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}/input`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        data: 'echo "AFTER_DISCONNECT"\n',
      }),
    });

    expect(inputResponse.ok).toBe(true);
  }, 10000);
});
