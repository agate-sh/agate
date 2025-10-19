import { useEffect } from "react";
import { TextAttributes } from "@opentui/core";
import { useRenderer } from "@opentui/react";
import { theme } from "./utils/theme.js";
import { AgentsPane } from "./components/AgentsPane/index.js";
import { useDialog } from "./hooks/useDialog.js";
import { TerminalPane } from "./components/TerminalPane.js";

export function App() {
  const dialog = useDialog();
  const renderer = useRenderer();

  // Set background color on mount
  useEffect(() => {
    if (renderer) {
      renderer.setBackgroundColor(theme.base);
    }
  }, [renderer]);

  return (
    <>
      <box flexDirection="column" width="100%" height="100%">
        {/* Main content area */}
        <box flexGrow={1} width="100%" flexDirection="row">
          {/* Agents sidebar */}
          <AgentsPane dialog={dialog} />

          {/* Terminal pane - hardcoded session for testing */}
          <TerminalPane sessionId="0072bbef-5011-4859-9f48-56049d449bdc" />
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
