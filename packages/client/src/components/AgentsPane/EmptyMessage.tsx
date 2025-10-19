import { TextAttributes } from "@opentui/core";

interface EmptyMessageProps {
  sectionType: "main" | "linked";
}

/**
 * Empty message shown when no sessions exist in a section
 */
export function EmptyMessage({ sectionType }: EmptyMessageProps) {
  const message = sectionType === "main" ? "No main session" : "No agents";

  return (
    <box paddingLeft={3}>
      <text attributes={TextAttributes.DIM}>{message}</text>
    </box>
  );
}
