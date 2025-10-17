import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createTestRepo, cleanupTestRepo, createFile } from './test-helpers';
import { GitStatusPoller } from '../poller';
import { EventBus } from '../../event-bus';
import { SimpleGit } from 'simple-git';
import type { GitStatusEvent } from '@agate/shared';

describe('GitStatusPoller', () => {
  let repoPath: string;
  let git: SimpleGit;
  let eventBus: EventBus;
  let poller: GitStatusPoller;

  beforeEach(async () => {
    const testRepo = await createTestRepo();
    repoPath = testRepo.path;
    git = testRepo.git;

    // Create initial commit
    await createFile(git, 'README.md', '# Test\n');
    await git.add('README.md');
    await git.commit('Initial commit');

    eventBus = new EventBus();
  });

  afterEach(async () => {
    if (poller) {
      poller.stop();
    }
    if (eventBus) {
      eventBus.destroy();
    }
    await cleanupTestRepo(repoPath);
  });

  describe('Basic functionality', () => {
    it('should create a GitStatusPoller instance', () => {
      poller = new GitStatusPoller(eventBus);
      expect(poller).toBeDefined();
    });

    it('should start and stop polling', () => {
      poller = new GitStatusPoller(eventBus);
      expect(() => poller.start()).not.toThrow();
      expect(() => poller.stop()).not.toThrow();
    });

    it('should accept custom polling interval', () => {
      const customInterval = 500;
      poller = new GitStatusPoller(eventBus, { interval: customInterval });
      expect(poller).toBeDefined();
    });
  });

  describe('Repository management', () => {
    it('should add a repository to watch', async () => {
      poller = new GitStatusPoller(eventBus);
      await expect(poller.watch('session-1', repoPath)).resolves.not.toThrow();
    });

    it('should remove a repository from watching', async () => {
      poller = new GitStatusPoller(eventBus);
      await poller.watch('session-1', repoPath);
      expect(() => poller.unwatch('session-1')).not.toThrow();
    });

    it('should handle multiple repositories', async () => {
      // Create a second test repo
      const testRepo2 = await createTestRepo();
      const repoPath2 = testRepo2.path;
      await createFile(testRepo2.git, 'README.md', '# Test 2\n');
      await testRepo2.git.add('README.md');
      await testRepo2.git.commit('Initial commit');

      poller = new GitStatusPoller(eventBus);
      await poller.watch('session-1', repoPath);
      await poller.watch('session-2', repoPath2);
      expect(() => poller.stop()).not.toThrow();

      // Cleanup second repo
      await cleanupTestRepo(repoPath2);
    });
  });

  describe('Change detection', () => {
    it('should detect file changes and publish event', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus, { interval: 100 });
      await poller.watch('session-1', repoPath);
      poller.start();

      // Wait for initial poll
      await new Promise(resolve => setTimeout(resolve, 150));

      // Make a change
      await createFile(git, 'test.txt', 'content\n');

      // Wait for next poll
      await new Promise(resolve => setTimeout(resolve, 150));

      expect(publishSpy).toHaveBeenCalled();

      const gitStatusCalls = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      );
      expect(gitStatusCalls.length).toBeGreaterThan(0);

      const lastEvent = gitStatusCalls[gitStatusCalls.length - 1]?.[0] as GitStatusEvent;
      expect(lastEvent?.sessionId).toBe('session-1');
      expect(lastEvent?.status).toBeDefined();
    });

    it('should not publish if no changes detected', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus, { interval: 100 });
      await poller.watch('session-1', repoPath);
      poller.start();

      // Wait for initial poll
      await new Promise(resolve => setTimeout(resolve, 150));

      const initialCallCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      // Wait for another poll without changes
      await new Promise(resolve => setTimeout(resolve, 150));

      const finalCallCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      // Should have at most one more call (smart polling)
      expect(finalCallCount - initialCallCount).toBeLessThanOrEqual(1);
    });
  });

  describe('Smart polling', () => {
    it('should skip polling when idle', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus, { interval: 100 });
      await poller.watch('session-1', repoPath);
      poller.start();

      // Wait for initial poll
      await new Promise(resolve => setTimeout(resolve, 150));

      const initialGitStatusCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      // Wait for several polling intervals without any changes
      await new Promise(resolve => setTimeout(resolve, 400));

      const finalGitStatusCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      // Should not poll excessively when nothing changes
      // Allow some margin for initial polls
      expect(finalGitStatusCount - initialGitStatusCount).toBeLessThan(3);
    });

    it('should resume polling after changes', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus, { interval: 100 });
      await poller.watch('session-1', repoPath);
      poller.start();

      // Wait for initial poll
      await new Promise(resolve => setTimeout(resolve, 150));

      // Make a change
      await createFile(git, 'file1.txt', 'content\n');
      await new Promise(resolve => setTimeout(resolve, 150));

      // Make another change
      await createFile(git, 'file2.txt', 'content\n');
      await new Promise(resolve => setTimeout(resolve, 150));

      const gitStatusEvents = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      );

      expect(gitStatusEvents.length).toBeGreaterThan(0);
    });
  });

  describe('Adaptive intervals', () => {
    it('should publish initial status immediately on watch', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus);
      await poller.watch('session-1', repoPath);

      // Should publish immediately without needing to start()
      const gitStatusEvents = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      );
      expect(gitStatusEvents.length).toBe(1);

      const event = gitStatusEvents[0]?.[0] as GitStatusEvent;
      expect(event.sessionId).toBe('session-1');
    });

    it('should poll at independent intervals for different repos', async () => {
      // Create a second test repo
      const testRepo2 = await createTestRepo();
      const repoPath2 = testRepo2.path;
      await createFile(testRepo2.git, 'README.md', '# Test 2\n');
      await testRepo2.git.add('README.md');
      await testRepo2.git.commit('Initial commit');

      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus);

      // Watch both repos - they should get their own intervals
      await poller.watch('session-1', repoPath);
      await poller.watch('session-2', repoPath2);

      poller.start();

      // Both should have published initial status
      const initialEvents = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      );
      expect(initialEvents.length).toBe(2);

      // Make changes to both repos
      await createFile(git, 'test1.txt', 'content\n');
      await createFile(testRepo2.git, 'test2.txt', 'content\n');

      // Wait for polling to detect changes
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Both repos should have published updates
      const allEvents = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      );

      const session1Events = allEvents.filter(
        (call) => (call[0] as GitStatusEvent).sessionId === 'session-1'
      );
      const session2Events = allEvents.filter(
        (call) => (call[0] as GitStatusEvent).sessionId === 'session-2'
      );

      expect(session1Events.length).toBeGreaterThan(1);
      expect(session2Events.length).toBeGreaterThan(1);

      // Cleanup
      await cleanupTestRepo(repoPath2);
    });

    it('should stop polling for a repo when unwatched', async () => {
      const publishSpy = vi.spyOn(eventBus, 'publish');

      poller = new GitStatusPoller(eventBus);
      await poller.watch('session-1', repoPath);
      poller.start();

      // Get initial event count
      const initialCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      // Unwatch the repo
      poller.unwatch('session-1');

      // Wait for what would be several polling intervals
      await new Promise(resolve => setTimeout(resolve, 3000));

      // Should not have any new events after unwatching
      const finalCount = publishSpy.mock.calls.filter(
        (call) => call[0]?.type === 'git.status'
      ).length;

      expect(finalCount).toBe(initialCount);
    });
  });

});
