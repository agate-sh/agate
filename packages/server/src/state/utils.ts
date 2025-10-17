import { homedir } from 'node:os';
import { join } from 'node:path';
import { mkdirSync } from 'node:fs';

/**
 * Get the agate configuration directory path
 * @returns Path to ~/.agate
 */
export function getAgateDir(): string {
  return join(homedir(), '.agate');
}

/**
 * Ensure the agate directory exists with proper permissions
 */
export function ensureAgateDir(): void {
  const agateDir = getAgateDir();
  mkdirSync(agateDir, { recursive: true, mode: 0o755 });
}

/**
 * Get the state file path
 * @returns Path to ~/.agate/state.json
 */
export function getStateFilePath(): string {
  return join(getAgateDir(), 'state.json');
}
