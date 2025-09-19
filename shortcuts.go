package main

// Shortcut represents a keyboard shortcut with its description
type Shortcut struct {
	Key         string
	Description string
	IsGlobal    bool // Whether this is a global shortcut
}

// TmuxShortcuts are the shortcuts available when the tmux pane is focused
var TmuxShortcuts = []Shortcut{
	{Key: "n", Description: "new", IsGlobal: false},
	{Key: "D", Description: "kill", IsGlobal: false},
	{Key: "↵/o", Description: "open", IsGlobal: false},
	{Key: "p", Description: "push branch", IsGlobal: false},
	{Key: "c", Description: "checkout", IsGlobal: false},
	{Key: "tab", Description: "switch tab", IsGlobal: false},
}

// LeftPaneShortcuts are the shortcuts available when the left pane is focused
var LeftPaneShortcuts = []Shortcut{
	{Key: "q", Description: "quit", IsGlobal: false},
	{Key: "tab", Description: "switch tab", IsGlobal: false},
}

// GlobalShortcuts are always available regardless of focused pane
var GlobalShortcuts = []Shortcut{
	{Key: "?", Description: "help", IsGlobal: true},
}

// AllShortcuts returns all available shortcuts for the help dialog
func AllShortcuts() map[string][]Shortcut {
	return map[string][]Shortcut{
		"Session Management": {
			{Key: "n", Description: "Create a new session", IsGlobal: false},
			{Key: "D", Description: "Kill (delete) the selected session", IsGlobal: false},
			{Key: "↵/o", Description: "Open/attach to the selected session", IsGlobal: false},
			{Key: "ctrl+a, a", Description: "Attach to tmux session full-screen", IsGlobal: false},
			{Key: "ctrl+q", Description: "Detach from attached tmux session", IsGlobal: false},
		},
		"Handoff": {
			{Key: "p", Description: "Push branch to GitHub", IsGlobal: false},
			{Key: "c", Description: "Checkout: commit changes and pause session", IsGlobal: false},
		},
		"Navigation": {
			{Key: "tab", Description: "Switch between panes", IsGlobal: false},
			{Key: "q", Description: "Quit the application (when left pane focused)", IsGlobal: false},
		},
		"Help": {
			{Key: "?", Description: "Show this help dialog", IsGlobal: true},
			{Key: "esc", Description: "Close dialogs and overlays", IsGlobal: true},
		},
	}
}