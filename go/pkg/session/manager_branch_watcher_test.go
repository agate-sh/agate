package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"agate/pkg/git"
)

func TestManagerEmitsBranchUpdatesOnHeadChange(t *testing.T) {
	repoPath := setupGitRepo(t)

	worktreeMgr, err := git.NewWorktreeManagerForPath(repoPath)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	mgr := NewManager(worktreeMgr, newFakeStateManager())
	defer mgr.ShutdownWatchers()

	worktree, err := worktreeMgr.GetMainWorktreeInfo()
	if err != nil {
		t.Fatalf("failed to get main worktree: %v", err)
	}

	session := &Session{Worktree: worktree}
	mgr.sessions[generateWorktreeKey(worktree)] = session

	mgr.ensureHeadWatcher(session)

	drainBranchUpdates(mgr)

	if err := exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/headwatcher").Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	select {
	case update := <-mgr.BranchUpdates():
		if update.Worktree == nil {
			t.Fatal("expected worktree info in update")
		}
		if got := update.Worktree.Branch; got != "feature/headwatcher" {
			t.Fatalf("expected branch name feature/headwatcher, got %s", got)
		}
		if !mgr.ApplyBranchUpdate(update) {
			t.Fatal("expected ApplyBranchUpdate to report update")
		}
		if session.Worktree.Branch != "feature/headwatcher" {
			t.Fatalf("session branch not updated, got %s", session.Worktree.Branch)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for branch update")
	}
}

func drainBranchUpdates(mgr *Manager) {
	for {
		select {
		case <-mgr.BranchUpdates():
			continue
		default:
			return
		}
	}
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()

	if err := exec.Command("git", "init", repoPath).Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure local git user
	configureGit(t, repoPath, "user.name", "Test User")
	configureGit(t, repoPath, "user.email", "test@example.com")

	// Create initial commit so branch checkout works
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	if err := exec.Command("git", "-C", repoPath, "add", "README.md").Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}
	if err := exec.Command("git", "-C", repoPath, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	return repoPath
}

func configureGit(t *testing.T, repoPath, key, value string) {
	t.Helper()
	if err := exec.Command("git", "-C", repoPath, "config", key, value).Run(); err != nil {
		t.Fatalf("failed to set git config %s: %v", key, err)
	}
}
