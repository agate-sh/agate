package common

import (
	"github.com/charmbracelet/bubbles/key"
)

// GlobalKeyMap defines global keybindings that work across all panes
//
// Note: This contains both truly global keybindings (like quit, help) and
// conceptually pane-specific keybindings that need to be globally accessible.
// For example:
// - 'w' (new worktree) conceptually belongs to the repos pane but works globally
// - 'r' (add repo) conceptually belongs to the repos pane but works globally
// - Git pane actions like "open in editor" are pane-specific and should be handled by the pane
//
// TODO: As pane components mature, consider moving more keybindings to individual panes
// while keeping them globally accessible through the pane interface.
type GlobalKeyMap struct {
	// Truly global keys - work from any pane, any context
	Quit        key.Binding // q, Ctrl+C - quit application
	Keybindings key.Binding // ? - show help
	DebugOverlay key.Binding // Ctrl+D - toggle debug overlay

	// Global navigation keys - work within any focusable pane
	Up   key.Binding // ↑, k - move up in active pane
	Down key.Binding // ↓, j - move down in active pane

	// Pane switching - always global for quick navigation
	FocusPaneRepos key.Binding // 0 - focus repos & worktrees pane
	FocusPaneTmux  key.Binding // 1 - focus tmux pane
	FocusPaneGit   key.Binding // 2 - focus git pane
	FocusPaneShell key.Binding // 3 - focus shell pane

	// Repository and worktree management - conceptually belong to repos pane
	// but are globally accessible for convenience
	AddRepo        key.Binding // r - add repository (repos pane action, but global)
	NewWorktree    key.Binding // w - create worktree (repos pane action, but global)
	DeleteWorktree key.Binding // d - delete worktree (repos pane action, context-sensitive)

	// Tmux interaction - conceptually belongs to tmux pane but globally accessible
	AttachTmux key.Binding // Enter - attach to tmux session
	DetachTmux key.Binding // Ctrl+Q - detach from tmux session

	// Dialog actions - global because dialogs overlay all content
	Confirm key.Binding // Enter, y - confirm dialog action
	Cancel  key.Binding // Esc, n - cancel dialog

	// List interaction - used by multiple panes (repos, git, etc.)
	Filter      key.Binding // / - start filtering
	ClearFilter key.Binding // Esc - clear filter
}

// NewGlobalKeyMap creates a new GlobalKeyMap with default keybindings
func NewGlobalKeyMap() *GlobalKeyMap {
	return &GlobalKeyMap{
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

		// Direct pane switching (zero-based indexing)
		FocusPaneRepos: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "focus repos & worktrees"),
		),
		FocusPaneTmux: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "focus tmux"),
		),
		FocusPaneGit: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "focus git"),
		),
		FocusPaneShell: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "focus shell"),
		),

		// Repository and Worktree management
		AddRepo: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "add repo"),
		),
		NewWorktree: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "new worktree"),
		),
		DeleteWorktree: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete worktree"),
		),

		// Tmux interaction
		AttachTmux: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "attach to tmux"),
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
	}
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
		{k.Quit, k.Keybindings},                             // Global
		{k.FocusPaneRepos, k.FocusPaneTmux, k.FocusPaneGit, k.FocusPaneShell}, // Direct pane switching
		{k.Up, k.Down},                               // Navigation
		{k.AddRepo, k.NewWorktree, k.DeleteWorktree}, // Repository & Worktree
		{k.AttachTmux, k.DetachTmux},                 // Tmux
		{k.Filter, k.ClearFilter},                    // Filtering
		{k.Confirm, k.Cancel},                        // Dialogs
	}
}

// GetHelpSections returns help sections with categorized keybindings
func (k *GlobalKeyMap) GetHelpSections() map[string][]key.Binding {
	return map[string][]key.Binding{
		"Global": {
			k.Quit,
			k.Keybindings,
		},
		"Navigation": {
			k.Up,
			k.Down,
		},
		"Direct Pane Switching": {
			k.FocusPaneRepos,
			k.FocusPaneTmux,
			k.FocusPaneGit,
			k.FocusPaneShell,
		},
		"Repository & Worktree Management": {
			k.AddRepo,
			k.NewWorktree,
			k.DeleteWorktree,
		},
		"Tmux Interaction": {
			k.AttachTmux,
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

// DisableWorktreeKeys disables worktree-specific keybindings
func (k *GlobalKeyMap) DisableWorktreeKeys() {
	k.DeleteWorktree.SetEnabled(false)
}

// EnableWorktreeKeys enables worktree-specific keybindings
func (k *GlobalKeyMap) EnableWorktreeKeys() {
	k.DeleteWorktree.SetEnabled(true)
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
