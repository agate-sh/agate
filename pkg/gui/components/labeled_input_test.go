package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestLabeledInput_LoadingState(t *testing.T) {
	input := NewLabeledInput("Test Label", "placeholder")

	// Initially should not be loading
	if input.IsLoading() {
		t.Error("Input should not be loading initially")
	}

	// View should show the label and input
	view := input.View()
	if !strings.Contains(view, "Test Label") {
		t.Error("View should contain label")
	}

	// Set loading state with a loader
	loader := NewLaunchAgentLoader("Loading...")
	input.SetLoading(true, loader)

	if !input.IsLoading() {
		t.Error("Input should be loading after SetLoading(true)")
	}

	// View should now show the loader instead of input
	view = input.View()
	if !strings.Contains(view, "Test Label") {
		t.Error("View should still contain label when loading")
	}
	// Loader should be visible (check for loader characters)
	if !strings.Contains(view, "█") && !strings.Contains(view, "░") {
		t.Error("View should contain loader spinner when loading")
	}

	// Update with a tick message should delegate to loader
	tickMsg := spinner.TickMsg{}
	cmd := input.Update(tickMsg)
	if cmd == nil {
		t.Error("Update should return loader tick command when loading")
	}

	// Turn off loading
	input.SetLoading(false, nil)
	if input.IsLoading() {
		t.Error("Input should not be loading after SetLoading(false)")
	}

	// View should show input again
	view = input.View()
	if !strings.Contains(view, "Test Label") {
		t.Error("View should contain label when not loading")
	}
}

func TestLabeledInput_UpdateDelegation(t *testing.T) {
	input := NewLabeledInput("Test", "placeholder")
	loader := NewLaunchAgentLoader("Loading...")

	// When not loading, keyboard input should update text input
	input.SetLoading(false, nil)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_ = input.Update(keyMsg)
	// Text input should process the key
	if input.Value() != "a" {
		t.Errorf("Input should contain 'a', got %q", input.Value())
	}

	// When loading, keyboard input should be ignored
	input.SetLoading(true, loader)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	input.Update(keyMsg)
	// Value should still be "a" (not updated)
	if input.Value() != "a" {
		t.Errorf("Input value should not change when loading, got %q", input.Value())
	}
}

func TestLabeledInput_LoaderRendering(t *testing.T) {
	input := NewLabeledInput("Commit message", "Summary")
	loader := NewLaunchAgentLoader("Claude is generating a commit message")

	// Set loading
	input.SetLoading(true, loader)

	// ViewInput should show loader text
	viewInput := input.ViewInput()
	if !strings.Contains(viewInput, "Claude is generating a commit message") {
		t.Error("ViewInput should contain loader label when loading")
	}

	// Full View should show both label and loader
	fullView := input.View()
	if !strings.Contains(fullView, "Commit message") {
		t.Error("Full view should contain input label")
	}
	if !strings.Contains(fullView, "Claude is generating a commit message") {
		t.Error("Full view should contain loader text when loading")
	}
}
