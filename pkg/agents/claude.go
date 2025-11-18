package agents

import (
	"fmt"
	"time"

	"agate/internal/debug"
)

// handleClaudePostLaunch waits for Claude Code to be ready using a 4-phase approach:
// Phase 1: Wait for TUI to start loading
// Phase 2: Handle trust prompt if present
// Phase 3: Wait for screen stability (with trust prompt detection integrated)
// Phase 4: Type prompt
func handleClaudePostLaunch(sessionName string, paneIndex int, prompt string) error {
	debug.DebugLog("handleClaudePostLaunch: Waiting for agent ready in session=%s, pane=%d", sessionName, paneIndex)

	pollInterval := 500 * time.Millisecond
	requiredStableCount := 3
	maxWaitTime := 15 * time.Second
	startTime := time.Now()

	// Phase 1: Wait for TUI to start loading
	// Claude doesn't use alternate screen, so use generic content-based detection
	if err := waitForTUIStart_Generic(sessionName, paneIndex, "claude", pollInterval, maxWaitTime, startTime); err != nil {
		return err
	}

	// Phase 2 & 3: Wait for screen stability with trust prompt handling integrated
	// We integrate trust prompt handling into stability polling because the prompt
	// appears during the loading process and needs to reset stability detection
	debug.DebugLog("handleClaudePostLaunch: Beginning stability polling with trust prompt handling...")

	var previousCaptures []string

	for {
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for claude to stabilize (screen kept changing after %v)", maxWaitTime)
		}

		time.Sleep(pollInterval)

		// Capture pane content
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("handleClaudePostLaunch: Failed to capture pane: %v", err)
			continue
		}

		// Phase 2: Check for and handle trust prompt
		promptHandled, err := handleTUIPrompts_Claude(sessionName, paneIndex, content, pollInterval)
		if err != nil {
			return err
		}
		if promptHandled {
			// Reset stability counter and continue polling
			previousCaptures = []string{}
			continue
		}

		// Check if content changed from previous capture - if so, reset stability counter
		if len(previousCaptures) > 0 && content != previousCaptures[len(previousCaptures)-1] {
			debug.DebugLog("handleClaudePostLaunch: Screen changed, resetting stability counter")
			previousCaptures = []string{}
		}

		// Check for screen stability
		if isScreenStable(content, previousCaptures, requiredStableCount) {
			debug.DebugLog("handleClaudePostLaunch: Screen stable for %d captures", requiredStableCount)
			debug.DebugLog("handleClaudePostLaunch: Stable content (first 300 chars): %q", truncateString(content, 300))
			break
		}

		// Add current capture to history
		previousCaptures = append(previousCaptures, content)
		debug.DebugLog("handleClaudePostLaunch: Screen stable (%d/%d captures)", len(previousCaptures), requiredStableCount)
	}

	// Phase 4: Type prompt (without Enter - manager will send Enter to all agents at once)
	if err := typePrompt(sessionName, paneIndex, prompt); err != nil {
		return err
	}

	return nil
}
