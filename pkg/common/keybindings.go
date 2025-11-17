package common

import (
	"agate/internal/debug"
	"agate/pkg/config"
	"strings"

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
	AgentsPane  key.Binding // Alt+A - jump to agents pane
	SessionPane key.Binding // Alt+S - jump to session pane
	ChangesPane key.Binding // Alt+C - jump to changes pane

	// Agent cycling (only works on session pane)
	NextAgent key.Binding // Tab - cycle to next agent (session pane only)
	PrevAgent key.Binding // Shift+Tab - cycle to previous agent (session pane only)

	// Agent selection (new session creation)
	AddRemoveAgents key.Binding // Tab - add/remove agents (new session view)

	// Repository and session management - conceptually belong to repos pane
	// but are globally accessible for convenience
	NewSession key.Binding // Alt+N - create new session (repos pane action, but global)

	// Session interaction - conceptually belongs to panes but globally accessible
	AttachAgent  key.Binding // a - attach to agent tmux session
	MergeChanges key.Binding // m - merge changes to main worktree (global)
	DetachTmux   key.Binding // Ctrl+Q - detach from tmux session

	// Dialog actions - global because dialogs overlay all content
	Confirm key.Binding // Enter, y - confirm dialog action
	Cancel  key.Binding // Esc, n - cancel dialog

	// List interaction - used by multiple panes (repos, git, etc.)
	Filter      key.Binding // / - start filtering
	ClearFilter key.Binding // Esc - clear filter

	// Git pane actions
	OpenInEditor key.Binding // Enter - open selected file in editor
}

