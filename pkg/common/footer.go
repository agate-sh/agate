package common

import (
	"agate/pkg/app"
	"strings"

	"agate/pkg/gui/theme"

	"github.com/charmbracelet/lipgloss"
)

// Footer manages the bottom footer bar with keyboard shortcuts
type Footer struct {
	width           int
	height          int
	focused         string // "left" or "right"
	showHelp        bool   // Whether help dialog is shown
	mode            string // "preview", "focused", "attached"
	shortcutOverlay *ShortcutOverlay
}

// Styling for footer elements
var (
	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription))

	footerDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription))

	footerSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.SeparatorColor))

	footerStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

// NewFooter creates a new footer component
func NewFooter() *Footer {
	return &Footer{
		height: 1,
	}
}

// SetShortcutOverlay sets the shortcut overlay for the footer
func (f *Footer) SetShortcutOverlay(overlay *ShortcutOverlay) {
	f.shortcutOverlay = overlay
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

// SetShowHelp updates whether help is shown
func (f *Footer) SetShowHelp(show bool) {
	f.showHelp = show
}

// SetMode updates the current interaction mode
func (f *Footer) SetMode(mode string) {
	f.mode = mode
}

// GetShortcuts returns the current shortcuts to display based on mode
func (f *Footer) GetShortcuts() []Shortcut {
	if f.shortcutOverlay != nil {
		f.shortcutOverlay.SetFocus(f.focused)
		f.shortcutOverlay.SetMode(f.mode)
		return f.shortcutOverlay.FormatShortcuts()
	}
	return []Shortcut{}
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

	// Find the core global shortcuts and help/quit shortcuts
	var coreShortcuts []Shortcut // n, a, s shortcuts with agent color
	var quitShortcut *Shortcut
	var helpShortcut *Shortcut

	for _, shortcut := range shortcuts {
		if shortcut.Key == "q" && shortcut.IsGlobal {
			quitShortcut = &shortcut
		} else if shortcut.Key == "?" && shortcut.IsGlobal {
			helpShortcut = &shortcut
		} else if shortcut.Key == "n" || shortcut.Key == "a" || shortcut.Key == "s" {
			// Core global shortcuts that should be styled with agent colors
			coreShortcuts = append(coreShortcuts, shortcut)
		}
	}

	var parts []string

	// Render core global shortcuts (n, a, s) with agent colors
	agentColor := app.GetCurrentAgentColor()
	for i, shortcut := range coreShortcuts {
		if i > 0 {
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		}

		keyStyle := footerKeyStyle.Bold(true).Foreground(lipgloss.Color(agentColor))
		descStyle := footerDescStyle.Foreground(lipgloss.Color(agentColor))

		part := keyStyle.Render(shortcut.Key) + " " + descStyle.Render(shortcut.Description)
		parts = append(parts, part)
	}

	// Add pipe separator before help/quit shortcuts
	if len(parts) > 0 && (quitShortcut != nil || helpShortcut != nil) {
		parts = append(parts, footerSeparatorStyle.Render(" │ "))
	}

	// Render quit shortcut
	if quitShortcut != nil {
		keyStyle := footerKeyStyle.Bold(true)
		descStyle := footerDescStyle
		part := keyStyle.Render(quitShortcut.Key) + " " + descStyle.Render(quitShortcut.Description)
		parts = append(parts, part)
	}

	// Add consistent spacing before help
	if helpShortcut != nil {
		if len(parts) > 0 {
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		}
		keyStyle := footerKeyStyle.Bold(true)
		descStyle := footerDescStyle
		part := keyStyle.Render(helpShortcut.Key) + " " + descStyle.Render(helpShortcut.Description)
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
