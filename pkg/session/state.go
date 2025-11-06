package session

import (
	"fmt"
	"os/exec"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/config"
	"agate/pkg/git"
	"agate/pkg/tmux"
)

// PersistSessions saves all sessions to config atomically
func (m *Manager) PersistSessions() error {
	if m.stateMgr == nil {
		debug.DebugLog("PersistSessions: StateManager is nil, cannot persist sessions")
		return fmt.Errorf("state manager is nil")
	}

	debug.DebugLog("PersistSessions: Persisting %d sessions", len(m.sessions))

	// Batch all session updates into a single atomic operation
	return m.stateMgr.UpdateSessions(func(s *config.SessionState) error {
		// Clear existing mappings and rebuild from current sessions
		s.SessionMappings = make(map[string]config.PersistedSession)

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

			s.SessionMappings[worktreeKey] = persistedSession

			debug.DebugLog("PersistSessions: Persisted session: %s (tmux: %s, agent: %s)",
				persistedSession.ID, persistedSession.TmuxName, persistedSession.AgentName)
		}

		// Update active session
		if m.activeSession != nil {
			s.ActiveSession = m.activeSession.WorktreeKey
			debug.DebugLog("PersistSessions: Set active session to: %s", m.activeSession.WorktreeKey)
		} else {
			s.ActiveSession = ""
			debug.DebugLog("PersistSessions: No active session to persist")
		}

		// Update pinned sessions
		s.PinnedSessions = m.pinnedSessions
		debug.DebugLog("PersistSessions: Persisted %d pinned sessions", len(m.pinnedSessions))

		debug.DebugLog("PersistSessions: Successfully persisted %d sessions", len(s.SessionMappings))
		return nil
	})
}

// LoadSessions restores sessions from config
func (m *Manager) LoadSessions() error {
	debug.DebugLog("LoadSessions: Starting session restoration")

	// Check if StateManager is available
	if m.stateMgr == nil {
		debug.DebugLog("LoadSessions: StateManager is nil, cannot load sessions")
		return fmt.Errorf("state manager is nil")
	}

	sessionMappings := m.stateMgr.GetSessionMappings()
	debug.DebugLog("LoadSessions: Retrieved %d session mappings from state", len(sessionMappings))

	if len(sessionMappings) == 0 {
		debug.DebugLog("LoadSessions: No persisted sessions to restore")
		return nil
	}

	orphanedKeys := []string{}
	restoredCount := 0

	for worktreeKey, persistedSession := range sessionMappings {
		debug.DebugLog("LoadSessions: Attempting to restore session for key: %s", worktreeKey)
		debug.DebugLog("LoadSessions: Checking if tmux session exists: %s", persistedSession.TmuxName)

		// Check if the tmux session still exists
		exists, err := m.checkTmuxSessionExists(persistedSession.TmuxName)
		if err != nil {
			debug.DebugLog("LoadSessions: Error checking tmux session %s: %v", persistedSession.TmuxName, err)
			orphanedKeys = append(orphanedKeys, worktreeKey)
			continue
		}
		if !exists {
			debug.DebugLog("LoadSessions: Tmux session %s does not exist, marking as orphaned", persistedSession.TmuxName)
			orphanedKeys = append(orphanedKeys, worktreeKey)
			continue
		}
		debug.DebugLog("LoadSessions: Tmux session %s exists! Proceeding to restore", persistedSession.TmuxName)

		// Recreate the session object (without creating a new tmux session)
		session, err := m.restoreSessionFromPersisted(persistedSession)
		if err != nil {
			debug.DebugLog("LoadSessions: Failed to restore session %s: %v", persistedSession.ID, err)
			continue
		}

		// Store in sessions map
		m.sessions[worktreeKey] = session
		restoredCount++
		debug.DebugLog("LoadSessions: Successfully restored session: %s (tmux: %s, agent: %s)",
			session.ID, session.GetTmuxSessionName(), session.Agent.Name)
	}

	debug.DebugLog("LoadSessions: Restored %d sessions, found %d orphaned", restoredCount, len(orphanedKeys))

	// Remove orphaned sessions atomically
	if len(orphanedKeys) > 0 {
		debug.DebugLog("LoadSessions: Removing %d orphaned session mappings", len(orphanedKeys))
		m.stateMgr.UpdateSessions(func(s *config.SessionState) error {
			for _, key := range orphanedKeys {
				delete(s.SessionMappings, key)
			}
			return nil
		})
	}

	// Restore active session
	activeSessionKey := m.stateMgr.GetActiveSession()
	if activeSessionKey != "" {
		if session, exists := m.sessions[activeSessionKey]; exists {
			m.activeSession = session
			debug.DebugLog("LoadSessions: Restored active session: %s", session.ID)
		} else {
			debug.DebugLog("LoadSessions: Active session key %s not found in restored sessions", activeSessionKey)
		}
	}

	// Restore pinned sessions
	pinnedSessionIDs := m.stateMgr.GetPinnedSessions()
	m.pinnedSessions = make([]string, 0, len(pinnedSessionIDs))
	for _, sessionID := range pinnedSessionIDs {
		// Verify the session still exists
		found := false
		for _, session := range m.sessions {
			if session.ID == sessionID {
				found = true
				break
			}
		}
		if found {
			m.pinnedSessions = append(m.pinnedSessions, sessionID)
			debug.DebugLog("LoadSessions: Restored pinned session: %s", sessionID)
		} else {
			debug.DebugLog("LoadSessions: Pinned session %s not found in restored sessions, skipping", sessionID)
		}
	}
	debug.DebugLog("LoadSessions: Restored %d pinned sessions (out of %d persisted)", len(m.pinnedSessions), len(pinnedSessionIDs))

	debug.DebugLog("LoadSessions: Session restoration complete")
	return nil
}

// checkTmuxSessionExists checks if a tmux session with the given name exists
func (m *Manager) checkTmuxSessionExists(sessionName string) (bool, error) {
	debug.DebugLog("checkTmuxSessionExists: Checking session: %s", sessionName)

	// Run tmux has-session directly with the exact name (don't use NewTmuxSession
	// which would transform the name)
	cmd := tmux.Command("has-session", "-t", sessionName)
	err := cmd.Run()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means session doesn't exist
			if exitErr.ExitCode() == 1 {
				debug.DebugLog("checkTmuxSessionExists: Session %s does not exist (exit code 1)", sessionName)
				return false, nil
			}
		}
		debug.DebugLog("checkTmuxSessionExists: Error checking session %s: %v", sessionName, err)
		return false, err
	}

	debug.DebugLog("checkTmuxSessionExists: Session %s exists!", sessionName)
	return true, nil
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

	// Get executable name from agent config
	executableName := agentConfig.ExecutableName
	if executableName == "" {
		executableName = persistedSession.AgentName
	}

	// Convert agent config env vars to tmux env vars
	envVars := make([]tmux.EnvVar, len(agentConfig.EnvVars))
	for i, envVar := range agentConfig.EnvVars {
		envVars[i] = tmux.EnvVar{
			Name:  envVar.Name,
			Value: envVar.Value,
		}
	}

	// Create tmux session object (connecting to existing session)
	// Note: env vars are passed but won't be used when restoring existing sessions
	tmuxSession := tmux.NewTmuxSession(persistedSession.TmuxName, executableName, envVars)
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
