package main

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all the keybindings for the application
type KeyMap struct {
	// Global keys
	Quit key.Binding
	Keybindings key.Binding

	// Debug
	DebugOverlay key.Binding

	// Navigation
	Up   key.Binding
	Down key.Binding

	// Direct pane switching (zero-based indexing)
	FocusPaneRepos key.Binding // Pane 0
	FocusPaneTmux  key.Binding // Pane 1
	FocusPaneGit   key.Binding // Pane 2
	FocusPaneShell key.Binding // Pane 3

	// Repository and Worktree management
	AddRepo        key.Binding
	NewWorktree    key.Binding
	DeleteWorktree key.Binding

	// Tmux interaction
	AttachTmux key.Binding
	DetachTmux key.Binding

	// List navigation
	Filter      key.Binding
	ClearFilter key.Binding

	// Dialog actions
	Confirm key.Binding
	Cancel  key.Binding

	// Git pane actions
	OpenInEditor key.Binding
}

// NewKeyMap creates a new KeyMap with default keybindings
func NewKeyMap() *KeyMap {
	return &KeyMap{
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

		// Git pane actions
		OpenInEditor: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open with $EDITOR"),
		),
	}
}

// ShortHelp returns a slice of key bindings to show in the short help view
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Keybindings,
		k.Quit,
	}
}

// FullHelp returns a slice of key bindings to show in the full help view
func (k *KeyMap) FullHelp() [][]key.Binding {
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
func (k *KeyMap) GetHelpSections() map[string][]key.Binding {
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
func (k *KeyMap) SetEnabled(binding *key.Binding, enabled bool) {
	binding.SetEnabled(enabled)
}

// DisableWorktreeKeys disables worktree-specific keybindings
func (k *KeyMap) DisableWorktreeKeys() {
	k.DeleteWorktree.SetEnabled(false)
}

// EnableWorktreeKeys enables worktree-specific keybindings
func (k *KeyMap) EnableWorktreeKeys() {
	k.DeleteWorktree.SetEnabled(true)
}

// DisableDialogKeys disables dialog-specific keybindings
func (k *KeyMap) DisableDialogKeys() {
	k.Confirm.SetEnabled(false)
	k.Cancel.SetEnabled(false)
}

// EnableDialogKeys enables dialog-specific keybindings
func (k *KeyMap) EnableDialogKeys() {
	k.Confirm.SetEnabled(true)
	k.Cancel.SetEnabled(true)
}
