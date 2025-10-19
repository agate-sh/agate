import { useEffect, useRef, useCallback } from 'react';
import WebSocket from 'ws';
import type { PersistedSession } from '@agate/shared';

interface WebSocketMessage {
  type: string;
  data?: any;
}

interface UseWebSocketOptions {
  url: string;
  onSessionCreated?: (session: PersistedSession) => void;
  onSessionDeleted?: (sessionId: string) => void;
  onSessionUpdated?: (session: PersistedSession) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
}

/**
 * WebSocket connection hook for real-time session updates
 * Manages connection lifecycle and event handling
 */
export function useWebSocket({
  url,
  onSessionCreated,
  onSessionDeleted,
  onSessionUpdated,
  onConnect,
  onDisconnect,
  onError,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    console.log('[WebSocket] Connecting to:', url);
    const ws = new WebSocket(url);

    ws.onopen = () => {
      console.log('[WebSocket] Connected');
      onConnect?.();
    };

    ws.onmessage = (event) => {
      try {
        const data = typeof event.data === 'string' ? event.data : event.data.toString();
        const message: WebSocketMessage = JSON.parse(data);
        console.log('[WebSocket] Message:', message);

        switch (message.type) {
          case 'session.created':
            if (message.data && onSessionCreated) {
              onSessionCreated(message.data as PersistedSession);
            }
            break;

          case 'session.deleted':
            if (message.data?.sessionId && onSessionDeleted) {
              onSessionDeleted(message.data.sessionId);
            }
            break;

          case 'session.updated':
            if (message.data && onSessionUpdated) {
              onSessionUpdated(message.data as PersistedSession);
            }
            break;

          default:
            console.log('[WebSocket] Unknown message type:', message.type);
        }
      } catch (error) {
        console.error('[WebSocket] Error parsing message:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('[WebSocket] Error:', error);
      onError?.(new Error(error.message || 'WebSocket error'));
    };

    ws.onclose = () => {
      console.log('[WebSocket] Disconnected');
      onDisconnect?.();

      // Reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('[WebSocket] Reconnecting...');
        connect();
      }, 3000);
    };

    wsRef.current = ws;
  }, [url, onSessionCreated, onSessionDeleted, onSessionUpdated, onConnect, onDisconnect, onError]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  const send = useCallback((message: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('[WebSocket] Cannot send message, not connected');
    }
  }, []);

  return {
    send,
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
  };
}
