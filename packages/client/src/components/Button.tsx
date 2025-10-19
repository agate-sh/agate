import { theme } from "../utils/theme.js";

interface ButtonProps {
  label: string;
  shortcut: string;
  disabled?: boolean;
  agentColor?: string;
  variant?: "agent" | "danger";
  fullWidth?: boolean;
}

/**
 * Styled button component
 * Matches the Go implementation's Button component
 */
export function Button({
  label,
  shortcut,
  disabled = false,
  agentColor,
  variant = "agent",
  fullWidth = false,
}: ButtonProps) {
  const text = `${label} (${shortcut})`;

  // Determine background color using theme
  let bgColor: string = theme.borderMuted; // default muted color
  if (!disabled) {
    if (variant === "agent" && agentColor) {
      bgColor = agentColor;
    } else if (variant === "danger") {
      bgColor = theme.error;
    }
  } else {
    bgColor = theme.borderMuted; // gray when disabled
  }

  const textColor: string = disabled ? theme.textMuted : theme.textPrimary;

  return (
    <box width={fullWidth ? "100%" : undefined} justifyContent="center">
      <box paddingLeft={2} paddingRight={2}>
        <text fg={textColor} bg={bgColor}>
          <strong>{text}</strong>
        </text>
      </box>
    </box>
  );
}
