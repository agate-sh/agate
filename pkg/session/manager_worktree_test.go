package session

import (
	"os/exec"
	"path/filepath"
	"testing"

	"agate/pkg/config"
	"agate/pkg/git"
)

// fakeStateManager implements the minimal subset of StateManager needed for tests.
type fakeStateManager struct {
	repoRefs map[string]config.WorktreeRef
}

func newFakeStateManager() *fakeStateManager {
	return &fakeStateManager{repoRefs: make(map[string]config.WorktreeRef)}
}

func (f *fakeStateManager) SaveSessionMapping(string, config.PersistedSession) error { return nil }
func (f *fakeStateManager) RemoveSessionMapping(string) error                        { return nil }
func (f *fakeStateManager) GetSessionMappings() map[string]config.PersistedSession {
	return map[string]config.PersistedSession{}
}
func (f *fakeStateManager) SetActiveSession(string) error { return nil }
func (f *fakeStateManager) GetActiveSession() string      { return "" }
func (f *fakeStateManager) GetPinnedSessions() []string   { return []string{} }
func (f *fakeStateManager) UpdateSessions(fn func(*config.SessionState) error) error {
	if fn != nil {
		return fn(&config.SessionState{})
	}
	return nil
}
func (f *fakeStateManager) GetLastWorktreeForRepo(repoName string) *config.WorktreeRef {
	ref, ok := f.repoRefs[repoName]
	if !ok {
		return nil
	}
	copy := ref
	return &copy
}

func TestManager_GetRepositoryPath_UsesMainSession(t *testing.T) {
	mgr := NewManager(nil, newFakeStateManager())

	repoPath := filepath.Join(t.TempDir(), "myrepo")
	repoName := "myrepo"
	worktree := &git.WorktreeInfo{
		RepoName: repoName,
		Path:     repoPath,
		Branch:   "main",
	}

	sess := &Session{Worktree: worktree}
	mgr.sessions[generateWorktreeKey(worktree)] = sess

	got, err := mgr.GetRepositoryPath(repoName)
	if err != nil {
		t.Fatalf("GetRepositoryPath returned error: %v", err)
	}
	if got != repoPath {
		t.Fatalf("GetRepositoryPath = %s, want %s", got, repoPath)
	}
}

func TestManager_GetRepositoryPath_FallsBackToState(t *testing.T) {
	fakeState := newFakeStateManager()
	mgr := NewManager(nil, fakeState)

	repoPath := filepath.Join(t.TempDir(), "fallbackrepo")
	repoName := "fallbackrepo"
	fakeState.repoRefs[repoName] = config.WorktreeRef{Path: repoPath}

	got, err := mgr.GetRepositoryPath(repoName)
	if err != nil {
		t.Fatalf("GetRepositoryPath returned error: %v", err)
	}
	if got != repoPath {
		t.Fatalf("GetRepositoryPath = %s, want %s", got, repoPath)
	}
}

func createGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "init", dir).Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
}

func TestManager_GetWorktreeManagerForRepo_CachesManagersPerRepo(t *testing.T) {
	fakeState := newFakeStateManager()
	mgr := NewManager(nil, fakeState)

	repoPathOne := filepath.Join(t.TempDir(), "repo-one")
	repoPathTwo := filepath.Join(t.TempDir(), "repo-two")
	repoNameOne := "repo-one"
	repoNameTwo := "repo-two"

	createGitRepo(t, repoPathOne)
	createGitRepo(t, repoPathTwo)

	fakeState.repoRefs[repoNameOne] = config.WorktreeRef{Path: repoPathOne}
	fakeState.repoRefs[repoNameTwo] = config.WorktreeRef{Path: repoPathTwo}

	mgrOneFirst, err := mgr.GetWorktreeManagerForRepo(repoNameOne)
	if err != nil {
		t.Fatalf("GetWorktreeManagerForRepo returned error for repo one: %v", err)
	}

	mgrOneSecond, err := mgr.GetWorktreeManagerForRepo(repoNameOne)
	if err != nil {
		t.Fatalf("second GetWorktreeManagerForRepo returned error for repo one: %v", err)
	}

	if mgrOneFirst != mgrOneSecond {
		t.Fatalf("expected cached manager for repo one to be reused")
	}

	mgrTwo, err := mgr.GetWorktreeManagerForRepo(repoNameTwo)
	if err != nil {
		t.Fatalf("GetWorktreeManagerForRepo returned error for repo two: %v", err)
	}

	if mgrTwo == mgrOneFirst {
		t.Fatalf("expected different manager for different repo")
	}

	gotPath := filepath.Clean(mgrTwo.GetRepositoryPath())
	gotEval, err := filepath.EvalSymlinks(gotPath)
	if err == nil {
		gotPath = gotEval
	}

	wantedPath := filepath.Clean(repoPathTwo)
	wantEval, err := filepath.EvalSymlinks(wantedPath)
	if err == nil {
		wantedPath = wantEval
	}

	if gotPath != wantedPath {
		t.Fatalf("repo two manager path = %s, want %s", gotPath, wantedPath)
	}
}

func TestManager_GetRepositoryPath_NotFound(t *testing.T) {
	mgr := NewManager(nil, newFakeStateManager())
	if _, err := mgr.GetRepositoryPath("missing"); err == nil {
		t.Fatalf("expected error when repo path is unknown")
	}
}
