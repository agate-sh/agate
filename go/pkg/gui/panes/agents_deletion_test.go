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

func TestDetermineDeletionSelectionPlan_LinkedPrefersBelowWhenAboveIsMain(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "linked-top", "feature/top", "/tmp/linked-top")}

	items := []AgentListItem{
		{Type: "main_session", RepoName: repo, Worktree: makeWorktree(repo, "main", "main", "/tmp/main")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-top", "feature/top", "/tmp/linked-top")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-below", "feature/below", "/tmp/linked-below")},
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

	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/linked-below") {
		t.Fatalf("expected first candidate to be below linked session, got %+v", first)
	}

	if second == nil || second.Worktree == nil || second.Worktree.Path != filepath.Clean("/tmp/main") {
		t.Fatalf("expected second candidate to be main worktree, got %+v", second)
	}
}

func TestDetermineDeletionSelectionPlan_LinkedKeepsAboveWhenAboveIsLinked(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "linked-mid", "feature/mid", "/tmp/linked-mid")}

	items := []AgentListItem{
		{Type: "main_session", RepoName: repo, Worktree: makeWorktree(repo, "main", "main", "/tmp/main")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-above", "feature/above", "/tmp/linked-above")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-mid", "feature/mid", "/tmp/linked-mid")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-below", "feature/below", "/tmp/linked-below")},
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

	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/linked-above") {
		t.Fatalf("expected first candidate to be the linked session above, got %+v", first)
	}

	if second == nil || second.Worktree == nil || second.Worktree.Path != filepath.Clean("/tmp/linked-below") {
		t.Fatalf("expected second candidate to be the linked session below, got %+v", second)
	}
}

func TestDetermineDeletionSelectionPlan_LinkedUsesAboveWhenNoBelow(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "linked-last", "feature/last", "/tmp/linked-last")}

	items := []AgentListItem{
		{Type: "main_session", RepoName: repo, Worktree: makeWorktree(repo, "main", "main", "/tmp/main")},
		{Type: "linked_session", RepoName: repo, Worktree: makeWorktree(repo, "linked-last", "feature/last", "/tmp/linked-last")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found")
	}

	if len(plan.candidates) != 1 {
		t.Fatalf("expected one candidate (main), got %d", len(plan.candidates))
	}

	first := plan.candidates[0].item
	if first == nil || first.Worktree == nil || first.Worktree.Path != filepath.Clean("/tmp/main") {
		t.Fatalf("expected sole candidate to be main worktree, got %+v", first)
	}
}

func TestDetermineDeletionSelectionPlan_NoCandidatesFallback(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "main", "main", "/tmp/main")}

	items := []AgentListItem{
		{Type: "main_session", RepoName: repo, Worktree: makeWorktree(repo, "main", "main", "/tmp/main")},
	}

	plan, found := determineDeletionSelectionPlan(deleted, items)
	if !found {
		t.Fatalf("expected plan to be found for main session")
	}

	if len(plan.candidates) != 0 {
		t.Fatalf("expected no candidates, got %d", len(plan.candidates))
	}
}

func TestDetermineDeletionSelectionPlan_DeletedSessionNotFound(t *testing.T) {
	repo := "agate"
	deleted := &session.Session{Worktree: makeWorktree(repo, "ghost", "feature/ghost", "/tmp/ghost")}

	items := []AgentListItem{
		{Type: "main_session", RepoName: repo, Worktree: makeWorktree(repo, "main", "main", "/tmp/main")},
	}

	_, found := determineDeletionSelectionPlan(deleted, items)
	if found {
		t.Fatalf("expected plan not to be found when deleted session is missing")
	}
}
