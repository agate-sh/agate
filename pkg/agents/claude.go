package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// handleClaudePostLaunch waits for Claude's trust prompt and auto-accepts it if it appears.
// Since Claude is now launched with the prompt as a CLI argument, we only need to watch
// for the trust prompt in the first few seconds after launch.
func handleClaudePostLaunch(sessionName string, paneIndex int, prompt string) error {
	debug.DebugLog("handleClaudePostLaunch: Watching for trust prompt in session=%s, pane=%d", sessionName, paneIndex)

	maxWaitTime := 10 * time.Second
	pollInterval := 500 * time.Millisecond
	startTime := time.Now()

	// Poll for up to 10 seconds looking for the trust prompt
	for time.Since(startTime) < maxWaitTime {
		time.Sleep(pollInterval)

		// Capture pane content
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleClaudePostLaunch: Failed to capture pane: %v", err)
			continue
		}

		debug.DebugLog("handleClaudePostLaunch: Captured content (first 200 chars): %q", truncateString(content, 200))

		// Check for trust prompt
		if strings.Contains(content, "Do you trust the files in this folder?") {
			debug.DebugLog("handleClaudePostLaunch: Trust prompt detected, sending Enter to accept")

			target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
			acceptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "Enter")
			if err := acceptCmd.Run(); err != nil {
				debug.DebugLog("handleClaudePostLaunch: Failed to send Enter: %v", err)
				return fmt.Errorf("failed to accept trust prompt: %w", err)
			}

			debug.DebugLog("handleClaudePostLaunch: Trust prompt accepted")
			return nil
		}

		// If the trust prompt text is gone or was never there, we're done
		// (agent already started or directory was already trusted)
		if !strings.Contains(content, "Do you trust") {
			debug.DebugLog("handleClaudePostLaunch: No trust prompt detected, agent ready")
			return nil
		}
	}

	// Timeout - but this is not a failure, trust prompt may not appear
	debug.DebugLog("handleClaudePostLaunch: Timeout waiting for trust prompt (probably already trusted)")
	return nil
}

// capturePane captures the tmux pane content including alternate screen buffer
func capturePane(sessionName string, paneIndex int) (string, error) {
	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
	captureCmd := exec.Command("tmux", "-L", "agate", "capture-pane", "-p", "-e", "-J", "-S", "-", "-t", target)
	output, err := captureCmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// truncateString truncates a string to maxLen characters for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
