package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/config"
	"agate/pkg/git"
	"agate/pkg/tmux"
)

const MaxPinnedSessions = 4

// Manager is a singleton that manages all sessions
type Manager struct {
	sessions       map[string]*Session  // WorktreeKey -> Session
	activeSession  *Session             // Currently active session
	pinnedSessions []string             // Ordered list of pinned session IDs (max MaxPinnedSessions)
	worktreeMgr    *git.WorktreeManager // Default Git worktree manager (launch repo)
	worktreeMap    map[string]*git.WorktreeManager
	stateMgr       StateManager // Thread-safe state persistence
}

// StateManager defines the interface for state persistence
type StateManager interface {
	SaveSessionMapping(worktreeKey string, session config.PersistedSession) error
	RemoveSessionMapping(worktreeKey string) error
	GetSessionMappings() map[string]config.PersistedSession
	SetActiveSession(sessionKey string) error
	GetActiveSession() string
	GetPinnedSessions() []string
	UpdateSessions(fn func(*config.SessionState) error) error
	GetLastWorktreeForRepo(repoName string) *config.WorktreeRef
}

// NewManager creates a new session manager
func NewManager(worktreeMgr *git.WorktreeManager, stateMgr StateManager) *Manager {
	if stateMgr == nil {
		debug.DebugLog("WARNING: SessionManager created with nil StateManager - persistence will not work")
	}

	mgr := &Manager{
		sessions:    make(map[string]*Session),
		worktreeMgr: worktreeMgr,
		worktreeMap: make(map[string]*git.WorktreeManager),
		stateMgr:    stateMgr,
	}

	if worktreeMgr != nil {
		repoName := worktreeMgr.GetRepositoryName()
		mgr.worktreeMap[repoName] = worktreeMgr
	}

	return mgr
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

	// Use the agent config's executable name for tmux session (not the raw agentName)
	executableName := agentConfig.ExecutableName
	if executableName == "" {
		// Fallback to agentName if ExecutableName is empty (shouldn't happen with DefaultAgent)
		executableName = agentName
		if executableName == "" {
			executableName = "default"
		}
	}

	// Create tmux session
	tmuxSession := tmux.NewTmuxSession(sessionName, executableName)
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
	if m.activeSession != nil && m.activeSession.Worktree != nil {
		debug.DebugLog("[SessionManager] GetActiveSession: returning path=%s, branch=%s",
			m.activeSession.Worktree.Path, m.activeSession.Worktree.Branch)
	} else if m.activeSession != nil {
		debug.DebugLog("[SessionManager] GetActiveSession: returning session with nil worktree")
	} else {
		debug.DebugLog("[SessionManager] GetActiveSession: returning nil")
	}
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

// ListSessions returns all sessions
func (m *Manager) ListSessions() []*Session {
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.Worktree != nil {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// GetRepositoryPath returns the absolute path for the given repository name.
func (m *Manager) GetRepositoryPath(repoName string) (string, error) {
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		if m.worktreeMgr != nil {
			return m.worktreeMgr.GetRepositoryPath(), nil
		}
		return "", fmt.Errorf("repository name cannot be empty")
	}

	if manager, ok := m.worktreeMap[repoName]; ok && manager != nil {
		return manager.GetRepositoryPath(), nil
	}

	if m.worktreeMgr != nil && m.worktreeMgr.GetRepositoryName() == repoName {
		return m.worktreeMgr.GetRepositoryPath(), nil
	}

	for _, session := range m.sessions {
		if session == nil || session.Worktree == nil {
			continue
		}
		if session.Worktree.RepoName != repoName {
			continue
		}
		path := strings.TrimSpace(session.Worktree.Path)
		if path != "" {
			return filepath.Clean(path), nil
		}
	}

	if m.stateMgr != nil {
		if ref := m.stateMgr.GetLastWorktreeForRepo(repoName); ref != nil {
			path := strings.TrimSpace(ref.Path)
			if path != "" {
				return filepath.Clean(path), nil
			}
		}
	}

	return "", fmt.Errorf("repository path not found for repo: %s", repoName)
}

// GetWorktreeManagerForRepo returns a worktree manager scoped to the given repository.
func (m *Manager) GetWorktreeManagerForRepo(repoName string) (*git.WorktreeManager, error) {
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		if m.worktreeMgr != nil {
			return m.worktreeMgr, nil
		}
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	if manager, ok := m.worktreeMap[repoName]; ok && manager != nil {
		return manager, nil
	}

	repoPath, err := m.GetRepositoryPath(repoName)
	if err != nil {
		return nil, err
	}

	manager, err := git.NewWorktreeManagerForPath(repoPath)
	if err != nil {
		return nil, err
	}

	if m.worktreeMap == nil {
		m.worktreeMap = make(map[string]*git.WorktreeManager)
	}
	key := manager.GetRepositoryName()
	m.worktreeMap[key] = manager
	if key != repoName {
		m.worktreeMap[repoName] = manager
	}

	return manager, nil
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

// PinSession pins a session to the grid display (max 4 sessions)
func (m *Manager) PinSession(sessionID string) error {
	// Check if already pinned
	for _, id := range m.pinnedSessions {
		if id == sessionID {
			debug.DebugLog("Session %s is already pinned", sessionID)
			return nil // Already pinned, not an error
		}
	}

	// Check if we've reached the max limit
	if len(m.pinnedSessions) >= MaxPinnedSessions {
		return fmt.Errorf("cannot pin more than %d sessions (currently have %d pinned)", MaxPinnedSessions, len(m.pinnedSessions))
	}

	// Verify the session exists
	found := false
	for _, session := range m.sessions {
		if session.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Add to pinned sessions
	m.pinnedSessions = append(m.pinnedSessions, sessionID)
	debug.DebugLog("Pinned session %s (total pinned: %d)", sessionID, len(m.pinnedSessions))

	// Persist the change
	if err := m.PersistSessions(); err != nil {
		debug.DebugLog("Failed to persist pinned session %s: %v", sessionID, err)
		// Don't fail the pin operation if persistence fails
	}

	return nil
}

// UnpinSession removes a session from the pinned list
func (m *Manager) UnpinSession(sessionID string) {
	for i, id := range m.pinnedSessions {
		if id == sessionID {
			// Remove from slice
			m.pinnedSessions = append(m.pinnedSessions[:i], m.pinnedSessions[i+1:]...)
			debug.DebugLog("Unpinned session %s (remaining pinned: %d)", sessionID, len(m.pinnedSessions))

			// Persist the change
			if err := m.PersistSessions(); err != nil {
				debug.DebugLog("Failed to persist after unpinning session %s: %v", sessionID, err)
				// Don't fail the unpin operation if persistence fails
			}
			return
		}
	}
	debug.DebugLog("Session %s was not pinned", sessionID)
}

// GetPinnedSessions returns an ordered list of pinned sessions
func (m *Manager) GetPinnedSessions() []*Session {
	result := make([]*Session, 0, len(m.pinnedSessions))
	for _, sessionID := range m.pinnedSessions {
		// Find the session by ID
		for _, session := range m.sessions {
			if session.ID == sessionID {
				result = append(result, session)
				break
			}
		}
	}
	return result
}

// IsPinned checks if a session is currently pinned
func (m *Manager) IsPinned(sessionID string) bool {
	for _, id := range m.pinnedSessions {
		if id == sessionID {
			return true
		}
	}
	return false
}
