import { useKeyboard } from "@opentui/react";
import { TextAttributes } from "@opentui/core";
import { useState } from "react";
import {
  AGENTS,
  type AgentName,
  generateBranchName,
  getAgentConfig,
  getAgentKeyFromCommand,
  isValidAgentCommand,
} from "@agate/shared";
import {
  Dialog,
  DIALOG_WIDTH,
  DIALOG_PADDING_X,
} from "./Dialog.js";
import { LabeledInput } from "./LabeledInput.js";
import { Button } from "./Button.js";

interface NewAgentDialogProps {
  repoName: string;
  onSelect: (branchName: string, agentName: AgentName) => void;
  onClose: () => void;
}

/**
 * Dialog for creating a new agent session
 * Matches the Go implementation's SessionDialog component
 */
export function NewAgentDialog({
  repoName,
  onSelect,
  onClose,
}: NewAgentDialogProps) {
  const [branchName, setBranchName] = useState("");
  const [agentCommand, setAgentCommand] = useState("");
  const [focusedField, setFocusedField] = useState<0 | 1>(0); // 0 = branch, 1 = agent
  const [error, setError] = useState("");

  const randomBranchPlaceholder = useState(() => generateBranchName())[0];

  const agentKey = getAgentKeyFromCommand(agentCommand);
  const selectedAgent =
    agentKey !== null ? AGENTS[agentKey] : getAgentConfig(agentCommand);
  const isValidAgent = agentKey !== null;
  const INNER_PADDING_X = 2;
  const SECTION_GAP = 2;
  const contentWidth =
    DIALOG_WIDTH - 2 * (DIALOG_PADDING_X + INNER_PADDING_X);
  const horizontalRule = "─".repeat(Math.max(0, contentWidth));

  useKeyboard((key) => {
    if (key.name === "tab") {
      // Tab to next field, Shift+Tab to previous field
      setFocusedField((current) => (current === 0 ? 1 : 0));
      return;
    }

    if (key.name === "return") {
      // Only create if agent is valid
      if (isValidAgentCommand(agentCommand) && agentKey) {
        const finalBranchName = branchName.trim() || randomBranchPlaceholder;
        onSelect(finalBranchName, agentKey);
      } else {
        setError("Please enter a valid agent command");
      }
      return;
    }
  });

  return (
    <Dialog title="" onClose={onClose}>
      <box
        flexDirection="column"
        paddingLeft={INNER_PADDING_X}
        paddingRight={INNER_PADDING_X}
        gap={SECTION_GAP}
      >
        {/* Header: repo > New agent */}
        <box width="100%" flexDirection="row" alignItems="center" gap={1}>
          <text attributes={TextAttributes.DIM}>{repoName}</text>
          <text fg="blue">
            <strong>{">"}</strong>
          </text>
          <text fg="blue">
            <strong>New agent</strong>
          </text>
        </box>

        {/* Divider */}
        <box>
          <text attributes={TextAttributes.DIM}>{horizontalRule}</text>
        </box>

        {/* Branch name input */}
        <LabeledInput
          label="Branch name"
          value={branchName}
          onChange={(value) => {
            setBranchName(value);
            setError("");
          }}
          placeholder={randomBranchPlaceholder}
          focused={focusedField === 0}
        />

        {/* Agent command input */}
        <LabeledInput
          label="Agent command"
          value={agentCommand}
          onChange={(value) => {
            setAgentCommand(value);
            setError("");
          }}
          placeholder="claude, codex, etc"
          focused={focusedField === 1}
        />

        {/* Create button */}
        <Button
          label="Create and attach"
          shortcut="↵"
          disabled={!isValidAgent}
          agentColor={selectedAgent.borderColor}
          variant="agent"
          fullWidth
        />

        {/* Error message */}
        {error && (
          <text fg="red">Error: {error}</text>
        )}

        {/* Help text */}
        <box justifyContent="center">
          <text attributes={TextAttributes.DIM}>tab: navigate fields • esc: cancel</text>
        </box>
      </box>
    </Dialog>
  );
}
