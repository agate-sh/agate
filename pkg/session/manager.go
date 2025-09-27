package session

import (
	"fmt"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/tmux"
)

// Manager is a singleton that manages all sessions
type Manager struct {
	sessions      map[string]*Session   // WorktreeKey -> Session
	activeSession *Session              // Currently active session
	worktreeMgr   *git.WorktreeManager  // Git worktree management
}

// NewManager creates a new session manager
func NewManager(worktreeMgr *git.WorktreeManager) *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		worktreeMgr: worktreeMgr,
	}
}

// CreateSession creates a new session for the given worktree and agent
func (m *Manager) CreateSession(worktree *git.WorktreeInfo, agentName string) (*Session, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree cannot be nil")
	}

	// Get agent configuration for this session
	agentConfig := app.GetAgentConfig(agentName)

	// Generate stable identifiers
	worktreeKey := generateWorktreeKey(worktree)
	sessionName := generateTmuxSessionName(worktree, agentName)

	// Check if session already exists
	if existing, exists := m.sessions[worktreeKey]; exists {
		debug.DebugLog("Session already exists for worktree key: %s", worktreeKey)
		return existing, nil
	}

	// Create tmux session
	tmuxSession := tmux.NewTmuxSession(sessionName, agentName)
	err := tmuxSession.Start(worktree.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to start tmux session: %w", err)
	}

	// Create session
	session := &Session{
		ID:           worktreeKey + "_" + agentName,
		Name:         sessionName,
		WorktreeKey:  worktreeKey,
		TmuxSession:  tmuxSession,
		Worktree:     worktree,
		Agent:        agentConfig,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		IsActive:     false,
	}

	// Store session
	m.sessions[worktreeKey] = session

	// Persist session to config
	if err := m.PersistSessions(); err != nil {
		debug.DebugLog("Failed to persist session %s: %v", session.ID, err)
		// Don't fail session creation if persistence fails
	}

	debug.DebugLog("Created new session: %s for worktree: %s with agent: %s",
		session.ID, worktree.Path, agentName)

	return session, nil
}

// GetOrCreateSession returns existing session or creates a new one
func (m *Manager) GetOrCreateSession(worktree *git.WorktreeInfo, agentName string) (*Session, error) {
	if worktree == nil {
		return nil, fmt.Errorf("worktree cannot be nil")
	}

	worktreeKey := generateWorktreeKey(worktree)

	// Check if session exists
	if session, exists := m.sessions[worktreeKey]; exists {
		// Update access time
		session.Update()
		debug.DebugLog("Reusing existing session: %s", session.ID)
		return session, nil
	}

	// Create new session
	return m.CreateSession(worktree, agentName)
}

// SwitchToSession activates the specified session
func (m *Manager) SwitchToSession(worktreeKey string) (*Session, error) {
	session, exists := m.sessions[worktreeKey]
	if !exists {
		return nil, fmt.Errorf("session not found for worktree key: %s", worktreeKey)
	}

	// Deactivate current session
	if m.activeSession != nil {
		m.activeSession.Deactivate()
	}

	// Activate new session
	session.Update()
	m.activeSession = session

	// Persist active session change
	if err := m.PersistSessions(); err != nil {
		debug.DebugLog("Failed to persist active session change: %v", err)
		// Don't fail session switch if persistence fails
	}

	debug.DebugLog("Switched to session: %s", session.ID)
	return session, nil
}

// GetActiveSession returns the currently active session
func (m *Manager) GetActiveSession() *Session {
	return m.activeSession
}

// GetSessionForWorktree returns the session associated with the given worktree
func (m *Manager) GetSessionForWorktree(worktree *git.WorktreeInfo) *Session {
	if worktree == nil {
		return nil
	}

	worktreeKey := generateWorktreeKey(worktree)
	return m.sessions[worktreeKey]
}

// DeleteSession removes and cleans up a session
func (m *Manager) DeleteSession(worktreeKey string) error {
	session, exists := m.sessions[worktreeKey]
	if !exists {
		return fmt.Errorf("session not found for worktree key: %s", worktreeKey)
	}

	debug.DebugLog("Deleting session: %s", session.ID)

	// Kill tmux session
	if session.TmuxSession != nil {
		if err := session.TmuxSession.Kill(); err != nil {
			debug.DebugLog("Failed to kill tmux session: %v", err)
			// Continue with deletion even if tmux kill fails
		}
	}

	// Delete worktree if we have a worktree manager
	if m.worktreeMgr != nil && session.Worktree != nil {
		if err := m.worktreeMgr.DeleteWorktree(*session.Worktree); err != nil {
			debug.DebugLog("Failed to delete worktree %s: %v", session.Worktree.Path, err)
			// Continue with session cleanup even if worktree deletion fails
		} else {
			debug.DebugLog("Successfully deleted worktree: %s", session.Worktree.Path)
		}
	}

	// Remove from sessions map
	delete(m.sessions, worktreeKey)

	// If this was the active session, clear it
	if m.activeSession == session {
		m.activeSession = nil
	}

	// Persist changes to config
	if err := m.PersistSessions(); err != nil {
		debug.DebugLog("Failed to persist sessions after deletion: %v", err)
		// Don't fail deletion if persistence fails
	}

	debug.DebugLog("Successfully deleted session: %s", session.ID)
	return nil
}

// ListSessions returns all sessions (both main and linked worktrees)
func (m *Manager) ListSessions() []*Session {
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.Worktree != nil {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// GetMainSession returns the main worktree session for a repository
func (m *Manager) GetMainSession(repoName string) *Session {
	for _, session := range m.sessions {
		if session.Worktree != nil &&
		   session.Worktree.RepoName == repoName &&
		   !m.isLinkedWorktree(session) {
			return session
		}
	}
	return nil
}

// GetLinkedSessions returns all linked worktree sessions for a repository
func (m *Manager) GetLinkedSessions(repoName string) []*Session {
	sessions := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.Worktree != nil &&
		   session.Worktree.RepoName == repoName &&
		   m.isLinkedWorktree(session) {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// isLinkedWorktree determines if a session is from a linked worktree
func (m *Manager) isLinkedWorktree(session *Session) bool {
	if session.Worktree == nil {
		return false
	}
	// Linked worktree sessions have paths like: /Users/user/.agate/worktrees/repo/branch
	// Main sessions have paths like: /Users/user/Git/repo
	return strings.Contains(session.Worktree.Path, "/.agate/worktrees/")
}

// RestoreSessions attempts to reconnect to existing tmux sessions on startup
func (m *Manager) RestoreSessions() error {
	return m.LoadSessions()
}

// CleanupOrphanedSessions removes sessions for tmux sessions that no longer exist
func (m *Manager) CleanupOrphanedSessions() {
	for worktreeKey, session := range m.sessions {
		if session.TmuxSession != nil {
			// Check if tmux session still exists
			exists, err := session.TmuxSession.SessionExists()
			if err != nil || !exists {
				debug.DebugLog("Removing orphaned session: %s", session.ID)
				delete(m.sessions, worktreeKey)
				if m.activeSession == session {
					m.activeSession = nil
				}
			}
		}
	}
}

// GetWorktreeManager returns the worktree manager
func (m *Manager) GetWorktreeManager() *git.WorktreeManager {
	return m.worktreeMgr
}

