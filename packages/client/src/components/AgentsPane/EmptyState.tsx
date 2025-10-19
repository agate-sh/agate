import { TextAttributes } from "@opentui/core";

/**
 * Empty state shown when no agents exist
 */
export function EmptyState() {
  return (
    <box
      flexGrow={1}
      flexDirection="column"
      justifyContent="center"
      alignItems="center"
    >
      <box marginBottom={1}>
        <text attributes={TextAttributes.DIM}>No agents yet</text>
      </box>
      <box>
        <text fg="gray">n to create a new agent</text>
      </box>
    </box>
  );
}
