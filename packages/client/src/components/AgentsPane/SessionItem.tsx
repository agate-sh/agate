import { TextAttributes } from "@opentui/core";
import { AGENTS } from "@agate/shared";
import { icons } from "../../utils/icons.js";
import type { WorktreeInfo } from "./types.js";

interface SessionItemProps {
  worktree: WorktreeInfo;
  isPinned: boolean;
  isLinked: boolean;
  isSelected: boolean;
  isHovered: boolean;
}

/**
 * Session item showing branch name with pin/repo icon
 * Shows contextual hints when hovered
 */
export function SessionItem({
  worktree,
  isPinned,
  isLinked,
  isHovered,
}: SessionItemProps) {
  const agent = AGENTS[worktree.agentName as keyof typeof AGENTS] || AGENTS.default;
  const icon = isPinned ? icons.Pin : icons.GitRepo;
  const iconColor = isPinned ? agent.borderColor : "gray";

  // Determine hint text based on state
  let hint = "";
  if (isHovered) {
    if (isPinned) {
      // Hovering a pinned session
      hint = isLinked ? " ↵ to pin, d to delete" : " ↵ to pin";
    } else {
      // Hovering an unpinned session
      hint = isLinked ? " ↵ to select, d to delete" : " ↵ to select";
    }
  }

  return (
    <box flexDirection="row" justifyContent="space-between" width="100%">
      <box flexDirection="row">
        <text fg={iconColor}>   {icon}  </text>
        <text>{worktree.branch}</text>
      </box>
      {hint && <text attributes={TextAttributes.DIM}>{hint}</text>}
    </box>
  );
}
