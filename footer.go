package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Footer manages the bottom footer bar with keyboard shortcuts
type Footer struct {
	width        int
	height       int
	focused      string // "left" or "right"
	agentConfig  AgentConfig
	showHelp     bool // Whether help dialog is shown
}

// Styling for footer elements
var (
	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	footerDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	footerSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

	footerActiveKeyStyle = lipgloss.NewStyle().
				Bold(true)

	footerStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

// NewFooter creates a new footer component
func NewFooter() *Footer {
	return &Footer{
		height: 1,
	}
}

// SetSize updates the footer dimensions
func (f *Footer) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetFocus updates which pane is focused
func (f *Footer) SetFocus(focused string) {
	f.focused = focused
}

// SetAgentConfig updates the agent configuration for coloring
func (f *Footer) SetAgentConfig(config AgentConfig) {
	f.agentConfig = config
}

// SetShowHelp updates whether help is shown
func (f *Footer) SetShowHelp(show bool) {
	f.showHelp = show
}

// GetShortcuts returns the current shortcuts to display
func (f *Footer) GetShortcuts() []Shortcut {
	var shortcuts []Shortcut

	// Add pane-specific shortcuts
	if f.focused == "right" {
		// Tmux pane shortcuts
		shortcuts = append(shortcuts, TmuxShortcuts...)
	} else if f.focused == "left" {
		// Left pane shortcuts
		shortcuts = append(shortcuts, LeftPaneShortcuts...)
	}

	// Always add global shortcuts at the end (after separator)
	shortcuts = append(shortcuts, GlobalShortcuts...)

	return shortcuts
}

// View renders the footer
func (f *Footer) View() string {
	if f.width == 0 {
		return ""
	}

	shortcuts := f.GetShortcuts()
	if len(shortcuts) == 0 {
		return ""
	}

	var parts []string
	globalStarted := false

	for i, shortcut := range shortcuts {
		// Check if this is a global shortcut
		isGlobal := shortcut.IsGlobal

		// Add separator before global shortcuts
		if isGlobal && !globalStarted {
			if len(parts) > 0 {
				parts = append(parts, footerSeparatorStyle.Render(" │ "))
			}
			globalStarted = true
		} else if i > 0 && !isGlobal {
			// Regular separator between non-global shortcuts
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		}

		// Style the shortcut based on context
		keyStyle := footerKeyStyle
		descStyle := footerDescStyle

		// Apply agent color to tmux shortcuts when right pane is focused
		if f.focused == "right" && !isGlobal {
			keyStyle = keyStyle.Copy().Foreground(lipgloss.Color(f.agentConfig.BorderColor))
			descStyle = descStyle.Copy().Foreground(lipgloss.Color(f.agentConfig.BorderColor))
		}

		// Make key bold
		keyStyle = keyStyle.Copy().Bold(true)

		// Render the shortcut
		part := keyStyle.Render(shortcut.Key) + " " + descStyle.Render(shortcut.Description)
		parts = append(parts, part)
	}

	content := strings.Join(parts, "")

	// Center the footer content
	return lipgloss.Place(
		f.width,
		f.height,
		lipgloss.Center,
		lipgloss.Center,
		footerStyle.Render(content),
	)
}