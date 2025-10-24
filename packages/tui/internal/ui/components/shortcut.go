package components

import (
	"strings"

	"agate/tui/internal/ui/theme"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// ShortcutVariant controls how shortcut text is styled.
type ShortcutVariant string

const (
	// ShortcutDefault renders muted key/hint text (matches legacy UI).
	ShortcutDefault ShortcutVariant = "default"
	// ShortcutAgent renders using the agent color (key bold).
	ShortcutAgent ShortcutVariant = "agent"
)

// RenderShortcut prints a single shortcut with desired styling.
// If variant is ShortcutAgent the agentColor must be provided.
func RenderShortcut(keyText, desc string, variant ShortcutVariant, agentColor string) string {
	var keyStyle, descStyle lipgloss.Style

	switch variant {
	case ShortcutAgent:
		color := lipgloss.Color(agentColor)
		keyStyle = lipgloss.NewStyle().Foreground(color).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(color)
	default:
		color := lipgloss.Color(theme.TextMuted)
		keyStyle = lipgloss.NewStyle().Foreground(color).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(color)
	}

	return keyStyle.Render(keyText) + " " + descStyle.Render(desc)
}

// RenderShortcuts renders multiple shortcuts separated by " • ".
func RenderShortcuts(shortcuts []struct{ Key, Desc string }, variant ShortcutVariant, agentColor string) string {
	if len(shortcuts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(shortcuts)*2)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted)).Render(" • ")

	for i, s := range shortcuts {
		if i > 0 {
			parts = append(parts, sep)
		}
		parts = append(parts, RenderShortcut(s.Key, s.Desc, variant, agentColor))
	}

	return strings.Join(parts, "")
}

// RenderShortcutFromBinding renders a shortcut from a key binding definition.
func RenderShortcutFromBinding(binding key.Binding, variant ShortcutVariant, agentColor string) string {
	help := binding.Help()
	return RenderShortcut(help.Key, help.Desc, variant, agentColor)
}

// RenderShortcutsFromBindings renders a slice of bindings using the default separator.
func RenderShortcutsFromBindings(bindings []key.Binding, variant ShortcutVariant, agentColor string) string {
	shortcuts := make([]struct{ Key, Desc string }, len(bindings))
	for i, binding := range bindings {
		help := binding.Help()
		shortcuts[i] = struct{ Key, Desc string }{Key: help.Key, Desc: help.Desc}
	}
	return RenderShortcuts(shortcuts, variant, agentColor)
}
