package config

import (
	"time"
)

// SessionState manages the persistent state of multi-agent sessions
type SessionState struct {
	SessionMappings map[string]PersistedSession `json:"session_mappings"` // SessionID -> PersistedSession
	ActiveSession   string                      `json:"active_session"`   // Currently active session ID
	PinnedSessions  []string                    `json:"pinned_sessions"`  // Ordered list of pinned session IDs (max 4)
	DefaultAgent    string                      `json:"default_agent"`    // Default agent for new sessions
}

// PersistedSession represents a session's persistent data
type PersistedSession struct {
	ID                  string                      `json:"id"`
	Prompt              string                      `json:"prompt"`
	Description         string                      `json:"description"` // Can be empty
	BranchBaseName      string                      `json:"branch_base_name"`
	ActiveInstanceIndex int                         `json:"active_instance_index"`
	Instances           []PersistedAgentInstance    `json:"instances"`
	CreatedAt           time.Time                   `json:"created_at"`
	LastAccessed        time.Time                   `json:"last_accessed"`
}

// PersistedAgentInstance represents an agent instance's persistent data
type PersistedAgentInstance struct {
	ID           string    `json:"id"`
	AgentName    string    `json:"agent_name"`
	TmuxName     string    `json:"tmux_name"`
	WorktreePath string    `json:"worktree_path"`
	Branch       string    `json:"branch"`
	RepoName     string    `json:"repo_name"`
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
func SaveSessionMapping(sessionID string, session PersistedSession) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Sessions.SessionMappings == nil {
		state.Sessions.SessionMappings = make(map[string]PersistedSession)
	}

	state.Sessions.SessionMappings[sessionID] = session
	return SaveState(state)
}

// RemoveSessionMapping removes a session mapping
func RemoveSessionMapping(sessionID string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	if state.Sessions.SessionMappings != nil {
		delete(state.Sessions.SessionMappings, sessionID)
	}

	return SaveState(state)
}

// GetActiveSession returns the active session ID
func GetActiveSession() (string, error) {
	state, err := LoadState()
	if err != nil {
		return "", err
	}
	return state.Sessions.ActiveSession, nil
}

// SetActiveSession sets the active session ID
func SetActiveSession(sessionID string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.Sessions.ActiveSession = sessionID
	return SaveState(state)
}

// GetPinnedSessions returns the list of pinned session IDs
func GetPinnedSessions() ([]string, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}
	if state.Sessions.PinnedSessions == nil {
		return []string{}, nil
	}
	return state.Sessions.PinnedSessions, nil
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
