import { useEffect, useState } from "react";
import { TextAttributes } from "@opentui/core";
import { useRenderer, useKeyboard } from "@opentui/react";
import { theme } from "./utils/theme.js";
import { AgentsPane } from "./components/AgentsPane/index.js";
import { useDialog } from "./hooks/useDialog.js";
import { TerminalPane } from "./components/TerminalPane.js";
import * as api from "./api.js";
import type { SessionGetResponse } from "@agate/sdk";

export function App() {
  const dialog = useDialog();
  const renderer = useRenderer();
  const [focusedPane, setFocusedPane] = useState<'agents' | 'terminal'>('agents');
  const [session, setSession] = useState<SessionGetResponse | null>(null);

  // Set background color on mount
  useEffect(() => {
    if (renderer) {
      renderer.setBackgroundColor(theme.base);
    }
  }, [renderer]);

  // Fetch session data on mount
  useEffect(() => {
    const fetchSession = async () => {
      const response = await api.sessionGet({
        path: { id: "7cc598c0-7374-47f3-b0ad-130664e4bc03" },
      });
      if (response.data) {
        setSession(response.data);
      }
    };
    fetchSession();
  }, []);

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

          {/* Terminal pane */}
          {session && (
            <TerminalPane
              sessionId={session.id}
              isFocused={focusedPane === 'terminal'}
              branch={session.name.split('_')[1] || 'unknown'}
            />
          )}
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
