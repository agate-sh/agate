package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

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

// isScreenStable checks if the last N screen captures are identical
func isScreenStable(currentCapture string, previousCaptures []string, requiredStableCount int) bool {
	if len(previousCaptures) < requiredStableCount-1 {
		return false
	}

	// Check if current capture matches all previous captures
	for i := len(previousCaptures) - (requiredStableCount - 1); i < len(previousCaptures); i++ {
		if previousCaptures[i] != currentCapture {
			return false
		}
	}

	return true
}

// detectClaudePrompt checks if the content contains a Claude Code prompt
func detectClaudePrompt(content string) bool {
	// Check for Claude Code prompts
	return strings.Contains(content, "Enter to confirm") ||
		strings.Contains(content, "Esc to exit") ||
		strings.Contains(content, "Do you trust the files in this folder?") ||
		strings.Contains(content, "No, and tell Claude what to do differently")
}

// waitForScreenStability polls screen captures until N consecutive identical captures are seen
func waitForScreenStability(sessionName string, paneIndex int, pollInterval time.Duration, requiredStableCount int, maxWaitTime time.Duration, startTime time.Time) error {
	debug.DebugLog("waitForScreenStability: Starting stability polling for pane %d", paneIndex)

	var previousCaptures []string

	for {
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for pane %d to stabilize (screen kept changing after %v)", paneIndex, maxWaitTime)
		}

		time.Sleep(pollInterval)

		// Capture pane content
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("waitForScreenStability: Failed to capture pane %d: %v", paneIndex, err)
			continue
		}

		// Check if content changed from previous capture - if so, reset stability counter
		if len(previousCaptures) > 0 && content != previousCaptures[len(previousCaptures)-1] {
			debug.DebugLog("waitForScreenStability: Pane %d screen changed, resetting stability counter", paneIndex)
			previousCaptures = []string{}
		}

		// Check for screen stability
		if isScreenStable(content, previousCaptures, requiredStableCount) {
			debug.DebugLog("waitForScreenStability: Pane %d stable for %d captures", paneIndex, requiredStableCount)
			debug.DebugLog("waitForScreenStability: Stable content (first 300 chars): %q", truncateString(content, 300))
			return nil
		}

		// Add current capture to history
		previousCaptures = append(previousCaptures, content)
		debug.DebugLog("waitForScreenStability: Pane %d stable (%d/%d captures)", paneIndex, len(previousCaptures), requiredStableCount)
	}
}

// typePrompt types the prompt text into the pane without pressing Enter
func typePrompt(sessionName string, paneIndex int, prompt string) error {
	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
	sendPromptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "-l", prompt)
	if err := sendPromptCmd.Run(); err != nil {
		debug.DebugLog("typePrompt: Failed to send prompt to pane %d: %v", paneIndex, err)
		return fmt.Errorf("failed to type prompt: %w", err)
	}

	debug.DebugLog("typePrompt: Prompt typed to pane %d, agent is ready!", paneIndex)
	return nil
}