// GlobalKeys is the single source of truth for all keybindings in the application.
// Components that use GlobalKeys (like ShortcutOverlay and HelpDialog) hold pointers
// to this map, so they automatically see changes when keybindings are modified via
// ApplyUserConfig. Since Bubble Tea calls View() on every update cycle, components
// will automatically reflect updated keybindings without requiring explicit refresh.
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
		key.WithKeys("ctrl+s"),
		key.WithHelp("⌃s", "sessions pane"),
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

	// Agent selection (new session creation)
	AddRemoveAgents: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "add/remove agents"),
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
	MergeChanges: key.NewBinding(
		key.WithKeys("alt+m"),
		key.WithHelp("⌥m", "merge changes"),
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
		{k.Quit, k.Keybindings},                       // Global
		{k.AgentsPane, k.SessionPane, k.ChangesPane},  // Pane navigation
		{k.NextAgent, k.PrevAgent},                    // Agent cycling
		{k.Up, k.Down},                                // List navigation
		{k.NewSession, k.AttachAgent, k.MergeChanges}, // Quick actions (n, a, m)
		{k.DetachTmux},                                // Session
		{k.Filter, k.ClearFilter},                     // Filtering
		{k.Confirm, k.Cancel},                         // Dialogs
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
			k.MergeChanges,
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

// formatHelpKey formats a list of keys into a help string
// Example: ["ctrl+c"] -> "^c", ["up", "k"] -> "↑/k", ["ctrl+alt+s"] -> "^⌥s"
func formatHelpKey(keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	formatted := make([]string, len(keys))
	for i, key := range keys {
		formatted[i] = formatKey(key)
	}
	return strings.Join(formatted, "/")
}

// formatKey converts a single key string to its display format
// Modifiers with '+' are converted without delimiter: "ctrl+y" -> "^y"
// Single keys are converted: "up" -> "↑", "enter" -> "↵"
func formatKey(key string) string {
	parts := strings.Split(key, "+")
	if len(parts) == 1 {
		// Single key - convert direction/enter keys
		return convertKey(parts[0])
	}

	// Multiple parts (modifiers + key) - convert each and join without delimiter
	var result strings.Builder
	for _, part := range parts {
		result.WriteString(convertKey(part))
	}
	return result.String()
}

// convertKey converts a single key part to its symbol
func convertKey(key string) string {
	switch key {
	case "ctrl":
		return "^"
	case "alt":
		return "⌥"
	case "shift":
		return "⇧"
	case "enter":
		return "↵"
	case "up":
		return "↑"
	case "down":
		return "↓"
	default:
		return key
	}
}

// formatHelpDesc extracts description from existing binding or uses default
func formatHelpDesc(existing key.Binding, defaultDesc string) string {
	if existing.Help().Desc != "" {
		return existing.Help().Desc
	}
	return defaultDesc
}

// ApplyUserConfig applies user configuration overrides to GlobalKeys
// This should be called once at application startup after loading user config.
//
// Note: Components that use GlobalKeys (like ShortcutOverlay and HelpDialog) hold
// pointers to GlobalKeyMap, so they will automatically see changes when GlobalKeys
// is modified. However, Bubble Tea only re-renders on messages/updates, so if this
// is called at runtime, ensure a refresh is triggered (e.g., by sending a message
// to the model or calling View() again).
func ApplyUserConfig(userConfig *config.UserConfig) {
	if userConfig == nil {
		debug.DebugLog("ApplyUserConfig: userConfig is nil, skipping")
		return
	}

	kb := userConfig.Keybindings
	overrideCount := 0

	// Apply overrides only if user provided them
	if len(kb.Quit) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'quit' keybinding with %v", kb.Quit)
		GlobalKeys.Quit = key.NewBinding(
			key.WithKeys(kb.Quit...),
			key.WithHelp(formatHelpKey(kb.Quit), formatHelpDesc(GlobalKeys.Quit, "quit")),
		)
		overrideCount++
	}

	if len(kb.Keybindings) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'keybindings' keybinding with %v", kb.Keybindings)
		GlobalKeys.Keybindings = key.NewBinding(
			key.WithKeys(kb.Keybindings...),
			key.WithHelp(formatHelpKey(kb.Keybindings), formatHelpDesc(GlobalKeys.Keybindings, "Commands")),
		)
		overrideCount++
	}

	if len(kb.DebugOverlay) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'debug_overlay' keybinding with %v", kb.DebugOverlay)
		GlobalKeys.DebugOverlay = key.NewBinding(
			key.WithKeys(kb.DebugOverlay...),
			key.WithHelp(formatHelpKey(kb.DebugOverlay), formatHelpDesc(GlobalKeys.DebugOverlay, "debug overlay")),
		)
		overrideCount++
	}

	if len(kb.Up) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'up' keybinding with %v", kb.Up)
		GlobalKeys.Up = key.NewBinding(
			key.WithKeys(kb.Up...),
			key.WithHelp(formatHelpKey(kb.Up), formatHelpDesc(GlobalKeys.Up, "move up")),
		)
		overrideCount++
	}

	if len(kb.Down) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'down' keybinding with %v", kb.Down)
		GlobalKeys.Down = key.NewBinding(
			key.WithKeys(kb.Down...),
			key.WithHelp(formatHelpKey(kb.Down), formatHelpDesc(GlobalKeys.Down, "move down")),
		)
		overrideCount++
	}

	if len(kb.AgentsPane) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'agents_pane' keybinding with %v", kb.AgentsPane)
		GlobalKeys.AgentsPane = key.NewBinding(
			key.WithKeys(kb.AgentsPane...),
			key.WithHelp(formatHelpKey(kb.AgentsPane), formatHelpDesc(GlobalKeys.AgentsPane, "sessions pane")),
		)
		overrideCount++
	}

	if len(kb.SessionPane) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'session_pane' keybinding with %v", kb.SessionPane)
		GlobalKeys.SessionPane = key.NewBinding(
			key.WithKeys(kb.SessionPane...),
			key.WithHelp(formatHelpKey(kb.SessionPane), formatHelpDesc(GlobalKeys.SessionPane, "agents pane")),
		)
		overrideCount++
	}

	if len(kb.ChangesPane) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'changes_pane' keybinding with %v", kb.ChangesPane)
		GlobalKeys.ChangesPane = key.NewBinding(
			key.WithKeys(kb.ChangesPane...),
			key.WithHelp(formatHelpKey(kb.ChangesPane), formatHelpDesc(GlobalKeys.ChangesPane, "changes pane")),
		)
		overrideCount++
	}

	if len(kb.NextAgent) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'next_agent' keybinding with %v", kb.NextAgent)
		GlobalKeys.NextAgent = key.NewBinding(
			key.WithKeys(kb.NextAgent...),
			key.WithHelp(formatHelpKey(kb.NextAgent), formatHelpDesc(GlobalKeys.NextAgent, "next agent")),
		)
		overrideCount++
	}

	if len(kb.PrevAgent) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'prev_agent' keybinding with %v", kb.PrevAgent)
		// Preserve shift+tab sequences for prev agent
		prevKeys := append(kb.PrevAgent, sessionShiftTabSequences...)
		GlobalKeys.PrevAgent = key.NewBinding(
			key.WithKeys(prevKeys...),
			key.WithHelp(formatHelpKey(kb.PrevAgent), formatHelpDesc(GlobalKeys.PrevAgent, "prev agent")),
		)
		overrideCount++
	}

	if len(kb.AddRemoveAgents) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'add_remove_agents' keybinding with %v", kb.AddRemoveAgents)
		GlobalKeys.AddRemoveAgents = key.NewBinding(
			key.WithKeys(kb.AddRemoveAgents...),
			key.WithHelp(formatHelpKey(kb.AddRemoveAgents), formatHelpDesc(GlobalKeys.AddRemoveAgents, "add/remove agents")),
		)
		overrideCount++
	}

	if len(kb.NewSession) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'new_session' keybinding with %v", kb.NewSession)
		GlobalKeys.NewSession = key.NewBinding(
			key.WithKeys(kb.NewSession...),
			key.WithHelp(formatHelpKey(kb.NewSession), formatHelpDesc(GlobalKeys.NewSession, "new agent")),
		)
		overrideCount++
	}

	if len(kb.AttachAgent) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'attach_agent' keybinding with %v", kb.AttachAgent)
		GlobalKeys.AttachAgent = key.NewBinding(
			key.WithKeys(kb.AttachAgent...),
			key.WithHelp(formatHelpKey(kb.AttachAgent), formatHelpDesc(GlobalKeys.AttachAgent, "attach to agent")),
		)
		overrideCount++
	}

	if len(kb.DetachTmux) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'detach_tmux' keybinding with %v", kb.DetachTmux)
		GlobalKeys.DetachTmux = key.NewBinding(
			key.WithKeys(kb.DetachTmux...),
			key.WithHelp(formatHelpKey(kb.DetachTmux), formatHelpDesc(GlobalKeys.DetachTmux, "detach from tmux")),
		)
		overrideCount++
	}

	if len(kb.Confirm) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'confirm' keybinding with %v", kb.Confirm)
		GlobalKeys.Confirm = key.NewBinding(
			key.WithKeys(kb.Confirm...),
			key.WithHelp(formatHelpKey(kb.Confirm), formatHelpDesc(GlobalKeys.Confirm, "confirm")),
		)
		overrideCount++
	}

	if len(kb.Cancel) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'cancel' keybinding with %v", kb.Cancel)
		GlobalKeys.Cancel = key.NewBinding(
			key.WithKeys(kb.Cancel...),
			key.WithHelp(formatHelpKey(kb.Cancel), formatHelpDesc(GlobalKeys.Cancel, "cancel")),
		)
		overrideCount++
	}

	if len(kb.Filter) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'filter' keybinding with %v", kb.Filter)
		GlobalKeys.Filter = key.NewBinding(
			key.WithKeys(kb.Filter...),
			key.WithHelp(formatHelpKey(kb.Filter), formatHelpDesc(GlobalKeys.Filter, "filter list")),
		)
		overrideCount++
	}

	if len(kb.ClearFilter) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'clear_filter' keybinding with %v", kb.ClearFilter)
		GlobalKeys.ClearFilter = key.NewBinding(
			key.WithKeys(kb.ClearFilter...),
			key.WithHelp(formatHelpKey(kb.ClearFilter), formatHelpDesc(GlobalKeys.ClearFilter, "clear filter")),
		)
		overrideCount++
	}

	if len(kb.OpenInEditor) > 0 {
		debug.DebugLog("ApplyUserConfig: overriding 'open_in_editor' keybinding with %v", kb.OpenInEditor)
		GlobalKeys.OpenInEditor = key.NewBinding(
			key.WithKeys(kb.OpenInEditor...),
			key.WithHelp(formatHelpKey(kb.OpenInEditor), formatHelpDesc(GlobalKeys.OpenInEditor, "open in editor")),
		)
		overrideCount++
	}

	debug.DebugLog("ApplyUserConfig: applied %d keybinding override(s)", overrideCount)

	// Log the complete structure of GlobalKeys after applying config
	LogGlobalKeys("ApplyUserConfig: GlobalKeys structure after applying config:")
}

