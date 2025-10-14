package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/config"
	"agate/pkg/git"
	"agate/pkg/naming"
	"agate/pkg/tmux"
)

// Manager is a singleton that manages all sessions
type Manager struct {
	sessions      map[string]*Session  // WorktreeKey -> Session
	activeSession *Session             // Currently active session
	worktreeMgr   *git.WorktreeManager // Default Git worktree manager (launch repo)
	worktreeMap   map[string]*git.WorktreeManager
	stateMgr      StateManager // Thread-safe state persistence
	headWatchers  map[string]*git.HeadWatcher
	branchUpdates chan BranchUpdate
}

// BranchUpdate represents a change to the main worktree's branch for a repository.
type BranchUpdate struct {
	RepoName string
	Worktree *git.WorktreeInfo
}

// StateManager defines the interface for state persistence
type StateManager interface {
	SaveSessionMapping(worktreeKey string, session config.PersistedSession) error
	RemoveSessionMapping(worktreeKey string) error
	GetSessionMappings() map[string]config.PersistedSession
	SetActiveSession(sessionKey string) error
	GetActiveSession() string
	UpdateSessions(fn func(*config.SessionState) error) error
	GetLastWorktreeForRepo(repoName string) *config.WorktreeRef
}

// NewManager creates a new session manager
func NewManager(worktreeMgr *git.WorktreeManager, stateMgr StateManager) *Manager {
	if stateMgr == nil {
		debug.DebugLog("WARNING: SessionManager created with nil StateManager - persistence will not work")
	}

	mgr := &Manager{
		sessions:      make(map[string]*Session),
		worktreeMgr:   worktreeMgr,
		worktreeMap:   make(map[string]*git.WorktreeManager),
		stateMgr:      stateMgr,
		headWatchers:  make(map[string]*git.HeadWatcher),
		branchUpdates: make(chan BranchUpdate, 16),
	}

	if worktreeMgr != nil {
		repoName := worktreeMgr.GetRepositoryName()
		mgr.worktreeMap[repoName] = worktreeMgr
	}

	return mgr
}

// BranchUpdates returns a read-only channel that emits branch updates for main worktrees.
func (m *Manager) BranchUpdates() <-chan BranchUpdate {
	if m == nil {
		return nil
	}
	return m.branchUpdates
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

	// Create shell tmux session with user's preferred shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	// Generate shell session name using naming generator
	nameGen := naming.NewGenerator()
	shellSessionName := nameGen.GenerateShellSessionName(sessionName)
	shellTmuxSession := tmux.NewTmuxSession(shellSessionName, shell)
	err = shellTmuxSession.Start(worktree.Path)
	if err != nil {
		// Clean up agent tmux session if shell session fails
		tmuxSession.Kill()
		return nil, fmt.Errorf("failed to start shell tmux session: %w", err)
	}

	// Create session
	session := &Session{
		ID:               worktreeKey + "_" + agentName,
		Name:             sessionName,
		WorktreeKey:      worktreeKey,
		TmuxSession:      tmuxSession,
		ShellTmuxSession: shellTmuxSession,
		Worktree:         worktree,
		Agent:            agentConfig,
		CreatedAt:        time.Now(),
		LastAccessed:     time.Now(),
		IsActive:         false,
	}

	// Store session
	m.sessions[worktreeKey] = session
	m.ensureHeadWatcher(session)

	// Persist session to config
	if err := m.PersistSessions(); err != nil {
		debug.DebugLog("Failed to persist session %s: %v", session.ID, err)
		// Don't fail session creation if persistence fails
	}

	debug.DebugLog("Created new session: %s for worktree: %s with agent: %s",
		session.ID, worktree.Path, agentName)

	return session, nil
}

// ApplyBranchUpdate updates the stored session metadata to reflect a branch change.
// Returns true if an existing session was updated.
func (m *Manager) ApplyBranchUpdate(update BranchUpdate) bool {
	if update.Worktree == nil {
		return false
	}

	updated := false
	for _, session := range m.sessions {
		if session == nil || session.Worktree == nil {
			continue
		}
		if m.isLinkedWorktree(session) {
			continue
		}
		if filepath.Clean(session.Worktree.Path) != filepath.Clean(update.Worktree.Path) {
			continue
		}

		session.Worktree.Branch = update.Worktree.Branch
		session.Worktree.Name = update.Worktree.Name
		session.Worktree.GitStatus = update.Worktree.GitStatus
		updated = true
	}

	if updated {
		if err := m.PersistSessions(); err != nil {
			debug.DebugLog("Failed to persist sessions after branch update: %v", err)
		}
	}

	return updated
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
		m.ensureHeadWatcher(session)
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

	// Kill shell tmux session
	if session.ShellTmuxSession != nil {
		if err := session.ShellTmuxSession.Kill(); err != nil {
			debug.DebugLog("Failed to kill shell tmux session: %v", err)
			// Continue with deletion even if shell kill fails
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
	m.stopHeadWatcher(session)
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

func (m *Manager) ensureHeadWatcher(session *Session) {
	if m == nil || session == nil || session.Worktree == nil {
		return
	}
	if m.headWatchers == nil {
		m.headWatchers = make(map[string]*git.HeadWatcher)
	}
	if m.isLinkedWorktree(session) {
		return
	}

	repoPath := filepath.Clean(session.Worktree.Path)
	if repoPath == "" {
		return
	}

	if _, exists := m.headWatchers[repoPath]; exists {
		return
	}

	repoName := session.Worktree.RepoName
	watcher, err := git.NewHeadWatcher(repoPath, func() {
		m.handleHeadChange(repoName, repoPath)
	})
	if err != nil {
		debug.DebugLog("Failed to start HEAD watcher for %s: %v", repoPath, err)
		return
	}

	m.headWatchers[repoPath] = watcher
}

func (m *Manager) handleHeadChange(repoName, repoPath string) {
	manager, err := m.GetWorktreeManagerForRepo(repoName)
	if err != nil {
		debug.DebugLog("handleHeadChange: failed to get worktree manager for %s: %v", repoName, err)
		return
	}

	worktree, err := manager.GetMainWorktreeInfo()
	if err != nil {
		debug.DebugLog("handleHeadChange: failed to get main worktree info for %s: %v", repoName, err)
		return
	}
	if worktree == nil {
		return
	}

	update := BranchUpdate{
		RepoName: repoName,
		Worktree: worktree,
	}

	select {
	case m.branchUpdates <- update:
		debug.DebugLog("handleHeadChange: dispatched branch update for repo=%s path=%s", repoName, repoPath)
	default:
		debug.DebugLog("handleHeadChange: branch update channel full, dropping update for repo=%s", repoName)
	}
}

func (m *Manager) stopHeadWatcher(session *Session) {
	if m == nil || session == nil || session.Worktree == nil {
		return
	}
	repoPath := filepath.Clean(session.Worktree.Path)
	if repoPath == "" {
		return
	}
	watcher, ok := m.headWatchers[repoPath]
	if !ok {
		return
	}
	if watcher != nil {
		if err := watcher.Close(); err != nil {
			debug.DebugLog("stopHeadWatcher: failed to close watcher for %s: %v", repoPath, err)
		}
	}
	delete(m.headWatchers, repoPath)
}

// ShutdownWatchers stops all active HEAD watchers. Mainly used for tests.
func (m *Manager) ShutdownWatchers() {
	for _, session := range m.sessions {
		m.stopHeadWatcher(session)
	}
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
		if m.isLinkedWorktree(session) {
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
