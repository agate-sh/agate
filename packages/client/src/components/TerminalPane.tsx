import { useEffect } from 'react';
import { useKeyboard, useTerminalDimensions } from '@opentui/react';
import { usePtyStream } from '../hooks/usePtyStream.js';
import * as api from '../api.js';
import { theme } from '../utils/theme.js';
import { logger } from '../logger.js';
import { StyledText } from '@opentui/core';

interface TerminalPaneProps {
  sessionId: string;
  isFocused: boolean;
  branch: string;
}

/**
 * TerminalPane component - Displays PTY output from a tmux session
 * Handles keyboard input and sends it to the PTY via WebSocket
 */
export function TerminalPane({ sessionId, isFocused, branch }: TerminalPaneProps) {
  const { width, height } = useTerminalDimensions();

  // Calculate terminal dimensions
  // Terminal pane is ~70% of screen width, minus borders and padding
  const cols = Math.floor(width * 0.7) - 4;
  const rows = height - 4; // Account for header, status bar, borders

  const { parsedLines, isConnected, sendInput } = usePtyStream({
    sessionId,
    cols,
    rows,
  });

  // Resize the PTY when terminal dimensions change
  useEffect(() => {
    if (!isConnected) return;

    const resize = async () => {
      logger.debug({ cols, rows, sessionId }, '📐 Resizing PTY session');
      const response = await api.sessionResize({
        path: { id: sessionId },
        body: { cols, rows },
      });
      if (response.error) {
        logger.error({ err: response.error }, 'Failed to resize PTY');
      }
    };
    resize();
  }, [cols, rows, sessionId, isConnected]);

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
      title={branch}
      padding={1}
    >
      {/* Output area with ANSI rendering */}
      <box flexGrow={1} flexDirection="column" overflow="hidden" backgroundColor={theme.base}>
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
