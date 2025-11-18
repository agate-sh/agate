// Package tmux provides tmux session management and monitoring functionality
// for creating, attaching to, and controlling tmux sessions.
package tmux

import (
	"crypto/sha256"
)

// StatusMonitor tracks changes in tmux session output
type StatusMonitor struct {
	prevOutputHash []byte
}

// newStatusMonitor creates a new status monitor for a tmux session
func newStatusMonitor() *StatusMonitor {
	return &StatusMonitor{
		prevOutputHash: make([]byte, 0),
	}
}

// hash generates a SHA256 hash of the given content
func (m *StatusMonitor) hash(content string) []byte {
	h := sha256.Sum256([]byte(content))
	return h[:]
}

// HasUpdated checks if the content has changed since the last check
func (m *StatusMonitor) HasUpdated(content string) bool {
	currentHash := m.hash(content)
	hasUpdated := !bytesEqual(currentHash, m.prevOutputHash)
	if hasUpdated {
		m.prevOutputHash = currentHash
	}
	// Debug: Log every 50th check to avoid spam
	// debug.DebugLog("[Monitor] HasUpdated check: updated=%v, content_len=%d", hasUpdated, len(content))
	return hasUpdated
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
