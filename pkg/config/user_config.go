// Package config provides user configuration management for the agate application.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KeybindingsConfig represents keybinding overrides from user config
type KeybindingsConfig struct {
	// Truly global keys
	Quit         []string `yaml:"quit,omitempty"`
	Keybindings  []string `yaml:"keybindings,omitempty"`
	DebugOverlay []string `yaml:"debug_overlay,omitempty"`

	// Global navigation keys
	Up   []string `yaml:"up,omitempty"`
	Down []string `yaml:"down,omitempty"`

	// Pane navigation
	AgentsPane  []string `yaml:"agents_pane,omitempty"`
	SessionPane []string `yaml:"session_pane,omitempty"`
	ChangesPane []string `yaml:"changes_pane,omitempty"`

	// Agent cycling
	NextAgent       []string `yaml:"next_agent,omitempty"`
	PrevAgent       []string `yaml:"prev_agent,omitempty"`
	AddRemoveAgents []string `yaml:"add_remove_agents,omitempty"`

	// Repository and session management
	NewSession []string `yaml:"new_session,omitempty"`

	// Session interaction
	AttachAgent []string `yaml:"attach_agent,omitempty"`
	Commit      []string `yaml:"commit,omitempty"`
	DetachTmux  []string `yaml:"detach_tmux,omitempty"`

	// Dialog actions
	Confirm []string `yaml:"confirm,omitempty"`
	Cancel  []string `yaml:"cancel,omitempty"`

	// List interaction
	Filter      []string `yaml:"filter,omitempty"`
	ClearFilter []string `yaml:"clear_filter,omitempty"`

	// Git pane actions
	OpenInEditor []string `yaml:"open_in_editor,omitempty"`
}

// UserConfig represents the user configuration file structure
type UserConfig struct {
	Keybindings KeybindingsConfig `yaml:"keybindings,omitempty"`
}

// getConfigFilePath returns the path to the config.yaml file
func getConfigFilePath() (string, error) {
	agateDir, err := GetAgateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(agateDir, "config.yaml"), nil
}

// LoadUserConfig loads the user configuration from ~/.agate/config.yaml
// If the file doesn't exist, returns a default empty config.
// The returned config should be merged with defaults from keybindings.go
func LoadUserConfig() (*UserConfig, error) {
	configFile, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	config := &UserConfig{}

	// If file doesn't exist, return empty config (will use defaults)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return config, nil
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
