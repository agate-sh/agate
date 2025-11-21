package components

import (
	"agate/pkg/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

// GlobalFooter represents the global footer with keyboard shortcuts
type GlobalFooter struct {
	width int
}

// NewGlobalFooter creates a new global footer component
func NewGlobalFooter() *GlobalFooter {
	return &GlobalFooter{
		width: 80,
	}
}

// SetWidth sets the width of the footer
func (f *GlobalFooter) SetWidth(width int) {
	f.width = width
}

// View renders the global footer
func (f *GlobalFooter) View() string {
	shortcutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.AgateColor)).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	bulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))

	// Build the shortcuts line: ⌥p Commands • ⌥n New session
	shortcuts := shortcutStyle.Render("⌥n") + 
		labelStyle.Render(" New session") + 
		bulletStyle.Render(" • ") +
		shortcutStyle.Render("⌥p") +
		labelStyle.Render(" Commands") 

	// Center the shortcuts horizontally
	centered := lipgloss.Place(
		f.width,
		1,
		lipgloss.Center,
		lipgloss.Center,
		shortcuts,
	)

	return centered
}
