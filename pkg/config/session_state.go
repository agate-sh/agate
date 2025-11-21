package config

import (
	"time"
)

// SessionState manages the persistent state of multi-agent sessions
type SessionState struct {
	SessionMappings map[string]PersistedSession `json:"session_mappings"` // SessionID -> PersistedSession
	PinnedSessions  []string                    `json:"pinned_sessions"`  // Ordered list of pinned session IDs (max 4)
	SelectedAgents  []string                    `json:"selected_agents"`  // Last selected agents for new sessions
}

// PersistedSession represents a session's persistent data
type PersistedSession struct {
	ID               string                   `json:"id"`
	Prompt           string                   `json:"prompt"`
	Description      string                   `json:"description"` // Can be empty
	BranchBaseName   string                   `json:"branch_base_name"`
	ActiveAgentIndex int                      `json:"active_agent_index"`
	Agents           []PersistedAgentInstance `json:"agents"`
	SharedTmuxName   string                   `json:"shared_tmux_name"` // Shared tmux session for all agents
	CreatedAt        time.Time                `json:"created_at"`
	LastAccessed     time.Time                `json:"last_accessed"`
}

// PersistedAgentInstance represents an agent's persistent data
type PersistedAgentInstance struct {
	ID           string    `json:"id"`
	AgentName    string    `json:"agent_name"`
	TmuxName     string    `json:"tmux_name"`      // Deprecated, kept for backward compat
	WorktreePath string    `json:"worktree_path"`
	Branch       string    `json:"branch"`
	RepoName     string    `json:"repo_name"`
	PaneIndex    int       `json:"pane_index"`     // Which pane in the shared tmux session
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

// GetSelectedAgents returns the selected agents for new sessions
func GetSelectedAgents() ([]string, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}
	return state.Sessions.SelectedAgents, nil
}

// SetSelectedAgents sets the selected agents for new sessions
func SetSelectedAgents(agents []string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	state.Sessions.SelectedAgents = agents
	return SaveState(state)
}
