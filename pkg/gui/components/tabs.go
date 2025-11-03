// Package components provides reusable UI components for the Agate TUI.
package components

import (
	"agate/pkg/gui/theme"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tabBorderWithBottom creates a custom border for tabs
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")

	inactiveTabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(lipgloss.Color(theme.BorderMuted)).
				AlignHorizontal(lipgloss.Center)

	activeTabStyle = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(lipgloss.Color(theme.BorderActive)).
			AlignHorizontal(lipgloss.Center)

	tabWindowStyle = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color(theme.BorderActive)).
			Border(lipgloss.NormalBorder(), false, true, true, true)
)

// Tab represents a single tab with name and content
type Tab struct {
	Name     string
	Content  string
	Trailing string
}

// TabbedPane represents a tabbed interface component
type TabbedPane struct {
	tabs        []Tab
	activeTab   int
	height      int
	width       int
	borderColor string
}

// NewTabbedPane creates a new tabbed pane with the given tabs
func NewTabbedPane(tabs []Tab) *TabbedPane {
	return &TabbedPane{
		tabs:        tabs,
		activeTab:   0,
		borderColor: theme.BorderMuted,
	}
}

// SetSize updates the dimensions of the tabbed pane
func (t *TabbedPane) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetBorderColor sets the border color for the tabbed pane
func (t *TabbedPane) SetBorderColor(color string) {
	t.borderColor = color
}

// SetActiveTab sets the active tab by index
func (t *TabbedPane) SetActiveTab(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeTab = index
	}
}

// GetActiveTab returns the current active tab index
func (t *TabbedPane) GetActiveTab() int {
	return t.activeTab
}

// NextTab switches to the next tab (wraps around)
func (t *TabbedPane) NextTab() {
	t.activeTab = (t.activeTab + 1) % len(t.tabs)
}

// PrevTab switches to the previous tab (wraps around)
func (t *TabbedPane) PrevTab() {
	t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
}

// UpdateTabContent updates the content of a specific tab
func (t *TabbedPane) UpdateTabContent(index int, content string) {
	if index >= 0 && index < len(t.tabs) {
		t.tabs[index].Content = content
	}
}

// View renders the tabbed pane
func (t *TabbedPane) View() string {
	if t.width == 0 || t.height == 0 || len(t.tabs) == 0 {
		return ""
	}

	row, windowStyle, windowContentWidth, windowContentHeight := t.layoutMetrics()
	if windowContentWidth == 0 || windowContentHeight == 0 {
		return row
	}

	// Get active tab content
	content := ""
	if t.activeTab >= 0 && t.activeTab < len(t.tabs) {
		content = t.tabs[t.activeTab].Content
	}

	// Clamp content to the window dimensions so long lines don't spill outside
	content = lipgloss.NewStyle().
		Width(windowContentWidth).
		MaxWidth(windowContentWidth).
		Height(windowContentHeight).
		MaxHeight(windowContentHeight).
		Render(content)

	window := windowStyle.
		Height(windowContentHeight).
		Render(content)

	// Join tabs and window vertically
	return lipgloss.JoinVertical(lipgloss.Left, row, window)
}

// ContentSize returns the inner content width and height for the tab window.
func (t *TabbedPane) ContentSize() (int, int) {
	_, _, width, height := t.layoutMetrics()
	return width, height
}

// layoutMetrics prepares the rendered tab row and shared layout values.
func (t *TabbedPane) layoutMetrics() (string, lipgloss.Style, int, int) {
	if t.width == 0 || t.height == 0 || len(t.tabs) == 0 {
		return "", lipgloss.NewStyle(), 0, 0
	}

	// Create window style with custom border color
	windowStyle := lipgloss.NewStyle().
		BorderForeground(lipgloss.Color(t.borderColor)).
		Border(lipgloss.NormalBorder(), false, true, true, true)

	// Create tab styles - active tab uses custom border color, inactive uses muted
	activeStyle := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(lipgloss.Color(t.borderColor)).
		AlignHorizontal(lipgloss.Center)

	inactiveStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(lipgloss.Color(theme.BorderMuted)).
		BorderBottomForeground(lipgloss.Color(t.borderColor)).
		AlignHorizontal(lipgloss.Center)

	var renderedTabs []string

	tabCount := len(t.tabs)
	tabWidth := t.width / tabCount
	if tabWidth < 1 {
		tabWidth = 1
	}
	lastTabWidth := t.width - tabWidth*(tabCount-1)
	if lastTabWidth < 1 {
		lastTabWidth = 1
	}

	// Render each tab header
	for i, tab := range t.tabs {
		width := tabWidth
		if i == tabCount-1 {
			width = lastTabWidth
		}
		if width < 1 {
			width = 1
		}

		var style lipgloss.Style
		isFirst := i == 0
		isLast := i == tabCount-1
		isActive := i == t.activeTab

		if isActive {
			style = activeStyle
		} else {
			style = inactiveStyle
		}

		// Adjust border corners based on position
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast {
			border.BottomRight = "┤"
		}
		style = style.Border(border)

		// Tab has left+right borders (2 chars), so content width = total width - frame
		tabContentWidth := width - style.GetHorizontalFrameSize()
		if tabContentWidth < 1 {
			tabContentWidth = 1
		}
		style = style.Width(tabContentWidth)

		var content string
		trailing := tab.Trailing
		if trailing == "" {
			content = lipgloss.NewStyle().
				Width(tabContentWidth).
				AlignHorizontal(lipgloss.Center).
				Render(tab.Name)
		} else {
			trailingWidth := lipgloss.Width(trailing)
			gapWidth := 1
			mainWidth := tabContentWidth - trailingWidth - gapWidth
			if mainWidth < 1 {
				mainWidth = 1
			}

			main := lipgloss.NewStyle().
				Width(mainWidth).
				AlignHorizontal(lipgloss.Center).
				Render(tab.Name)
			gap := strings.Repeat(" ", gapWidth)
			trailingRendered := lipgloss.NewStyle().
				Width(trailingWidth).
				AlignHorizontal(lipgloss.Right).
				Render(trailing)
			content = main + gap + trailingRendered
		}

		renderedTabs = append(renderedTabs, style.Render(content))
	}

	// Join tabs horizontally and measure actual dimensions
	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Calculate window dimensions to fill remaining space after tabs
	windowBorderHeight := windowStyle.GetVerticalFrameSize()
	windowBorderWidth := windowStyle.GetHorizontalFrameSize()

	tabHeight := lipgloss.Height(row)
	windowContentHeight := t.height - tabHeight - windowBorderHeight
	if windowContentHeight < 1 {
		windowContentHeight = 1
	}

	windowContentWidth := t.width - windowBorderWidth
	if windowContentWidth < 1 {
		windowContentWidth = 1
	}

	return row, windowStyle, windowContentWidth, windowContentHeight
}
