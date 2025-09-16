package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
)

//go:embed ascii-art.txt
var asciiArt string

type model struct {
	width       int
	height      int
	leftContent string
	rightOutput []byte // Terminal output buffer
	ptmx        *os.File
	cmd         *exec.Cmd
	ready       bool
	focused     string // "left" or "right"
	err         error
	subprocess  string // Command to run in right pane
}

func initialModel(subprocess string) model {
	// Create styled ASCII art with the specified color
	asciiStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9d87ae"))

	styledAscii := asciiStyle.Render(asciiArt)

	return model{
		focused:     "right", // Focus on right pane for subprocess interaction
		leftContent: styledAscii + fmt.Sprintf("\n\nStarting %s...\n\nUse Tab to switch between panes.\nType 'q' in left pane to quit.", subprocess),
		subprocess:  subprocess,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		startSubprocess(m.subprocess),
		tea.EnterAltScreen,
	)
}

func startSubprocess(subprocess string) tea.Cmd {
	return func() tea.Msg {
		// Start the specified subprocess in a PTY
		cmd := exec.Command(subprocess)

		// Set environment variables to help crossterm work in PTY
		cmd.Env = append(os.Environ(),
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
		)

		ptmx, err := pty.Start(cmd)
		if err != nil {
			return errMsg{err}
		}

		return subprocessStartedMsg{
			cmd:  cmd,
			ptmx: ptmx,
		}
	}
}

func waitForOutput(ptmx *os.File, terminal vt10x.Terminal, useTerminal bool) tea.Cmd {
	return func() tea.Msg {
		// Read directly from PTY
		buf := make([]byte, 1024)
		n, err := ptmx.Read(buf)
		if err != nil {
			if err == io.EOF {
				return subprocessExitedMsg{}
			}
			return outputMsg{data: nil, terminalError: false}
		}

		if n > 0 {
			// Try terminal emulation first if enabled
			if useTerminal && terminal != nil {
				_, writeErr := terminal.Write(buf[:n])
				if writeErr != nil {
					// Terminal emulation failed, return raw data and disable terminal
					return outputMsg{data: buf[:n], terminalError: true}
				}
				// Terminal emulation succeeded
				return outputMsg{data: nil, terminalError: false}
			} else {
				// Use raw output
				return outputMsg{data: buf[:n], terminalError: false}
			}
		}

		return outputMsg{data: nil, terminalError: false}
	}
}

