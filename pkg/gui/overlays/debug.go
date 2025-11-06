package overlays

import (
	"agate/internal/debug"
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"agate/pkg/gui/theme"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DebugOverlay provides a full-screen scrollable view of debug logs
type DebugOverlay struct {
	viewport    viewport.Model
	logger      *debug.Logger
	width       int
	height      int
}

// NewDebugOverlay creates a new debug overlay
func NewDebugOverlay(logger *debug.Logger) *DebugOverlay {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.White)). // White border
		Padding(1)

	return &DebugOverlay{
		viewport: vp,
		logger:   logger,
	}
}

// SetSize updates the overlay dimensions
func (d *DebugOverlay) SetSize(width, height int) {
	d.width = width
	d.height = height

	// Leave some margin for the overlay
	overlayWidth := width - 4
	overlayHeight := height - 4

	d.viewport.Width = overlayWidth - 4   // Account for border and padding
	d.viewport.Height = overlayHeight - 4 // Account for border and padding
}

// Update handles messages for the debug overlay
func (d *DebugOverlay) Update(msg tea.Msg) (*DebugOverlay, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+d":
			// Close overlay
			return d, func() tea.Msg {
				return DebugOverlayClosedMsg{}
			}
		case "q":
			// Close overlay
			return d, func() tea.Msg {
				return DebugOverlayClosedMsg{}
			}
		case "o":
			// Open debug log file in default editor
			return d, d.openDebugLogFile()
		default:
			// Let viewport handle scrolling
			d.viewport, cmd = d.viewport.Update(msg)
		}
	default:
		d.viewport, cmd = d.viewport.Update(msg)
	}

	return d, cmd
}

// View renders the debug overlay
func (d *DebugOverlay) View() string {
	if d.logger == nil {
		return ""
	}

	// Read logs directly from agate.log file
	logs := d.readDebugLogFile()

	// Get log path for display (current directory)
	workDir, _ := os.Getwd()
	logPath := filepath.Join(workDir, "agate.log")

	// Create header with title and path on same row
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.White)). // White
		Bold(true)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextMuted)) // File path uses muted color

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription)). // Help text uses description color
		Align(lipgloss.Center)

	titleRow := titleStyle.Render("Debug Log Viewer") + " " + pathStyle.Render("("+logPath+")")
	helpRow := helpStyle.Render("Use ↑/↓ to scroll • o to open in editor • ESC to close")

	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(titleRow) + "\n" + helpRow

	// Prepare log content
	var content strings.Builder
	for _, log := range logs {
		content.WriteString(log)
		content.WriteString("\n")
	}

	if len(logs) == 0 {
		content.WriteString("No debug logs available")
	}

	// Set viewport content
	d.viewport.SetContent(content.String())

	// Auto-scroll to bottom for new content
	d.viewport.GotoBottom()

	// Create the full overlay
	overlayContent := header + "\n\n" + d.viewport.View()

	// Apply overlay styling
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.White)). // White border
		Padding(1, 2).
		Width(d.width - 4).
		Height(d.height - 4)

	return overlayStyle.Render(overlayContent)
}

// readDebugLogFile reads all lines from the agate.log file
func (d *DebugOverlay) readDebugLogFile() []string {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return []string{"Error: Could not get current directory"}
	}

	// Create log file path
	logPath := filepath.Join(workDir, "agate.log")

	// Open the log file
	file, err := os.Open(logPath)
	if err != nil {
		return []string{"Error: Could not open agate.log file"}
	}
	defer func() { _ = file.Close() }()

	// Read all lines from the file
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return []string{"Error: Could not read agate.log file"}
	}

	if len(lines) == 0 {
		return []string{"No logs available"}
	}

	return lines
}

// openDebugLogFile opens the log file in the default editor
func (d *DebugOverlay) openDebugLogFile() tea.Cmd {
	return func() tea.Msg {
		// Get current working directory
		workDir, err := os.Getwd()
		if err != nil {
			return nil
		}
		logPath := filepath.Join(workDir, "agate.log")

		// Cross-platform file opening
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", logPath)
		case "linux":
			cmd = exec.Command("xdg-open", logPath)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", logPath)
		default:
			return nil
		}

		// Run the command in the background
		_ = cmd.Start() // Ignore error as this is best-effort
		return nil
	}
}

// DebugOverlayClosedMsg indicates the debug overlay was closed
type DebugOverlayClosedMsg struct{}
