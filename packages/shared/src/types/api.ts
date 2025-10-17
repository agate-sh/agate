import type { TmuxSession } from './tmux.js';
import type { GitStatus } from './git.js';
import type { AgentName } from './agent.js';

/**
 * API request/response types
 */

// Session Management (new REST API)
export interface CreateSessionRequest {
  worktreePath: string;
  branch: string;
  agentName: AgentName;
}

export interface CreateSessionResponse {
  sessionId: string;
}

export interface GetSessionResponse {
  id: string;
  name: string;
  agent: AgentName;
  cwd: string;
  cols: number;
  rows: number;
  pid: number;
  isAlive: boolean;
}

export interface SendSessionInputRequest {
  data: string;
}

export interface SendSessionInputResponse {
  success: boolean;
}

export interface ResizeSessionRequest {
  cols: number;
  rows: number;
}

export interface ResizeSessionResponse {
  success: boolean;
}

export interface DeleteSessionResponse {
  success: boolean;
}

// Legacy Session Management (to be deprecated)
export interface AttachSessionRequest {
  sessionId: string;
}

export interface AttachSessionResponse {
  tmuxSession: TmuxSession;
}

// Tmux Operations
export interface SendInputRequest {
  paneId: string;
  input: string;
}

export interface SendInputResponse {
  success: boolean;
}

export interface ResizePaneRequest {
  paneId: string;
  width: number;
  height: number;
}

export interface ResizePaneResponse {
  success: boolean;
}

// Git Operations
export interface GetGitStatusRequest {
  path: string;
}

export interface GetGitStatusResponse {
  status: GitStatus;
}
