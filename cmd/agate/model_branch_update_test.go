package main

import (
	"path/filepath"
	"testing"
	"unsafe"

	"agate/pkg/git"
	"agate/pkg/gui/panes"
	"agate/pkg/session"
)

func TestModelHandlesBranchUpdateMsg(t *testing.T) {
	mgr := session.NewManager(nil, nil)

	worktreePath := filepath.Clean("/tmp/agate-test-repo")
	worktree := &git.WorktreeInfo{
		RepoName: "agaterepo",
		Path:     worktreePath,
		Branch:   "main",
		Name:     "main",
	}
	sess := &session.Session{
		Worktree:    worktree,
		WorktreeKey: worktree.RepoName + ":" + worktree.Path,
	}

	injectSessionIntoManager(mgr, sess)

	pane := panes.NewAgentsPane(mgr)

	m := model{
		sessionManager: mgr,
		repoPane:       pane,
	}

	update := session.BranchUpdate{
		RepoName: worktree.RepoName,
		Worktree: &git.WorktreeInfo{
			RepoName: worktree.RepoName,
			Path:     worktree.Path,
			Branch:   "feature/branch",
			Name:     "feature/branch",
		},
	}

	newModel, cmd := m.Update(branchUpdateMsg{update: update})
	if cmd == nil {
		t.Fatal("expected command to continue listening for branch updates")
	}

	updatedModel, ok := newModel.(model)
	if !ok {
		t.Fatalf("expected model type after update, got %T", newModel)
	}

	mainSession := updatedModel.sessionManager.GetMainSession(worktree.RepoName)
	if mainSession == nil || mainSession.Worktree.Branch != "feature/branch" {
		t.Fatalf("expected session branch to update, got %v", mainSession)
	}

	items := pane.SnapshotItems()
	found := false
	for _, item := range items {
		if item.Type == "main_session" && item.Worktree != nil {
			if item.Worktree.Branch == "feature/branch" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected agents pane items to reflect new branch; items: %+v", items)
	}
}

func injectSessionIntoManager(mgr *session.Manager, sess *session.Session) {
	sessionsField := reflectValueOfSessions(mgr)
	key := sess.WorktreeKey
	sessionsField[key] = sess
}

type managerView struct {
	Sessions      map[string]*session.Session
	ActiveSession *session.Session
	WorktreeMgr   *git.WorktreeManager
	WorktreeMap   map[string]*git.WorktreeManager
	StateMgr      session.StateManager
	HeadWatchers  map[string]*git.HeadWatcher
	BranchUpdates chan session.BranchUpdate
}

func reflectValueOfSessions(mgr *session.Manager) map[string]*session.Session {
	view := (*managerView)(unsafe.Pointer(mgr))
	return view.Sessions
}
