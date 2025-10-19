import { useKeyboard } from "@opentui/react";
import { theme } from "../utils/theme.js";

interface DialogProps {
  title?: string;
  onClose: () => void;
  children: React.ReactNode;
}

/**
 * Dialog overlay component
 * Centers a modal dialog with opaque background and border using absolute positioning
 * This solves the layout issue where dialog was appearing as a sibling instead of overlay
 */
export function Dialog({ title, onClose, children }: DialogProps) {
  // Close on Escape
  useKeyboard((key) => {
    if (key.name === "escape") {
      onClose();
    }
  });

  return (
    <box
      style={{
        position: "absolute",
        left: 0,
        right: 0,
        top: 0,
        bottom: 0,
        margin: "auto",
        width: "60%",
        height: "auto",
        maxHeight: "80%",
        flexDirection: "column",
        borderStyle: "rounded",
        borderColor: theme.borderActive,
        backgroundColor: theme.mantle,
        paddingLeft: 2,
        paddingRight: 2,
        paddingTop: 1,
        paddingBottom: 1,
      }}
    >
      {title && <box marginBottom={1}><text>{title}</text></box>}
      {children}
    </box>
  );
}
