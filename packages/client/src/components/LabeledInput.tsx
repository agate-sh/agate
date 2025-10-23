import { TextAttributes } from "@opentui/core";
import { theme } from "../utils/theme.js";

interface LabeledInputProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  focused?: boolean;
}

/**
 * Text input with label above it
 * Uses OpenTUI's native <input> component
 */
export function LabeledInput({
  label,
  value,
  onChange,
  placeholder,
  focused = false,
}: LabeledInputProps) {
  return (
    <box flexDirection="column" width="100%" gap={1}>
      <text attributes={TextAttributes.DIM} fg={theme.textDescription}>
        {label}
      </text>
      <box
        borderStyle="single"
        borderColor={focused ? theme.borderActive : theme.borderMuted}
        paddingLeft={1}
        paddingRight={1}
      >
        <input
          value={value}
          placeholder={placeholder}
          onInput={onChange}
          focused={focused}
          width="100%"
        />
      </box>
    </box>
  );
}
