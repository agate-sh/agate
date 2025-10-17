import { createHash } from 'crypto';

export interface StatusUpdateResult {
  updated: boolean;
  hasPrompt: boolean;
}

/**
 * Monitors tmux session output for changes and prompt detection
 * Uses hash-based change detection for efficiency
 * Ported from go/pkg/tmux/monitor.go
 */
export class StatusMonitor {
  private prevOutputHash: string | null = null;

  /**
   * Check if content has changed and if there's a prompt waiting
   * @param content - Terminal output content to check
   * @returns Object with updated (content changed) and hasPrompt flags
   */
  hasUpdated(content: string): StatusUpdateResult {
    // Detect generic prompt patterns
    const trimmed = content.trim();
    const hasPrompt =
      trimmed.endsWith('>') || trimmed.endsWith('$') || trimmed.endsWith(':');

    // Check if content has changed using hash comparison
    const currentHash = this.hash(content);
    if (this.prevOutputHash !== currentHash) {
      this.prevOutputHash = currentHash;
      return { updated: true, hasPrompt };
    }

    return { updated: false, hasPrompt };
  }

  /**
   * Generate SHA-256 hash of content for efficient comparison
   */
  private hash(content: string): string {
    return createHash('sha256').update(content).digest('hex');
  }
}
