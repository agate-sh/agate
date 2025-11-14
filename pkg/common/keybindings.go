package common

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	// sessionShiftTabSequences captures the escape sequences emitted by common
	// terminals (including Ghostty, iTerm2, kitty, and xterm) when pressing
	// Shift+Tab. This keeps the navigation working even when the terminal sends
	// raw escape codes instead of Bubble Tea's named key.
	sessionShiftTabSequences = []string{
		"\x1b[Z", // Shift+Tab (reverse tab)
	}
)

// GlobalKeyMap defines global keybindings that work across all panes
//
// Note: This contains both truly global keybindings (like quit, help) and
// conceptually pane-specific keybindings that need to be globally accessible.
// For example:
// - 'n' (new agent) conceptually belongs to the repos pane but works globally
// - Git pane actions like "open in editor" are pane-specific and should be handled by the pane
//
// TODO: As pane components mature, consider moving more keybindings to individual panes
// while keeping them globally accessible through the pane interface.
type GlobalKeyMap struct {
	// Truly global keys - work from any pane, any context
	Quit         key.Binding // Ctrl+C - quit application
	Keybindings  key.Binding // ? - show help
	DebugOverlay key.Binding // Ctrl+D - toggle debug overlay

	// Global navigation keys - work within any focusable pane
	Up   key.Binding // ↑, k - move up in active pane
	Down key.Binding // ↓, j - move down in active pane

	// Pane navigation with Option/Alt keys
	AgentsPane    key.Binding // Alt+A - jump to agents pane
	SessionPane   key.Binding // Alt+S - jump to session pane
	ChangesPane   key.Binding // Alt+C - jump to changes pane

	// Agent cycling (only works on session pane)
	NextAgent     key.Binding // Tab - cycle to next agent (session pane only)
	PrevAgent     key.Binding // Shift+Tab - cycle to previous agent (session pane only)

	// Repository and session management - conceptually belong to repos pane
	// but are globally accessible for convenience
	NewSession key.Binding // Alt+N - create new session (repos pane action, but global)

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
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Keybindings: key.NewBinding(
		key.WithKeys("alt+p"),
		key.WithHelp("⌥p", "Commands"),
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

	// Pane navigation with Option/Alt keys
	AgentsPane: key.NewBinding(
		key.WithKeys("alt+s"),
		key.WithHelp("⌥s", "sessions pane"),
	),
	SessionPane: key.NewBinding(
		key.WithKeys("alt+a"),
		key.WithHelp("⌥a", "agents pane"),
	),
	ChangesPane: key.NewBinding(
		key.WithKeys("alt+c"),
		key.WithHelp("⌥c", "changes pane"),
	),

	// Agent cycling (only works on session pane)
	NextAgent: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next agent"),
	),
	PrevAgent: key.NewBinding(
		key.WithKeys(append([]string{"shift+tab"}, sessionShiftTabSequences...)...),
		key.WithHelp("⇧tab", "prev agent"),
	),

	// Repository and Session management
	NewSession: key.NewBinding(
		key.WithKeys("alt+n"),
		key.WithHelp("⌥n", "new agent"),
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
		{k.Quit, k.Keybindings},                           // Global
		{k.AgentsPane, k.SessionPane, k.ChangesPane},      // Pane navigation
		{k.NextAgent, k.PrevAgent},                        // Agent cycling
		{k.Up, k.Down},                                    // List navigation
		{k.NewSession, k.AttachAgent, k.Commit},           // Quick actions (n, a, c)
		{k.DetachTmux},                                    // Session
		{k.Filter, k.ClearFilter},                         // Filtering
		{k.Confirm, k.Cancel},                             // Dialogs
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
			k.AgentsPane,
			k.SessionPane,
			k.ChangesPane,
		},
		"Agent Cycling": {
			k.NextAgent,
			k.PrevAgent,
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

// IsPrevAgentKey reports whether the provided key message should trigger a
// transition to the previous agent. This explicitly handles common
// escape sequences for Shift+Tab emitted by popular terminals.
func IsPrevAgentKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, GlobalKeys.PrevAgent) || matchesSequence(msg, sessionShiftTabSequences)
}

func matchesSequence(msg tea.KeyMsg, sequences []string) bool {
	keyStr := msg.String()
	for _, seq := range sequences {
		if keyStr == seq {
			return true
		}
	}
	return false
}
