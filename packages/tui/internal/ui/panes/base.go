package panes

import (
	"strings"

	"agate/tui/internal/ui/theme"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	paneContentHorizontalPadding = 1
	paneContentVerticalPadding   = 1
)

var (
	// PaneBaseStyle mirrors the chrome used in the legacy Bubble Tea UI.
	PaneBaseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderMuted)).
		Padding(paneContentVerticalPadding, 0)
)

// Pane provides the shared behaviours each UI pane must implement.
type Pane interface {
	SetSize(width, height int)
	SetActive(active bool)
	IsActive() bool
	GetIndex() int
	GetTitle() string
	GetTitleStyle() TitleStyle
	View() string
	Update(msg tea.Msg) (Pane, tea.Cmd)
	HandleKey(key string) (handled bool, cmd tea.Cmd)
	MoveUp() bool
	MoveDown() bool
	GetPaneSpecificKeybindings() []key.Binding
}

// TitleStyle describes how a pane's title should be presented.
type TitleStyle struct {
	Type      string
	Color     string
	Text      string
	Shortcuts string
}

// BasePane bundles shared state and default behaviours.
type BasePane struct {
	index    int
	width    int
	height   int
	isActive bool
	title    string
}

// NewBasePane constructs a base implementation with the provided index and title.
func NewBasePane(index int, title string) *BasePane {
	return &BasePane{
		index: index,
		title: title,
	}
}

// SetSize stores the width/height so child panes can react during rendering.
func (p *BasePane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetActive flips the focus flag that panes can read during rendering.
func (p *BasePane) SetActive(active bool) {
	p.isActive = active
}

// IsActive returns whether the pane currently owns focus.
func (p *BasePane) IsActive() bool {
	return p.isActive
}

// GetIndex exposes the pane's index which underpins numeric keybindings.
func (p *BasePane) GetIndex() int {
	return p.index
}

// GetTitle returns the configured pane title.
func (p *BasePane) GetTitle() string {
	return p.title
}

// GetTitleStyle mirrors the legacy behaviour where inactive panes surface their index.
func (p *BasePane) GetTitleStyle() TitleStyle {
	shortcuts := ""
	if !p.isActive {
		shortcuts = "[" + string(rune('0'+p.index)) + "]"
	}
	return TitleStyle{
		Type:      "plain",
		Text:      p.title,
		Shortcuts: shortcuts,
	}
}

// View provides a default empty view. Concrete panes should override it.
func (p *BasePane) View() string {
	return ""
}

// Update offers a default no-op implementation.
func (p *BasePane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	return p, nil
}

// HandleKey is a no-op by default.
func (p *BasePane) HandleKey(string) (bool, tea.Cmd) {
	return false, nil
}

// MoveUp returns false so panes can opt-in to navigation.
func (p *BasePane) MoveUp() bool {
	return false
}

// MoveDown returns false so panes can opt-in to navigation.
func (p *BasePane) MoveDown() bool {
	return false
}

// GetPaneSpecificKeybindings defaults to an empty slice.
func (p *BasePane) GetPaneSpecificKeybindings() []key.Binding {
	return nil
}

// GetWidth exposes the last configured width.
func (p *BasePane) GetWidth() int {
	return p.width
}

// GetHeight exposes the last configured height.
func (p *BasePane) GetHeight() int {
	return p.height
}

// PaneContentHorizontalPadding exposes the shared horizontal padding value.
func PaneContentHorizontalPadding() int {
	return paneContentHorizontalPadding
}

// PaneContentVerticalPadding exposes the shared vertical padding value.
func PaneContentVerticalPadding() int {
	return paneContentVerticalPadding
}

// PaneFullWidth converts an inner width into the full width including padding.
func PaneFullWidth(innerWidth int) int {
	if innerWidth < 0 {
		innerWidth = 0
	}
	return innerWidth + paneContentHorizontalPadding*2
}

// ApplyPaneContentPadding wraps each line with the standard horizontal padding.
func ApplyPaneContentPadding(content string, innerWidth int) string {
	if innerWidth < 0 {
		innerWidth = 0
	}
	lines := strings.Split(content, "\n")
	leftPad := strings.Repeat(" ", paneContentHorizontalPadding)
	rightPad := strings.Repeat(" ", paneContentHorizontalPadding)
	contentStyle := lipgloss.NewStyle().Width(innerWidth)

	padded := make([]string, len(lines))
	for i, line := range lines {
		padded[i] = leftPad + contentStyle.Render(line) + rightPad
	}
	return strings.Join(padded, "\n")
}
