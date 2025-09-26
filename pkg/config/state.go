// Package config provides configuration management and state persistence
// for the agate application.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const currentStateVersion = 1

// AppState represents the application's persistent state.
type AppState struct {
	Version   int            `json:"version"`
	UI        UIState        `json:"ui"`
	Workspace WorkspaceState `json:"workspace"`
}

func defaultAppState() AppState {
	return AppState{
		Version: currentStateVersion,
		UI: UIState{
			Welcome: WelcomeState{Shown: false},
		},
		Workspace: WorkspaceState{
			Repositories:   []string{},
			RepoSelections: map[string]RepoSelection{},
		},
	}
}

func (s *AppState) normalize() {
	if s == nil {
		return
	}
	if s.Version == 0 {
		s.Version = currentStateVersion
	}
	if s.Workspace.Repositories == nil {
		s.Workspace.Repositories = []string{}
	}
	if s.Workspace.RepoSelections == nil {
		s.Workspace.RepoSelections = map[string]RepoSelection{}
	}
}

// GetAgateDir returns the path to the .agate directory
func GetAgateDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".agate"), nil
}

// EnsureAgateDir creates the .agate directory if it doesn't exist
func EnsureAgateDir() error {
	agateDir, err := GetAgateDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(agateDir, 0755)
}

// getStateFilePath returns the path to the state.json file
func getStateFilePath() (string, error) {
	agateDir, err := GetAgateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(agateDir, "state.json"), nil
}

// LoadState loads the application state from disk
func LoadState() (*AppState, error) {
	stateFile, err := getStateFilePath()
	if err != nil {
		return nil, err
	}

	state := defaultAppState()

	// If file doesn't exist, return default state
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return &state, nil
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		state.normalize()
		return &state, nil
	}

	if err := json.Unmarshal(data, &state); err != nil {
		state.normalize()
		return &state, nil
	}

	state.normalize()
	return &state, nil
}

// SaveState saves the application state to disk
func SaveState(state *AppState) error {
	if state == nil {
		return errors.New("state cannot be nil")
	}

	if err := EnsureAgateDir(); err != nil {
		return err
	}

	state.normalize()

	stateFile, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}