// LogGlobalKeys logs the complete structure of GlobalKeys to the debug log.
// This function can be called from anywhere to inspect the current state of keybindings.
func LogGlobalKeys(prefix string) {
	debug.DebugLog("%s", prefix)
	debug.DebugLog("  Quit: key=%s, desc=%s", GlobalKeys.Quit.Help().Key, GlobalKeys.Quit.Help().Desc)
	debug.DebugLog("  Keybindings: key=%s, desc=%s", GlobalKeys.Keybindings.Help().Key, GlobalKeys.Keybindings.Help().Desc)
	debug.DebugLog("  DebugOverlay: key=%s, desc=%s", GlobalKeys.DebugOverlay.Help().Key, GlobalKeys.DebugOverlay.Help().Desc)
	debug.DebugLog("  Up: key=%s, desc=%s", GlobalKeys.Up.Help().Key, GlobalKeys.Up.Help().Desc)
	debug.DebugLog("  Down: key=%s, desc=%s", GlobalKeys.Down.Help().Key, GlobalKeys.Down.Help().Desc)
	debug.DebugLog("  AgentsPane: key=%s, desc=%s", GlobalKeys.AgentsPane.Help().Key, GlobalKeys.AgentsPane.Help().Desc)
	debug.DebugLog("  SessionPane: key=%s, desc=%s", GlobalKeys.SessionPane.Help().Key, GlobalKeys.SessionPane.Help().Desc)
	debug.DebugLog("  ChangesPane: key=%s, desc=%s", GlobalKeys.ChangesPane.Help().Key, GlobalKeys.ChangesPane.Help().Desc)
	debug.DebugLog("  NextAgent: key=%s, desc=%s", GlobalKeys.NextAgent.Help().Key, GlobalKeys.NextAgent.Help().Desc)
	debug.DebugLog("  PrevAgent: key=%s, desc=%s", GlobalKeys.PrevAgent.Help().Key, GlobalKeys.PrevAgent.Help().Desc)
	debug.DebugLog("  AddRemoveAgents: key=%s, desc=%s", GlobalKeys.AddRemoveAgents.Help().Key, GlobalKeys.AddRemoveAgents.Help().Desc)
	debug.DebugLog("  NewSession: key=%s, desc=%s", GlobalKeys.NewSession.Help().Key, GlobalKeys.NewSession.Help().Desc)
	debug.DebugLog("  AttachAgent: key=%s, desc=%s", GlobalKeys.AttachAgent.Help().Key, GlobalKeys.AttachAgent.Help().Desc)
	debug.DebugLog("  DetachTmux: key=%s, desc=%s", GlobalKeys.DetachTmux.Help().Key, GlobalKeys.DetachTmux.Help().Desc)
	debug.DebugLog("  Confirm: key=%s, desc=%s", GlobalKeys.Confirm.Help().Key, GlobalKeys.Confirm.Help().Desc)
	debug.DebugLog("  Cancel: key=%s, desc=%s", GlobalKeys.Cancel.Help().Key, GlobalKeys.Cancel.Help().Desc)
	debug.DebugLog("  Filter: key=%s, desc=%s", GlobalKeys.Filter.Help().Key, GlobalKeys.Filter.Help().Desc)
	debug.DebugLog("  ClearFilter: key=%s, desc=%s", GlobalKeys.ClearFilter.Help().Key, GlobalKeys.ClearFilter.Help().Desc)
	debug.DebugLog("  OpenInEditor: key=%s, desc=%s", GlobalKeys.OpenInEditor.Help().Key, GlobalKeys.OpenInEditor.Help().Desc)
}
