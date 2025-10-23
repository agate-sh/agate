import { AGATE_SERVER_PORT } from '@agate/shared';
import { execa } from 'execa';
import chalk from 'chalk';
import { createRequire } from 'module';

const PORT = process.env.PORT ? parseInt(process.env.PORT, 10) : AGATE_SERVER_PORT;
const SERVER_URL = `http://localhost:${PORT}`;
const SERVER_START_TIMEOUT_MS = 10_000;
const SERVER_POLL_INTERVAL_MS = 500;
const require = createRequire(import.meta.url);

async function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Check if the server is running by hitting the health endpoint
 */
async function isServerRunning(): Promise<boolean> {
  try {
    const response = await fetch(`${SERVER_URL}/health`);
    return response.ok;
  } catch {
    return false;
  }
}

/**
 * Wait for the server to report healthy within a timeout
 */
async function waitForServer(): Promise<void> {
  const startedAt = Date.now();
  while (Date.now() - startedAt < SERVER_START_TIMEOUT_MS) {
    if (await isServerRunning()) {
      return;
    }
    await delay(SERVER_POLL_INTERVAL_MS);
  }

  throw new Error(`Agate server did not become ready within ${SERVER_START_TIMEOUT_MS / 1000}s`);
}

/**
 * Start the Agate server in the background if it is not already running
 */
async function startServer(): Promise<void> {
  require.resolve('@agate/server');
  try {
    const subprocess = execa('agate-server', [], {
      detached: true,
      stdio: 'ignore',
      env: {
        ...process.env,
        PORT: String(PORT),
      },
    });

    subprocess.unref();
  } catch (error) {
    throw new Error(`Failed to start Agate server: ${(error as Error).message}`);
  }

  await waitForServer();
}

/**
 * Ensure the Agate server is running, prompting the user to start it if necessary
 * Returns the server URL
 */
export async function ensureServer(): Promise<string> {
  if (await isServerRunning()) {
    return SERVER_URL;
  }

  console.log(chalk.gray('Agate server not detected, starting...'));

  try {
    await startServer();
    console.log(chalk.gray('Agate server started.'));
  } catch (error) {
    console.error(chalk.red((error as Error).message));
    process.exit(1);
  }

  return SERVER_URL;
}

/**
 * Get the server URL (assumes server is already running)
 */
export function getServerUrl(): string {
  return SERVER_URL;
}
