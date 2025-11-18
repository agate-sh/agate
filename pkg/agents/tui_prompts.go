package agents

import (
	"strings"
)

// detectClaudePrompt checks if the content contains a Claude Code prompt
// This is used by the DetectPrompt method to determine if Claude is waiting for user input
func detectClaudePrompt(content string) bool {
	// Check for Claude Code prompts
	return strings.Contains(content, "Enter to confirm") ||
		strings.Contains(content, "Esc to exit") ||
		strings.Contains(content, "Do you trust the files in this folder?") ||
		strings.Contains(content, "No, and tell Claude what to do differently")
}
