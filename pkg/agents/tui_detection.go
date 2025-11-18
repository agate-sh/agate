package agents

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
)

// waitForTUIStart_Gemini waits for gemini's TUI to start by polling alternate_on
func waitForTUIStart_Gemini(sessionName string, paneIndex int, pollInterval time.Duration, maxWaitTime time.Duration, startTime time.Time) error {
	debug.DebugLog("waitForTUIStart_Gemini: Waiting for alternate screen mode...")

	target := fmt.Sprintf("%s.%d", sessionName, paneIndex)

	for {
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for gemini TUI to start (alternate_on never switched after %v)", maxWaitTime)
		}

		time.Sleep(pollInterval)

		// Check if alternate screen is on
		cmd := exec.Command("tmux", "-L", "agate", "display-message", "-p", "-t", target, "#{alternate_on}")
		output, err := cmd.Output()
		if err != nil {
			debug.DebugLog("waitForTUIStart_Gemini: Failed to check alternate_on: %v", err)
			continue
		}

		alternateOn := strings.TrimSpace(string(output))
		if alternateOn == "1" {
			debug.DebugLog("waitForTUIStart_Gemini: Alternate screen activated, TUI loaded!")
			return nil
		}
	}
}

// waitForTUIStart_Generic waits for a generic agent's TUI to start by detecting shell prompt disappearance
func waitForTUIStart_Generic(sessionName string, paneIndex int, agentName string, pollInterval time.Duration, maxWaitTime time.Duration, startTime time.Time) error {
	debug.DebugLog("waitForTUIStart_Generic: [%s] Waiting for shell to echo command...", agentName)

	var shellEchoContent string

	// First, wait for shell to echo the command
	for {
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for %s to start (shell never echoed command after %v)", agentName, maxWaitTime)
		}

		time.Sleep(pollInterval)
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("waitForTUIStart_Generic: [%s] Failed to capture pane: %v", agentName, err)
			continue
		}

		if strings.Contains(content, agentName) {
			debug.DebugLog("waitForTUIStart_Generic: [%s] Shell echoed command", agentName)
			debug.DebugLog("waitForTUIStart_Generic: [%s] Shell echo content (first 200 chars): %q", agentName, truncateString(content, 200))
			shellEchoContent = content
			break
		}
	}

	// Now wait for TUI to actually start (shell prompt disappears)
	debug.DebugLog("waitForTUIStart_Generic: [%s] Waiting for TUI to start loading...", agentName)
	time.Sleep(1 * time.Second) // Let command start hanging

	for {
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for %s TUI to start (command kept hanging after %v)", agentName, maxWaitTime)
		}

		time.Sleep(pollInterval)
		content, err := capturePane(sessionName, paneIndex)
		if err != nil {
			debug.DebugLog("waitForTUIStart_Generic: [%s] Failed to capture pane: %v", agentName, err)
			continue
		}

		// Check if TUI has actually started by ensuring shell prompt is gone
		// Shell prompts contain indicators like "❯" or the agent name followed by ANSI codes
		contentChanged := content != shellEchoContent
		shellPromptGone := !strings.Contains(content, "❯") && !strings.Contains(content, agentName+"\n\x1b")

		if contentChanged && shellPromptGone {
			debug.DebugLog("waitForTUIStart_Generic: [%s] TUI started loading!", agentName)
			debug.DebugLog("waitForTUIStart_Generic: [%s] TUI loading content (first 300 chars): %q", agentName, truncateString(content, 300))
			return nil
		}
	}
}
