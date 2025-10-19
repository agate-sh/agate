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
  onChange,
  placeholder,
  focused = false,
}: LabeledInputProps) {
  return (
    <box flexDirection="column">
      <text fg="white">
        <strong>{label}</strong>
      </text>
      <input
        placeholder={placeholder}
        onInput={onChange}
        focused={focused}
      />
    </box>
  );
}
