package main

// Shortcut represents a keyboard shortcut with its description
type Shortcut struct {
	Key         string
	Description string
	IsGlobal    bool // Whether this is a global shortcut
}

// PreviewModeShortcuts are available when viewing tmux in preview mode
var PreviewModeShortcuts = []Shortcut{
	{Key: "↵", Description: "attach to tmux", IsGlobal: false},
}


// LeftPaneShortcuts are the shortcuts available when the left pane is focused
var LeftPaneShortcuts = []Shortcut{
	{Key: "q", Description: "quit", IsGlobal: false},
}

// GlobalShortcuts are always available regardless of focused pane
var GlobalShortcuts = []Shortcut{
	{Key: "?", Description: "help", IsGlobal: true},
}

// AllShortcuts returns all available shortcuts for the help dialog
func AllShortcuts() map[string][]Shortcut {
	return map[string][]Shortcut{
		"Two-Mode System": {
			{Key: "↵ (Enter)", Description: "Attach to full tmux (when right pane focused)", IsGlobal: false},
			{Key: "ctrl+q", Description: "Detach from tmux (return to preview)", IsGlobal: false},
		},
		"Navigation": {
			{Key: "q", Description: "Quit application (when left pane focused)", IsGlobal: false},
		},
		"Mode Benefits": {
			{Key: "Preview", Description: "Read-only, fast rendering, no typing lag", IsGlobal: false},
			{Key: "Attached", Description: "Full tmux experience, complete terminal control", IsGlobal: false},
		},
		"Help": {
			{Key: "?", Description: "Show this help dialog", IsGlobal: true},
			{Key: "any key", Description: "Close help dialog", IsGlobal: true},
		},
	}
}