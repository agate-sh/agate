package common

import (
	"github.com/charmbracelet/bubbles/key"
)

// ShortcutOverlay manages the display of contextual shortcuts
type ShortcutOverlay struct {
	keyMap  *GlobalKeyMap
	focused string
	mode    string // "preview" or "attached"
}

// NewShortcutOverlay creates a new shortcut overlay
func NewShortcutOverlay(keyMap *GlobalKeyMap) *ShortcutOverlay {
	return &ShortcutOverlay{
		keyMap:  keyMap,
		focused: "left",
		mode:    "preview",
	}
}

// SetFocus updates the focused pane
func (s *ShortcutOverlay) SetFocus(focus string) {
	s.focused = focus
}

// SetMode updates the interaction mode
func (s *ShortcutOverlay) SetMode(mode string) {
	s.mode = mode
}

// GetContextualShortcuts returns shortcuts relevant to current context
func (s *ShortcutOverlay) GetContextualShortcuts() []key.Binding {
	shortcuts := []key.Binding{}

	// Always show global shortcuts
	shortcuts = append(shortcuts, s.keyMap.Quit, s.keyMap.Keybindings)

	// Add context-specific shortcuts
	switch s.mode {
	case "preview":
		switch s.focused {
		case "reposAndWorktrees":
			// Left pane (repos & worktrees list) shortcuts
			shortcuts = append(shortcuts,
				s.keyMap.AddRepo,
				s.keyMap.NewWorktree,
				s.keyMap.DeleteWorktree,
				s.keyMap.FocusPaneTmux,
				s.keyMap.FocusPaneGit,
				s.keyMap.FocusPaneShell,
			)
		case "tmux", "git", "shell":
			// Right panes (tmux/git/shell) shortcuts
			shortcuts = append(shortcuts,
				s.keyMap.AttachTmux,
				s.keyMap.DetachTmux, // Show detach hint for reference
				s.keyMap.FocusPaneRepos,
			)
		}
	case "attached":
		// When attached to tmux, only show detach option
		shortcuts = append(shortcuts, s.keyMap.DetachTmux)
	}

	return shortcuts
}

// FormatShortcuts formats the shortcuts for display
func (s *ShortcutOverlay) FormatShortcuts() []Shortcut {
	bindings := s.GetContextualShortcuts()
	shortcuts := make([]Shortcut, 0, len(bindings))

	for _, binding := range bindings {
		if binding.Enabled() {
			shortcuts = append(shortcuts, Shortcut{
				Key:         binding.Help().Key,
				Description: binding.Help().Desc,
				IsGlobal:    s.isGlobalKey(binding),
			})
		}
	}

	return shortcuts
}

// isGlobalKey checks if a keybinding is global
func (s *ShortcutOverlay) isGlobalKey(binding key.Binding) bool {
	// Compare by the key help text since we can't compare structs directly
	helpKey := binding.Help().Key
	return helpKey == s.keyMap.Quit.Help().Key || helpKey == s.keyMap.Keybindings.Help().Key
}

// Shortcut represents a keyboard shortcut with its description (for compatibility)
type Shortcut struct {
	Key             string
	Description     string
	IsGlobal        bool // Whether this is a global shortcut
	IsAgentShortcut bool // Whether this is an agent/tmux-specific shortcut
}

// PreviewModeShortcuts returns shortcuts for preview mode (for compatibility)
func (s *ShortcutOverlay) PreviewModeShortcuts() []Shortcut {
	s.SetMode("preview")
	return s.FormatShortcuts()
}

// LeftPaneShortcuts returns shortcuts for left pane (for compatibility)
func (s *ShortcutOverlay) LeftPaneShortcuts() []Shortcut {
	s.SetFocus("left")
	s.SetMode("preview")
	return s.FormatShortcuts()
}

// GlobalShortcuts returns global shortcuts (for compatibility)
func (s *ShortcutOverlay) GlobalShortcuts() []Shortcut {
	return []Shortcut{
		{Key: s.keyMap.Quit.Help().Key, Description: s.keyMap.Quit.Help().Desc, IsGlobal: true},
		{Key: s.keyMap.Keybindings.Help().Key, Description: s.keyMap.Keybindings.Help().Desc, IsGlobal: true},
	}
}

// AllShortcuts returns all available shortcuts for the help dialog
func AllShortcuts(keyMap *GlobalKeyMap) map[string][]Shortcut {
	sections := keyMap.GetHelpSections()
	result := make(map[string][]Shortcut)

	for sectionName, bindings := range sections {
		shortcuts := make([]Shortcut, 0, len(bindings))
		for _, binding := range bindings {
			shortcuts = append(shortcuts, Shortcut{
				Key:         binding.Help().Key,
				Description: binding.Help().Desc,
				IsGlobal:    binding.Help().Key == keyMap.Quit.Help().Key || binding.Help().Key == keyMap.Keybindings.Help().Key,
			})
		}
		result[sectionName] = shortcuts
	}

	return result
}
