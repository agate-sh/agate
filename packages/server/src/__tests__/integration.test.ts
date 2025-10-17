import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import express, { type Express } from 'express';
import { Server } from 'http';
import { EventBus } from '../event-bus.js';
import { TmuxSessionManager } from '../tmux/session.js';
import { WorktreeManager } from '../git/worktree.js';
import { createTestRepo, cleanupTestRepo, createInitialCommit } from '../git/__tests__/test-helpers.js';
import type { PtyOutputEvent } from '@agate/shared';
import { randomUUID } from 'crypto';
import { join } from 'path';

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
  let app: Express;
  let server: Server;
  let eventBus: EventBus;
  let sessionManager: TmuxSessionManager;
  const PORT = 3001; // Use different port to avoid conflicts

  beforeAll(async () => {
    // 1. Create test git repository
    const { path, git } = await createTestRepo();
    testRepoPath = path;
    await createInitialCommit(git);

    // 2. Create worktree
    const worktreeManager = new WorktreeManager(testRepoPath);
    const branchName = `test-branch-${Date.now()}`;
    worktreePath = join(testRepoPath, '..', `worktree-${branchName}`);
    await worktreeManager.create(worktreePath, branchName);

    // 3. Setup Express server with EventBus
    eventBus = new EventBus();
    app = express();

    app.get('/events', (req, res) => {
      const clientId = (req.query.clientId as string) || randomUUID();
      eventBus.subscribe(clientId, res);
    });

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
});
