package common

import (
	"github.com/charmbracelet/bubbles/key"
)

// GlobalKeyMap defines global keybindings that work across all panes
//
// Note: This contains both truly global keybindings (like quit, help) and
// conceptually pane-specific keybindings that need to be globally accessible.
// For example:
// - 'n' (new agent) conceptually belongs to the repos pane but works globally
// - 'r' (add repo) conceptually belongs to the repos pane but works globally
// - Git pane actions like "open in editor" are pane-specific and should be handled by the pane
//
// TODO: As pane components mature, consider moving more keybindings to individual panes
// while keeping them globally accessible through the pane interface.
type GlobalKeyMap struct {
	// Truly global keys - work from any pane, any context
	Quit         key.Binding // q, Ctrl+C - quit application
	Keybindings  key.Binding // ? - show help
	DebugOverlay key.Binding // Ctrl+D - toggle debug overlay

	// Global navigation keys - work within any focusable pane
	Up   key.Binding // ↑, k - move up in active pane
	Down key.Binding // ↓, j - move down in active pane

	// Two-level navigation system
	TabNextPane key.Binding // Tab - cycle between top-level panes (Agents ↔ Session)
	NextSubPane key.Binding // Ctrl+] - cycle forward within session sub-panes (Tmux → Git → Shell)
	PrevSubPane key.Binding // Ctrl+[ - cycle backward within session sub-panes (Shell → Git → Tmux)

	// Repository and session management - conceptually belong to repos pane
	// but are globally accessible for convenience
	AddRepo    key.Binding // r - add repository (repos pane action, but global)
	NewSession key.Binding // n - create new session (repos pane action, but global)

	// Session interaction - conceptually belongs to panes but globally accessible
	AttachAgent key.Binding // a - attach to agent tmux session
	Commit      key.Binding // c - create commit (git pane action, but global)
	DetachTmux  key.Binding // Ctrl+Q - detach from tmux session

	// Dialog actions - global because dialogs overlay all content
	Confirm key.Binding // Enter, y - confirm dialog action
	Cancel  key.Binding // Esc, n - cancel dialog

	// List interaction - used by multiple panes (repos, git, etc.)
	Filter      key.Binding // / - start filtering
	ClearFilter key.Binding // Esc - clear filter

	// Git pane actions
	OpenInEditor key.Binding // Enter - open selected file in editor
}

// GlobalKeys is the single source of truth for all keybindings in the application
var GlobalKeys = &GlobalKeyMap{
	// Global keys
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Keybindings: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "keybindings"),
	),

	// Debug
	DebugOverlay: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "debug overlay"),
	),

	// Navigation
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),

	// Two-level navigation system
	TabNextPane: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "cycle pane"),
	),
	NextSubPane: key.NewBinding(
		key.WithKeys("ctrl+]"),
		key.WithHelp("ctrl+]", "next sub-pane"),
	),
	PrevSubPane: key.NewBinding(
		key.WithKeys("ctrl+["),
		key.WithHelp("ctrl+[", "prev sub-pane"),
	),

	// Repository and Session management
	AddRepo: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "add repo"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new agent"),
	),

	// Session interaction
	AttachAgent: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "attach to agent"),
	),
	Commit: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "commit"),
	),
	DetachTmux: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "detach from tmux"),
	),

	// List navigation
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter list"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear filter"),
	),

	// Dialog actions
	Confirm: key.NewBinding(
		key.WithKeys("enter", "y"),
		key.WithHelp("↵/y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "n"),
		key.WithHelp("esc/n", "cancel"),
	),

	// Git pane actions
	OpenInEditor: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "open in editor"),
	),
}

// FormatTitleShortcut formats a keybinding for display in pane title bars
// Example: "[a to attach]", "[r to add repo]"
func FormatTitleShortcut(binding key.Binding) string {
	help := binding.Help()
	// Special case for multi-key shortcuts
	if help.Key == "↑/k" {
		return "[↑/k to " + help.Desc + "]"
	}
	if help.Key == "↓/j" {
		return "[↓/j to " + help.Desc + "]"
	}
	// For single keys, add "to" for better readability
	return "[" + help.Key + " to " + help.Desc + "]"
}

// FormatFooterShortcut formats a keybinding for display in the footer
// Example: "a: attach", "r: add repo"
func FormatFooterShortcut(binding key.Binding) string {
	help := binding.Help()
	return help.Key + ": " + help.Desc
}

// FormatCompactShortcut formats a keybinding in compact form for title bars
// Example: "[a]", "[ctrl+q]"
func FormatCompactShortcut(binding key.Binding) string {
	return "[" + binding.Help().Key + "]"
}

// ShortHelp returns a slice of key bindings to show in the short help view
func (k *GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Keybindings,
		k.Quit,
	}
}

// FullHelp returns a slice of key bindings to show in the full help view
func (k *GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Keybindings},                     // Global
		{k.TabNextPane, k.NextSubPane, k.PrevSubPane}, // Two-level navigation
		{k.Up, k.Down},                              // List navigation
		{k.NewSession, k.AttachAgent, k.Commit},     // Quick actions (n, a, c)
		{k.AddRepo},                                 // Repository
		{k.DetachTmux},                              // Session
		{k.Filter, k.ClearFilter},                   // Filtering
		{k.Confirm, k.Cancel},                       // Dialogs
	}
}

// GetHelpSections returns help sections with categorized keybindings
func (k *GlobalKeyMap) GetHelpSections() map[string][]key.Binding {
	return map[string][]key.Binding{
		"Global": {
			k.Quit,
			k.Keybindings,
		},
		"Pane Navigation": {
			k.TabNextPane,
			k.NextSubPane,
			k.PrevSubPane,
		},
		"List Navigation": {
			k.Up,
			k.Down,
		},
		"Quick Actions": {
			k.NewSession,
			k.AttachAgent,
			k.Commit,
		},
		"Repository Management": {
			k.AddRepo,
		},
		"Session Interaction": {
			k.DetachTmux,
		},
		"List Controls": {
			k.Filter,
			k.ClearFilter,
		},
		"Dialog Actions": {
			k.Confirm,
			k.Cancel,
		},
		"Help": {
			k.DebugOverlay,
		},
	}
}

// SetEnabled allows enabling/disabling specific keybindings based on context
func (k *GlobalKeyMap) SetEnabled(binding *key.Binding, enabled bool) {
	binding.SetEnabled(enabled)
}

// DisableDialogKeys disables dialog-specific keybindings
func (k *GlobalKeyMap) DisableDialogKeys() {
	k.Confirm.SetEnabled(false)
	k.Cancel.SetEnabled(false)
}

// EnableDialogKeys enables dialog-specific keybindings
func (k *GlobalKeyMap) EnableDialogKeys() {
	k.Confirm.SetEnabled(true)
	k.Cancel.SetEnabled(true)
}
