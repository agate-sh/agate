// Package components provides reusable UI components for the Agate TUI.
//
// Toast Notification System
//
// The toast system provides non-intrusive, temporary notifications that appear
// in the bottom-right corner of the screen. Unlike modal overlays, toasts:
//   - Do NOT dim the background
//   - Do NOT block user interaction
//   - Automatically dismiss after a duration (default 3 seconds)
//   - Show one at a time (queued if multiple are triggered)
//
// Usage Example:
//
//	// In your model initialization:
//	toast := components.NewToast()
//
//	// To show a toast with default 3s duration:
//	cmd := toast.Show("Operation completed successfully", 0)
//
//	// To show a toast with custom duration:
//	cmd := toast.Show("Custom message", 5*time.Second)
//
//	// In your Update function:
//	case components.ToastTickMsg:
//	    cmd := m.toast.Update(msg)
//	    return m, cmd
//
//	// In your View function:
//	mainView := /* your main UI */
//	if m.toast.IsVisible() {
//	    return m.toast.PlaceOverlay(mainView, width, height)
//	}
//	return mainView
//
// Queue Behavior:
//
// When multiple toasts are triggered, they are queued and shown one at a time.
// Each toast displays for its full duration before the next one appears.
// This keeps the UI clean and prevents information overload.
package components

import (
	"strings"
	"time"

	"agate/pkg/gui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
)

const (
	// DefaultToastDuration is the default time a toast is visible (3 seconds)
	DefaultToastDuration = 3 * time.Second

	// ToastBottomMargin is the space from the bottom of the screen
	ToastBottomMargin = 2

	// ToastRightMargin is the space from the right edge of the screen
	ToastRightMargin = 2
)

// ToastTickMsg is sent periodically to check if the current toast should be dismissed
type ToastTickMsg struct{}

// toastItem represents a single queued toast notification
type toastItem struct {
	message  string
	duration time.Duration
}

// Toast manages toast notifications with a simple FIFO queue.
// Only one toast is visible at a time.
type Toast struct {
	// current is the currently displayed toast (nil if none)
	current *toastItem

	// startTime is when the current toast was first displayed
	startTime time.Time

	// queue holds pending toasts that will be shown after current one
	queue []toastItem
}

// NewToast creates a new toast notification manager
func NewToast() *Toast {
	return &Toast{
		current: nil,
		queue:   []toastItem{},
	}
}

// Show adds a toast to the queue and returns a command to start the timer.
// If duration is 0, DefaultToastDuration (3s) is used.
// If no toast is currently visible, the new toast is shown immediately.
func (t *Toast) Show(message string, duration time.Duration) tea.Cmd {
	if duration == 0 {
		duration = DefaultToastDuration
	}

	item := toastItem{
		message:  message,
		duration: duration,
	}

	// If no current toast, show this one immediately
	if t.current == nil {
		t.current = &item
		t.startTime = time.Now()
		return t.tick()
	}

	// Otherwise, add to queue
	t.queue = append(t.queue, item)
	return nil
}

// Update handles timer messages and manages toast lifecycle
func (t *Toast) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case ToastTickMsg:
		// Check if current toast should be dismissed
		if t.current != nil {
			elapsed := time.Since(t.startTime)
			if elapsed >= t.current.duration {
				// Dismiss current toast
				t.current = nil

				// Show next toast in queue if any
				if len(t.queue) > 0 {
					t.current = &t.queue[0]
					t.queue = t.queue[1:]
					t.startTime = time.Now()
					return t.tick()
				}
			} else {
				// Continue ticking while toast is visible
				return t.tick()
			}
		}
	}
	return nil
}

// tick returns a command that sends a ToastTickMsg after a short interval
func (t *Toast) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return ToastTickMsg{}
	})
}

// IsVisible returns true if a toast is currently being displayed
func (t *Toast) IsVisible() bool {
	return t.current != nil
}

// View renders the current toast notification
func (t *Toast) View() string {
	if !t.IsVisible() {
		return ""
	}

	// Style for the toast container
	toastStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.HighlightBg)).
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Padding(1, 3).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderActive))

	return toastStyle.Render(t.current.message)
}

// PlaceOverlay renders the toast in the bottom-right corner over the main view.
// Unlike modal overlays, this does NOT dim the background.
func (t *Toast) PlaceOverlay(mainView string, termWidth, termHeight int) string {
	if !t.IsVisible() {
		return mainView
	}

	toastContent := t.View()

	// Split both into lines
	toastLines, toastWidth := getToastLines(toastContent)
	mainLines := strings.Split(mainView, "\n")

	// Calculate bottom-right position
	toastHeight := len(toastLines)
	x := termWidth - toastWidth - ToastRightMargin
	y := termHeight - toastHeight - ToastBottomMargin

	// Ensure toast stays on screen
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if y >= len(mainLines) {
		y = len(mainLines) - toastHeight
		if y < 0 {
			y = 0
		}
	}

	// Build output by overlaying toast on main view
	var result strings.Builder
	for i, line := range mainLines {
		if i > 0 {
			result.WriteByte('\n')
		}

		// Check if this line should have toast content overlaid
		toastLineIdx := i - y
		if toastLineIdx >= 0 && toastLineIdx < len(toastLines) {
			// This line intersects with the toast
			toastLine := toastLines[toastLineIdx]

			// Get the left part of the main line (before toast)
			lineWidth := ansi.PrintableRuneWidth(line)
			if x < lineWidth {
				// Truncate main line and append toast
				leftPart := truncateToWidth(line, x)
				result.WriteString(leftPart)
				result.WriteString(toastLine)
			} else {
				// Main line is shorter than toast x position, pad with spaces
				result.WriteString(line)
				result.WriteString(strings.Repeat(" ", x-lineWidth))
				result.WriteString(toastLine)
			}
		} else {
			// No toast on this line
			result.WriteString(line)
		}
	}

	return result.String()
}

// getToastLines splits a string into lines and returns the widest line width
func getToastLines(s string) ([]string, int) {
	lines := strings.Split(s, "\n")
	widest := 0
	for _, line := range lines {
		w := ansi.PrintableRuneWidth(line)
		if w > widest {
			widest = w
		}
	}
	return lines, widest
}

// truncateToWidth truncates a string (with ANSI codes) to a specific printable width
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	var result strings.Builder
	currentWidth := 0
	inAnsi := false

	for _, r := range s {
		// Detect ANSI escape sequences
		if r == '\x1b' {
			inAnsi = true
		}

		if inAnsi {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inAnsi = false
			}
			continue
		}

		// Count printable character width
		w := ansi.PrintableRuneWidth(string(r))
		if currentWidth+w > width {
			break
		}

		result.WriteRune(r)
		currentWidth += w
	}

	return result.String()
}
