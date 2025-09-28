package panes

import (
	"agate/pkg/gui/components"
	"agate/pkg/gui/theme"
	"agate/pkg/tmux"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShellPane manages the display of shell content using tmux like the TmuxPane
type ShellPane struct {
	*components.BasePane
	session *tmux.TmuxSession
	content string
}

// NewShellPane creates a new ShellPane instance
func NewShellPane() *ShellPane {
	return &ShellPane{
		BasePane: components.NewBasePane(3, "Shell"), // Pane index 3
		content:  "",
	}
}

// SetSession sets the tmux session for this pane
func (s *ShellPane) SetSession(session *tmux.TmuxSession) {
	s.session = session
	s.updateContent()
}

// SetContent updates the shell content to display
func (s *ShellPane) SetContent(content string) {
	s.content = content
}

// updateContent refreshes the content from the tmux session
func (s *ShellPane) updateContent() {
	if s.session != nil {
		content, err := s.session.CapturePaneContent()
		if err == nil {
			s.content = content
		}
	}
}

// SetSize updates the dimensions and resizes the tmux session
func (s *ShellPane) SetSize(width, height int) {
	s.BasePane.SetSize(width, height)
	if s.session != nil {
		// Set tmux session size to match pane content area
		s.session.SetDetachedSize(width, height)
	}
}

// GetTitleStyle returns the plain title style for the shell pane
func (s *ShellPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if s.IsActive() {
		// When active, show enter to attach and ctrl+q to detach
		shortcuts = "[↵ attach, ctrl+q detach]"
	} else {
		// When not active, show pane number
		shortcuts = "[3]"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Shell",
		Shortcuts: shortcuts,
	}
}

// View renders the shell pane content
func (s *ShellPane) View() string {
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
func (s *ShellPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	// ShellPane doesn't handle specific update messages currently
	return s, nil
}

// HandleKey processes keyboard input when the pane is active
func (s *ShellPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	// ShellPane uses tmux attach/detach like TmuxPane - no direct key handling
	return false, nil
}

// GetPaneSpecificKeybindings returns shell pane specific keybindings
func (s *ShellPane) GetPaneSpecificKeybindings() []key.Binding {
	// No shell-specific keybindings currently
	return []key.Binding{}
}
