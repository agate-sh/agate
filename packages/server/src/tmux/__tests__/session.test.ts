import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { TmuxSessionManager } from '../session.js';
import { EventBus } from '../../event-bus.js';
import type { PtyOutputEvent, PtyExitEvent, PtyResizeEvent } from '@agate/shared';

describe('TmuxSessionManager', () => {
  let eventBus: EventBus;
  let sessionManager: TmuxSessionManager;
  const sessionId = 'test-session-123';
  const uniqueSuffix = Date.now() + '-' + Math.random().toString(36).substring(7);

  beforeEach(() => {
    eventBus = new EventBus();
    sessionManager = new TmuxSessionManager(eventBus, sessionId);
  });

  afterEach(async () => {
    if (sessionManager.isAlive()) {
      await sessionManager.kill();
    }
    eventBus.destroy();
  });

  describe('createSession', () => {
    it('should create a new tmux session with specified options', async () => {
      const options = {
        name: `test-tmux-session-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
        cols: 100,
        rows: 30,
      };

      await sessionManager.createSession(options);

      const info = sessionManager.getInfo();
      expect(info.id).toBe(sessionId);
      expect(info.name).toBe(options.name);
      expect(info.agent).toBe(options.agent);
      expect(info.cwd).toBe(options.cwd);
      expect(info.cols).toBe(options.cols);
      expect(info.rows).toBe(options.rows);
      expect(info.isAlive).toBe(true);
      expect(info.pid).toBeGreaterThan(0);
    });

    it('should use default cols/rows if not specified', async () => {
      const options = {
        name: `test-tmux-session-defaults-${uniqueSuffix}`,
        agent: 'gemini' as const,
        cwd: process.cwd(),
      };

      await sessionManager.createSession(options);

      const info = sessionManager.getInfo();
      expect(info.cols).toBe(80);
      expect(info.rows).toBe(24);
    });

    it('should throw if session already created', async () => {
      const options = {
        name: `test-duplicate-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      };

      await sessionManager.createSession(options);

      await expect(sessionManager.createSession(options)).rejects.toThrow(
        'Session already created',
      );
    });
  });

  describe('attachToSession', () => {
    it('should attach to an existing tmux session', async () => {
      // First create a session directly with tmux
      const sessionName = `test-attach-session-${uniqueSuffix}`;
      const { spawn } = await import('node-pty');
      const createProcess = spawn('tmux', ['new-session', '-d', '-s', sessionName], {
        name: 'xterm-256color',
      });

      // Wait for session creation
      await new Promise<void>((resolve) => {
        createProcess.onExit(() => resolve());
      });

      // Now attach to it
      await sessionManager.attachToSession(sessionName, 'claude', process.cwd());

      const info = sessionManager.getInfo();
      expect(info.name).toBe(sessionName);
      expect(info.isAlive).toBe(true);

      // Cleanup
      await sessionManager.kill();
    });

    it('should throw if session already created', async () => {
      await sessionManager.createSession({
        name: `test-session-1-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      await expect(
        sessionManager.attachToSession(`other-session-${uniqueSuffix}`, 'gemini', process.cwd()),
      ).rejects.toThrow('Session already created');
    });
  });

  describe('PTY event streaming', () => {
    it('should publish pty.output events when data is received', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      await sessionManager.createSession({
        name: `test-output-session-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      // Send a command to generate output
      sessionManager.write('echo "test output"\n');

      // Wait for output
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Verify output events were published
      const outputEvents = publishSpy.mock.calls
        .map((call) => call[0])
        .filter((event) => event.type === 'pty.output') as PtyOutputEvent[];

      expect(outputEvents.length).toBeGreaterThan(0);
      expect(outputEvents[0]?.sessionId).toBe(sessionId);
      expect(outputEvents[0]?.data).toBeDefined();
    });

    it('should publish pty.exit event when session exits', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      await sessionManager.createSession({
        name: `test-exit-session-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      await sessionManager.kill();

      // Wait for exit event
      await new Promise((resolve) => setTimeout(resolve, 200));

      const exitEvents = publishSpy.mock.calls
        .map((call) => call[0])
        .filter((event) => event.type === 'pty.exit') as PtyExitEvent[];

      expect(exitEvents.length).toBeGreaterThan(0);
      expect(exitEvents[0]?.sessionId).toBe(sessionId);
      expect(exitEvents[0]?.exitCode).toBeDefined();
    });

    it('should publish pty.resize event when terminal is resized', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      await sessionManager.createSession({
        name: `test-resize-session-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      sessionManager.resize(120, 40);

      const resizeEvents = publishSpy.mock.calls
        .map((call) => call[0])
        .filter((event) => event.type === 'pty.resize') as PtyResizeEvent[];

      expect(resizeEvents.length).toBe(1);
      expect(resizeEvents[0]?.sessionId).toBe(sessionId);
      expect(resizeEvents[0]?.cols).toBe(120);
      expect(resizeEvents[0]?.rows).toBe(40);
    });
  });

  describe('resize', () => {
    it('should resize the PTY and publish event', async () => {
      await sessionManager.createSession({
        name: `test-resize-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      sessionManager.resize(150, 50);

      const info = sessionManager.getInfo();
      expect(info.cols).toBe(150);
      expect(info.rows).toBe(50);
    });

    it('should throw if no active PTY process', () => {
      expect(() => sessionManager.resize(100, 30)).toThrow('No active PTY process');
    });
  });

  describe('write', () => {
    it('should send data to the PTY', async () => {
      await sessionManager.createSession({
        name: `test-write-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      expect(() => sessionManager.write('test input')).not.toThrow();
    });

    it('should throw if no active PTY process', () => {
      expect(() => sessionManager.write('test')).toThrow('No active PTY process');
    });
  });

  describe('sendKeys', () => {
    it('should send keys to tmux session', async () => {
      await sessionManager.createSession({
        name: `test-send-keys-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      await expect(sessionManager.sendKeys('C-c')).resolves.not.toThrow();
    });

    it('should throw if no session name set', async () => {
      await expect(sessionManager.sendKeys('C-c')).rejects.toThrow(
        'No session name set',
      );
    });
  });

  describe('capturePane', () => {
    it('should capture pane content', async () => {
      await sessionManager.createSession({
        name: `test-capture-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      // Send some output
      sessionManager.write('echo "capture this"\n');
      await new Promise((resolve) => setTimeout(resolve, 300));

      const content = await sessionManager.capturePane();
      expect(typeof content).toBe('string');
      expect(content.length).toBeGreaterThan(0);
    });

    it('should throw if no session name set', async () => {
      await expect(sessionManager.capturePane()).rejects.toThrow(
        'No session name set',
      );
    });
  });

  describe('kill', () => {
    it('should kill the PTY process and tmux session', async () => {
      await sessionManager.createSession({
        name: `test-kill-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      expect(sessionManager.isAlive()).toBe(true);

      await sessionManager.kill();

      expect(sessionManager.isAlive()).toBe(false);
      const info = sessionManager.getInfo();
      expect(info.isAlive).toBe(false);
    });

    it('should not throw if session already dead', async () => {
      await sessionManager.createSession({
        name: `test-already-dead-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      await sessionManager.kill();
      await expect(sessionManager.kill()).resolves.not.toThrow();
    });
  });

  describe('getInfo', () => {
    it('should return session information', async () => {
      const options = {
        name: `test-info-${uniqueSuffix}`,
        agent: 'codex' as const,
        cwd: '/tmp',
        cols: 90,
        rows: 25,
      };

      await sessionManager.createSession(options);

      const info = sessionManager.getInfo();
      expect(info).toMatchObject({
        id: sessionId,
        name: options.name,
        agent: options.agent,
        cwd: options.cwd,
        cols: options.cols,
        rows: options.rows,
        isAlive: true,
      });
      expect(info.pid).toBeGreaterThan(0);
    });
  });

  describe('isAlive', () => {
    it('should return true for active session', async () => {
      await sessionManager.createSession({
        name: `test-alive-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      expect(sessionManager.isAlive()).toBe(true);
    });

    it('should return false for killed session', async () => {
      await sessionManager.createSession({
        name: `test-dead-${uniqueSuffix}`,
        agent: 'claude' as const,
        cwd: process.cwd(),
      });

      await sessionManager.kill();

      expect(sessionManager.isAlive()).toBe(false);
    });

    it('should return false for uninitialized session', () => {
      expect(sessionManager.isAlive()).toBe(false);
    });
  });
});
