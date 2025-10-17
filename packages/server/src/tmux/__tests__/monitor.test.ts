import { describe, it, expect, beforeEach } from 'vitest';
import { StatusMonitor } from '../monitor.js';

describe('StatusMonitor', () => {
  let monitor: StatusMonitor;

  beforeEach(() => {
    monitor = new StatusMonitor();
  });

  describe('hasUpdated', () => {
    it('should return updated=true on first check', () => {
      const content = 'Initial terminal output';
      const { updated, hasPrompt } = monitor.hasUpdated(content);

      expect(updated).toBe(true);
      expect(hasPrompt).toBe(false);
    });

    it('should return updated=false when content unchanged', () => {
      const content = 'Same content';

      monitor.hasUpdated(content);
      const { updated, hasPrompt } = monitor.hasUpdated(content);

      expect(updated).toBe(false);
      expect(hasPrompt).toBe(false);
    });

    it('should return updated=true when content changes', () => {
      const content1 = 'Original content';
      const content2 = 'Modified content';

      monitor.hasUpdated(content1);
      const { updated } = monitor.hasUpdated(content2);

      expect(updated).toBe(true);
    });

    it('should detect generic prompt with >', () => {
      const content = 'Some output\n>';
      const { hasPrompt } = monitor.hasUpdated(content);

      expect(hasPrompt).toBe(true);
    });

    it('should detect dollar sign prompt', () => {
      const content = 'user@hostname ~/project $';
      const { hasPrompt } = monitor.hasUpdated(content);

      expect(hasPrompt).toBe(true);
    });

    it('should detect colon prompt', () => {
      const content = 'Enter your choice:';
      const { hasPrompt } = monitor.hasUpdated(content);

      expect(hasPrompt).toBe(true);
    });

    it('should not detect prompt in regular output', () => {
      const content = 'Just regular terminal output';
      const { hasPrompt } = monitor.hasUpdated(content);

      expect(hasPrompt).toBe(false);
    });

    it('should use efficient hash comparison for large content', () => {
      const largeContent = 'A'.repeat(100000);

      const { updated: updated1 } = monitor.hasUpdated(largeContent);
      expect(updated1).toBe(true);

      const { updated: updated2 } = monitor.hasUpdated(largeContent);
      expect(updated2).toBe(false);

      const largeContent2 = 'A'.repeat(100000) + 'B';
      const { updated: updated3 } = monitor.hasUpdated(largeContent2);
      expect(updated3).toBe(true);
    });

    it('should detect even small changes in content', () => {
      const content1 = 'Hello World';
      const content2 = 'Hello World!';

      monitor.hasUpdated(content1);
      const { updated } = monitor.hasUpdated(content2);

      expect(updated).toBe(true);
    });

    it('should return both updated=true and hasPrompt=true when appropriate', () => {
      const content1 = 'Initial output';
      const content2 = 'New output\nuser@host $';

      monitor.hasUpdated(content1);
      const { updated, hasPrompt } = monitor.hasUpdated(content2);

      expect(updated).toBe(true);
      expect(hasPrompt).toBe(true);
    });

    it('should return updated=false and hasPrompt=true when content unchanged but has prompt', () => {
      const content = 'Output with prompt $';

      monitor.hasUpdated(content);
      const { updated, hasPrompt } = monitor.hasUpdated(content);

      expect(updated).toBe(false);
      expect(hasPrompt).toBe(true);
    });
  });
});
