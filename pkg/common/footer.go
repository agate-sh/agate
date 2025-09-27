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

	// Group shortcuts into categories with new logic
	var paneSpecificShortcuts []Shortcut         // Shortcuts that should be highlighted for current pane
	var globalNonHighlightedShortcuts []Shortcut // Shortcuts that are shown but not highlighted
	var quitShortcut *Shortcut
	var helpShortcut *Shortcut

	for _, shortcut := range shortcuts {
		if shortcut.Key == "q" && shortcut.IsGlobal {
			quitShortcut = &shortcut
		} else if shortcut.Key == "?" && shortcut.IsGlobal {
			helpShortcut = &shortcut
		} else if !shortcut.IsGlobal {
			// Determine if this shortcut should be highlighted based on current focus
			shouldHighlight := false

			if (f.focused == "tmux" || f.focused == "git" || f.focused == "shell") && shortcut.Key == "a" {
				// Only highlight "attach to tmux" when any right pane is focused
				shouldHighlight = true
			} else if f.focused == "agents" && (shortcut.Key == "n" || shortcut.Key == "d") {
				// Highlight worktree shortcuts when left pane focused
				shouldHighlight = true
			}

			if shouldHighlight {
				paneSpecificShortcuts = append(paneSpecificShortcuts, shortcut)
			} else {
				globalNonHighlightedShortcuts = append(globalNonHighlightedShortcuts, shortcut)
			}
		}
	}

	var parts []string

	// Render pane-specific shortcuts (highlighted)
	for i, shortcut := range paneSpecificShortcuts {
		if i > 0 {
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		}

		keyStyle := footerKeyStyle.Bold(true)
		descStyle := footerDescStyle

		// Apply appropriate highlighting color based on current pane focus
		switch f.focused {
		case "tmux":
			// Apply agent color when tmux pane is focused
			agentColor := app.GetCurrentAgentColor()
			keyStyle = keyStyle.Foreground(lipgloss.Color(agentColor))
			descStyle = descStyle.Foreground(lipgloss.Color(agentColor))
		case "git", "shell":
			// Apply gray color when git/shell panes are focused
			keyStyle = keyStyle.Foreground(lipgloss.Color(theme.TextDescription))
			descStyle = descStyle.Foreground(lipgloss.Color(theme.TextDescription))
		case "agents":
			// Apply white color when left pane is focused
			keyStyle = keyStyle.Foreground(lipgloss.Color(theme.TextPrimary))
			descStyle = descStyle.Foreground(lipgloss.Color(theme.TextPrimary))
		}

		part := keyStyle.Render(shortcut.Key) + " " + descStyle.Render(shortcut.Description)
		parts = append(parts, part)
	}

	// Add pipe separator before non-highlighted shortcuts if we have highlighted ones
	if len(paneSpecificShortcuts) > 0 && len(globalNonHighlightedShortcuts) > 0 {
		parts = append(parts, footerSeparatorStyle.Render(" │ "))
	}

	// Render non-highlighted shortcuts (in default gray color)
	for i, shortcut := range globalNonHighlightedShortcuts {
		if len(parts) > 0 && len(paneSpecificShortcuts) == 0 && i > 0 {
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		} else if i > 0 {
			parts = append(parts, footerSeparatorStyle.Render(" • "))
		}

		keyStyle := footerKeyStyle.Bold(true)
		descStyle := footerDescStyle
		// Keep default gray styling

		part := keyStyle.Render(shortcut.Key) + " " + descStyle.Render(shortcut.Description)
		parts = append(parts, part)
	}

	// Add separator before global shortcuts (quit/help)
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
