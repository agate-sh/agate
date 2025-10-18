import { randomUUID } from 'crypto';
import { EventBus } from './event-bus.js';
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
 * Create WebSocket handler using @hono/node-ws
 * This replaces the standalone WebSocket server with Hono-integrated WebSocket handling
 */
export function createWebSocketHandler(
  eventBus: EventBus,
  sessions: Map<string, TmuxSessionManager>
) {
  return () => {
    const state: ClientState = {
      id: randomUUID(),
      currentSessionId: null,
      unsubscribeCallback: null,
    };

    return {
        onMessage(event: MessageEvent, ws: any) {
          try {
            const msg = JSON.parse(event.data.toString()) as WebSocketMessage;

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

                  // Event callback
                  const eventCallback = (event: ServerEvent) => {
                    // Filter for PTY output events matching this session
                    if (
                      event.type === 'pty.output' &&
                      'sessionId' in event &&
                      event.sessionId === state.currentSessionId
                    ) {
                      ws.send(
                        JSON.stringify({
                          type: 'pty:output',
                          sessionId: event.sessionId,
                          data: event.data,
                        })
                      );
                    }

                    // Handle git status events for this worktree
                    if (event.type === 'git.status' && 'worktreePath' in event) {
                      ws.send(JSON.stringify(event));
                    }

                    // Handle session lifecycle events
                    if (event.type === 'session.created' || event.type === 'session.deleted') {
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
        },

        onClose() {
          // Cleanup event listeners on disconnect
          if (state.unsubscribeCallback) {
            state.unsubscribeCallback();
            state.unsubscribeCallback = null;
          }
        },

        onError(event: any) {
          console.error(`WebSocket error for client ${state.id}:`, event);
        },
    };
  };
}
