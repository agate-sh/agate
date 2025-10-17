import { WebSocketServer, WebSocket } from 'ws';
import { Server } from 'http';
import { EventBus } from './event-bus.js';
import { randomUUID } from 'crypto';
import type { TmuxSessionManager } from './tmux/session.js';
import type { ServerEvent } from '@agate/shared';

interface WebSocketMessage {
  type: string;
  sessionId?: string;
  data?: string;
}

interface ClientState {
  id: string;
  currentSessionId: string | null;
  unsubscribeCallback: (() => void) | null;
}

/**
 * Setup WebSocket server for real-time PTY I/O streaming
 * Reference: Phase 2.10 implementation
 */
export function setupWebSocket(
  httpServer: Server,
  eventBus: EventBus,
  sessions: Map<string, TmuxSessionManager>
): WebSocketServer {
  const wss = new WebSocketServer({
    server: httpServer,
    path: '/ws'
  });

  wss.on('connection', (ws: WebSocket) => {
    const state: ClientState = {
      id: randomUUID(),
      currentSessionId: null,
      unsubscribeCallback: null
    };

    ws.on('message', (data: Buffer) => {
      try {
        const msg = JSON.parse(data.toString()) as WebSocketMessage;

        switch (msg.type) {
          case 'pty:input':
            // Forward input to session
            if (msg.sessionId && msg.data !== undefined) {
              const sessionManager = sessions.get(msg.sessionId);
              if (sessionManager) {
                sessionManager.write(msg.data);
              }
            }
            break;

          case 'subscribe':
            // Subscribe to PTY output for a session
            if (msg.sessionId) {
              // Cleanup previous subscription if exists
              if (state.unsubscribeCallback) {
                state.unsubscribeCallback();
                state.unsubscribeCallback = null;
              }

              state.currentSessionId = msg.sessionId;

              // Simple callback - no more SSE buffer needed!
              const eventCallback = (event: ServerEvent) => {
                // Filter for PTY output events matching this session
                if (event.type === 'pty.output' &&
                    'sessionId' in event &&
                    event.sessionId === state.currentSessionId &&
                    ws.readyState === WebSocket.OPEN) {
                  ws.send(JSON.stringify({
                    type: 'pty:output',
                    sessionId: event.sessionId,
                    data: event.data
                  }));
                }

                // Handle git status events for this worktree
                if (event.type === 'git.status' &&
                    'worktreePath' in event &&
                    ws.readyState === WebSocket.OPEN) {
                  ws.send(JSON.stringify(event));
                }

                // Handle session lifecycle events
                if ((event.type === 'session.created' || event.type === 'session.deleted') &&
                    ws.readyState === WebSocket.OPEN) {
                  ws.send(JSON.stringify(event));
                }
              };

              // Subscribe to event bus
              eventBus.subscribe(state.id, eventCallback);

              // Store unsubscribe callback
              state.unsubscribeCallback = () => {
                eventBus.unsubscribe(state.id);
              };
            }
            break;

          case 'unsubscribe':
            // Unsubscribe from session
            if (state.unsubscribeCallback) {
              state.unsubscribeCallback();
              state.unsubscribeCallback = null;
            }
            state.currentSessionId = null;
            break;

          default:
            // Ignore unknown message types (gracefully)
            console.warn(`Unknown WebSocket message type: ${msg.type}`);
            break;
        }
      } catch (error) {
        console.error(`WebSocket message parsing error for client ${state.id}:`, error);
      }
    });

    ws.on('close', () => {
      // Cleanup event listeners on disconnect
      if (state.unsubscribeCallback) {
        state.unsubscribeCallback();
        state.unsubscribeCallback = null;
      }
    });

    ws.on('error', (error) => {
      console.error(`WebSocket error for client ${state.id}:`, error);
    });
  });

  return wss;
}