type subprocessStartedMsg struct {
	cmd                  *exec.Cmd
	ptmx                 *os.File
	terminal             vt10x.Terminal
	useTerminalEmulation bool
}
type outputMsg struct {
	data          []byte
	terminalError bool
}
type subprocessExitedMsg struct{}
type errMsg struct{ error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update PTY and terminal size if they exist
		if m.ptmx != nil {
			rightWidth := int(float64(m.width) * 0.4)
			pty.Setsize(m.ptmx, &pty.Winsize{
				Rows: uint16(m.height - 4),
				Cols: uint16(rightWidth - 6),
			})
		}
		if m.terminal != nil {
			rightWidth := int(float64(m.width) * 0.4)
			m.terminal.Resize(rightWidth-6, m.height-4)
		}

	case subprocessStartedMsg:
		m.cmd = msg.cmd
		m.ptmx = msg.ptmx
		m.terminal = msg.terminal
		if !msg.useTerminalEmulation {
			m.useTerminal = false
		}

		// Keep the ASCII art with updated status
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9d87ae"))
		styledAscii := asciiStyle.Render(asciiArt)
		mode := "terminal emulation"
		if !m.useTerminal {
			mode = "raw mode"
		}
		m.leftContent = styledAscii + fmt.Sprintf("\n\n%s started successfully! (%s)\n\nUse Tab to switch between panes.\nType 'q' in left pane to quit.", m.subprocess, mode)

		// Set initial PTY and terminal size
		if m.ready {
			rightWidth := int(float64(m.width) * 0.4)
			pty.Setsize(m.ptmx, &pty.Winsize{
				Rows: uint16(m.height - 4),
				Cols: uint16(rightWidth - 6),
			})
			if m.terminal != nil {
				m.terminal.Resize(rightWidth-6, m.height-4)
			}
		}

		return m, waitForOutput(m.ptmx, m.terminal, m.useTerminal)

	case outputMsg:
		// If terminal emulation failed, switch to raw mode
		if msg.terminalError {
			m.useTerminal = false
		}

		// Append raw output to buffer (used when terminal emulation is disabled)
		if msg.data != nil {
			m.rightOutput = append(m.rightOutput, msg.data...)
			// Keep only last 100KB to prevent unlimited growth
			if len(m.rightOutput) > 100000 {
				m.rightOutput = m.rightOutput[len(m.rightOutput)-100000:]
			}
		}

		// Continue reading
		return m, tea.Tick(time.Millisecond*50, func(time.Time) tea.Msg {
			return waitForOutput(m.ptmx, m.terminal, m.useTerminal)()
		})

	case subprocessExitedMsg:
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9d87ae"))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\n%s has exited.\n\nPress 'q' to quit.", m.subprocess)

	case errMsg:
		m.err = msg.error
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9d87ae"))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\nError: %v\n\nPress 'q' to quit.", msg.error)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.focused == "left" || m.cmd == nil {
				// Cleanup
				if m.cmd != nil && m.cmd.Process != nil {
					m.cmd.Process.Kill()
				}
				if m.ptmx != nil {
					m.ptmx.Close()
				}
				return m, tea.Quit
			}

		case "tab":
			// Switch focus between panes
			if m.focused == "left" {
				m.focused = "right"
			} else {
				m.focused = "left"
			}

		default:
			// Forward input to subprocess if right pane is focused
			if m.focused == "right" && m.ptmx != nil {
				// Handle special keys first
				switch msg.Type {
				case tea.KeyEnter:
					m.ptmx.Write([]byte("\r"))
				case tea.KeyBackspace:
					m.ptmx.Write([]byte("\x7f"))
				case tea.KeyCtrlD:
					m.ptmx.Write([]byte("\x04"))
				case tea.KeyCtrlC:
					m.ptmx.Write([]byte("\x03"))
				case tea.KeyUp:
					m.ptmx.Write([]byte("\x1b[A"))
				case tea.KeyDown:
					m.ptmx.Write([]byte("\x1b[B"))
				case tea.KeyLeft:
					m.ptmx.Write([]byte("\x1b[D"))
				case tea.KeyRight:
					m.ptmx.Write([]byte("\x1b[C"))
				case tea.KeyHome:
					m.ptmx.Write([]byte("\x1b[H"))
				case tea.KeyEnd:
					m.ptmx.Write([]byte("\x1b[F"))
				case tea.KeyPgUp:
					m.ptmx.Write([]byte("\x1b[5~"))
				case tea.KeyPgDown:
					m.ptmx.Write([]byte("\x1b[6~"))
				case tea.KeyDelete:
					m.ptmx.Write([]byte("\x1b[3~"))
				case tea.KeyTab:
					m.ptmx.Write([]byte("\t"))
				case tea.KeyEsc:
					m.ptmx.Write([]byte("\x1b"))
				default:
					// Handle space key explicitly
					if msg.String() == " " {
						m.ptmx.Write([]byte(" "))
					} else if msg.String() != "" && msg.Type == tea.KeyRunes {
						// Regular character input
						m.ptmx.Write([]byte(msg.String()))
					}
				}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate pane sizes (60/40 split)
	leftWidth := int(float64(m.width) * 0.6)
	rightWidth := m.width - leftWidth

	// Define styles
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	activeBorderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("86"))

	// Apply borders based on focus
	leftStyle := borderStyle
	rightStyle := borderStyle

	if m.focused == "left" {
		leftStyle = activeBorderStyle
	} else {
		rightStyle = activeBorderStyle
	}

	// Prepare left pane content
	leftContent := leftStyle.
		Padding(1, 2).
		Width(leftWidth - 2).
		Render(m.leftContent)

	// Render right pane with terminal or raw output
	rightContent := ""
	if m.useTerminal && m.terminal != nil {
		// Use terminal emulation
		rightContent = renderTerminal(m.terminal, rightWidth-6, m.height-4)
	}

	// If terminal emulation is disabled or failed, use raw output
	if !m.useTerminal || rightContent == "" {
		rightContent = renderRawOutput(m.rightOutput, rightWidth-6, m.height-4)
	}

	rightRendered := rightStyle.
		Padding(1, 2).
		Width(rightWidth - 2).
		Render(rightContent)

	// Join panes horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightRendered)
}

// renderRawOutput renders raw terminal output
func renderRawOutput(rawData []byte, width, height int) string {
	if len(rawData) == 0 {
		return ""
	}

	// Convert to string
	content := string(rawData)

	// Split into lines and take the last N lines that fit in height
	lines := strings.Split(content, "\n")

	// Take only the lines that fit in our display area
	startIdx := 0
	if len(lines) > height {
		startIdx = len(lines) - height
	}

	visibleLines := lines[startIdx:]

	// Truncate lines that are too wide
	for i, line := range visibleLines {
		if len(line) > width {
			visibleLines[i] = line[:width]
		}
	}

	return strings.Join(visibleLines, "\n")
}

