package components

import (
	"agate/pkg/gui/theme"
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ButtonVariant defines the visual style of a button
type ButtonVariant string

const (
	ButtonVariantDefault ButtonVariant = "default" // Default gray button
	ButtonVariantAgent   ButtonVariant = "agent"   // Agent color button (requires agentColor)
	ButtonVariantDanger  ButtonVariant = "danger"  // Red/error button for destructive actions
)

// Button represents a styled button component
type Button struct {
	label       string
	shortcut    string        // Key to trigger the button (e.g., "c", "d", "n")
	variant     ButtonVariant // Visual style variant
	disabled    bool          // Whether the button is disabled
	agentColor  string        // Color for agent variant (optional)
	width       int           // Button width
}

// NewButton creates a new button
func NewButton(label, shortcut string, variant ButtonVariant) *Button {
	return &Button{
		label:    label,
		shortcut: shortcut,
		variant:  variant,
		disabled: false,
	}
}

// SetDisabled sets whether the button is disabled
func (b *Button) SetDisabled(disabled bool) {
	b.disabled = disabled
}

// SetAgentColor sets the color for agent variant buttons
func (b *Button) SetAgentColor(color string) {
	b.agentColor = color
}

// SetWidth sets the button width
func (b *Button) SetWidth(width int) {
	b.width = width
}

// Render renders the button
func (b *Button) Render() string {
	// Button text with shortcut
	text := fmt.Sprintf("%s (%s)", b.label, b.shortcut)

	// Base style
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		Bold(true)

	if b.width > 0 {
		style = style.Width(b.width).Align(lipgloss.Center)
	}

	// Apply colors based on state
	if b.disabled {
		// Disabled state - same for all variants
		style = style.
			Background(lipgloss.Color(theme.TextDescription)).
			Foreground(lipgloss.Color(theme.TextMuted))
	} else {
		// Enabled state - variant-specific colors
		switch b.variant {
		case ButtonVariantAgent:
			// Use agent color if provided, otherwise fall back to default
			if b.agentColor != "" {
				style = style.Background(lipgloss.Color(b.agentColor))
			} else {
				style = style.Background(lipgloss.Color(theme.BorderMuted))
			}
		case ButtonVariantDanger:
			// Red/error color for destructive actions
			style = style.Background(lipgloss.Color(theme.ErrorStatus))
		case ButtonVariantDefault:
			fallthrough
		default:
			// Default gray button
			style = style.Background(lipgloss.Color(theme.BorderMuted))
		}
	}

	return style.Render(text)
}
