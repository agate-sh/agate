package session

import (
	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/config"
	"agate/pkg/git"
	"agate/pkg/tmux"
)


// PersistSessions saves all sessions to config
func (m *Manager) PersistSessions() error {
	for worktreeKey, session := range m.sessions {
		persistedSession := config.PersistedSession{
			ID:           session.ID,
			WorktreeKey:  session.WorktreeKey,
			TmuxName:     session.GetTmuxSessionName(),
			AgentName:    session.Agent.Name,
			CreatedAt:    session.CreatedAt,
			LastAccessed: session.LastAccessed,
		}

		if session.Worktree != nil {
			persistedSession.WorktreePath = session.Worktree.Path
			persistedSession.Branch = session.Worktree.Branch
			persistedSession.RepoName = session.Worktree.RepoName
		}

		if err := config.SaveSessionMapping(worktreeKey, persistedSession); err != nil {
			debug.DebugLog("Failed to persist session %s: %v", session.ID, err)
			return err
		}

		debug.DebugLog("Persisted session: %s (tmux: %s, agent: %s)",
			persistedSession.ID, persistedSession.TmuxName, persistedSession.AgentName)
	}

	// Update active session
	if m.activeSession != nil {
		if err := config.SetActiveSession(m.activeSession.WorktreeKey); err != nil {
			debug.DebugLog("Failed to persist active session: %v", err)
		}
	}

	return nil
}

// LoadSessions restores sessions from config
func (m *Manager) LoadSessions() error {
	sessionMappings, err := config.GetSessionMappings()
	if err != nil {
		debug.DebugLog("Failed to load session mappings: %v", err)
		return err
	}

	debug.DebugLog("Loading %d persisted sessions", len(sessionMappings))

	for worktreeKey, persistedSession := range sessionMappings {
		// Check if the tmux session still exists
		exists, err := m.checkTmuxSessionExists(persistedSession.TmuxName)
		if err != nil || !exists {
			debug.DebugLog("Tmux session %s no longer exists, removing mapping", persistedSession.TmuxName)
			config.RemoveSessionMapping(worktreeKey)
			continue
		}

		// Recreate the session object (without creating a new tmux session)
		session, err := m.restoreSessionFromPersisted(persistedSession)
		if err != nil {
			debug.DebugLog("Failed to restore session %s: %v", persistedSession.ID, err)
			continue
		}

		// Store in sessions map
		m.sessions[worktreeKey] = session
		debug.DebugLog("Restored session: %s (tmux: %s, agent: %s)",
			session.ID, session.GetTmuxSessionName(), session.Agent.Name)
	}

	// Restore active session
	activeSessionKey, err := config.GetActiveSession()
	if err == nil && activeSessionKey != "" {
		if session, exists := m.sessions[activeSessionKey]; exists {
			m.activeSession = session
			debug.DebugLog("Restored active session: %s", session.ID)
		}
	}

	return nil
}

// checkTmuxSessionExists checks if a tmux session with the given name exists
func (m *Manager) checkTmuxSessionExists(sessionName string) (bool, error) {
	// Create a temporary tmux session object to check existence
	tempSession := tmux.NewTmuxSession(sessionName, "dummy")
	return tempSession.SessionExists()
}

// restoreSessionFromPersisted recreates a session object from persisted data
func (m *Manager) restoreSessionFromPersisted(persistedSession config.PersistedSession) (*Session, error) {
	// Get agent configuration
	agentConfig := app.GetAgentConfig(persistedSession.AgentName)

	// Recreate worktree info
	worktree := &git.WorktreeInfo{
		Name:     persistedSession.Branch, // Use branch as name
		Path:     persistedSession.WorktreePath,
		Branch:   persistedSession.Branch,
		RepoName: persistedSession.RepoName,
	}

	// Create tmux session object (connecting to existing session)
	tmuxSession := tmux.NewTmuxSession(persistedSession.TmuxName, persistedSession.AgentName)
	err := tmuxSession.Restore() // Connect to existing session
	if err != nil {
		return nil, err
	}

	// Recreate session object
	session := &Session{
		ID:           persistedSession.ID,
		Name:         persistedSession.TmuxName,
		WorktreeKey:  persistedSession.WorktreeKey,
		TmuxSession:  tmuxSession,
		Worktree:     worktree,
		Agent:        agentConfig,
		CreatedAt:    persistedSession.CreatedAt,
		LastAccessed: persistedSession.LastAccessed,
		IsActive:     false, // Will be set during activation
	}

	return session, nil
}

