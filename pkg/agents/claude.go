package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeSettings represents the structure of Claude Code's settings.local.json
type ClaudeSettings struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
		Ask   []string `json:"ask"`
	} `json:"permissions"`
	FolderPermissions struct {
		TrustedFolders []string `json:"trustedFolders"`
	} `json:"folderPermissions"`
	OutputStyle string `json:"outputStyle"`
}


// setupClaudeWorktree creates the Claude Code configuration directory and settings file
func setupClaudeWorktree(worktreePath string) error {
	claudeDir := filepath.Join(worktreePath, ".claude")

	// Create .claude directory if it doesn't exist
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Create settings with folder trust pre-configured
	settings := ClaudeSettings{
		OutputStyle: "default",
	}

	// Pre-trust this worktree directory
	settings.FolderPermissions.TrustedFolders = []string{worktreePath}

	// Initialize empty permission arrays
	settings.Permissions.Allow = []string{}
	settings.Permissions.Deny = []string{}
	settings.Permissions.Ask = []string{}

	// Write settings file
	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}
