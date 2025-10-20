import { useEffect, useRef, useState, useCallback } from 'react';
import WebSocket from 'ws';
import { AGATE_SERVER_PORT } from '@agate/shared';
import { logger } from '../logger.js';
import { AnsiParser, type AnsiLine } from '../utils/ansiParser.js';

interface UsePtyStreamOptions {
  sessionId: string;
  cols?: number;
  rows?: number;
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
  cols = 120,
  rows = 50,
  url = `ws://localhost:${AGATE_SERVER_PORT}/ws`,
  onConnect,
  onDisconnect,
  onError,
}: UsePtyStreamOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const parserRef = useRef<AnsiParser | null>(null);
  const [output, setOutput] = useState<string>('');
  const [parsedLines, setParsedLines] = useState<AnsiLine[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  // Resize the parser when dimensions change
  useEffect(() => {
    if (parserRef.current) {
      parserRef.current.resize(cols, rows);
      logger.debug({ cols, rows }, '📐 Resizing terminal parser');
    }
  }, [cols, rows]);

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
      parserRef.current = new AnsiParser(cols, rows);
    }

    ws.onopen = () => {
      logger.info('✓ WebSocket CONNECTED');
      logger.info({ sessionId }, '📡 Subscribing to session');
      setIsConnected(true);
      onConnect?.();

      // Clear the parser buffer to prevent duplicate initial content
      if (parserRef.current) {
        parserRef.current.clear();
        logger.debug('🧹 Cleared terminal buffer before subscribing');
      }

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

          // Log raw ANSI codes (escaped for visibility)
          const escapedData = message.data
            .replace(/\x1b/g, '\\x1b')
            .replace(/\n/g, '\\n')
            .replace(/\r/g, '\\r');
          logger.info({ raw: escapedData.substring(0, 300) }, '🔍 RAW ANSI');

          // Also log hex dump of first 100 bytes
          const hexDump = Array.from(message.data.substring(0, 100))
            .map(char => char.charCodeAt(0).toString(16).padStart(2, '0'))
            .join(' ');
          logger.info({ hex: hexDump }, '🔢 HEX DUMP');

          // Append new output to existing output (for raw debugging)
          setOutput((prev) => {
            const newOutput = prev + message.data;
            logger.debug({ totalLength: newOutput.length }, '📝 Total output length');
            return newOutput;
          });

          // Parse ANSI codes using headless terminal
          if (parserRef.current) {
            parserRef.current.write(message.data);
            const lines = parserRef.current.getVisibleLines(); // Get viewport lines
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
