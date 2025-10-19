import { useEffect, useRef, useState, useCallback } from 'react';
import WebSocket from 'ws';
import { logger } from '../logger.js';
import { AnsiParser, type AnsiLine } from '../utils/ansiParser.js';

interface UsePtyStreamOptions {
  sessionId: string;
  url?: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
}

/**
 * Hook for managing PTY I/O streaming via WebSocket
 * Subscribes to a session's output and provides input sending capability
 */
export function usePtyStream({
  sessionId,
  url = 'ws://localhost:3000/ws',
  onConnect,
  onDisconnect,
  onError,
}: UsePtyStreamOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const parserRef = useRef<AnsiParser | null>(null);
  const [output, setOutput] = useState<string>('');
  const [parsedLines, setParsedLines] = useState<AnsiLine[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  const sendInput = useCallback(
    (data: string) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        logger.debug({ data: JSON.stringify(data) }, '⌨️  Sending input');
        wsRef.current.send(
          JSON.stringify({
            type: 'pty:input',
            sessionId,
            data,
          })
        );
      } else {
        logger.warn('⚠️  Cannot send input, not connected');
      }
    },
    [sessionId]
  );

  useEffect(() => {
    logger.info({ url }, 'Connecting to WebSocket');
    const ws = new WebSocket(url);

    // Initialize ANSI parser
    if (!parserRef.current) {
      parserRef.current = new AnsiParser(120, 50);
    }

    ws.onopen = () => {
      logger.info('✓ WebSocket CONNECTED');
      logger.info({ sessionId }, '📡 Subscribing to session');
      setIsConnected(true);
      onConnect?.();

      // Subscribe to PTY output for this session
      const subscribeMsg = {
        type: 'subscribe',
        sessionId,
      };
      logger.debug({ subscribeMsg }, '📤 Sending subscribe message');
      ws.send(JSON.stringify(subscribeMsg));
    };

    ws.onmessage = (event) => {
      try {
        const data =
          typeof event.data === 'string' ? event.data : event.data.toString();
        logger.debug({ preview: data.substring(0, 200) }, '📥 Raw message received');

        const message = JSON.parse(data);
        logger.debug({
          type: message.type,
          receivedSessionId: message.sessionId,
          expectedSessionId: sessionId,
          dataLength: message.data?.length
        }, '📦 Parsed message');

        if (message.type === 'pty:output' && message.sessionId === sessionId) {
          logger.info({ preview: message.data.substring(0, 100) }, '✅ PTY OUTPUT RECEIVED');

          // Append new output to existing output (for raw debugging)
          setOutput((prev) => {
            const newOutput = prev + message.data;
            logger.debug({ totalLength: newOutput.length }, '📝 Total output length');
            return newOutput;
          });

          // Parse ANSI codes using headless terminal
          if (parserRef.current) {
            parserRef.current.write(message.data);
            const lines = parserRef.current.getVisibleLines(50);
            setParsedLines(lines);
            logger.debug({ linesCount: lines.length }, '🎨 Parsed lines');
          }
        } else {
          logger.warn('⚠️  Message ignored - type or sessionId mismatch');
        }
      } catch (error) {
        logger.error({ err: error }, '❌ Error parsing message');
        logger.error({ rawData: event.data }, '❌ Raw data');
      }
    };

    ws.onerror = (error) => {
      logger.error({ err: error }, 'WebSocket error');
      onError?.(new Error('WebSocket error'));
    };

    ws.onclose = () => {
      logger.info('Disconnected');
      setIsConnected(false);
      onDisconnect?.();
    };

    wsRef.current = ws;

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        // Unsubscribe before closing
        ws.send(
          JSON.stringify({
            type: 'unsubscribe',
          })
        );
      }
      ws.close();
    };
  }, [sessionId, url, onConnect, onDisconnect, onError]);

  return {
    output,
    parsedLines,
    sendInput,
    isConnected,
  };
}
