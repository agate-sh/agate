// Package state provides thread-safe state management for the agate application.
// It uses a centralized StateManager with mutex protection to prevent race conditions
// in the read-modify-write cycle of state persistence.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"agate/internal/debug"
	"agate/pkg/config"
)

// Manager provides thread-safe access to application state.
// It maintains an in-memory copy of the state protected by a mutex
// and provides atomic writes to disk.
type Manager struct {
	mu       sync.RWMutex
	state    *config.AppState
	filePath string
}

// NewManager creates a new state manager and loads the initial state from disk.
func NewManager() (*Manager, error) {
	if err := config.EnsureAgateDir(); err != nil {
		return nil, err
	}

	agateDir, err := config.GetAgateDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(agateDir, "state.json")

	state, err := loadStateFromDisk(filePath)
	if err != nil {
		return nil, err
	}

	// Note: Can't use debug.DebugLog here as it might not be initialized yet
	// Will log in GetSessionMappings when it's called

	return &Manager{
		state:    state,
		filePath: filePath,
	}, nil
}

// loadStateFromDisk loads state from the given file path.
// If the file doesn't exist, returns a default state.
func loadStateFromDisk(filePath string) (*config.AppState, error) {
	state := defaultAppState()

	// If file doesn't exist, return default state
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &state, nil
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		state.Normalize()
		return &state, nil
	}

	if err := json.Unmarshal(data, &state); err != nil {
		state.Normalize()
		return &state, nil
	}

	state.Normalize()
	return &state, nil
}

// defaultAppState returns a new AppState with default values.
func defaultAppState() config.AppState {
	return config.AppState{
		Version: 1,
		UI:      config.UIState{},
		Workspace: config.WorkspaceState{
			Repositories:   []string{},
			RepoSelections: map[string]config.RepoSelection{},
		},
		Sessions: config.SessionState{
			SessionMappings: map[string]config.PersistedSession{},
		},
	}
}

// atomicSave performs an atomic write of the state to disk.
// It writes to a temporary file first, then atomically renames it.
// This ensures that the state file is never corrupted or partially written.
//
// Must be called with mu held (either read or write lock).
func (m *Manager) atomicSave() error {
	m.state.Normalize()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file in same directory (ensures same filesystem for atomic rename)
	dir := filepath.Dir(m.filePath)
	tmp, err := os.CreateTemp(dir, ".state.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	// Ensure cleanup on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	// Atomic rename (POSIX guarantee)
	if err := os.Rename(tmpPath, m.filePath); err != nil {
		return err
	}

	success = true
	return nil
}

// GetState returns a deep copy of the current state for read-only access.
func (m *Manager) GetState() config.AppState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a deep copy to prevent external modifications
	return *m.state
}

