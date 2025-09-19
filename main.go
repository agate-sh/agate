package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"agate/tmux"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed ascii-art.txt
var asciiArt string

var (
	paneBaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)
	paneActiveStyle = paneBaseStyle.Copy().BorderForeground(lipgloss.Color("86"))
	asciiArtColor = "#9d87ae" // Color used for ASCII art and left pane
	activeBorderGray = "250" // Brighter gray for active non-tmux pane borders
)

type model struct {
	width        int
	height       int
	leftContent  string
	rightOutput  string // Raw tmux pane content with ANSI codes
	tmuxSession  *tmux.TmuxSession
	ready        bool
	focused      string // "left" or "right"
	err          error
	subprocess   string // Command to run in right pane
	attached     bool   // Whether we're attached to tmux session
	ctrlAPressed bool   // Whether Ctrl+A was recently pressed
	agentConfig  AgentConfig // Configuration for the specific agent
}

func initialModel(subprocess string) model {
	// Get agent configuration based on subprocess name
	agentConfig := GetAgentConfig(subprocess)

	// Create styled ASCII art with the specified color
	asciiStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(asciiArtColor))

	styledAscii := asciiStyle.Render(asciiArt)

	return model{
		focused:     "right", // Focus on right pane for subprocess interaction
		leftContent: styledAscii + fmt.Sprintf("\n\nStarting %s in tmux session...\n\nUse Tab to switch between panes.\nType 'q' in left pane to quit.\nPress Ctrl+A, A to attach full-screen.", subprocess),
		subprocess:  subprocess,
		agentConfig: agentConfig,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		startTmuxSession(m.subprocess),
		tea.EnterAltScreen,
	)
}

func startTmuxSession(subprocess string) tea.Cmd {
	return func() tea.Msg {
		// Create a new tmux session
		session := tmux.NewTmuxSession(subprocess, subprocess)

		// Get current working directory
		workDir, err := os.Getwd()
		if err != nil {
			workDir = os.Getenv("HOME")
		}

		// Start the tmux session
		err = session.Start(workDir)
		if err != nil {
			return errMsg{err}
		}

		return tmuxSessionStartedMsg{
			session: session,
		}
	}
}

func waitForTmuxOutput(session *tmux.TmuxSession) tea.Cmd {
	return func() tea.Msg {
		// Capture tmux pane content with ANSI codes preserved
		content, err := session.CapturePaneContent()
		if err != nil {
			return tmuxOutputMsg{content: ""}
		}

		// Check if output has changed
		updated, hasPrompt := session.HasUpdated()
		if !updated {
			return tmuxOutputMsg{content: ""}
		}

		// Return the raw content with ANSI codes
		return tmuxOutputMsg{content: content, hasPrompt: hasPrompt}
	}
}

type tmuxSessionStartedMsg struct {
	session *tmux.TmuxSession
}

type tmuxOutputMsg struct {
	content   string
	hasPrompt bool
}

type ctrlATimeoutMsg struct{}

type tmuxAttachedMsg struct {
	detachCh chan struct{}
}

type tmuxDetachedMsg struct{}

