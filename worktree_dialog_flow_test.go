package main

import (
	"testing"

	"agate/git"
)

func TestWorktreeDialogTransitionsToInitializingAfterWorktreeCreated(t *testing.T) {
	dialog := NewWorktreeDialog(nil, CodexAgent)
	dialog.creating = true

	initialModel := model{
		showWorktreeDialog: true,
		worktreeDialog:     dialog,
		agentConfig:        CodexAgent,
	}

	msg := WorktreeCreatedMsg{Worktree: &git.WorktreeInfo{Branch: "new-branch"}}

	updatedModel, _ := initialModel.Update(msg)
	got := updatedModel.(model)

	if got.worktreeDialog == nil {
		t.Fatalf("expected worktree dialog to remain visible")
	}

	if got.worktreeDialog.creating {
		t.Fatalf("expected creating to be false, got true")
	}

	if !got.worktreeDialog.initializing {
		t.Fatalf("expected initializing to be true")
	}

}

func TestWorktreeDialogShowsErrorAfterWorktreeCreationFailure(t *testing.T) {
	dialog := NewWorktreeDialog(nil, CodexAgent)
	dialog.creating = true

	initialModel := model{
		showWorktreeDialog: true,
		worktreeDialog:     dialog,
	}

	msg := WorktreeCreationErrorMsg{Error: "branch already exists"}

	updatedModel, _ := initialModel.Update(msg)
	got := updatedModel.(model)

	if got.worktreeDialog == nil {
		t.Fatalf("expected worktree dialog to remain visible")
	}

	if got.worktreeDialog.creating {
		t.Fatalf("expected creating to be false after error")
	}

	if got.worktreeDialog.err != msg.Error {
		t.Fatalf("expected error message %q, got %q", msg.Error, got.worktreeDialog.err)
	}
}
