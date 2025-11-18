package agents

import (
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

// handleGenericPostLaunch waits for a generic agent to be ready using a 4-phase approach:
// Phase 1: Wait for TUI to start (gemini uses alternate_on, others use content detection)
// Phase 2: Handle TUI prompts (no-op for generic agents)
// Phase 3: Wait for screen stability
// Phase 4: Type prompt
func handleGenericPostLaunch(sessionName string, paneIndex int, agentName string, prompt string) error {
	debug.DebugLog("handleGenericPostLaunch: Waiting for %s to be ready in session=%s, pane=%d", agentName, sessionName, paneIndex)

	pollInterval := 500 * time.Millisecond
	requiredStableCount := 3
	maxWaitTime := 15 * time.Second
	startTime := time.Now()

	// Phase 1: Wait for TUI to start loading
	if agentName == "gemini" {
		// Gemini uses alternate screen mode - more reliable detection
		if err := waitForTUIStart_Gemini(sessionName, paneIndex, pollInterval, maxWaitTime, startTime); err != nil {
			return err
		}
	} else {
		// Other agents use content-based detection
		if err := waitForTUIStart_Generic(sessionName, paneIndex, agentName, pollInterval, maxWaitTime, startTime); err != nil {
			return err
		}
	}

	// Phase 2: Handle TUI prompts (no-op for generic agents)
	// Generic agents don't have intermediate prompts like Claude's trust prompt

	// Phase 3: Wait for screen stability
	if err := waitForScreenStability(sessionName, paneIndex, pollInterval, requiredStableCount, maxWaitTime, startTime); err != nil {
		return err
	}

	// Phase 4: Type prompt (without Enter - manager will send Enter to all agents at once)
	if err := typePrompt(sessionName, paneIndex, prompt); err != nil {
		return err
	}

	return nil
}
