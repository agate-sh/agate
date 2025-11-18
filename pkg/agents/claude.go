package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// detectClaudePrompt checks if the content contains a Claude Code prompt
func detectClaudePrompt(content string) bool {
	// Check for Claude Code prompts
	return strings.Contains(content, "Enter to confirm") ||
		strings.Contains(content, "Esc to exit") ||
		strings.Contains(content, "Do you trust the files in this folder?") ||
		strings.Contains(content, "No, and tell Claude what to do differently")
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

// handleClaudePostLaunch waits for Claude Code to be ready by:
// 1. Waiting for screen content to change (agent started)
// 2. Checking for and accepting the trust prompt if present
// 3. Waiting for screen stability (3 consecutive identical captures at 500ms intervals)
// 4. Attempting to type the prompt and verifying it appears on screen
// 5. If prompt appears, submitting it with Enter; if not, resuming stability polling
func handleClaudePostLaunch(sessionName string, paneIndex int, prompt string) error {
	debug.DebugLog("handleClaudePostLaunch: Waiting for agent ready in session=%s, pane=%d", sessionName, paneIndex)

	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
	pollInterval := 500 * time.Millisecond
	requiredStableCount := 3

	// Wait for screen content to change (agent starting)
	debug.DebugLog("handleClaudePostLaunch: Waiting for screen content to change (agent starting)...")
	initialContent, err := capturePane(sessionName, paneIndex)
	if err != nil {
		debug.DebugLog("handleClaudePostLaunch: Failed initial capture: %v", err)
		initialContent = ""
	}

	for {
		time.Sleep(pollInterval)
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleClaudePostLaunch: Failed to capture pane: %v", err)
			continue
		}

		if content != initialContent {
			debug.DebugLog("handleClaudePostLaunch: Screen content changed, agent started!")
			break
		}
	}

	// Now begin stability polling
	debug.DebugLog("handleClaudePostLaunch: Beginning stability polling...")
	var previousCaptures []string

	for {
		time.Sleep(pollInterval)

		// Capture pane content
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleClaudePostLaunch: Failed to capture pane: %v", err)
			continue
		}

		// Check for trust prompt FIRST - if present, accept it
		if strings.Contains(content, "Do you trust the files in this folder?") {
			debug.DebugLog("handleClaudePostLaunch: Trust prompt detected, sending Enter to accept")

			// Send Enter to accept (option 1 "Yes, proceed" is pre-selected)
			acceptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "Enter")
			if err := acceptCmd.Run(); err != nil {
				debug.DebugLog("handleClaudePostLaunch: Failed to send Enter: %v", err)
			} else {
				debug.DebugLog("handleClaudePostLaunch: Sent Enter to accept trust prompt")
			}

			// Wait for prompt to clear and reset stability detection
			time.Sleep(pollInterval)
			previousCaptures = []string{}
			continue
		}

		// Check if content changed from previous capture - if so, reset stability counter
		if len(previousCaptures) > 0 && content != previousCaptures[len(previousCaptures)-1] {
			debug.DebugLog("handleClaudePostLaunch: Screen changed during polling, resetting stability counter")
			previousCaptures = []string{}
			// Don't continue - let it fall through to add this capture to the array
		}

		// Check for screen stability
		if isScreenStable(content, previousCaptures, requiredStableCount) {
			debug.DebugLog("handleClaudePostLaunch: Screen stable for %d captures, attempting to verify with prompt...", requiredStableCount)

			// Try sending the prompt text (without Enter)
			sendPromptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "-l", prompt)
			if err := sendPromptCmd.Run(); err != nil {
				debug.DebugLog("handleClaudePostLaunch: Failed to send prompt text: %v", err)
				previousCaptures = []string{}
				continue
			}

			// Wait for prompt to appear
			time.Sleep(pollInterval)

			// Capture screen and check if prompt appears
			verifyContent, err := capturePane(sessionName, paneIndex)
			if err != nil {
				debug.DebugLog("handleClaudePostLaunch: Failed to capture for verification: %v", err)
				previousCaptures = []string{}
				continue
			}

			if strings.Contains(verifyContent, prompt) {
				debug.DebugLog("handleClaudePostLaunch: Prompt verified on screen, agent is ready!")
				// Don't send Enter yet - manager will send it to all agents at once
				return nil
			}

			debug.DebugLog("handleClaudePostLaunch: Prompt not visible on screen, resetting stability counter")
			previousCaptures = []string{}
			continue
		}

		// Add current capture to history
		previousCaptures = append(previousCaptures, content)

		debug.DebugLog("handleClaudePostLaunch: Screen stable (%d/%d captures)", len(previousCaptures), requiredStableCount)
	}
}