type errMsg struct{ error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update tmux session size if it exists
		if m.tmuxSession != nil && !m.attached {
			if contentWidth, contentHeight := m.tmuxPreviewDimensions(); contentWidth > 0 && contentHeight > 0 {
				m.tmuxSession.SetDetachedSize(contentWidth, contentHeight)
			}
		}

	case tmuxSessionStartedMsg:
		m.tmuxSession = msg.session

		// Keep the ASCII art with updated status
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		statusText := fmt.Sprintf("\n\nTmux session '%s' started!\n\nUse Tab to switch between panes.\nType 'q' in left pane to quit.\nPress Ctrl+A, A to attach full-screen.\nPress Ctrl+Q when attached to detach.", m.tmuxSession.GetSessionName())

		if m.ctrlAPressed {
			statusText += "\n\n🔄 Press A to attach full-screen..."
		}

		m.leftContent = styledAscii + statusText

		// Set initial tmux session size using consistent calculation
		if m.ready {
			if contentWidth, contentHeight := m.tmuxPreviewDimensions(); contentWidth > 0 && contentHeight > 0 {
				m.tmuxSession.SetDetachedSize(contentWidth, contentHeight)
			}
		}

		// Start monitoring tmux output
		return m, waitForTmuxOutput(m.tmuxSession)

	case tmuxOutputMsg:
		// Update output if there's new content
		if msg.content != "" {
			m.rightOutput = msg.content
		}

		// Continue monitoring
		return m, tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
			if m.tmuxSession != nil && !m.attached {
				return waitForTmuxOutput(m.tmuxSession)()
			}
			return nil
		})

	case ctrlATimeoutMsg:
		// Reset Ctrl+A state on timeout
		m.ctrlAPressed = false

	case tmuxAttachedMsg:
		m.attached = true
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\nAttached to tmux session!\n\nPress Ctrl+Q to detach and return here.")

		// Wait for detach in background and send message when it happens
		return m, func() tea.Msg {
			<-msg.detachCh
			return tmuxDetachedMsg{}
		}

	case tmuxDetachedMsg:
		m.attached = false
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\nDetached from tmux session.\n\nUse Tab to switch between panes.\nType 'q' in left pane to quit.\nPress Ctrl+A, A to attach full-screen.")

		// Immediately resize the tmux session to current window dimensions
		if m.tmuxSession != nil && m.ready {
			if contentWidth, contentHeight := m.tmuxPreviewDimensions(); contentWidth > 0 && contentHeight > 0 {
				m.tmuxSession.SetDetachedSize(contentWidth, contentHeight)
			}
		}

		// Resume monitoring and trigger window size recalculation
		return m, tea.Batch(
			waitForTmuxOutput(m.tmuxSession),
			tea.WindowSize(), // Trigger complete UI layout recalculation
		)

	case errMsg:
		m.err = msg.error
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\nError: %v\n\nPress 'q' to quit.", msg.error)

	case tea.KeyMsg:
		// If attached, only handle detach key
		if m.attached {
			// When attached, don't process keys in Bubble Tea - tmux handles them
			return m, nil
		}

		// Handle Ctrl+A sequence first
		switch msg.String() {
		case "ctrl+a":
			// Start Ctrl+A sequence
			m.ctrlAPressed = true
			return m, tea.Tick(time.Second*2, func(time.Time) tea.Msg {
				return ctrlATimeoutMsg{}
			})

		case "a":
			if m.ctrlAPressed {
				// Complete Ctrl+A, A sequence - attach to tmux session
				m.ctrlAPressed = false
				if m.tmuxSession != nil && !m.attached {
					return m, func() tea.Msg {
						detachCh, err := m.tmuxSession.Attach()
						if err != nil {
							return errMsg{err}
						}
						return tmuxAttachedMsg{detachCh: detachCh}
					}
				}
				return m, nil
			}
			// If not in Ctrl+A sequence, fall through to normal handling

		case "ctrl+c", "q":
			if m.focused == "left" {
				// Cleanup
				if m.tmuxSession != nil {
					m.tmuxSession.Kill()
				}
				return m, tea.Quit
			}

		case "tab":
			// Reset Ctrl+A state on tab
			m.ctrlAPressed = false
			// Switch focus between panes
			if m.focused == "left" {
				m.focused = "right"
			} else {
				m.focused = "left"
			}
			return m, nil
		}

		// Reset Ctrl+A state for any other key (except those already handled)
		if msg.String() != "ctrl+a" && msg.String() != "a" && msg.String() != "tab" && msg.String() != "ctrl+c" && msg.String() != "q" {
			m.ctrlAPressed = false
		}

		// Handle input forwarding to tmux session when not attached
		if m.tmuxSession != nil && !m.attached {
			// Don't forward 'a' if we're in Ctrl+A sequence
			if msg.String() == "a" && m.ctrlAPressed {
				return m, nil
			}

			// Use SendKeys for detached sessions
			var keys string
			switch msg.Type {
			case tea.KeyEnter:
				keys = "Enter"
			case tea.KeyBackspace:
				keys = "BSpace"
			case tea.KeyCtrlD:
				keys = "C-d"
			case tea.KeyCtrlC:
				keys = "C-c"
			case tea.KeyUp:
				keys = "Up"
			case tea.KeyDown:
				keys = "Down"
			case tea.KeyLeft:
				keys = "Left"
			case tea.KeyRight:
				keys = "Right"
			case tea.KeyHome:
				keys = "Home"
			case tea.KeyEnd:
				keys = "End"
			case tea.KeyPgUp:
				keys = "PageUp"
			case tea.KeyPgDown:
				keys = "PageDown"
			case tea.KeyDelete:
				keys = "Delete"
			case tea.KeyTab:
				keys = "Tab"
			case tea.KeyEsc:
				keys = "Escape"
			default:
				// Regular character input
				if msg.String() == " " {
					keys = "Space"
				} else if msg.String() != "" && msg.Type == tea.KeyRunes {
					// For regular characters, just send them directly
					m.tmuxSession.SendKeys(msg.String())
					return m, nil
				}
			}

			if keys != "" {
				m.tmuxSession.SendKeys(keys)
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// If attached to tmux, show full screen message
	if m.attached {
		attachedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#86")).
			Bold(true).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width).
			Height(m.height)

		return attachedStyle.Render("Attached to tmux session\n\nPress Ctrl+Q to detach")
	}

	// Reserve space for proper border rendering
	// Subtract 2 from height (1 for top, 1 for bottom of terminal)
	availableHeight := m.height - 2

	// Calculate the actual frame sizes to be precise
	frameWidth := paneBaseStyle.GetHorizontalFrameSize()

	// We need space for both panes' frames plus a small buffer for the right edge
	totalFrameWidth := frameWidth * 2 + 4 // 2 panes + 4 char buffer for right border

	// Available width for actual content across both panes
	availableContentWidth := m.width - totalFrameWidth

	// Split content 60/40, then add frame back to each pane
	leftContentWidth := int(float64(availableContentWidth) * 0.6)
	rightContentWidth := availableContentWidth - leftContentWidth

	leftWidth := leftContentWidth + frameWidth
	rightWidth := rightContentWidth + frameWidth

	// Start with base style (light gray borders) for both panes
	leftStyle := paneBaseStyle
	rightStyle := paneBaseStyle

	// Apply focus styling:
	// - Left pane uses brighter gray when active
	// - Right pane uses agent color when active
	if m.focused == "left" {
		leftStyle = paneBaseStyle.Copy().
			BorderForeground(lipgloss.Color(activeBorderGray))
	} else if m.focused == "right" {
		rightStyle = paneBaseStyle.Copy().
			BorderForeground(lipgloss.Color(m.agentConfig.BorderColor))
	}

	// Prepare left pane content
	leftContent := leftStyle.Copy().
		Width(leftWidth).
		Height(availableHeight).
		Render(m.leftContent)

	// Render right pane with tmux output
	// The rightOutput already contains ANSI codes which lipgloss will handle
	rightContent := m.rightOutput

	rightRendered := rightStyle.Copy().
		Width(rightWidth).
		Height(availableHeight).
		Render(rightContent)

	// Join panes horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightRendered)
}

