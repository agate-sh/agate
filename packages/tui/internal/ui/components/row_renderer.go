package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderRowWithHint mirrors the legacy row renderer that keeps hint text pinned.
func RenderRowWithHint(content, hint string, width, paddingCount int, bgColor, contentColor, hintColor string) string {
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))

	var body string
	if hint != "" {
		hintStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(bgColor)).
			Foreground(lipgloss.Color(hintColor))
		hintRendered := hintStyle.Render(hint)
		hintWidth := lipgloss.Width(hintRendered)

		available := width - hintWidth
		if available < 0 {
			available = 0
		}

		display := content
		if lipgloss.Width(display) > available {
			runes := []rune(display)
			for lipgloss.Width(string(runes)) > available-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			if available >= 3 {
				display = string(runes) + "..."
			} else {
				display = string(runes)
			}
		}

		contentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(contentColor)).
			Background(lipgloss.Color(bgColor))
		body = contentStyle.Width(available).Render(display) + hintRendered

		if lipgloss.Width(body) < width {
			body += bgStyle.Width(width - lipgloss.Width(body)).Render("")
		}
	} else {
		contentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(contentColor)).
			Background(lipgloss.Color(bgColor))
		body = contentStyle.Width(width).Render(content)
	}

	leftPad := bgStyle.Render(strings.Repeat(" ", paddingCount))
	rightPad := bgStyle.Render(strings.Repeat(" ", paddingCount))
	fullRow := leftPad + body + rightPad

	fullWidth := width + paddingCount*2
	if lipgloss.Width(fullRow) < fullWidth {
		fullRow += bgStyle.Render(strings.Repeat(" ", fullWidth-lipgloss.Width(fullRow)))
	}
	return fullRow
}
