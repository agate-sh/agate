import * as pty from 'node-pty';
import { EventBus } from '../event-bus.js';
import type { AgentName } from '@agate/shared';
import type {
  PtyOutputEvent,
  PtyResizeEvent,
  PtyExitEvent,
} from '@agate/shared';

export interface TmuxSessionOptions {
  /** Session name (must be unique and valid tmux name) */
  name: string;
  /** Agent to launch in the session */
  agent: AgentName;
  /** Working directory for the session */
  cwd: string;
  /** Initial terminal size */
  cols?: number;
  rows?: number;
}

export interface TmuxSessionInfo {
  id: string;
  name: string;
  agent: AgentName;
  cwd: string;
  cols: number;
  rows: number;
  pid: number;
  isAlive: boolean;
}

/**
 * Manages a single tmux session via node-pty
 * Handles lifecycle, I/O streaming, and event emission
 */
export class TmuxSessionManager {
  private ptyProcess: pty.IPty | null = null;
  private eventBus: EventBus;
  private sessionId: string;
  private sessionName: string;
  private agent: AgentName;
  private cwd: string;
  private isAliveFlag = false;

  constructor(eventBus: EventBus, sessionId: string) {
    this.eventBus = eventBus;
    this.sessionId = sessionId;
    this.sessionName = '';
    this.agent = 'default';
    this.cwd = process.cwd();
  }

  /**
   * Create a new tmux session
   */
  async createSession(options: TmuxSessionOptions): Promise<void> {
    if (this.ptyProcess) {
      throw new Error('Session already created');
    }

    this.sessionName = options.name;
    this.agent = options.agent;
    this.cwd = options.cwd;

    const cols = options.cols ?? 80;
    const rows = options.rows ?? 24;

    // Spawn tmux with new session
    this.ptyProcess = pty.spawn(
      'tmux',
      ['new-session', '-s', options.name, '-c', options.cwd, options.agent],
      {
        name: 'xterm-256color',
        cols,
        rows,
        cwd: options.cwd,
        env: process.env as { [key: string]: string },
      },
    );

    this.isAliveFlag = true;
    this.setupEventHandlers();
  }

  /**
   * Attach to an existing tmux session
   */
  async attachToSession(
    name: string,
    agent: AgentName,
    cwd: string,
    cols = 80,
    rows = 24,
  ): Promise<void> {
    if (this.ptyProcess) {
      throw new Error('Session already created');
    }

    this.sessionName = name;
    this.agent = agent;
    this.cwd = cwd;

    // Attach to existing tmux session
    this.ptyProcess = pty.spawn('tmux', ['attach-session', '-t', name], {
      name: 'xterm-256color',
      cols,
      rows,
      cwd,
      env: process.env as { [key: string]: string },
    });

    this.isAliveFlag = true;
    this.setupEventHandlers();
  }

  /**
   * Setup PTY event handlers for output, exit, etc.
   */
  private setupEventHandlers(): void {
    if (!this.ptyProcess) {
      return;
    }

    // Stream PTY output to event bus
    this.ptyProcess.onData((data: string) => {
      const event: PtyOutputEvent = {
        type: 'pty.output',
        sessionId: this.sessionId,
        data,
        timestamp: new Date().toISOString(),
      };
      this.eventBus.publish(event);
    });

    // Handle PTY exit
    this.ptyProcess.onExit(({ exitCode }) => {
      this.isAliveFlag = false;
      const event: PtyExitEvent = {
        type: 'pty.exit',
        sessionId: this.sessionId,
        exitCode,
        timestamp: new Date().toISOString(),
      };
      this.eventBus.publish(event);
    });
  }

  /**
   * Resize the terminal
   */
  resize(cols: number, rows: number): void {
    if (!this.ptyProcess) {
      throw new Error('No active PTY process');
    }

    this.ptyProcess.resize(cols, rows);

    const event: PtyResizeEvent = {
      type: 'pty.resize',
      sessionId: this.sessionId,
      cols,
      rows,
      timestamp: new Date().toISOString(),
    };
    this.eventBus.publish(event);
  }

  /**
   * Send input to the PTY
   */
  write(data: string): void {
    if (!this.ptyProcess) {
      throw new Error('No active PTY process');
    }
    this.ptyProcess.write(data);
  }

  /**
   * Send keys to tmux session
   */
  async sendKeys(keys: string): Promise<void> {
    if (!this.sessionName) {
      throw new Error('No session name set');
    }

    return new Promise((resolve, reject) => {
      const child = pty.spawn(
        'tmux',
        ['send-keys', '-t', this.sessionName, keys],
        {
          name: 'xterm-256color',
          cwd: this.cwd,
        },
      );

      child.onExit(({ exitCode }) => {
        if (exitCode === 0) {
          resolve();
        } else {
          reject(new Error(`tmux send-keys failed with code ${exitCode}`));
        }
      });
    });
  }

  /**
   * Capture pane content
   */
  async capturePane(): Promise<string> {
    if (!this.sessionName) {
      throw new Error('No session name set');
    }

    return new Promise((resolve, reject) => {
      let output = '';

      const child = pty.spawn(
        'tmux',
        ['capture-pane', '-t', this.sessionName, '-p'],
        {
          name: 'xterm-256color',
          cwd: this.cwd,
        },
      );

      child.onData((data: string) => {
        output += data;
      });

      child.onExit(({ exitCode }) => {
        if (exitCode === 0) {
          resolve(output);
        } else {
          reject(new Error(`tmux capture-pane failed with code ${exitCode}`));
        }
      });
    });
  }

  /**
   * Kill the session and clean up
   */
  async kill(): Promise<void> {
    if (this.ptyProcess) {
      this.ptyProcess.kill();
      this.ptyProcess = null;
    }

    // Also kill the tmux session if it exists
    if (this.sessionName) {
      try {
        await new Promise<void>((resolve, reject) => {
          const child = pty.spawn(
            'tmux',
            ['kill-session', '-t', this.sessionName],
            {
              name: 'xterm-256color',
              cwd: this.cwd,
            },
          );

          child.onExit(({ exitCode }) => {
            // Exit code 1 often means session doesn't exist, which is fine
            if (exitCode === 0 || exitCode === 1) {
              resolve();
            } else {
              reject(
                new Error(`tmux kill-session failed with code ${exitCode}`),
              );
            }
          });
        });
      } catch (error) {
        // Ignore errors - session might already be dead
        console.error('Error killing tmux session:', error);
      }
    }

    this.isAliveFlag = false;
  }

  /**
   * Get current session information
   */
  getInfo(): TmuxSessionInfo {
    return {
      id: this.sessionId,
      name: this.sessionName,
      agent: this.agent,
      cwd: this.cwd,
      cols: this.ptyProcess?.cols ?? 80,
      rows: this.ptyProcess?.rows ?? 24,
      pid: this.ptyProcess?.pid ?? -1,
      isAlive: this.isAliveFlag && this.ptyProcess !== null,
    };
  }

  /**
   * Check if session is alive
   */
  isAlive(): boolean {
    return this.isAliveFlag && this.ptyProcess !== null;
  }
}
