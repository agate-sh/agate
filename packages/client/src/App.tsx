import React, { useEffect, useState } from 'react';
import { useTerminalDimensions, useKeyboard, useRenderer } from '@opentui/react';
import { TextAttributes } from '@opentui/core';

export function App() {
  const dimensions = useTerminalDimensions();
  const renderer = useRenderer();
  const [counter, setCounter] = useState(0);

  // Handle keyboard input
  useKeyboard((evt) => {
    // Cmd/Ctrl + T for debug overlay
    if (evt.meta && evt.name === 't') {
      renderer.toggleDebugOverlay();
      return;
    }

    // Cmd/Ctrl + D for console
    if (evt.meta && evt.name === 'd') {
      renderer.console.toggle();
      return;
    }

    // Space to increment counter (for testing)
    if (evt.name === 'space') {
      setCounter((c) => c + 1);
      return;
    }
  });

  // Log dimensions changes for debugging
  useEffect(() => {
    console.log('Terminal dimensions:', dimensions);
  }, [dimensions]);

  return (
    <box
      width={dimensions.width}
      height={dimensions.height}
      backgroundColor="#1e1e2e"
      flexDirection="column"
    >
      {/* Main content area */}
      <box flexGrow={1} flexDirection="column" justifyContent="center" alignItems="center">
        <box marginBottom={2}>
          <text fg="#cdd6f4" attributes={TextAttributes.BOLD}>
            🚀 Agate OpenTUI Client
          </text>
        </box>
        <box marginBottom={1}>
          <text fg="#89b4fa">Terminal: {dimensions.width}x{dimensions.height}</text>
        </box>
        <box marginBottom={1}>
          <text fg="#a6e3a1">Counter: {counter}</text>
        </box>
        <box>
          <text fg="#585b70">Press Space to increment • Cmd+T for debug • Cmd+D for console</text>
        </box>
      </box>

      {/* Status bar */}
      <box
        height={1}
        backgroundColor="#181825"
        flexDirection="row"
        justifyContent="space-between"
        flexShrink={0}
      >
        <box flexDirection="row">
          <box backgroundColor="#313244" paddingLeft={1} paddingRight={1}>
            <text fg="#89b4fa" attributes={TextAttributes.BOLD}>agate</text>
          </box>
          <box paddingLeft={1} paddingRight={1}>
            <text fg="#585b70">v0.1.0</text>
          </box>
        </box>
        <box paddingRight={1}>
          <text fg="#f38ba8">● READY</text>
        </box>
      </box>
    </box>
  );
}
