package panes

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)


// TitleStyle defines how a pane's title should be rendered
type TitleStyle struct {
	Type      string // "plain" or "badge"
	Color     string // hex color for badge background (used for badge type)
	Text      string // the title text
	Shortcuts string // shown when active, e.g. "[↵ open]", "[2]"
}

// Pane represents a common interface for all UI panes in the application
type Pane interface {
	// Core pane functionality
	SetSize(width, height int)
	SetActive(active bool)
	IsActive() bool
	GetIndex() int

	// Title management
	GetTitle() string
	GetTitleStyle() TitleStyle

	// Content and rendering
	View() string
	Update(msg tea.Msg) (Pane, tea.Cmd)

	// Navigation and key handling
	HandleKey(key string) (handled bool, cmd tea.Cmd)
	MoveUp() bool   // returns true if navigation occurred, false if not supported
	MoveDown() bool // returns true if navigation occurred, false if not supported

	// Keybindings - pane-specific shortcuts that should be shown when this pane is active
	GetPaneSpecificKeybindings() []key.Binding
}

// BasePane provides default implementations for common pane functionality
type BasePane struct {
	index    int
	width    int
	height   int
	isActive bool
	title    string
}

// NewBasePane creates a new BasePane with the given index and title
func NewBasePane(index int, title string) *BasePane {
	return &BasePane{
		index: index,
		title: title,
	}
}

// SetSize updates the pane dimensions
func (p *BasePane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetActive sets whether this pane is currently focused
func (p *BasePane) SetActive(active bool) {
	p.isActive = active
}

// IsActive returns whether this pane is currently focused
func (p *BasePane) IsActive() bool {
	return p.isActive
}

// GetIndex returns the pane's index (used for keybindings like 0, 1, 2, 3)
func (p *BasePane) GetIndex() int {
	return p.index
}

// GetTitle returns the pane's title
func (p *BasePane) GetTitle() string {
	return p.title
}

// GetTitleStyle returns the default title style (plain text with pane number)
func (p *BasePane) GetTitleStyle() TitleStyle {
	shortcuts := ""
	if p.isActive {
		// When active, could show pane-specific shortcuts - override in implementations
		shortcuts = ""
	} else {
		// When not active, show pane number
		shortcuts = "[" + string(rune('0'+p.index)) + "]"
	}

	return TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      p.title,
		Shortcuts: shortcuts,
	}
}

// View returns a default empty view - should be overridden by implementations
func (p *BasePane) View() string {
	return "Empty pane - override View() method"
}

// Update handles tea.Msg updates - default implementation does nothing
func (p *BasePane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	return p, nil
}

// HandleKey processes keyboard input - default implementation handles nothing
func (p *BasePane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	return false, nil
}

// MoveUp provides default no-navigation implementation
func (p *BasePane) MoveUp() bool {
	return false // no navigation supported by default
}

// MoveDown provides default no-navigation implementation
func (p *BasePane) MoveDown() bool {
	return false // no navigation supported by default
}

// GetPaneSpecificKeybindings returns pane-specific keybindings - default is empty
func (p *BasePane) GetPaneSpecificKeybindings() []key.Binding {
	return []key.Binding{} // no pane-specific keybindings by default
}

// GetWidth returns the current width
func (p *BasePane) GetWidth() int {
	return p.width
}

// GetHeight returns the current height
func (p *BasePane) GetHeight() int {
	return p.height
}

// SetTitle updates the pane's title
func (p *BasePane) SetTitle(title string) {
	p.title = title
}