package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// handleGenericPostLaunch waits for a generic agent to be ready by:
// 1. Waiting for screen content to change (agent started)
// 2. Waiting for screen stability (3 consecutive identical captures at 500ms intervals)
// 3. Attempting to type the prompt and verifying it appears on screen
// 4. If prompt appears, submitting it with Enter; if not, resuming stability polling
// This works for agents that don't have special trust prompts (gemini, codex, etc.)
func handleGenericPostLaunch(sessionName string, paneIndex int, agentName string, prompt string) error {
	debug.DebugLog("handleGenericPostLaunch: Waiting for %s to be ready in session=%s, pane=%d", agentName, sessionName, paneIndex)

	pollInterval := 500 * time.Millisecond
	requiredStableCount := 3

	// Step 1: Wait for shell to echo the command back
	debug.DebugLog("handleGenericPostLaunch: [%s] Waiting for shell to echo command...", agentName)
	var shellEchoContent string
	for {
		time.Sleep(pollInterval)
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleGenericPostLaunch: [%s] Failed to capture pane: %v", agentName, err)
			continue
		}

		if strings.Contains(content, agentName) {
			debug.DebugLog("handleGenericPostLaunch: [%s] Shell echoed command", agentName)
			shellEchoContent = content
			break
		}
	}

	// Step 2: Wait for content to change from shell echo (agent UI loading)
	debug.DebugLog("handleGenericPostLaunch: [%s] Waiting for agent UI to load...", agentName)
	for {
		time.Sleep(pollInterval)
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleGenericPostLaunch: [%s] Failed to capture pane: %v", agentName, err)
			continue
		}

		if content != shellEchoContent {
			debug.DebugLog("handleGenericPostLaunch: [%s] Agent UI loaded!", agentName)
			debug.DebugLog("handleGenericPostLaunch: [%s] Content (first 200 chars): %q", agentName, truncateString(content, 200))
			break
		}
	}

	// Now begin stability polling
	debug.DebugLog("handleGenericPostLaunch: [%s] Beginning stability polling...", agentName)
	var previousCaptures []string

	for {
		time.Sleep(pollInterval)

		// Capture pane content
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleGenericPostLaunch: [%s] Failed to capture pane: %v", agentName, err)
			continue
		}

		// Check if content changed from previous capture - if so, reset stability counter
		if len(previousCaptures) > 0 && content != previousCaptures[len(previousCaptures)-1] {
			debug.DebugLog("handleGenericPostLaunch: [%s] Screen changed during polling, resetting stability counter", agentName)
			previousCaptures = []string{}
			// Don't continue - let it fall through to add this capture to the array
		}

		// Check for screen stability
		if isScreenStable(content, previousCaptures, requiredStableCount) {
			debug.DebugLog("handleGenericPostLaunch: [%s] Screen stable for %d captures, attempting to verify with prompt...", agentName, requiredStableCount)

			// Try sending the prompt text (without Enter)
			target := fmt.Sprintf("%s.%d", sessionName, paneIndex)
			sendPromptCmd := exec.Command("tmux", "-L", "agate", "send-keys", "-t", target, "-l", prompt)
			if err := sendPromptCmd.Run(); err != nil {
				debug.DebugLog("handleGenericPostLaunch: [%s] Failed to send prompt text: %v", agentName, err)
				previousCaptures = []string{}
				continue
			}

			// Wait for prompt to appear
			time.Sleep(pollInterval)

			// Capture screen and check if prompt appears
			verifyContent, err := capturePane(sessionName, paneIndex)
			if err != nil {
				debug.DebugLog("handleGenericPostLaunch: [%s] Failed to capture for verification: %v", agentName, err)
				previousCaptures = []string{}
				continue
			}

			if strings.Contains(verifyContent, prompt) {
				debug.DebugLog("handleGenericPostLaunch: [%s] Prompt verified on screen, agent is ready!", agentName)
				// Don't send Enter yet - manager will send it to all agents at once
				return nil
			}

			debug.DebugLog("handleGenericPostLaunch: [%s] Prompt not visible on screen, resetting stability counter", agentName)
			previousCaptures = []string{}
			continue
		}

		// Add current capture to history
		previousCaptures = append(previousCaptures, content)

		debug.DebugLog("handleGenericPostLaunch: [%s] Screen stable (%d/%d captures)", agentName, len(previousCaptures), requiredStableCount)
	}
}
