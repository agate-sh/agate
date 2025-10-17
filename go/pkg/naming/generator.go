package naming

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
)

// Generator handles all tmux session name transformations.
// This is the single source of truth for name generation.
type Generator struct{}

// NewGenerator creates a new name generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateFromWorktree creates a tmux session name from worktree components.
// This is the primary entry point for creating session names.
func (g *Generator) GenerateFromWorktree(repoName, branch, agentName string) string {
	// Compose the base name from components
	baseName := fmt.Sprintf("%s_%s_%s", repoName, branch, agentName)

	// Sanitize the composed name
	return g.Sanitize(baseName)
}

// Sanitize transforms a raw string into a valid tmux session name.
// This function adds the "agate_" prefix and hash suffix.
//
// IMPORTANT: This is NOT idempotent - calling it twice will produce different results.
// Use NormalizeName() if you need idempotency.
func (g *Generator) Sanitize(raw string) string {
	original := strings.TrimSpace(raw)
	if original == "" {
		original = "default"
	}

	// Replace unsupported characters with underscores to keep tmux happy
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-' || r == '.':
			return r
		default:
			return '_'
		}
	}, original)

	sanitized = strings.Trim(sanitized, "_-.")
	if sanitized == "" {
		sanitized = "session"
	}

	const maxSanitizedLen = 80
	if len(sanitized) > maxSanitizedLen {
		sanitized = sanitized[:maxSanitizedLen]
	}

	// Add hash for uniqueness
	hash := sha1.Sum([]byte(original))
	hashSuffix := fmt.Sprintf("%x", hash[:4])

	return fmt.Sprintf("agate_%s_%s", sanitized, hashSuffix)
}

// IsAlreadySanitized checks if a name has already been transformed by Sanitize().
// A sanitized name has the pattern: "agate_*_[0-9a-f]{8}"
func (g *Generator) IsAlreadySanitized(name string) bool {
	// Check for the "agate_" prefix
	if !strings.HasPrefix(name, "agate_") {
		return false
	}

	// Check for the hash suffix (8 hex characters at the end)
	pattern := regexp.MustCompile(`^agate_.+_[0-9a-f]{8}$`)
	return pattern.MatchString(name)
}

// NormalizeName ensures idempotency: only sanitize if not already sanitized.
// This is useful when you're not sure if a name has already been processed.
//
// Example:
//   NormalizeName("foo") -> "agate_foo_a1b2c3d4"
//   NormalizeName("agate_foo_a1b2c3d4") -> "agate_foo_a1b2c3d4" (unchanged)
func (g *Generator) NormalizeName(name string) string {
	if g.IsAlreadySanitized(name) {
		return name // Already transformed, return as-is
	}
	return g.Sanitize(name)
}

// GenerateShellSessionName creates a shell session name based on the agent session name.
// Shell sessions are prefixed with "shell_" to distinguish them from agent sessions.
func (g *Generator) GenerateShellSessionName(agentSessionName string) string {
	// Compose the name with shell prefix
	composedName := "shell_" + agentSessionName

	// Sanitize the composed name
	return g.Sanitize(composedName)
}
