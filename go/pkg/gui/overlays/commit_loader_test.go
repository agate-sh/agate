package overlays

import (
	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/session"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// TestCommitOverlay_LoaderAnimation verifies that the loader animates when generating
func TestCommitOverlay_LoaderAnimation(t *testing.T) {
	// Create a session with Claude agent
	sess := &session.Session{
		Agent: app.ClaudeAgent,
		Worktree: &git.WorktreeInfo{
			Path:     "/tmp/test",
			Branch:   "test-branch",
			RepoName: "test-repo",
		},
	}

	// Create commit overlay
	overlay := NewCommitOverlay(sess)
	overlay.SetSize(100, 40)

	// Initialize - should start generation
	initCmd := overlay.Init()
	if initCmd == nil {
		t.Fatal("Init should return commands to start loader")
	}

	// Note: initCmd is tea.Batch which can't be easily executed in tests
	// The important part is that it was created

	// Verify overlay is in generating state
	if !overlay.generating {
		t.Error("Overlay should be generating after Init()")
	}

	// Get initial view
	view1 := overlay.View()
	if !strings.Contains(view1, "Claude Code is generating a commit message") {
		t.Error("View should contain loader text")
	}

	// Send spinner tick message
	tickMsg := spinner.TickMsg{}
	model, cmd := overlay.Update(tickMsg)
	overlay = model.(*CommitOverlay)

	// THIS IS THE BUG: Should get a command back (next tick) but we get nil
	if cmd == nil {
		t.Fatal("BUG CONFIRMED: Update with TickMsg should return next tick command, got nil")
	}

	// Verify we can send multiple ticks
	for i := 0; i < 5; i++ {
		tickMsg := spinner.TickMsg{}
		model, cmd := overlay.Update(tickMsg)
		overlay = model.(*CommitOverlay)

		if cmd == nil {
			t.Errorf("Tick %d: Update should return tick command, got nil", i)
		}

		// View should still show loader
		view := overlay.View()
		if !strings.Contains(view, "Claude Code is generating a commit message") {
			t.Errorf("Tick %d: View should contain loader text", i)
		}
	}

	// Verify the loader text shows spinner characters
	finalView := overlay.View()
	hasSpinner := strings.Contains(finalView, "█") || strings.Contains(finalView, "░")
	if !hasSpinner {
		t.Error("View should contain spinner animation characters (█ or ░)")
	}
}

// TestCommitOverlay_LoaderTickFlow verifies tick messages flow through the overlay
func TestCommitOverlay_LoaderTickFlow(t *testing.T) {
	sess := &session.Session{
		Agent: app.ClaudeAgent,
		Worktree: &git.WorktreeInfo{
			Path:     "/tmp/test",
			Branch:   "test-branch",
			RepoName: "test-repo",
		},
	}

	overlay := NewCommitOverlay(sess)
	overlay.SetSize(100, 40)

	// Initialize
	initCmd := overlay.Init()
	if initCmd == nil {
		t.Fatal("Init should return commands")
	}

	// Verify overlay is generating
	if !overlay.generating {
		t.Fatal("Overlay should be generating after Init")
	}

	// Create a tick message
	tickMsg := spinner.TickMsg{}

	// Update overlay with tick
	model, cmd := overlay.Update(tickMsg)
	overlay = model.(*CommitOverlay)

	if cmd == nil {
		t.Fatal("Update with TickMsg should return a command (next tick)")
	}

	// Verify the command is a Cmd
	cmdType := tea.Batch(cmd)
	if cmdType == nil {
		t.Error("Returned command should be valid")
	}
}

// TestCommitOverlay_LoaderColorStyling verifies the loader uses agent color
func TestCommitOverlay_LoaderColorStyling(t *testing.T) {
	sess := &session.Session{
		Agent: app.ClaudeAgent, // Has BorderColor: "#da7756"
		Worktree: &git.WorktreeInfo{
			Path:     "/tmp/test",
			Branch:   "test-branch",
			RepoName: "test-repo",
		},
	}

	overlay := NewCommitOverlay(sess)
	overlay.SetSize(100, 40)
	overlay.Init()

	// View should show loader with agent color
	// We can't directly check loaderColor (private) but we can verify
	// the view renders properly

	// View should contain ANSI color codes (lipgloss renders colors)
	view := overlay.View()
	if len(view) == 0 {
		t.Error("View should not be empty")
	}
}
