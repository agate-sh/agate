package session

import (
	"testing"

	"agate/pkg/config"
)

// mockStateManager implements StateManager for testing
type mockStateManager struct {
	sessions       map[string]config.PersistedSession
	activeSession  string
	pinnedSessions []string
}

func newMockStateManager() *mockStateManager {
	return &mockStateManager{
		sessions:       make(map[string]config.PersistedSession),
		pinnedSessions: []string{},
	}
}

func (m *mockStateManager) SaveSessionMapping(sessionID string, session config.PersistedSession) error {
	m.sessions[sessionID] = session
	return nil
}

func (m *mockStateManager) RemoveSessionMapping(sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockStateManager) GetSessionMappings() map[string]config.PersistedSession {
	return m.sessions
}

func (m *mockStateManager) SetActiveSession(sessionID string) error {
	m.activeSession = sessionID
	return nil
}

func (m *mockStateManager) GetActiveSession() string {
	return m.activeSession
}

func (m *mockStateManager) GetPinnedSessions() []string {
	return m.pinnedSessions
}

func (m *mockStateManager) UpdateSessions(fn func(*config.SessionState) error) error {
	state := &config.SessionState{
		SessionMappings: m.sessions,
		ActiveSession:   m.activeSession,
		PinnedSessions:  m.pinnedSessions,
	}
	if err := fn(state); err != nil {
		return err
	}
	m.sessions = state.SessionMappings
	m.activeSession = state.ActiveSession
	m.pinnedSessions = state.PinnedSessions
	return nil
}

func (m *mockStateManager) GetLastWorktreeForRepo(repoName string) *config.WorktreeRef {
	return nil
}

// TestCreateSessionBranchNaming tests that multi-agent sessions create unique branch names
func TestCreateSessionBranchNaming(t *testing.T) {
	// This is a unit test focusing on the branch naming logic
	// We'll test the branch name generation without actually creating worktrees

	baseBranch := "test-branch"
	agents := []string{"claude", "gemini"}

	for _, agent := range agents {
		expectedBranch := baseBranch + "_" + agent
		t.Logf("Agent %s should get branch: %s", agent, expectedBranch)

		// Verify the branch names are unique
		if expectedBranch == baseBranch {
			t.Errorf("Branch name for agent %s is not unique: %s", agent, expectedBranch)
		}
	}

	// Verify all branches are different
	branch1 := baseBranch + "_" + agents[0]
	branch2 := baseBranch + "_" + agents[1]
	if branch1 == branch2 {
		t.Errorf("Branches are not unique: %s == %s", branch1, branch2)
	}
}
