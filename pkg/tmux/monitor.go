// Package tmux provides tmux session management and monitoring functionality
// for creating, attaching to, and controlling tmux sessions.
package tmux

import (
	"crypto/sha256"
	"strings"
)

// StatusMonitor tracks changes in tmux session output
type StatusMonitor struct {
	prevOutputHash []byte
	program        string
}

// newStatusMonitor creates a new status monitor for a tmux session
func newStatusMonitor(program string) *StatusMonitor {
	return &StatusMonitor{
		prevOutputHash: make([]byte, 0),
		program:        program,
	}
}

// hash generates a SHA256 hash of the given content
func (m *StatusMonitor) hash(content string) []byte {
	h := sha256.Sum256([]byte(content))
	return h[:]
}

// HasUpdated checks if the content has changed and if there's a prompt waiting
func (m *StatusMonitor) HasUpdated(content string) (updated bool, hasPrompt bool) {
	// Check for prompts specific to different AI programs
	switch m.program {
	case "claude":
		hasPrompt = strings.Contains(content, "No, and tell Claude what to do differently")
	case "aider":
		hasPrompt = strings.Contains(content, "(Y)es/(N)o/(D)on't ask again")
	case "codex":
		// Add specific prompt detection for codex if needed
		hasPrompt = false
	default:
		// Generic prompt detection - look for common patterns
		hasPrompt = strings.HasSuffix(strings.TrimSpace(content), ">") ||
			strings.HasSuffix(strings.TrimSpace(content), "$") ||
			strings.HasSuffix(strings.TrimSpace(content), ":")
	}

	// Check if content has changed
	currentHash := m.hash(content)
	if !bytesEqual(currentHash, m.prevOutputHash) {
		m.prevOutputHash = currentHash
		return true, hasPrompt
	}

	return false, hasPrompt
}

// bytesEqual compares two byte slices for equality
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
