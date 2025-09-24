package panes

import (
	"agate/theme"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShellPane manages the display of shell content (placeholder for future implementation)
type ShellPane struct {
	*BasePane
	content string
}

// NewShellPane creates a new ShellPane instance
func NewShellPane() *ShellPane {
	return &ShellPane{
		BasePane: NewBasePane(3, "Shell"), // Pane index 3
		content:  "",
	}
}

// SetContent updates the shell content to display
func (s *ShellPane) SetContent(content string) {
	s.content = content
}

// GetTitleStyle returns the plain title style for the shell pane
func (s *ShellPane) GetTitleStyle() TitleStyle {
	shortcuts := ""
	if s.IsActive() {
		// When active, could show shell-specific shortcuts in the future
		shortcuts = ""
	} else {
		// When not active, show pane number
		shortcuts = "[3]"
	}

	return TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Shell",
		Shortcuts: shortcuts,
	}
}

// View renders the shell pane content
func (s *ShellPane) View() string {
	if s.content == "" {
		// Show placeholder message
		style := lipgloss.NewStyle().
			Width(s.GetWidth()).
			Height(s.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextMuted))

		return style.Render("Shell pane - coming soon")
	}

	// Show shell content
	return s.content
}

// Update handles tea.Msg updates for the shell pane
func (s *ShellPane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	// ShellPane doesn't handle specific update messages currently
	return s, nil
}

// HandleKey processes keyboard input when the pane is active
func (s *ShellPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	// ShellPane doesn't handle any keys currently
	return false, nil
}

// GetPaneSpecificKeybindings returns shell pane specific keybindings
func (s *ShellPane) GetPaneSpecificKeybindings() []key.Binding {
	// No shell-specific keybindings currently
	return []key.Binding{}
}