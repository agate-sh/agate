package session

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/agents"

	tea "github.com/charmbracelet/bubbletea"
)

// GenerateBranchNameFromPrompt generates a deterministic git branch name.
// Format: {repo-name}-{unix-timestamp} (e.g., "agate-1737139852")
// This is instant and requires no LLM calls.
func GenerateBranchNameFromPrompt(repoName string) string {
	// Sanitize repo name: lowercase, replace invalid chars with hyphens
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r - 'A' + 'a' // convert to lowercase
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, repoName)

	// Trim hyphens from start/end
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		sanitized = "branch"
	}

	// Generate unix timestamp
	timestamp := time.Now().Unix()

	branchName := fmt.Sprintf("%s-%d", sanitized, timestamp)
	debug.DebugLog("Generated deterministic branch name: %s", branchName)
	return branchName
}

// SessionDescriptionGeneratedMsg is sent when async description generation completes
type SessionDescriptionGeneratedMsg struct {
	SessionID   string
	Description string
	Error       error
}

// GenerateSessionDescription returns a tea.Cmd that generates a session description asynchronously.
// This is NON-BLOCKING - the session can proceed without the description.
// When complete, it sends a SessionDescriptionGeneratedMsg with the result.
func GenerateSessionDescription(prompt string, agent agents.AgentConfig, session *Session, manager *Manager) tea.Cmd {
	return func() tea.Msg {
		debug.DebugLog("Generating session description asynchronously for session: %s", session.ID)

		// Get the headless command for the agent
		cmdArgs := agent.HeadlessCommand(fmt.Sprintf(
			"Generate a concise 1-2 sentence description of the work to be done for this task: %s\n\nRespond with ONLY the description, nothing else.",
			prompt,
		))

		if cmdArgs == nil || len(cmdArgs) == 0 {
			debug.DebugLog("Agent %s does not support headless mode for description", agent.Name)
			return SessionDescriptionGeneratedMsg{
				SessionID:   session.ID,
				Description: "",
				Error:       fmt.Errorf("agent does not support headless mode"),
			}
		}

		// Create command with timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

		// Run the command and capture output
		output, err := cmd.CombinedOutput()
		if err != nil {
			debug.DebugLog("Failed to generate description: %v, output: %s", err, string(output))
			return SessionDescriptionGeneratedMsg{
				SessionID:   session.ID,
				Description: "",
				Error:       fmt.Errorf("failed to generate description: %w", err),
			}
		}

		// Clean up the description
		description := strings.TrimSpace(string(output))
		debug.DebugLog("Generated description for session %s: %s", session.ID, description)

		// Update the session description in the manager
		if manager != nil {
			if err := manager.UpdateSessionDescription(session.ID, description); err != nil {
				debug.DebugLog("Failed to update session description: %v", err)
			}
		}

		return SessionDescriptionGeneratedMsg{
			SessionID:   session.ID,
			Description: description,
			Error:       nil,
		}
	}
}
