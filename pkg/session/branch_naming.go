package session

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/agents"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	// adjectives for random branch names
	adjectives = []string{
		"quick", "lazy", "clever", "brave", "calm", "eager",
		"gentle", "happy", "jolly", "kind", "lively", "merry",
		"nice", "proud", "silly", "witty", "zany", "bold",
	}

	// nouns for random branch names
	nouns = []string{
		"fox", "dog", "cat", "bear", "lion", "tiger",
		"eagle", "hawk", "owl", "deer", "wolf", "panda",
		"koala", "zebra", "giraffe", "elephant", "dolphin", "shark",
	}

	// regex to extract branch names from AI output
	branchNameRegex = regexp.MustCompile(`[a-z0-9][-a-z0-9]*[a-z0-9]`)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomBranchName generates a random fallback branch name
// Format: adjective-noun-number (e.g., "quick-fox-42")
func GenerateRandomBranchName() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(100)
	return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}

// GenerateBranchNameFromPrompt uses an AI agent to generate a git branch name from a prompt.
// This is a SYNCHRONOUS operation that blocks until the agent responds or times out.
// Returns the generated branch name or an error if generation fails.
// The caller should use GenerateRandomBranchName() as a fallback if this returns an error.
func GenerateBranchNameFromPrompt(prompt string, agent agents.AgentConfig) (string, error) {
	debug.DebugLog("Generating branch name from prompt using agent: %s", agent.Name)

	// Get the headless command for the agent
	cmdArgs := agent.HeadlessCommand(fmt.Sprintf(
		"Generate a short git branch name (kebab-case, lowercase, no special characters, max 30 chars) for this task: %s\n\nRespond with ONLY the branch name, nothing else.",
		prompt,
	))

	if cmdArgs == nil || len(cmdArgs) == 0 {
		debug.DebugLog("Agent %s does not support headless mode", agent.Name)
		return "", fmt.Errorf("agent %s does not support headless mode", agent.Name)
	}

	// Create command with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

	// Run the command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		debug.DebugLog("Failed to run agent command: %v, output: %s", err, string(output))
		return "", fmt.Errorf("agent command failed: %w", err)
	}

	// Parse the output to extract a valid branch name
	branchName := parseBranchName(string(output))
	if branchName == "" {
		debug.DebugLog("Failed to parse branch name from agent output: %s", string(output))
		return "", fmt.Errorf("failed to extract branch name from agent output")
	}

	debug.DebugLog("Generated branch name: %s", branchName)
	return branchName, nil
}

// parseBranchName extracts a valid git branch name from AI agent output
func parseBranchName(output string) string {
	// Clean up the output
	output = strings.TrimSpace(output)
	output = strings.ToLower(output)

	// Remove common prefixes/suffixes
	output = strings.TrimPrefix(output, "branch name:")
	output = strings.TrimPrefix(output, "branch:")
	output = strings.TrimSpace(output)

	// Try to find a valid branch name pattern
	matches := branchNameRegex.FindAllString(output, -1)
	if len(matches) == 0 {
		return ""
	}

	// Use the first match, but prefer longer matches
	var best string
	for _, match := range matches {
		if len(match) > len(best) && len(match) <= 30 {
			best = match
		}
	}

	// Ensure it doesn't start or end with a hyphen
	best = strings.Trim(best, "-")

	return best
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
