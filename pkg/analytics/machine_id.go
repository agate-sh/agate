package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"agate/pkg/config"
)

// GetMachineID returns a stable machine identifier.
// It first checks for an existing ID in ~/.agate/user_id.
// If not found, generates a new ID from hardware fingerprint and saves it.
func GetMachineID() (string, error) {
	agateDir, err := config.GetAgateDir()
	if err != nil {
		return "", fmt.Errorf("failed to get agate directory: %w", err)
	}
	userIDPath := filepath.Join(agateDir, "user_id")

	// Try to read existing ID
	if data, err := os.ReadFile(userIDPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	}

	// Generate new ID from hardware fingerprint
	fingerprint, err := getHardwareFingerprint()
	if err != nil {
		return "", fmt.Errorf("failed to generate hardware fingerprint: %w", err)
	}

	// Hash the fingerprint to create a stable ID
	hash := sha256.Sum256([]byte(fingerprint))
	id := hex.EncodeToString(hash[:])

	// Save the ID
	if err := os.MkdirAll(filepath.Dir(userIDPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(userIDPath, []byte(id), 0600); err != nil {
		return "", fmt.Errorf("failed to save user ID: %w", err)
	}

	return id, nil
}

// getHardwareFingerprint generates a fingerprint from hardware characteristics.
func getHardwareFingerprint() (string, error) {
	var parts []string

	switch runtime.GOOS {
	case "darwin":
		// Get hardware UUID on macOS
		if out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output(); err == nil {
			outStr := string(out)
			if idx := strings.Index(outStr, "IOPlatformUUID"); idx != -1 {
				line := outStr[idx:]
				if end := strings.Index(line, "\n"); end != -1 {
					line = line[:end]
				}
				// Extract UUID from format: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
				if idx := strings.Index(line, "\""); idx != -1 {
					if idx2 := strings.LastIndex(line, "\""); idx2 > idx {
						uuid := line[idx+1 : idx2]
						parts = append(parts, uuid)
					}
				}
			}
		}

	case "linux":
		// Try multiple sources on Linux
		sources := []string{
			"/etc/machine-id",
			"/var/lib/dbus/machine-id",
			"/sys/class/dmi/id/product_uuid",
		}
		for _, path := range sources {
			if data, err := os.ReadFile(path); err == nil {
				parts = append(parts, strings.TrimSpace(string(data)))
				break
			}
		}

	case "windows":
		// Get machine GUID on Windows
		if out, err := exec.Command("wmic", "csproduct", "get", "UUID").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				uuid := strings.TrimSpace(lines[1])
				if uuid != "" {
					parts = append(parts, uuid)
				}
			}
		}
	}

	// Fallback: use hostname if no hardware ID found
	if len(parts) == 0 {
		hostname, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("failed to get hostname: %w", err)
		}
		parts = append(parts, hostname)
	}

	// Add username for additional uniqueness on shared machines
	if username := os.Getenv("USER"); username != "" {
		parts = append(parts, username)
	} else if username := os.Getenv("USERNAME"); username != "" {
		parts = append(parts, username)
	}

	return strings.Join(parts, "-"), nil
}
