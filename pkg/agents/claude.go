package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// handleClaudePostLaunch waits for the Claude Code trust prompt and auto-accepts it
func handleClaudePostLaunch(sessionName string, paneIndex int) error {
	debug.DebugLog("handleClaudePostLaunch: Waiting for trust prompt in pane %d", paneIndex)

	// Poll for trust prompt with timeout
	timeout := 10 * time.Second
	pollInterval := 200 * time.Millisecond
	startTime := time.Now()

	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)

	for {
		// Check timeout
		if time.Since(startTime) > timeout {
			debug.DebugLog("handleClaudePostLaunch: Timeout waiting for trust prompt (agent may have started normally)")
			return nil // Not an error - agent might not show prompt
		}

		// Capture pane content directly using tmux command
		cmd := exec.Command("tmux", "capture-pane", "-p", "-e", "-J", "-t", target)
		output, err := cmd.Output()
		if err != nil {
			debug.DebugLog("handleClaudePostLaunch: Failed to capture pane: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		content := string(output)

		// Check for trust prompt
		if strings.Contains(content, "Do you trust the files in this folder?") {
			debug.DebugLog("handleClaudePostLaunch: Trust prompt detected, sending '1' to accept")

			// Send "1" to accept (option 1: Yes, proceed)
			sendCmd := exec.Command("tmux", "send-keys", "-t", target, "1")
			if err := sendCmd.Run(); err != nil {
				return fmt.Errorf("failed to send accept key: %w", err)
			}

			debug.DebugLog("handleClaudePostLaunch: Successfully accepted trust prompt")
			return nil
		}

		// Wait before next poll
		time.Sleep(pollInterval)
	}
}