// renderTerminal converts the VT10x terminal state to a string for display
func renderTerminal(term vt10x.Terminal, width, height int) string {
	if term == nil {
		return ""
	}

	// Lock the terminal state while reading
	term.Lock()
	defer term.Unlock()

	var output strings.Builder

	// Get terminal dimensions
	cols, rows := term.Size()

	// Render each row
	for row := 0; row < rows && row < height; row++ {
		if row > 0 {
			output.WriteString("\n")
		}

		// Render each column in the row
		for col := 0; col < cols && col < width; col++ {
			glyph := term.Cell(col, row)

			// Apply styling based on glyph attributes
			style := lipgloss.NewStyle()

			// Apply foreground color
			if glyph.FG != vt10x.DefaultFG {
				style = style.Foreground(convertColor(glyph.FG))
			}

			// Apply background color
			if glyph.BG != vt10x.DefaultBG {
				style = style.Background(convertColor(glyph.BG))
			}

			// Check mode flags for styling
			// Bold
			if glyph.Mode&0x01 != 0 {
				style = style.Bold(true)
			}
			// Underline
			if glyph.Mode&0x02 != 0 {
				style = style.Underline(true)
			}
			// Reverse
			if glyph.Mode&0x04 != 0 {
				style = style.Reverse(true)
			}
			// Blink
			if glyph.Mode&0x08 != 0 {
				style = style.Blink(true)
			}
			// Dim/Faint
			if glyph.Mode&0x10 != 0 {
				style = style.Faint(true)
			}
			// Italic
			if glyph.Mode&0x20 != 0 {
				style = style.Italic(true)
			}
			// Strikethrough
			if glyph.Mode&0x200 != 0 {
				style = style.Strikethrough(true)
			}

			if glyph.Char != 0 {
				output.WriteString(style.Render(string(glyph.Char)))
			} else {
				output.WriteString(" ")
			}
		}
	}

	return output.String()
}

// convertColor converts vt10x colors to lipgloss colors
func convertColor(c vt10x.Color) lipgloss.Color {
	// Check if it's a default color (no color set)
	if c == vt10x.DefaultFG || c == vt10x.DefaultBG {
		return lipgloss.Color("")
	}

	// Extract RGB components from the color
	// vt10x.Color encodes RGB in the lower 24 bits for true colors
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF

	// Check if this is a true color (RGB) by seeing if upper byte is set
	if c > 0xFFFFFF {
		// This is a special color (default or other)
		// Handle basic ANSI colors (0-15)
		if c <= 15 {
			return lipgloss.Color(fmt.Sprintf("%d", c))
		}
		// Handle 256 color palette
		if c <= 255 {
			return lipgloss.Color(fmt.Sprintf("%d", c))
		}
		return lipgloss.Color("")
	}

	// Handle basic ANSI colors by value
	switch c {
	case 0: // Black
		return lipgloss.Color("0")
	case 1: // Red
		return lipgloss.Color("1")
	case 2: // Green
		return lipgloss.Color("2")
	case 3: // Yellow
		return lipgloss.Color("3")
	case 4: // Blue
		return lipgloss.Color("4")
	case 5: // Magenta
		return lipgloss.Color("5")
	case 6: // Cyan
		return lipgloss.Color("6")
	case 7: // White/Light Grey
		return lipgloss.Color("7")
	case 8: // Bright Black/Dark Grey
		return lipgloss.Color("8")
	case 9: // Bright Red
		return lipgloss.Color("9")
	case 10: // Bright Green
		return lipgloss.Color("10")
	case 11: // Bright Yellow
		return lipgloss.Color("11")
	case 12: // Bright Blue
		return lipgloss.Color("12")
	case 13: // Bright Magenta
		return lipgloss.Color("13")
	case 14: // Bright Cyan
		return lipgloss.Color("14")
	case 15: // Bright White
		return lipgloss.Color("15")
	}

	// For 256 colors (16-255)
	if c >= 16 && c <= 255 {
		return lipgloss.Color(fmt.Sprintf("%d", c))
	}

	// For RGB true colors, format as hex
	if r > 0 || g > 0 || b > 0 {
		return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
	}

	// Default fallback
	return lipgloss.Color(fmt.Sprintf("%d", c))
}

func main() {
	// Check for command-line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: agate <command>")
		fmt.Println("Example: agate claude")
		fmt.Println("Example: agate codex")
		os.Exit(1)
	}

	subprocess := os.Args[1]

	p := tea.NewProgram(initialModel(subprocess), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}