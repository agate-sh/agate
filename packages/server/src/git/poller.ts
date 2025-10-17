import { simpleGit, SimpleGit } from 'simple-git';
import { EventBus } from '../event-bus';
import { getFileStatus } from './status';
import type { GitStatusEvent } from '@agate/shared';

interface PollerOptions {
  interval?: number; // Polling interval in milliseconds
}

interface WatchedRepo {
  sessionId: string;
  path: string;
  git: SimpleGit;
  lastHash: string | null; // Hash of last known status
  interval: number; // Polling interval in milliseconds (adaptive based on repo size)
  timerId: NodeJS.Timeout | null; // Individual timer for this repo
}

/**
 * GitStatusPoller automatically polls git repositories for status changes
 * and publishes events to the EventBus when changes are detected.
 */
export class GitStatusPoller {
  private repos: Map<string, WatchedRepo> = new Map();
  private isRunning = false;

  constructor(
    private eventBus: EventBus,
    options: PollerOptions = {}
  ) {
    // Options preserved for backward compatibility but intervals are now adaptive
  }

  /**
   * Start the polling loop for all watched repositories
   */
  start(): void {
    if (this.isRunning) {
      return;
    }

    this.isRunning = true;

    // Start timers for all existing repos
    for (const repo of this.repos.values()) {
      this.startRepoTimer(repo);
    }
  }

  /**
   * Stop the polling loop for all repositories
   */
  stop(): void {
    this.isRunning = false;

    // Stop all repo timers
    for (const repo of this.repos.values()) {
      if (repo.timerId) {
        clearInterval(repo.timerId);
        repo.timerId = null;
      }
    }
  }

  /**
   * Start polling timer for a specific repository
   */
  private startRepoTimer(repo: WatchedRepo): void {
    if (repo.timerId) {
      clearInterval(repo.timerId);
    }

    // Start interval timer for this repo
    repo.timerId = setInterval(() => {
      this.pollRepo(repo);
    }, repo.interval);

    // Do an initial poll immediately
    this.pollRepo(repo);
  }

  /**
   * Add a repository to watch for changes
   * Times the initial git status call to determine optimal polling interval
   */
  async watch(sessionId: string, repoPath: string): Promise<void> {
    const git = simpleGit(repoPath);

    // Time the first git status call to determine repo speed
    const start = Date.now();
    const initialStatus = await getFileStatus(git);
    const duration = Date.now() - start;

    const interval = chooseInterval(duration);
    const initialHash = this.createStatusHash(initialStatus);

    const repo: WatchedRepo = {
      sessionId,
      path: repoPath,
      git,
      lastHash: initialHash,
      interval,
      timerId: null,
    };

    this.repos.set(sessionId, repo);

    // If poller is already running, start timer for this repo
    if (this.isRunning) {
      this.startRepoTimer(repo);
    }

    // Publish initial status
    const event: GitStatusEvent = {
      type: 'git.status',
      timestamp: new Date().toISOString(),
      sessionId,
      status: {
        branch: initialStatus.current || 'unknown',
        ahead: initialStatus.ahead,
        behind: initialStatus.behind,
        staged: initialStatus.staged.length,
        modified: initialStatus.modified.length,
        untracked: initialStatus.not_added.length,
      },
    };
    this.eventBus.publish(event);
  }

  /**
   * Remove a repository from watching
   */
  unwatch(sessionId: string): void {
    const repo = this.repos.get(sessionId);
    if (repo?.timerId) {
      clearInterval(repo.timerId);
    }
    this.repos.delete(sessionId);
  }

  /**
   * Poll a single repository for changes
   */
  private async pollRepo(repo: WatchedRepo): Promise<void> {
    try {
      const status = await getFileStatus(repo.git);

      // Create a hash of the status to detect changes
      const statusHash = this.createStatusHash(status);

      // Only publish if status has changed (smart polling)
      if (statusHash !== repo.lastHash) {
        const event: GitStatusEvent = {
          type: 'git.status',
          timestamp: new Date().toISOString(),
          sessionId: repo.sessionId,
          status: {
            branch: status.current || 'unknown',
            ahead: status.ahead,
            behind: status.behind,
            staged: status.staged.length,
            modified: status.modified.length,
            untracked: status.not_added.length,
          },
        };

        this.eventBus.publish(event);
        repo.lastHash = statusHash;
      }
    } catch (error) {
      // Silently ignore errors for invalid repositories
      // This allows the poller to continue working for other repos
      console.error(`Error polling repo ${repo.path}:`, error);
    }
  }

  /**
   * Create a hash of the git status to detect changes
   */
  private createStatusHash(status: ReturnType<typeof getFileStatus> extends Promise<infer T> ? T : never): string {
    return JSON.stringify({
      current: status.current,
      tracking: status.tracking,
      ahead: status.ahead,
      behind: status.behind,
      files: status.files.map((f) => ({
        path: f.path,
        index: f.index,
        working_dir: f.working_dir,
      })),
    });
  }
}

/**
 * Choose polling interval based on how long git status takes
 * Matches Go implementation logic (without file count check)
 */
function chooseInterval(statusDurationMs: number): number {
  if (statusDurationMs > 1500) {
    return 4000; // 4 seconds for slow repos
  } else if (statusDurationMs > 700) {
    return 2000; // 2 seconds for medium repos
  } else {
    return 1000; // 1 second for fast repos
  }
}
