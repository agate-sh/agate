import type { TmuxSession } from './tmux.js';
import type { GitStatus } from './git.js';
import type { PersistedSession } from './session.js';

/**
 * API request/response types
 */

// Session Management
export interface CreateSessionRequest {
  worktreePath: string;
  branch: string;
  agentName: string;
}

export interface CreateSessionResponse {
  session: PersistedSession;
  tmuxSession: TmuxSession;
}

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

// Output Streaming (SSE)
export interface PaneOutputEvent {
  paneId: string;
  data: string;
  timestamp: string;
}

export interface SessionStateChangeEvent {
  sessionId: string;
  state: 'created' | 'attached' | 'detached' | 'destroyed';
  timestamp: string;
}

export type ServerEvent =
  | { type: 'pane_output'; payload: PaneOutputEvent }
  | { type: 'session_state'; payload: SessionStateChangeEvent };
