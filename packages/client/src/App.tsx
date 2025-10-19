import { useEffect, useState } from "react";
import { TextAttributes } from "@opentui/core";
import { useRenderer, useKeyboard } from "@opentui/react";
import { theme } from "./utils/theme.js";
import { AgentsPane } from "./components/AgentsPane/index.js";
import { useDialog } from "./hooks/useDialog.js";
import { TerminalPane } from "./components/TerminalPane.js";

export function App() {
  const dialog = useDialog();
  const renderer = useRenderer();
  const [focusedPane, setFocusedPane] = useState<'agents' | 'terminal'>('agents');

  // Set background color on mount
  useEffect(() => {
    if (renderer) {
      renderer.setBackgroundColor(theme.base);
    }
  }, [renderer]);

  // Handle Tab for pane focus cycling
  useKeyboard((key) => {
    if (key.name === 'tab') {
      setFocusedPane(prev => prev === 'agents' ? 'terminal' : 'agents');
    }
  });

  return (
    <>
      <box flexDirection="column" width="100%" height="100%">
        {/* Main content area */}
        <box flexGrow={1} width="100%" flexDirection="row">
          {/* Agents sidebar */}
          <AgentsPane dialog={dialog} isFocused={focusedPane === 'agents'} />

          {/* Terminal pane - hardcoded session for testing */}
          <TerminalPane sessionId="26f1eea4-3351-4ff0-8936-a63aaa9ac8f1" isFocused={focusedPane === 'terminal'} />
        </box>

        {/* Status bar */}
        <box
          height={1}
          width="100%"
          flexDirection="row"
          alignItems="center"
          flexGrow={0}
          flexShrink={0}
        >
          <text style={{ fg: theme.agate }}> agate </text>
          <text attributes={TextAttributes.DIM}>v0.1.0 </text>
          <box flexGrow={1} />
          <text fg={theme.status.ready}>● READY </text>
        </box>
      </box>

      {/* Dialog overlay - rendered outside main layout */}
      {dialog.isOpen && dialog.current}
    </>
  );
}