// UpdateSessions executes the given function with exclusive access to the Sessions state.
// The function can modify the SessionState, and changes will be atomically saved to disk.
func (m *Manager) UpdateSessions(fn func(*config.SessionState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := fn(&m.state.Sessions); err != nil {
		return err
	}

	return m.atomicSave()
}

// UpdateWorkspace executes the given function with exclusive access to the Workspace state.
// The function can modify the WorkspaceState, and changes will be atomically saved to disk.
func (m *Manager) UpdateWorkspace(fn func(*config.WorkspaceState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := fn(&m.state.Workspace); err != nil {
		return err
	}

	return m.atomicSave()
}

// UpdateUI executes the given function with exclusive access to the UI state.
// The function can modify the UIState, and changes will be atomically saved to disk.
func (m *Manager) UpdateUI(fn func(*config.UIState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := fn(&m.state.UI); err != nil {
		return err
	}

	return m.atomicSave()
}

// ReadSessions executes the given function with read-only access to the Sessions state.
// Multiple readers can access the state concurrently.
func (m *Manager) ReadSessions(fn func(*config.SessionState) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fn(&m.state.Sessions)
}

// ReadWorkspace executes the given function with read-only access to the Workspace state.
// Multiple readers can access the state concurrently.
func (m *Manager) ReadWorkspace(fn func(*config.WorkspaceState) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fn(&m.state.Workspace)
}

// ReadUI executes the given function with read-only access to the UI state.
// Multiple readers can access the state concurrently.
func (m *Manager) ReadUI(fn func(*config.UIState) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fn(&m.state.UI)
}

// Session-specific convenience methods

// SaveSessionMapping persists a session mapping.
func (m *Manager) SaveSessionMapping(worktreeKey string, session config.PersistedSession) error {
	return m.UpdateSessions(func(s *config.SessionState) error {
		if s.SessionMappings == nil {
			s.SessionMappings = make(map[string]config.PersistedSession)
		}
		s.SessionMappings[worktreeKey] = session
		return nil
	})
}

// RemoveSessionMapping removes a session mapping.
func (m *Manager) RemoveSessionMapping(worktreeKey string) error {
	return m.UpdateSessions(func(s *config.SessionState) error {
		if s.SessionMappings != nil {
			delete(s.SessionMappings, worktreeKey)
		}
		return nil
	})
}

// GetSessionMappings returns a copy of the stored session mappings.
// Always returns a valid map, never nil.
func (m *Manager) GetSessionMappings() map[string]config.PersistedSession {
	// Initialize with empty map to ensure we never return nil
	result := make(map[string]config.PersistedSession)

	if m == nil {
		debug.DebugLog("GetSessionMappings: StateManager is nil, returning empty map")
		return result
	}

	err := m.ReadSessions(func(s *config.SessionState) error {
		// Create new map with correct size
		result = make(map[string]config.PersistedSession, len(s.SessionMappings))
		for key, value := range s.SessionMappings {
			result[key] = value
		}
		debug.DebugLog("GetSessionMappings: Returning %d session mappings from state", len(result))
		return nil
	})

	if err != nil {
		debug.DebugLog("GetSessionMappings: Error reading sessions: %v, returning empty map", err)
		return make(map[string]config.PersistedSession)
	}

	return result
}

// SetActiveSession sets the active session key.
func (m *Manager) SetActiveSession(sessionKey string) error {
	return m.UpdateSessions(func(s *config.SessionState) error {
		s.ActiveSession = sessionKey
		return nil
	})
}

// GetActiveSession returns the active session key.
func (m *Manager) GetActiveSession() string {
	var result string
	m.ReadSessions(func(s *config.SessionState) error {
		result = s.ActiveSession
		return nil
	})
	return result
}

// GetPinnedSessions returns the list of pinned session IDs.
func (m *Manager) GetPinnedSessions() []string {
	var result []string
	m.ReadSessions(func(s *config.SessionState) error {
		if s.PinnedSessions == nil {
			result = []string{}
		} else {
			result = s.PinnedSessions
		}
		return nil
	})
	return result
}

// SetSelectedAgents sets the selected agents for new sessions.
func (m *Manager) SetSelectedAgents(agents []string) error {
	return m.UpdateSessions(func(s *config.SessionState) error {
		s.SelectedAgents = agents
		return nil
	})
}

// GetSelectedAgents returns the selected agents for new sessions.
// Returns empty slice if no agents are selected.
func (m *Manager) GetSelectedAgents() []string {
	var result []string
	m.ReadSessions(func(s *config.SessionState) error {
		result = s.SelectedAgents
		return nil
	})
	return result
}

// Workspace-specific convenience methods

// AddRepository adds a repository to the list if it's not already present.
func (m *Manager) AddRepository(repoPath string) error {
	return m.UpdateWorkspace(func(w *config.WorkspaceState) error {
		for _, existing := range w.Repositories {
			if existing == repoPath {
				return nil // Already exists
			}
		}
		w.Repositories = append(w.Repositories, repoPath)
		return nil
	})
}

// RemoveRepository removes a repository from the list.
func (m *Manager) RemoveRepository(repoPath string) error {
	return m.UpdateWorkspace(func(w *config.WorkspaceState) error {
		filtered := w.Repositories[:0]
		for _, existing := range w.Repositories {
			if existing != repoPath {
				filtered = append(filtered, existing)
			}
		}
		w.Repositories = append([]string{}, filtered...)
		delete(w.RepoSelections, repoPath)
		return nil
	})
}

// GetRepositories returns the list of user-added repositories.
func (m *Manager) GetRepositories() []string {
	var result []string
	m.ReadWorkspace(func(w *config.WorkspaceState) error {
		result = append([]string{}, w.Repositories...)
		return nil
	})
	return result
}

// SetLastWorktreeForRepo persists the latest worktree reference for the given repo name.
func (m *Manager) SetLastWorktreeForRepo(repoName string, worktree config.WorktreeRef) error {
	if repoName == "" {
		return errors.New("repo name cannot be empty")
	}

	return m.UpdateWorkspace(func(w *config.WorkspaceState) error {
		w.RepoSelections[repoName] = config.RepoSelection{Worktree: worktree}
		w.LastRepo = repoName
		return nil
	})
}

// GetLastWorktreeForRepo returns the stored worktree reference for the given repo name.
func (m *Manager) GetLastWorktreeForRepo(repoName string) *config.WorktreeRef {
	var result *config.WorktreeRef
	m.ReadWorkspace(func(w *config.WorkspaceState) error {
		selection, ok := w.RepoSelections[repoName]
		if ok {
			ref := selection.Worktree
			result = &ref
		}
		return nil
	})
	return result
}

// GetLastActiveRepo returns the name of the last repo that was active.
func (m *Manager) GetLastActiveRepo() string {
	var result string
	m.ReadWorkspace(func(w *config.WorkspaceState) error {
		result = w.LastRepo
		return nil
	})
	return result
}

// GetRepoSelections returns a copy of the stored repo selections.
func (m *Manager) GetRepoSelections() map[string]config.RepoSelection {
	var result map[string]config.RepoSelection
	m.ReadWorkspace(func(w *config.WorkspaceState) error {
		result = make(map[string]config.RepoSelection, len(w.RepoSelections))
		for key, value := range w.RepoSelections {
			result[key] = value
		}
		return nil
	})
	return result
}
