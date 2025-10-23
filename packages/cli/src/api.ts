import { client, gitRepoInfo, sessionCreate } from '@agate/sdk';
import { type AgentName, generateBranchName } from '@agate/shared';
import { getServerUrl } from './bootstrap.js';

/**
 * Initialize the SDK client with the server URL
 */
export function initClient(): void {
  client.setConfig({
    baseUrl: getServerUrl(),
  });
}

/**
 * Validate that the current directory is inside a git repository
 * Returns repository information
 */
export async function validateGitRepo(dir: string): Promise<{
  repoRoot: string;
  repoName: string;
  currentBranch: string;
}> {
  const { data, error } = await gitRepoInfo({
    query: { dir },
  });

  if (error || !data) {
    throw new Error(`Not in a git repository: ${dir}`);
  }

  return data;
}

/**
 * Create a new session with automatic branch name generation
 */
export async function createAgentSession(
  dir: string,
  agentName: AgentName,
  branchName?: string
): Promise<{
  sessionId: string;
  tmuxName: string;
  worktreePath: string;
  branchName: string;
  repoName: string;
}> {
  const branch = branchName ?? generateBranchName();

  const { data, error } = await sessionCreate({
    body: {
      dir,
      branchName: branch,
      agentName,
    },
  });

  if (error || !data) {
    throw new Error(`Failed to create session: ${error}`);
  }

  return data;
}

/**
 * Send input to a session via WebSocket
 */
export async function sendToSession(sessionId: string, input: string): Promise<void> {
  const ws = await import('ws');
  const WebSocket = ws.default;

  const serverUrl = getServerUrl().replace('http', 'ws');
  const socket = new WebSocket(`${serverUrl}/ws`);

  return new Promise((resolve, reject) => {
    socket.on('open', () => {
      // Subscribe to session
      socket.send(JSON.stringify({
        type: 'subscribe',
        sessionId,
      }));

      // Send input
      socket.send(JSON.stringify({
        type: 'pty:input',
        sessionId,
        data: input + '\n',
      }));

      // Close after a short delay to allow message to be sent
      setTimeout(() => {
        socket.close();
        resolve();
      }, 100);
    });

    socket.on('error', (err) => {
      reject(err);
    });
  });
}
