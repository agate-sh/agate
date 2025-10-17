import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, rmSync, existsSync, statSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { getAgateDir, ensureAgateDir, getStateFilePath } from '../utils.js';

describe('State Utils', () => {
  const testHome = join(homedir(), '.agate-test-' + Date.now());
  const originalHome = process.env.HOME;

  beforeEach(() => {
    // Override HOME for testing
    process.env.HOME = testHome;
    // Clean up any existing test directory
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
    mkdirSync(testHome, { recursive: true });
  });

  afterEach(() => {
    // Restore original HOME
    process.env.HOME = originalHome;
    // Clean up test directory
    if (existsSync(testHome)) {
      rmSync(testHome, { recursive: true, force: true });
    }
  });

  describe('getAgateDir', () => {
    it('should return ~/.agate path', () => {
      const agateDir = getAgateDir();
      expect(agateDir).toBe(join(testHome, '.agate'));
    });
  });

  describe('ensureAgateDir', () => {
    it('should create .agate directory if it does not exist', () => {
      const agateDir = getAgateDir();
      expect(existsSync(agateDir)).toBe(false);

      ensureAgateDir();

      expect(existsSync(agateDir)).toBe(true);
      const stats = statSync(agateDir);
      expect(stats.isDirectory()).toBe(true);
    });

    it('should not throw if directory already exists', () => {
      const agateDir = getAgateDir();
      mkdirSync(agateDir, { recursive: true });

      expect(() => ensureAgateDir()).not.toThrow();
      expect(existsSync(agateDir)).toBe(true);
    });

    it('should create directory with proper permissions (0o755)', () => {
      ensureAgateDir();
      const agateDir = getAgateDir();
      const stats = statSync(agateDir);

      // On Unix systems, check permissions (skip on Windows)
      if (process.platform !== 'win32') {
        const mode = stats.mode & parseInt('777', 8);
        expect(mode).toBe(parseInt('755', 8));
      }
    });
  });

  describe('getStateFilePath', () => {
    it('should return ~/.agate/state.json path', () => {
      const stateFile = getStateFilePath();
      expect(stateFile).toBe(join(testHome, '.agate', 'state.json'));
    });
  });
});
