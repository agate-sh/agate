import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { mkdirSync, rmSync, writeFileSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';
import { execSync } from 'child_process';
import { EventBus } from '../../event-bus.js';
import { StateManager } from '../../state/manager.js';
import { createHonoServer } from '../../server.hono.js';

describe('Git Routes', () => {
  let testRepoPath: string;
  let app: ReturnType<typeof createHonoServer>;
  let eventBus: EventBus;
  let stateManager: StateManager;

  beforeAll(async () => {
    // Initialize dependencies
    eventBus = new EventBus();
    stateManager = await StateManager.create();
    app = createHonoServer(eventBus, stateManager);

    // Create a temporary git repository for testing
    testRepoPath = join(homedir(), '.agate-test-git-routes-' + Date.now());
    mkdirSync(testRepoPath, { recursive: true });

    // Initialize git repo
    execSync('git init', { cwd: testRepoPath });
    execSync('git config user.email "test@example.com"', { cwd: testRepoPath });
    execSync('git config user.name "Test User"', { cwd: testRepoPath });

    // Create initial commit on main branch
    writeFileSync(join(testRepoPath, 'README.md'), '# Test Repo');
    execSync('git add README.md', { cwd: testRepoPath });
    execSync('git commit -m "Initial commit"', { cwd: testRepoPath });
  });

  afterAll(() => {
    // Cleanup
    eventBus?.destroy();

    // Cleanup test repo
    if (testRepoPath) {
      rmSync(testRepoPath, { recursive: true, force: true });
    }
  });

  describe('GET /git/head', () => {
    it('should return the current branch', async () => {
      const response = await app.request(
        `/git/head?repoPath=${encodeURIComponent(testRepoPath)}`
      );

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data).toHaveProperty('branch');
      expect(data.branch).toBe('main');
    });

    it('should return master branch when on master', async () => {
      // Checkout a new branch
      execSync('git checkout -b master', { cwd: testRepoPath });

      const response = await app.request(
        `/git/head?repoPath=${encodeURIComponent(testRepoPath)}`
      );

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.branch).toBe('master');

      // Switch back to main
      execSync('git checkout main', { cwd: testRepoPath });
    });

    it('should return the feature branch when checked out', async () => {
      // Create and checkout a feature branch
      execSync('git checkout -b feature/test-branch', { cwd: testRepoPath });

      const response = await app.request(
        `/git/head?repoPath=${encodeURIComponent(testRepoPath)}`
      );

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.branch).toBe('feature/test-branch');

      // Switch back to main
      execSync('git checkout main', { cwd: testRepoPath });
      execSync('git branch -D feature/test-branch', { cwd: testRepoPath });
    });

    it('should return 400 when repoPath is missing', async () => {
      const response = await app.request('/git/head');

      expect(response.status).toBe(400);
    });

    it('should return 500 when repo path does not exist', async () => {
      const invalidPath = '/path/that/does/not/exist';
      const response = await app.request(
        `/git/head?repoPath=${encodeURIComponent(invalidPath)}`
      );

      expect(response.status).toBe(500);
      const data = await response.json();
      expect(data).toHaveProperty('error');
    });

    it('should return 500 when repo path is not a git repository', async () => {
      const notARepo = join(homedir(), '.agate-test-not-a-repo-' + Date.now());
      mkdirSync(notARepo, { recursive: true });

      try {
        const response = await app.request(
          `/git/head?repoPath=${encodeURIComponent(notARepo)}`
        );

        expect(response.status).toBe(500);
        const data = await response.json();
        expect(data).toHaveProperty('error');
      } finally {
        rmSync(notARepo, { recursive: true, force: true });
      }
    });
  });
});
