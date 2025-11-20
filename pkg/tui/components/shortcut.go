package components

import (
	"agate/pkg/tui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// ShortcutVariant defines the styling variant for shortcuts
type ShortcutVariant string

const (
	// ShortcutDefault uses bubbles default styling (all muted, matching "? toggle help • q quit")
	ShortcutDefault ShortcutVariant = "default"

	// ShortcutAgent uses agent color for both key and desc with bold key
	ShortcutAgent ShortcutVariant = "agent"
)

// RenderShortcut renders a single shortcut with the specified variant
// For agent variant, agentColor must be provided (hex color string)
func RenderShortcut(key, desc string, variant ShortcutVariant, agentColor string) string {
	var keyStyle, descStyle lipgloss.Style

	switch variant {
	case ShortcutDefault:
		// Both key and desc use muted color, but key is bold (bubbles default rendering style)
		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted))

	case ShortcutAgent:
		// Both key and desc use agent color, key is bold
		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(agentColor)).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(agentColor))
	}

	return keyStyle.Render(key) + " " + descStyle.Render(desc)
}

// RenderShortcuts renders multiple shortcuts with bullet separators
// shortcuts is a slice of {Key, Desc} pairs
func RenderShortcuts(shortcuts []struct{ Key, Desc string }, variant ShortcutVariant, agentColor string) string {
	if len(shortcuts) == 0 {
		return ""
	}

	var parts []string
	// Use muted color for separator to match bubbles default styling
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted))

	for i, s := range shortcuts {
		if i > 0 {
			parts = append(parts, separatorStyle.Render(" • "))
		}
		parts = append(parts, RenderShortcut(s.Key, s.Desc, variant, agentColor))
	}

	return strings.Join(parts, "")
}

// RenderShortcutFromBinding renders a shortcut from a key.Binding
func RenderShortcutFromBinding(binding key.Binding, variant ShortcutVariant, agentColor string) string {
	help := binding.Help()
	return RenderShortcut(help.Key, help.Desc, variant, agentColor)
}

// RenderShortcutsFromBindings renders multiple shortcuts from key.Bindings
func RenderShortcutsFromBindings(bindings []key.Binding, variant ShortcutVariant, agentColor string) string {
	shortcuts := make([]struct{ Key, Desc string }, len(bindings))
	for i, binding := range bindings {
		help := binding.Help()
		shortcuts[i] = struct{ Key, Desc string }{Key: help.Key, Desc: help.Desc}
	}
	return RenderShortcuts(shortcuts, variant, agentColor)
}

// ParseAndRenderShortcuts parses a shortcut string like "c commit • ↵ attach" and renders it
// This is useful for maintaining compatibility with existing string-based shortcuts
func ParseAndRenderShortcuts(shortcutStr string, variant ShortcutVariant, agentColor string) string {
	parts := strings.Split(shortcutStr, " • ")
	var shortcuts []struct{ Key, Desc string }

	for _, part := range parts {
		tokens := strings.SplitN(strings.TrimSpace(part), " ", 2)
		if len(tokens) >= 2 {
			shortcuts = append(shortcuts, struct{ Key, Desc string }{
				Key:  tokens[0],
				Desc: tokens[1],
			})
		} else if len(tokens) == 1 {
			// Just a key with no description
			shortcuts = append(shortcuts, struct{ Key, Desc string }{
				Key:  tokens[0],
				Desc: "",
			})
		}
	}

	return RenderShortcuts(shortcuts, variant, agentColor)
}
