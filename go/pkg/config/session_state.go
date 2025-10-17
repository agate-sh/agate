package config

import (
	"time"
)

// SessionState manages the persistent state of sessions
type SessionState struct {
	SessionMappings map[string]PersistedSession `json:"session_mappings"` // WorktreeKey -> PersistedSession
	ActiveSession   string                      `json:"active_session"`   // Currently active session key
	DefaultAgent    string                      `json:"default_agent"`    // Default agent for new sessions
}

// PersistedSession represents a session's persistent data
type PersistedSession struct {
	ID           string    `json:"id"`
	WorktreeKey  string    `json:"worktree_key"`
	TmuxName     string    `json:"tmux_name"`     // Tmux session name
	AgentName    string    `json:"agent_name"`    // Agent used for this session
	WorktreePath string    `json:"worktree_path"` // Path to worktree
	Branch       string    `json:"branch"`        // Branch name
	RepoName     string    `json:"repo_name"`     // Repository name
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// GetSessionMappings returns a copy of the stored session mappings
func GetSessionMappings() (map[string]PersistedSession, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}

	result := make(map[string]PersistedSession, len(state.Sessions.SessionMappings))
	for key, value := range state.Sessions.SessionMappings {
		result[key] = value
	}
	return result, nil
}

// SaveSessionMapping persists a session mapping
func SaveSessionMapping(worktreeKey string, session PersistedSession) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Sessions.SessionMappings == nil {
		state.Sessions.SessionMappings = make(map[string]PersistedSession)
	}

	state.Sessions.SessionMappings[worktreeKey] = session
	return SaveState(state)
}

// RemoveSessionMapping removes a session mapping
func RemoveSessionMapping(worktreeKey string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Sessions.SessionMappings != nil {
		delete(state.Sessions.SessionMappings, worktreeKey)
	}

	return SaveState(state)
}

// GetActiveSession returns the active session key
func GetActiveSession() (string, error) {
	state, err := LoadState()
	if err != nil {
		return "", err
	}
	return state.Sessions.ActiveSession, nil
}

// SetActiveSession sets the active session key
func SetActiveSession(sessionKey string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.Sessions.ActiveSession = sessionKey
	return SaveState(state)
}

// GetDefaultAgent returns the default agent for new sessions
func GetDefaultAgent() (string, error) {
	state, err := LoadState()
	if err != nil {
		return "", err
	}
	return state.Sessions.DefaultAgent, nil
}

// SetDefaultAgent sets the default agent for new sessions
func SetDefaultAgent(agent string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.Sessions.DefaultAgent = agent
	return SaveState(state)
}
