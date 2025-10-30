package panes

import (
	"path/filepath"
	"testing"

	"agate/pkg/git"
	"agate/pkg/session"
)

func makeWorktree(repo, name, branch, path string) *git.WorktreeInfo {
	return &git.WorktreeInfo{
		RepoName: repo,
		Name:     name,
		Branch:   branch,
		Path:     filepath.Clean(path),
	}
}

func TestDetermineDeletionSelectionPlan_PrefersAbove(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "session-top", "feature/top", "/tmp/session-top")}

	items := []AgentListItem{
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-1", "feature/1", "/tmp/session-1")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-top", "feature/top", "/tmp/session-top")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-below", "feature/below", "/tmp/session-below")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found")
	}

	if plan.deletedIdx != 1 {
		t.Fatalf("expected deletedIdx=1, got %d", plan.deletedIdx)
	}

	if len(plan.candidates) != 2 {
		t.Fatalf("expected two candidates, got %d", len(plan.candidates))
	}

	first := plan.candidates[0].item
	second := plan.candidates[1].item

	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/session-1") {
		t.Fatalf("expected first candidate to be above session, got %+v", first)
	}

	if second == nil || second.Worktree == nil || second.Worktree.Path != filepath.Clean("/tmp/session-below") {
		t.Fatalf("expected second candidate to be below session, got %+v", second)
	}
}

func TestDetermineDeletionSelectionPlan_MiddleSession(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "session-mid", "feature/mid", "/tmp/session-mid")}

	items := []AgentListItem{
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-1", "feature/1", "/tmp/session-1")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-above", "feature/above", "/tmp/session-above")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-mid", "feature/mid", "/tmp/session-mid")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-below", "feature/below", "/tmp/session-below")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found")
	}

	if len(plan.candidates) != 2 {
		t.Fatalf("expected two candidates, got %d", len(plan.candidates))
	}

	first := plan.candidates[0].item
	second := plan.candidates[1].item

	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/session-above") {
		t.Fatalf("expected first candidate to be the session above, got %+v", first)
	}

	if second == nil || second.Worktree == nil || second.Worktree.Path != filepath.Clean("/tmp/session-below") {
		t.Fatalf("expected second candidate to be the session below, got %+v", second)
	}
}

func TestDetermineDeletionSelectionPlan_UsesAboveWhenNoBelow(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "session-last", "feature/last", "/tmp/session-last")}

	items := []AgentListItem{
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-1", "feature/1", "/tmp/session-1")},
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-last", "feature/last", "/tmp/session-last")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found")
	}

	if len(plan.candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(plan.candidates))
	}

	first := plan.candidates[0].item
	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/session-1") {
		t.Fatalf("expected sole candidate to be the session above, got %+v", first)
	}
}

func TestDetermineDeletionSelectionPlan_NoCandidatesFallback(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "session-only", "feature/only", "/tmp/session-only")}

	items := []AgentListItem{
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-only", "feature/only", "/tmp/session-only")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found for session")
	}

	if len(plan.candidates) != 0 {
		t.Fatalf("expected no candidates, got %d", len(plan.candidates))
	}
}

func TestDetermineDeletionSelectionPlan_DeletedSessionNotFound(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "ghost", "feature/ghost", "/tmp/ghost")}

	items := []AgentListItem{
		{Type: "session", RepoName: repo, Worktree: makeWorktree(repo, "session-1", "feature/1", "/tmp/session-1")},
	}

	_, found := determineDeletionSelectionPlan(deleted, items)
	if found {
		t.Fatalf("expected plan not to be found when deleted session is missing")
	}
}
