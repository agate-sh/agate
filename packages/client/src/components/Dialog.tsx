import { useKeyboard, useTerminalDimensions } from "@opentui/react";
import { theme } from "../utils/theme.js";

export const DIALOG_WIDTH = 60;
export const DIALOG_PADDING_X = 4;
export const DIALOG_PADDING_Y = 3;

interface DialogProps {
  title?: string;
  onClose: () => void;
  children: React.ReactNode;
}

/**
 * Dialog overlay component
 * Uses a shared overlay container to dim the background slightly and
 * centers the modal content.
 */
export function Dialog({ title, onClose, children }: DialogProps) {
  const { width, height } = useTerminalDimensions();

  useKeyboard((key) => {
    if (key.name === "escape") {
      onClose();
    }
  });

  const dialogWidth = Math.min(DIALOG_WIDTH, Math.max(20, width - 2));

  return (
    <box
      width={width}
      height={height}
      position="absolute"
      left={0}
      top={0}
      backgroundColor={theme.overlayScrim}
      alignItems="center"
      paddingTop={Math.floor(height / 4)}
    >
      <box
        flexDirection="column"
        backgroundColor={theme.mantle}
        paddingLeft={DIALOG_PADDING_X}
        paddingRight={DIALOG_PADDING_X}
        paddingTop={DIALOG_PADDING_Y}
        paddingBottom={DIALOG_PADDING_Y}
        width={dialogWidth}
        maxWidth={Math.max(20, width - 2)}
      >
        {title && (
          <box marginBottom={1}>
            <text>{title}</text>
          </box>
        )}
        {children}
      </box>
    </box>
  );
}
