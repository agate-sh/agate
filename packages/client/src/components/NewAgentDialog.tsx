import { useKeyboard } from "@opentui/react";
import { TextAttributes } from "@opentui/core";
import { useState } from "react";
import { AGENTS } from "@agate/shared";
import { Dialog } from "./Dialog.js";
import { LabeledInput } from "./LabeledInput.js";
import { Button } from "./Button.js";
import { generateRandomBranchName } from "../utils/git.js";

interface NewAgentDialogProps {
  repoName: string;
  onSelect: (branchName: string, agentName: string) => void;
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

  const randomBranchPlaceholder = useState(() => generateRandomBranchName())[0];

  // Get the selected agent config based on current agent command
  const selectedAgent = Object.values(AGENTS).find(
    (a) => a.name === agentCommand.trim().toLowerCase()
  ) || AGENTS.default;

  // Check if agent command is valid
  const isValidAgent = Object.values(AGENTS).some(
    (a) => a.name === agentCommand.trim().toLowerCase() && a.name !== "default"
  );

  useKeyboard((key) => {
    if (key.name === "tab") {
      // Tab to next field, Shift+Tab to previous field
      if (key.shift) {
        setFocusedField(focusedField === 0 ? 1 : 0);
      } else {
        setFocusedField(focusedField === 0 ? 1 : 0);
      }
      return;
    }

    if (key.name === "return") {
      // Only create if agent is valid
      if (isValidAgent) {
        const finalBranchName = branchName.trim() || randomBranchPlaceholder;
        onSelect(finalBranchName, agentCommand.trim().toLowerCase());
      } else {
        setError("Please enter a valid agent command");
      }
      return;
    }
  });

  return (
    <Dialog title="" onClose={onClose}>
      <box flexDirection="column" paddingLeft={1} paddingRight={1}>
        {/* Header: repo > New agent */}
        <box marginBottom={1}>
          <text attributes={TextAttributes.DIM}>{repoName}</text>
          <text fg="blue">
            <strong> {"> "} </strong>
          </text>
          <text fg="blue">
            <strong>New agent</strong>
          </text>
        </box>

        {/* Divider */}
        <box marginBottom={1}>
          <text attributes={TextAttributes.DIM}>{"─".repeat(56)}</text>
        </box>

        {/* Branch name input */}
        <box marginBottom={1}>
          <LabeledInput
            label="Branch name"
            value={branchName}
            onChange={setBranchName}
            placeholder={randomBranchPlaceholder}
            focused={focusedField === 0}
          />
        </box>

        {/* Agent command input */}
        <box marginBottom={2}>
          <LabeledInput
            label="Agent command"
            value={agentCommand}
            onChange={setAgentCommand}
            placeholder="claude, codex, etc"
            focused={focusedField === 1}
          />
        </box>

        {/* Create button */}
        <box marginBottom={1}>
          <Button
            label="Create and attach"
            shortcut="↵"
            disabled={!isValidAgent}
            agentColor={selectedAgent.borderColor}
            variant="agent"
            fullWidth
          />
        </box>

        {/* Error message */}
        {error && (
          <box marginBottom={1}>
            <text fg="red">Error: {error}</text>
          </box>
        )}

        {/* Help text */}
        <box justifyContent="center">
          <text attributes={TextAttributes.DIM}>tab: navigate fields • esc: cancel</text>
        </box>
      </box>
    </Dialog>
  );
}
