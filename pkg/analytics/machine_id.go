package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agate/pkg/config"
)

// GetMachineID returns a stable machine identifier.
// It first checks for an existing ID in ~/.agate/user_id.
// If not found, generates a new random UUID and saves it.
func GetMachineID() (string, error) {
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
