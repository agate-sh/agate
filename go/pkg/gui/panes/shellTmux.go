package panes

import (
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/tmux"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShellTmuxPane manages the display of shell content using tmux like the AgentTmuxPane
type ShellTmuxPane struct {
	*components.BasePane
	session *tmux.TmuxSession
	content string
}

// NewShellTmuxPane creates a new ShellTmuxPane instance
func NewShellTmuxPane() *ShellTmuxPane {
	return &ShellTmuxPane{
		BasePane: components.NewBasePane(3, "Shell"), // Pane index 3
		content:  "",
	}
}

// SetSession sets the tmux session for this pane
func (s *ShellTmuxPane) SetSession(session *tmux.TmuxSession) {
	s.session = session
	s.updateContent()
}

// SetContent updates the shell content to display
func (s *ShellTmuxPane) SetContent(content string) {
	s.content = content
}

// updateContent refreshes the content from the tmux session
func (s *ShellTmuxPane) updateContent() {
	if s.session != nil {
		content, err := s.session.CapturePaneContent()
		if err == nil {
			s.content = content
		}
	}
}

// SetSize updates the dimensions and resizes the tmux session
func (s *ShellTmuxPane) SetSize(width, height int) {
	s.BasePane.SetSize(width, height)
	if s.session != nil {
		// Set tmux session size to match pane content area
		s.session.SetDetachedSize(width, height)
	}
}

// GetTitleStyle returns the plain title style for the shell pane
func (s *ShellTmuxPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if s.IsActive() {
		// When active, format shortcuts like the footer (without brackets)
		shortcuts = "↵ attach • ctrl+q detach"
	} else {
		// When not active, show pane number
		shortcuts = "(3)"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Shell",
		Shortcuts: shortcuts,
	}
}

// View renders the shell pane content
func (s *ShellTmuxPane) View() string {
	// Update content from session if available
	if s.session != nil {
		s.updateContent()
	}

	if s.content == "" {
		// Show placeholder message
		style := lipgloss.NewStyle().
			Width(s.GetWidth()).
			Height(s.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextMuted))

		if s.session == nil {
			return style.Render("No shell session")
		} else {
			return style.Render("Shell starting...")
		}
	}

	// Show shell content
	return s.content
}

// Update handles tea.Msg updates for the shell pane
func (s *ShellTmuxPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	// ShellTmuxPane doesn't handle specific update messages currently
	return s, nil
}

// HandleKey processes keyboard input when the pane is active
func (s *ShellTmuxPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	// ShellTmuxPane uses tmux attach/detach like AgentTmuxPane - no direct key handling
	return false, nil
}

// GetPaneSpecificKeybindings returns shell pane specific keybindings
func (s *ShellTmuxPane) GetPaneSpecificKeybindings() []key.Binding {
	// No shell-specific keybindings currently
	return []key.Binding{}
}