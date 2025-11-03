package components

import (
	"fmt"

	"agate/pkg/gui/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// RenderChangeCountBadge returns a muted, rounded notification badge.
func RenderChangeCountBadge(count int) string {
	if count == 0 {
		return ""
	}

	bg := theme.RowHighlight
	fg := theme.TextMuted
	bg = theme.SeparatorColor
	fg = theme.TextDescription

	display := fmt.Sprintf("%d", count)
	if count > 99 {
		display = "99+"
	} else if count > 9 {
		display = "9+"
	}

	centerWidth := runewidth.StringWidth(display)
	if centerWidth < 1 {
		centerWidth = 1
	}

	left := lipgloss.NewStyle().
		Foreground(lipgloss.Color(bg)).
		Render("")

	middle := lipgloss.NewStyle().
		Bold(count > 0).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Width(centerWidth).
		AlignHorizontal(lipgloss.Center).
		Render(display)

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color(bg)).
		Render("")

	return left + middle + right
}
