/**
 * Tmux pane information
 */
export interface TmuxPane {
  /** Pane ID */
  id: string;
  /** Pane index within window */
  index: number;
  /** Current working directory */
  cwd: string;
  /** Width in characters */
  width: number;
  /** Height in characters */
  height: number;
  /** Whether this pane is active */
  isActive: boolean;
  /** Process ID running in pane */
  pid: number;
  /** Current command/process name */
  command: string;
}

/**
 * Tmux window information
 */
export interface TmuxWindow {
  /** Window ID */
  id: string;
  /** Window index */
  index: number;
  /** Window name */
  name: string;
  /** Panes in this window */
  panes: TmuxPane[];
  /** Whether this window is active */
  isActive: boolean;
}

/**
 * Tmux session information
 */
export interface TmuxSession {
  /** Session ID */
  id: string;
  /** Session name */
  name: string;
  /** Windows in this session */
  windows: TmuxWindow[];
  /** Whether this session is attached */
  isAttached: boolean;
  /** Creation timestamp */
  createdAt: string;
}
