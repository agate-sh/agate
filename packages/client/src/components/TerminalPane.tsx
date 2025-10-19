import { useEffect } from 'react';
import { useKeyboard } from '@opentui/react';
import { usePtyStream } from '../hooks/usePtyStream.js';
import { theme } from '../utils/theme.js';
import { logger } from '../logger.js';
import { StyledText } from '@opentui/core';

interface TerminalPaneProps {
  sessionId: string;
  isFocused: boolean;
}

/**
 * TerminalPane component - Displays PTY output from a tmux session
 * Handles keyboard input and sends it to the PTY via WebSocket
 */
export function TerminalPane({ sessionId, isFocused }: TerminalPaneProps) {
  const { output, parsedLines, isConnected, sendInput } = usePtyStream({ sessionId });

  // Debug: Log output changes
  useEffect(() => {
    logger.debug({ outputLength: output.length }, '🖥️  Output length');
    logger.debug({ preview: output.substring(0, 200) }, '🖥️  Output preview');
    logger.debug({ parsedLinesCount: parsedLines.length }, '🎨 Parsed lines count');
  }, [output, parsedLines]);

  // Handle keyboard input using OpenTUI's useKeyboard hook
  useKeyboard((key) => {
    // Only handle keyboard input when this pane is focused
    if (!isFocused) return;
    if (!isConnected) return;

    logger.debug({ key: key.name, sequence: key.sequence }, 'Keypress received');

    // Tab is reserved for focus management - don't send it to terminal
    if (key.name === 'tab') {
      return;
    }

    // Send all other keys directly via their raw sequence
    if (key.sequence) {
      sendInput(key.sequence);
    }
  });

  return (
    <box
      flexGrow={1}
      flexDirection="column"
      border
      borderStyle="single"
      borderColor={isFocused ? theme.agate : theme.borderDefault}
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
