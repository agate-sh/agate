package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderRowWithHint renders a row with content on the left and hint text on the right,
// ensuring the hint never wraps by measuring it first and allocating remaining space to content.
// This is the proven algorithm from the agents pane.
//
// Parameters:
//   - content: Plain text content to display on the left (will be truncated if needed)
//   - hint: Plain text hint to display on the right (always fully visible)
//   - width: Inner content width (excluding padding)
//   - paddingCount: Number of spaces for left and right padding
//   - bgColor: Background color for the entire row
//   - contentColor: Foreground color for the content
//   - hintColor: Foreground color for the hint text
//
// Returns: A fully rendered row string with background, padding, and proper width
func RenderRowWithHint(content, hint string, width, paddingCount int, bgColor, contentColor, hintColor string) string {
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))

	var body string

	if hint != "" {
		// Render hint first to measure its width (the key insight from agents pane)
		hintStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(bgColor)).
			Foreground(lipgloss.Color(hintColor))
		hintRendered := hintStyle.Render(hint)
		hintWidth := lipgloss.Width(hintRendered)

		// DEBUG
		// fmt.Printf("DEBUG RenderRowWithHint: width=%d, hintWidth=%d, contentLen=%d\n", width, hintWidth, lipgloss.Width(content))

		// Calculate available width for content (excluding hint)
		available := width - hintWidth
		if available < 0 {
			available = 0
		}

		// Render content with width limit
		// Truncate content if it's too long to prevent wrapping
		truncatedContent := content
		if lipgloss.Width(content) > available {
			// Manually truncate to fit available space
			runes := []rune(content)
			for lipgloss.Width(string(runes)) > available-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			truncatedContent = string(runes) + "..."
		}

		contentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(contentColor)).
			Background(lipgloss.Color(bgColor))
		bodySegment := contentStyle.Width(available).Render(truncatedContent)

		// Combine content + hint
		body = bodySegment + hintRendered

		// Pad to full inner width if needed
		if lipgloss.Width(body) < width {
			body += bgStyle.Width(width - lipgloss.Width(body)).Render("")
		}
	} else {
		// No hint - just render content
		contentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(contentColor)).
			Background(lipgloss.Color(bgColor))
		body = contentStyle.Width(width).Render(content)
	}

	// Add left and right padding
	leftPad := bgStyle.Render(strings.Repeat(" ", paddingCount))
	rightPad := bgStyle.Render(strings.Repeat(" ", paddingCount))
	fullRow := leftPad + body + rightPad

	// Calculate full width (width + 2*paddingCount)
	fullWidth := width + 2*paddingCount

	// Pad to full width if needed
	finalWidth := lipgloss.Width(fullRow)
	if finalWidth < fullWidth {
		fullRow += bgStyle.Render(strings.Repeat(" ", fullWidth-finalWidth))
	}

	return fullRow
}
