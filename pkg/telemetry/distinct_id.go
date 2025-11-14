package telemetry

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agate/pkg/config"
)

// GetDistinctId returns a stable user identifier for analytics.
// Priority order:
// 1. Hashed git email (if configured) - persists across ~/.agate deletions
// 2. Existing ID from ~/.agate/user_id
// 3. New random UUID saved to ~/.agate/user_id
func GetDistinctId() (string, error) {
	// Try git email first (most persistent)
	if gitEmail := getGitEmail(); gitEmail != "" {
		return hashEmail(gitEmail), nil
	}

	// Fall back to file-based ID
	agateDir, err := config.GetAgateDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}
	userIDPath := filepath.Join(agateDir, "user_id")

	// Try to read existing ID
	if data, err := os.ReadFile(userIDPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	}

	// Generate new random UUID
	id, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Ensure directory exists
	if err := config.EnsureAgateDir(); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save the ID
	if err := os.WriteFile(userIDPath, []byte(id), 0600); err != nil {
		return "", fmt.Errorf("failed to save user ID: %w", err)
	}

	return id, nil
}

// getGitEmail returns the user's git email if configured
func getGitEmail() string {
	cmd := exec.Command("git", "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	email := strings.TrimSpace(string(output))
	return email
}

// hashEmail creates a SHA256 hash of an email for privacy
func hashEmail(email string) string {
	hash := sha256.Sum256([]byte(email))
	return hex.EncodeToString(hash[:])
}

// generateUUID generates a random UUID v4
func generateUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Set version (4) and variant bits
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return hex.EncodeToString(bytes), nil
}
