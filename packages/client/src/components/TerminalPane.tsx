import { useEffect, useRef } from 'react';
import { usePtyStream } from '../hooks/usePtyStream.js';
import { theme } from '../utils/theme.js';
import { logger } from '../logger.js';
import { StyledText } from '@opentui/core';

interface TerminalPaneProps {
  sessionId: string;
}

/**
 * TerminalPane component - Displays PTY output from a tmux session
 * Handles keyboard input and sends it to the PTY via WebSocket
 */
export function TerminalPane({ sessionId }: TerminalPaneProps) {
  const { output, parsedLines, isConnected, sendInput } = usePtyStream({ sessionId });
  const boxRef = useRef<any>(null);

  // Debug: Log output changes
  useEffect(() => {
    logger.debug({ outputLength: output.length }, '🖥️  Output length');
    logger.debug({ preview: output.substring(0, 200) }, '🖥️  Output preview');
    logger.debug({ parsedLinesCount: parsedLines.length }, '🎨 Parsed lines count');
  }, [output, parsedLines]);

  // Handle keyboard input
  useEffect(() => {
    const box = boxRef.current;
    if (!box) return;

    // Focus the box to receive keyboard events
    box.focus();

    const handleKeypress = (ch: string, key: any) => {
      if (!isConnected) return;

      logger.debug({ ch, key: key?.name }, 'Keypress received');

      // Handle special keys
      if (key.name === 'return') {
        sendInput('\n');
      } else if (key.name === 'backspace') {
        sendInput('\x7f');
      } else if (key.name === 'tab') {
        sendInput('\t');
      } else if (key.name === 'escape') {
        sendInput('\x1b');
      } else if (key.ctrl && key.name === 'c') {
        sendInput('\x03');
      } else if (key.ctrl && key.name === 'd') {
        sendInput('\x04');
      } else if (ch) {
        sendInput(ch);
      }
    };

    // Listen for keypress events on the box
    box.on('keypress', handleKeypress);

    return () => {
      box.off('keypress', handleKeypress);
    };
  }, [isConnected, sendInput]);

  return (
    <box
      ref={boxRef}
      flexGrow={1}
      flexDirection="column"
      border
      borderStyle="single"
      borderColor={isConnected ? theme.success : theme.error}
      padding={1}
    >
      {/* Header */}
      <box height={1} width="100%" flexShrink={0}>
        <text fg={theme.agate}>Terminal </text>
        <text fg={theme.textMuted}>{sessionId.substring(0, 20)}...</text>
        <box flexGrow={1} />
        <text fg={isConnected ? theme.success : theme.error}>
          {isConnected ? '● Connected' : '○ Disconnected'}
        </text>
      </box>

      {/* Output area with ANSI rendering */}
      <box flexGrow={1} flexDirection="column" overflow="hidden" bg={theme.base}>
        {parsedLines.length === 0 ? (
          <text fg={theme.textMuted}>Waiting for output...</text>
        ) : (
          parsedLines.map((line, i) => (
            <box key={i} height={1}>
              <text content={new StyledText(line.chunks)} />
            </box>
          ))
        )}
      </box>
    </box>
  );
}
