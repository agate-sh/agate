/**
 * Event types for Server-Sent Events
 */

export type EventType =
  | 'pty.output'
  | 'pty.resize'
  | 'pty.exit'
  | 'git.status'
  | 'git.branch'
  | 'session.created'
  | 'session.updated'
  | 'session.deleted'
  | 'keepalive';

export interface BaseEvent {
  type: EventType;
  timestamp: string;
}

export interface PtyOutputEvent extends BaseEvent {
  type: 'pty.output';
  sessionId: string;
  data: string;
}

export interface PtyResizeEvent extends BaseEvent {
  type: 'pty.resize';
  sessionId: string;
  cols: number;
  rows: number;
}

export interface PtyExitEvent extends BaseEvent {
  type: 'pty.exit';
  sessionId: string;
  exitCode: number;
}

export interface GitStatusEvent extends BaseEvent {
  type: 'git.status';
  sessionId: string;
  status: {
    branch: string;
    ahead: number;
    behind: number;
    staged: number;
    modified: number;
    untracked: number;
  };
}

export interface GitBranchEvent extends BaseEvent {
  type: 'git.branch';
  sessionId: string;
  branch: string;
}

export interface SessionCreatedEvent extends BaseEvent {
  type: 'session.created';
  sessionId: string;
}

export interface SessionUpdatedEvent extends BaseEvent {
  type: 'session.updated';
  sessionId: string;
}

export interface SessionDeletedEvent extends BaseEvent {
  type: 'session.deleted';
  sessionId: string;
}

export interface KeepaliveEvent extends BaseEvent {
  type: 'keepalive';
}

export type ServerEvent =
  | PtyOutputEvent
  | PtyResizeEvent
  | PtyExitEvent
  | GitStatusEvent
  | GitBranchEvent
  | SessionCreatedEvent
  | SessionUpdatedEvent
  | SessionDeletedEvent
  | KeepaliveEvent;
