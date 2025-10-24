package panes

import (
	"strings"
	"time"

	"agate/tui/internal/ws"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BlinkingCursor is a simple on/off cursor spinner
var BlinkingCursor = spinner.Spinner{
	Frames: []string{"█", "░"},
	FPS:    time.Millisecond * 500, // Blink every 500ms
}

// TmuxPane displays live PTY output from a tmux session via WebSocket
type TmuxPane struct {
	*BasePane
	sessionID    string
	agentName    string
	agentColor   string
	content      strings.Builder
	isSubscribed bool
	wsClient     *ws.Client
	spinner      spinner.Model
}

// NewTmuxPane creates a new tmux pane instance
func NewTmuxPane(index int) *TmuxPane {
	s := spinner.New(spinner.WithSpinner(BlinkingCursor))
	return &TmuxPane{
		BasePane: NewBasePane(index, "Terminal"),
		spinner:  s,
	}
}

// SetSession sets the active session for this pane
func (p *TmuxPane) SetSession(sessionID, agentName, agentColor string) {
	p.sessionID = sessionID
	p.agentName = agentName
	p.agentColor = agentColor
}

// ClearSession clears the active session
func (p *TmuxPane) ClearSession() {
	p.sessionID = ""
	p.agentName = ""
	p.agentColor = ""
	p.content.Reset()
	p.isSubscribed = false
}

// AppendContent appends PTY output to the buffer
func (p *TmuxPane) AppendContent(data string) {
	p.content.WriteString(data)
}

// SetSubscribed marks the pane as subscribed to WebSocket events
func (p *TmuxPane) SetSubscribed(subscribed bool) {
	p.isSubscribed = subscribed
}

// SetWSClient sets the WebSocket client for input forwarding
func (p *TmuxPane) SetWSClient(client *ws.Client) {
	p.wsClient = client
}

// GetTitleStyle returns badge-style title with agent color when active
func (p *TmuxPane) GetTitleStyle() TitleStyle {
	shortcuts := ""
	if p.IsActive() {
		// When active, show relevant shortcuts
		shortcuts = "↵ attach • ctrl+q detach"
	} else {
		// When not active, show pane number
		shortcuts = "[" + string(rune('0'+p.index)) + "]"
	}

	// Use badge style with agent color if we have an active session
	titleType := "plain"
	color := ""
	titleText := "Terminal"

	if p.sessionID != "" && p.agentName != "" {
		titleType = "badge"
		color = p.agentColor
		titleText = p.agentName
	}

	return TitleStyle{
		Type:      titleType,
		Color:     color,
		Text:      titleText,
		Shortcuts: shortcuts,
	}
}

// View renders the tmux pane content
func (p *TmuxPane) View() string {
	if p.sessionID == "" {
		// No session selected - show placeholder
		style := lipgloss.NewStyle().
			Width(p.GetWidth()).
			Height(p.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#666666"))
		return style.Render("No session selected")
	}

	// Show buffered content
	content := p.content.String()
	if content == "" && p.isSubscribed {
		// Subscribed but no content yet - show blinking cursor with agent name
		spinnerView := p.spinner.View()
		loadingText := spinnerView + " " + p.agentName

		style := lipgloss.NewStyle().
			Width(p.GetWidth()).
			Height(p.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(p.agentColor))
		return style.Render(loadingText)
	}

	if content == "" {
		// Not subscribed yet - show empty
		return ""
	}

	// Render the buffered output
	// For now, just return the raw content - we'll add proper formatting later
	return content
}

// Update handles Bubble Tea messages
func (p *TmuxPane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	// Handle spinner tick messages
	switch msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	}

	return p, nil
}

// HandleKey processes keyboard input and forwards it to the active session via WebSocket
func (p *TmuxPane) HandleKey(keyStr string) (handled bool, cmd tea.Cmd) {
	// Only forward input if we have an active session and WebSocket client
	if p.sessionID == "" || p.wsClient == nil {
		return false, nil
	}

	// Map Bubble Tea key strings to PTY input
	input := mapKeyToPTYInput(keyStr)
	if input == "" {
		// Unknown or unmapped key
		return false, nil
	}

	// Send input to WebSocket
	if err := p.wsClient.SendInput(p.sessionID, input); err != nil {
		// Error sending input - for now, just ignore
		// Could trigger an error message in the future
		return true, nil
	}

	return true, nil
}

// mapKeyToPTYInput maps Bubble Tea key strings to PTY input format
func mapKeyToPTYInput(keyStr string) string {
	switch keyStr {
	// Special keys
	case "enter":
		return "\r"
	case "backspace", "ctrl+h":
		return "\x7f"
	case "tab":
		return "\t"
	case "esc":
		return "\x1b"
	case "space":
		return " "

	// Arrow keys
	case "up":
		return "\x1b[A"
	case "down":
		return "\x1b[B"
	case "right":
		return "\x1b[C"
	case "left":
		return "\x1b[D"

	// Ctrl keys
	case "ctrl+a":
		return "\x01"
	case "ctrl+b":
		return "\x02"
	case "ctrl+c":
		return "\x03"
	case "ctrl+d":
		return "\x04"
	case "ctrl+e":
		return "\x05"
	case "ctrl+f":
		return "\x06"
	case "ctrl+g":
		return "\x07"
	case "ctrl+k":
		return "\x0b"
	case "ctrl+l":
		return "\x0c"
	case "ctrl+n":
		return "\x0e"
	case "ctrl+p":
		return "\x10"
	case "ctrl+q":
		return "\x11"
	case "ctrl+r":
		return "\x12"
	case "ctrl+s":
		return "\x13"
	case "ctrl+t":
		return "\x14"
	case "ctrl+u":
		return "\x15"
	case "ctrl+v":
		return "\x16"
	case "ctrl+w":
		return "\x17"
	case "ctrl+x":
		return "\x18"
	case "ctrl+y":
		return "\x19"
	case "ctrl+z":
		return "\x1a"

	// Home/End/Page keys
	case "home":
		return "\x1b[H"
	case "end":
		return "\x1b[F"
	case "pgup":
		return "\x1b[5~"
	case "pgdown":
		return "\x1b[6~"
	case "delete":
		return "\x1b[3~"
	case "insert":
		return "\x1b[2~"

	default:
		// Regular printable characters - just return as-is
		// This includes letters, numbers, and symbols
		if len(keyStr) == 1 {
			return keyStr
		}
		// Unknown multi-character key
		return ""
	}
}

// GetPaneSpecificKeybindings returns tmux-specific keybindings
func (p *TmuxPane) GetPaneSpecificKeybindings() []key.Binding {
	// Will add attach/detach keybindings later
	return nil
}

// TickCmd returns the spinner tick command to start the animation
func (p *TmuxPane) TickCmd() tea.Cmd {
	return p.spinner.Tick
}
