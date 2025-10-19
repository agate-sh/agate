import { TextAttributes } from "@opentui/core";
import { icons } from "../../utils/icons.js";

interface SectionHeaderProps {
  title: "Main worktree" | "Linked worktrees";
  isHovered: boolean;
}

/**
 * Section header for main/linked worktrees
 * Shows "n new agent" hint when hovered on linked worktrees section
 */
export function SectionHeader({ title, isHovered }: SectionHeaderProps) {
  const icon = title === "Main worktree" ? icons.Home : icons.Link;
  const showShortcut = isHovered && title === "Linked worktrees";

  return (
    <box flexDirection="row" justifyContent="space-between" width="100%">
      <box flexDirection="row">
        <text attributes={TextAttributes.DIM}>{icon}  </text>
        <text fg="gray">{title}</text>
      </box>
      {showShortcut && <text attributes={TextAttributes.DIM}>n new agent</text>}
    </box>
  );
}
