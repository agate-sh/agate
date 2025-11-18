package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// handleTUIPrompts_Claude checks for and accepts Claude's trust prompt if present
// This runs in a loop during stability polling, resetting the counter if prompt is found
func handleTUIPrompts_Claude(sessionName string, paneIndex int, content string, pollInterval time.Duration) (promptHandled bool, err error) {
	// Check for trust prompt
	if !strings.Contains(content, "Do you trust the files in this folder?") {
		return false, nil // No prompt found
	}

	debug.DebugLog("handleTUIPrompts_Claude: Trust prompt detected, sending Enter to accept")

	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
	acceptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "Enter")
	if err := acceptCmd.Run(); err != nil {
		debug.DebugLog("handleTUIPrompts_Claude: Failed to send Enter: %v", err)
		return false, fmt.Errorf("failed to accept trust prompt: %w", err)
	}

	debug.DebugLog("handleTUIPrompts_Claude: Sent Enter to accept trust prompt")

	// Wait for prompt to clear
	time.Sleep(pollInterval)
	return true, nil
}
