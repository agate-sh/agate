package session

import (
	"fmt"
	"os/exec"
	"sort"

	"agate/internal/debug"
	"agate/pkg/agents"
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
			// Convert agents to persisted format
			persistedAgents := make([]config.PersistedAgentInstance, 0, len(session.Agents))
			for _, agent := range session.Agents {
				persistedAgent := config.PersistedAgentInstance{
					ID:           agent.ID,
					AgentName:    agent.AgentConfig.Name,
					TmuxName:     "", // Empty - will use session's shared tmux name
					CreatedAt:    agent.CreatedAt,
					LastAccessed: agent.LastAccessed,
					PaneIndex:    agent.PaneIndex,
				}

				if agent.Worktree != nil {
					persistedAgent.WorktreePath = agent.Worktree.Path
					persistedAgent.Branch = agent.Worktree.Branch
					persistedAgent.RepoName = agent.Worktree.RepoName
				}

				persistedAgents = append(persistedAgents, persistedAgent)
			}

			// Sort agents for consistent ordering
			sort.Slice(persistedAgents, func(i, j int) bool {
				return persistedAgents[i].AgentName < persistedAgents[j].AgentName
			})

			// Get shared tmux session name
			sharedTmuxName := ""
			if session.SharedTmux != nil {
				sharedTmuxName = session.SharedTmux.GetSessionName()
			}

			persistedSession := config.PersistedSession{
				ID:                session.ID,
				Prompt:            session.Prompt,
				Description:       session.Description,
				BranchBaseName:    session.BranchBaseName,
				ActiveAgentIndex:  session.ActiveAgentIndex,
				Agents:            persistedAgents,
				SharedTmuxName:    sharedTmuxName,
				CreatedAt:         session.CreatedAt,
				LastAccessed:      session.LastAccessed,
			}

			s.SessionMappings[sessionID] = persistedSession

			debug.DebugLog("PersistSessions: Persisted session: %s with %d agents",
				sessionID, len(persistedAgents))
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
		debug.DebugLog("LoadSessions: Successfully restored session: %s with %d agents",
			sessionID, len(session.Agents))
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


	debug.DebugLog("LoadSessions: Session restoration complete")
	return nil
}

// restoreSessionFromPersisted recreates a session object from persisted data
func (m *Manager) restoreSessionFromPersisted(persistedSession config.PersistedSession) (*Session, error) {
	session := &Session{
		ID:               persistedSession.ID,
		Prompt:           persistedSession.Prompt,
		Description:      persistedSession.Description,
		BranchBaseName:   persistedSession.BranchBaseName,
		Agents:           make(map[string]*Agent),
		ActiveAgentIndex: persistedSession.ActiveAgentIndex,
		CreatedAt:        persistedSession.CreatedAt,
		LastAccessed:     persistedSession.LastAccessed,
	}

	// Check if the shared tmux session still exists
	if persistedSession.SharedTmuxName != "" {
		exists, err := m.checkTmuxSessionExists(persistedSession.SharedTmuxName)
		if err != nil {
			return nil, fmt.Errorf("error checking tmux session: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("shared tmux session %s does not exist", persistedSession.SharedTmuxName)
		}

		// Restore the shared tmux session
		tmuxSession := tmux.NewTmuxSession(persistedSession.SharedTmuxName)
		if err := tmuxSession.Restore(); err != nil {
			return nil, fmt.Errorf("failed to restore shared tmux session: %w", err)
		}
		session.SharedTmux = tmuxSession
	}

	// Restore each agent
	for _, persistedAgent := range persistedSession.Agents {
		agent, err := m.restoreAgentFromPersisted(persistedAgent)
		if err != nil {
			debug.DebugLog("Failed to restore agent %s: %v", persistedAgent.ID, err)
			// Clean up any agents we've already restored and the tmux session
			if session.SharedTmux != nil {
				session.SharedTmux.Kill()
			}
			for _, a := range session.Agents {
				m.cleanupAgent(a)
			}
			return nil, fmt.Errorf("failed to restore agent %s: %w", persistedAgent.ID, err)
		}

		session.Agents[persistedAgent.AgentName] = agent
	}

	return session, nil
}

// restoreAgentFromPersisted recreates an agent from persisted data
func (m *Manager) restoreAgentFromPersisted(persistedAgent config.PersistedAgentInstance) (*Agent, error) {
	// Get agent configuration
	agentConfig := agents.GetAgentConfig(persistedAgent.AgentName)

	// Recreate worktree info
	worktree := &git.WorktreeInfo{
		Name:     persistedAgent.Branch,
		Path:     persistedAgent.WorktreePath,
		Branch:   persistedAgent.Branch,
		RepoName: persistedAgent.RepoName,
	}

	// Recreate agent object (no individual tmux session)
	agent := &Agent{
		ID:           persistedAgent.ID,
		AgentConfig:  agentConfig,
		SessionID:    "", // Will be set by caller
		Worktree:     worktree,
		PaneIndex:    persistedAgent.PaneIndex,
		CreatedAt:    persistedAgent.CreatedAt,
		LastAccessed: persistedAgent.LastAccessed,
	}

	return agent, nil
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
