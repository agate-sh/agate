import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { serve, type ServerType } from '@hono/node-server';
import { createNodeWebSocket } from '@hono/node-ws';
import { EventBus } from '../event-bus.js';
import { WorktreeManager } from '../git/worktree.js';
import { StateManager } from '../state/manager.js';
import { createHonoServer } from '../server.hono.js';
import { createWebSocketHandler } from '../websocket.hono.js';
import { sessions } from '../routes/session.hono.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type { CreateSessionResponse } from '@agate/shared';
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
  let server: ServerType;
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

    console.log(`WebSocket test server listening on port ${PORT}`);
  });

  afterEach(async () => {
    // Cleanup all sessions after each test to prevent interference
    for (const [sessionId, sessionManager] of sessions.entries()) {
      await sessionManager.kill();
      sessions.delete(sessionId);
    }
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

    // Wait for PTY to fully initialize
    await new Promise(resolve => setTimeout(resolve, 200));

    // 2. Connect WebSocket
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    // 3. Subscribe to session
    ws.send(JSON.stringify({
      type: 'subscribe',
      sessionId
    }));

    // Wait for subscription to be established
    await new Promise(resolve => setTimeout(resolve, 100));

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

    // Wait for PTY to fully initialize
    await new Promise(resolve => setTimeout(resolve, 200));

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

    // Wait for subscriptions to be established
    await new Promise(resolve => setTimeout(resolve, 100));

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

    // Wait for PTY to fully initialize
    await new Promise(resolve => setTimeout(resolve, 200));

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

    // Wait for PTY to fully initialize
    await new Promise(resolve => setTimeout(resolve, 200));

    // Connect and subscribe
    const ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    await new Promise(resolve => ws.once('open', resolve));

    ws.send(JSON.stringify({ type: 'subscribe', sessionId }));

    // Wait for subscription
    await new Promise(resolve => setTimeout(resolve, 100));

    // Close connection
    ws.close();
    await new Promise(resolve => ws.once('close', resolve));

    // Trigger session operation via HTTP (should not crash server even though WS is closed)
    const resizeResponse = await fetch(`http://localhost:${PORT}/session/${sessionId}/resize`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        cols: 100,
        rows: 30,
      }),
    });

    expect(resizeResponse.ok).toBe(true);
  }, 10000);
});
