package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppState represents the application's persistent state
type AppState struct {
	WelcomeShown bool     `json:"welcome_shown"`
	Repositories []string `json:"repositories"` // User-added repository paths
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

	// If file doesn't exist, return default state
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return &AppState{WelcomeShown: false}, nil
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		// If JSON is invalid, return default state
		return &AppState{WelcomeShown: false}, nil
	}

	return &state, nil
}

// SaveState saves the application state to disk
func SaveState(state *AppState) error {
	if err := EnsureAgateDir(); err != nil {
		return err
	}

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

// GetWelcomeShownState returns whether the welcome overlay has been shown
func GetWelcomeShownState() (bool, error) {
	state, err := LoadState()
	if err != nil {
		return false, err
	}
	return state.WelcomeShown, nil
}

// SetWelcomeShown sets the welcome shown state
func SetWelcomeShown(shown bool) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.WelcomeShown = shown
	return SaveState(state)
}

// GetRepositories returns the list of user-added repositories
func GetRepositories() ([]string, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}
	if state.Repositories == nil {
		return []string{}, nil
	}
	return state.Repositories, nil
}

// AddRepository adds a repository to the list if it's not already present
func AddRepository(repoPath string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Repositories == nil {
		state.Repositories = []string{}
	}

	// Check if repository is already in the list
	for _, existing := range state.Repositories {
		if existing == repoPath {
			return nil // Already exists, no need to add
		}
	}

	// Add the new repository
	state.Repositories = append(state.Repositories, repoPath)
	return SaveState(state)
}

// RemoveRepository removes a repository from the list
func RemoveRepository(repoPath string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Repositories == nil {
		return nil // Nothing to remove
	}

	// Filter out the repository to remove
	var filtered []string
	for _, existing := range state.Repositories {
		if existing != repoPath {
			filtered = append(filtered, existing)
		}
	}

	state.Repositories = filtered
	return SaveState(state)
}