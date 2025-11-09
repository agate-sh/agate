package session

import (
	"fmt"
	"os/exec"
	"sort"

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

		for sessionID, session := range m.sessions {
			// Convert instances to persisted format
			persistedInstances := make([]config.PersistedAgentInstance, 0, len(session.Instances))
			for _, instance := range session.Instances {
				persistedInstance := config.PersistedAgentInstance{
					ID:           instance.ID,
					AgentName:    instance.AgentConfig.Name,
					TmuxName:     instance.GetTmuxSessionName(),
					CreatedAt:    instance.CreatedAt,
					LastAccessed: instance.LastAccessed,
				}

				if instance.Worktree != nil {
					persistedInstance.WorktreePath = instance.Worktree.Path
					persistedInstance.Branch = instance.Worktree.Branch
					persistedInstance.RepoName = instance.Worktree.RepoName
				}

				persistedInstances = append(persistedInstances, persistedInstance)
			}

			// Sort instances for consistent ordering
			sort.Slice(persistedInstances, func(i, j int) bool {
				return persistedInstances[i].AgentName < persistedInstances[j].AgentName
			})

			persistedSession := config.PersistedSession{
				ID:                  session.ID,
				Prompt:              session.Prompt,
				Description:         session.Description,
				BranchBaseName:      session.BranchBaseName,
				ActiveInstanceIndex: session.ActiveInstanceIndex,
				Instances:           persistedInstances,
				CreatedAt:           session.CreatedAt,
				LastAccessed:        session.LastAccessed,
			}

			s.SessionMappings[sessionID] = persistedSession

			debug.DebugLog("PersistSessions: Persisted session: %s with %d instances",
				sessionID, len(persistedInstances))
		}

		// Update active session
		if m.activeSession != nil {
			s.ActiveSession = m.activeSession.ID
			debug.DebugLog("PersistSessions: Set active session to: %s", m.activeSession.ID)
		} else {
			s.ActiveSession = ""
			debug.DebugLog("PersistSessions: No active session to persist")
		}

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

	for sessionID, persistedSession := range sessionMappings {
		debug.DebugLog("LoadSessions: Attempting to restore session: %s", sessionID)

		// Restore the session
		session, err := m.restoreSessionFromPersisted(persistedSession)
		if err != nil {
			debug.DebugLog("LoadSessions: Failed to restore session %s: %v", sessionID, err)
			orphanedKeys = append(orphanedKeys, sessionID)
			continue
		}

		// Store in sessions map
		m.sessions[sessionID] = session
		restoredCount++
		debug.DebugLog("LoadSessions: Successfully restored session: %s with %d instances",
			sessionID, len(session.Instances))
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
	activeSessionID := m.stateMgr.GetActiveSession()
	if activeSessionID != "" {
		if session, exists := m.sessions[activeSessionID]; exists {
			m.activeSession = session
			debug.DebugLog("LoadSessions: Restored active session: %s", session.ID)
		} else {
			debug.DebugLog("LoadSessions: Active session ID %s not found in restored sessions", activeSessionID)
		}
	}

	debug.DebugLog("LoadSessions: Session restoration complete")
	return nil
}

// restoreSessionFromPersisted recreates a session object from persisted data
func (m *Manager) restoreSessionFromPersisted(persistedSession config.PersistedSession) (*Session, error) {
	session := &Session{
		ID:                  persistedSession.ID,
		Prompt:              persistedSession.Prompt,
		Description:         persistedSession.Description,
		BranchBaseName:      persistedSession.BranchBaseName,
		Instances:           make(map[string]*AgentInstance),
		ActiveInstanceIndex: persistedSession.ActiveInstanceIndex,
		CreatedAt:           persistedSession.CreatedAt,
		LastAccessed:        persistedSession.LastAccessed,
	}

	// Restore each agent instance
	for _, persistedInstance := range persistedSession.Instances {
		instance, err := m.restoreAgentInstanceFromPersisted(persistedInstance)
		if err != nil {
			debug.DebugLog("Failed to restore instance %s: %v", persistedInstance.ID, err)
			// Clean up any instances we've already restored
			for _, inst := range session.Instances {
				m.cleanupAgentInstance(inst)
			}
			return nil, fmt.Errorf("failed to restore instance %s: %w", persistedInstance.ID, err)
		}

		session.Instances[persistedInstance.AgentName] = instance
	}

	return session, nil
}

// restoreAgentInstanceFromPersisted recreates an agent instance from persisted data
func (m *Manager) restoreAgentInstanceFromPersisted(persistedInstance config.PersistedAgentInstance) (*AgentInstance, error) {
	// Check if the tmux session still exists
	exists, err := m.checkTmuxSessionExists(persistedInstance.TmuxName)
	if err != nil {
		return nil, fmt.Errorf("error checking tmux session: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("tmux session %s does not exist", persistedInstance.TmuxName)
	}

	// Get agent configuration
	agentConfig := app.GetAgentConfig(persistedInstance.AgentName)

	// Recreate worktree info
	worktree := &git.WorktreeInfo{
		Name:     persistedInstance.Branch,
		Path:     persistedInstance.WorktreePath,
		Branch:   persistedInstance.Branch,
		RepoName: persistedInstance.RepoName,
	}

	// Create tmux session object (connecting to existing session)
	tmuxSession := tmux.NewTmuxSession(persistedInstance.TmuxName, persistedInstance.AgentName)
	err = tmuxSession.Restore()
	if err != nil {
		return nil, fmt.Errorf("failed to restore tmux session: %w", err)
	}

	// Recreate instance object
	instance := &AgentInstance{
		ID:           persistedInstance.ID,
		AgentConfig:  agentConfig,
		SessionID:    "", // Will be set by caller
		TmuxSession:  tmuxSession,
		Worktree:     worktree,
		CreatedAt:    persistedInstance.CreatedAt,
		LastAccessed: persistedInstance.LastAccessed,
	}

	return instance, nil
}

// checkTmuxSessionExists checks if a tmux session with the given name exists
func (m *Manager) checkTmuxSessionExists(sessionName string) (bool, error) {
	debug.DebugLog("checkTmuxSessionExists: Checking session: %s", sessionName)

	// Run tmux has-session directly with the exact name
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
