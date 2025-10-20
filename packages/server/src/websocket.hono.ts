import { randomUUID } from 'crypto';
import { EventBus } from './event-bus.js';
import type { TmuxSessionManager } from './tmux/session.js';
import type { ServerEvent } from '@agate/shared';
import { logger } from './logger.js';

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

interface SessionMetadata {
  manager: TmuxSessionManager;
  worktreeKey: string;
  worktreePath: string;
  branch: string;
  repoName: string;
  agentName: string;
}

/**
 * Create WebSocket handler using @hono/node-ws
 * This replaces the standalone WebSocket server with Hono-integrated WebSocket handling
 */
export function createWebSocketHandler(
  eventBus: EventBus,
  sessions: Map<string, SessionMetadata>
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
                  const sessionMetadata = sessions.get(msg.sessionId);
                  if (sessionMetadata) {
                    sessionMetadata.manager.write(msg.data);
                  }
                }
                break;

              case 'subscribe':
                // Subscribe to PTY output for a session
                if (msg.sessionId) {
                  logger.debug({ clientId: state.id, sessionId: msg.sessionId }, 'Client subscribing to session');

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

                  // Send initial content from tmux buffer FIRST
                  const sessionMetadata = sessions.get(msg.sessionId);
                  logger.debug({ sessionId: msg.sessionId, found: !!sessionMetadata }, 'Looking up session manager');

                  if (sessionMetadata) {
                    logger.debug({ sessionId: msg.sessionId }, 'Capturing initial tmux buffer content');
                    sessionMetadata.manager.captureCurrentContent().then((content) => {
                      logger.debug({ sessionId: msg.sessionId, bytes: content.length }, 'Captured initial content');
                      ws.send(
                        JSON.stringify({
                          type: 'pty:output',
                          sessionId: msg.sessionId,
                          data: content,
                        })
                      );
                      logger.debug({ sessionId: msg.sessionId }, 'Sent initial content to client');

                      // Subscribe to event bus AFTER sending initial content
                      // This prevents race condition where new events arrive before initial state
                      eventBus.subscribe(state.id, eventCallback);
                      logger.debug({ clientId: state.id, sessionId: msg.sessionId }, 'Subscribed to event bus after sending initial content');

                      // Store unsubscribe callback
                      state.unsubscribeCallback = () => {
                        eventBus.unsubscribe(state.id);
                      };
                    }).catch((error) => {
                      logger.error({ sessionId: msg.sessionId, err: error }, 'Failed to capture initial content');
                    });
                  } else {
                    logger.warn({
                      sessionId: msg.sessionId,
                      availableSessions: Array.from(sessions.keys())
                    }, 'Session not found in sessions map');
                  }
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
                logger.warn({ type: msg.type }, 'Unknown WebSocket message type');
                break;
            }
          } catch (error) {
            logger.error({ clientId: state.id, err: error }, 'WebSocket message parsing error');
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
          logger.error({ clientId: state.id, event }, 'WebSocket error');
        },
    };
  };
}
