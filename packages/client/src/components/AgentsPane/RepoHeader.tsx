import { TextAttributes } from "@opentui/core";
import { icons } from "../../utils/icons.js";

interface RepoHeaderProps {
  repoName: string;
  isExpanded: boolean;
  isHovered: boolean;
  isCurrent: boolean;
}

/**
 * Repository header with collapse/expand arrow
 * Shows "n new agent" hint when hovered and collapsed
 */
export function RepoHeader({
  repoName,
  isExpanded,
  isHovered,
  isCurrent,
}: RepoHeaderProps) {
  const arrow = isExpanded ? icons.ArrowDown : icons.ArrowRight;
  const showShortcut = isHovered && !isExpanded;

  return (
    <box flexDirection="row" justifyContent="space-between" width="100%">
      <text fg={isCurrent ? "#89b4fa" : undefined}>
        <strong>{repoName}</strong>
      </text>
      {showShortcut && <text attributes={TextAttributes.DIM}>n new agent </text>}
      <text>{arrow}</text>
    </box>
  );
}
