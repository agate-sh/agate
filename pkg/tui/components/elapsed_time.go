package components

import (
	"agate/pkg/tui/theme"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// FormatElapsedTime formats an elapsed time display with a cancel/quit option
// Format: "Elapsed <time> • <key> <action>"
// Example: "Elapsed 5s • c cancel" or "Elapsed 1:23 • q quit"
func FormatElapsedTime(startTime time.Time, cancelKey, cancelAction string) string {
	elapsed := time.Since(startTime)
	seconds := int(elapsed.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60

	var timeStr string
	if minutes > 0 {
		timeStr = fmt.Sprintf("%d:%02d", minutes, seconds)
	} else {
		timeStr = fmt.Sprintf("%ds", seconds)
	}

	// Style elapsed text and separator (elapsed uses same muted color as shortcut descriptions)
	elapsedTextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted))

	// Use shortcut component with default variant (bubbles style)
	shortcut := RenderShortcut(cancelKey, cancelAction, ShortcutDefault, "")

	return elapsedTextStyle.Render("Elapsed "+timeStr) + " " +
		separatorStyle.Render("•") + " " +
		shortcut
}