func (m model) tmuxPreviewDimensions() (int, int) {
	if m.width == 0 || m.height == 0 {
		return 0, 0
	}

	// Use the same adjusted dimensions as in View()
	availableHeight := m.height - 2

	// Same calculation as View() - calculate actual content width
	frameWidth := paneBaseStyle.GetHorizontalFrameSize()
	totalFrameWidth := frameWidth * 2 + 4
	availableContentWidth := m.width - totalFrameWidth
	rightContentWidth := availableContentWidth - int(float64(availableContentWidth) * 0.6)

	// Use the actual content width for tmux (this is the usable space)
	contentWidth := rightContentWidth
	if contentWidth < 1 {
		contentWidth = 1
	}

	frameHeight := paneBaseStyle.GetVerticalFrameSize()
	contentHeight := availableHeight - frameHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	return contentWidth, contentHeight
}

func main() {
	// Check for command-line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: agate <command>")
		fmt.Println("Example: agate claude")
		fmt.Println("Example: agate codex")
		fmt.Println("\nAgate will create a tmux session for the specified command.")
		fmt.Println("Press Ctrl+A, A to attach to the tmux session full-screen.")
		fmt.Println("Press Ctrl+Q when attached to detach back to the preview.")
		os.Exit(1)
	}

	subprocess := os.Args[1]

	// Check if tmux is available
	if _, err := os.Stat("/usr/local/bin/tmux"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/tmux"); os.IsNotExist(err) {
			fmt.Println("Error: tmux is not installed. Please install tmux to use Agate.")
			fmt.Println("On macOS: brew install tmux")
			fmt.Println("On Ubuntu/Debian: sudo apt-get install tmux")
			os.Exit(1)
		}
	}

	p := tea.NewProgram(initialModel(subprocess), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
