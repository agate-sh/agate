package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"agate/tmux"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
	"github.com/spf13/cobra"
)

//go:embed ascii-art.txt
var asciiArt string

var (
	paneBaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)
	paneActiveStyle = paneBaseStyle.Copy().BorderForeground(lipgloss.Color("86"))
	asciiArtColor = "#9d87ae" // Color used for ASCII art and left pane
	activeBorderGray = "250" // Brighter gray for active non-tmux pane borders
)

type sessionMode int

const (
	modePreview sessionMode = iota // Read-only preview
	modeAttached                   // Full screen tmux
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
	mode         sessionMode // Current interaction mode
	agentConfig  AgentConfig // Configuration for the specific agent
	footer       *Footer     // Footer component for shortcuts
	helpDialog   *HelpDialog // Help dialog overlay
	showHelp     bool        // Whether help dialog is visible
}

func initialModel(subprocess string) model {
	// Get agent configuration based on subprocess name
	agentConfig := GetAgentConfig(subprocess)

	// Create styled ASCII art with the specified color
	asciiStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(asciiArtColor))

	styledAscii := asciiStyle.Render(asciiArt)

	// Create footer and help components
	footer := NewFooter()
	footer.SetAgentConfig(agentConfig)
	footer.SetFocus("right") // Start with right pane focused
	footer.SetMode("preview") // Start in preview mode

	return model{
		focused:     "right", // Focus on right pane for subprocess interaction
		leftContent: styledAscii, // Just show ASCII art, instructions moved to footer
		subprocess:  subprocess,
		mode:        modePreview, // Start in preview mode
		agentConfig: agentConfig,
		footer:      footer,
		helpDialog:  NewHelpDialog(),
		showHelp:    false,
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

		// Update component sizes
		m.footer.SetSize(msg.Width, 1)
		m.helpDialog.SetSize(msg.Width, msg.Height)

		// Update tmux session size if it exists and we're in preview mode
		if m.tmuxSession != nil && m.mode == modePreview {
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

		// Just show ASCII art - instructions are in footer now
		m.leftContent = styledAscii

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

		// Continue monitoring (increased frequency for better responsiveness)
		return m, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
			if m.tmuxSession != nil && m.mode == modePreview {
				return waitForTmuxOutput(m.tmuxSession)()
			}
			return nil
		})


	case tmuxAttachedMsg:
		m.mode = modeAttached
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii

		// Update footer for attached mode
		m.footer.SetMode("attached")

		// Wait for detach in background and send message when it happens
		return m, func() tea.Msg {
			<-msg.detachCh
			return tmuxDetachedMsg{}
		}

	case tmuxDetachedMsg:
		m.mode = modePreview
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(asciiArtColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii

		// Update footer back to preview mode
		m.footer.SetMode("preview")

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
		m.leftContent = styledAscii + fmt.Sprintf("\n\nError: %v", msg.error)

	case tea.KeyMsg:
		// If help dialog is visible, any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Handle different modes
		switch m.mode {
		case modeAttached:
			// When fully attached, don't process keys in Bubble Tea - tmux handles them
			return m, nil


		case modePreview:
			// Preview mode - handle navigation and mode switches only
			switch msg.String() {
			case "enter":
				// Enter key attaches to full tmux when right pane is focused
				if m.focused == "right" && m.tmuxSession != nil {
					return m, func() tea.Msg {
						detachCh, err := m.tmuxSession.Attach()
						if err != nil {
							return errMsg{err}
						}
						return tmuxAttachedMsg{detachCh: detachCh}
					}
				}

			case "ctrl+c", "q":
				if m.focused == "left" {
					// Cleanup
					if m.tmuxSession != nil {
						m.tmuxSession.Kill()
					}
					return m, tea.Quit
				}

			case "?":
				// Show help dialog
				m.showHelp = true
				return m, nil

			case "0":
				// Switch to right pane (tmux pane)
				m.focused = "right"
				m.footer.SetFocus(m.focused)
				return m, nil

			case "1":
				// Switch to left pane (info pane)
				m.focused = "left"
				m.footer.SetFocus(m.focused)
				return m, nil

			}
			// In preview mode, we don't forward any keys to tmux
			// This eliminates the SendKeys latency issue
		}
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Handle attached mode
	if m.mode == modeAttached {
		// Full screen tmux attachment
		attachedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#86")).
			Bold(true).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width).
			Height(m.height)
		return attachedStyle.Render("Attached to tmux session\n\nPress Ctrl+Q to detach")
	}

	// Reserve space for proper border rendering, footer, titles, and top padding
	// Subtract 5 from height (1 for top, 1 for bottom of terminal, 1 for footer, 1 for titles, 1 for top padding)
	availableHeight := m.height - 5

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

	// Create pane titles with index and company name
	// Create title strings with proper focus styling
	var leftTitle, rightTitle string

	if m.focused == "left" {
		// Left pane focused: both "Info" and "[1]" turn white
		focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // White
		leftTitle = focusStyle.Render("Info") + " " + focusStyle.Render("[1]")
		// Right pane unfocused: company name white, number gray
		rightTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(m.agentConfig.CompanyName) + " " +
					lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("[0]")
	} else {
		// Right pane focused: both company name and "[0]" use agent color
		focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.agentConfig.BorderColor))
		rightTitle = focusStyle.Render(m.agentConfig.CompanyName) + " " + focusStyle.Render("[0]")
		// Left pane unfocused: "Info" white, "[1]" gray
		leftTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render("Info") + " " +
					lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("[1]")
	}

	// Start with base style (light gray borders) for both panes
	leftStyle := paneBaseStyle
	rightStyle := paneBaseStyle

	// Apply focus styling:
	if m.focused == "left" {
		leftStyle = paneBaseStyle.Copy().
			BorderForeground(lipgloss.Color(activeBorderGray))
	} else if m.focused == "right" {
		// Right pane uses agent color when focused
		rightStyle = paneBaseStyle.Copy().
			BorderForeground(lipgloss.Color(m.agentConfig.BorderColor))
	}

	// Render pane content WITHOUT titles (titles will be separate)
	leftContent := leftStyle.Copy().
		Width(leftWidth).
		Height(availableHeight).
		Render(m.leftContent)

	rightRendered := rightStyle.Copy().
		Width(rightWidth).
		Height(availableHeight).
		Render(m.rightOutput)

	// Add left padding to titles and combine with panes
	leftTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(leftTitle)
	rightTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(rightTitle)

	// Add titles above the bordered panes
	leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitleWithPadding, leftContent)
	rightWithTitle := lipgloss.JoinVertical(lipgloss.Left, rightTitleWithPadding, rightRendered)

	// Join panes horizontally (now with titles above)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, rightWithTitle)

	// Add top padding to the entire pane layout
	panesWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(panes)

	// Add footer at the bottom
	mainView := lipgloss.JoinVertical(lipgloss.Left, panesWithPadding, m.footer.View())

	// If help dialog is visible, overlay it
	if m.showHelp {
		// Create a simple background model that just returns the main view
		bgModel := &simpleModel{content: mainView}

		// Create overlay with help dialog on top of main view
		overlayModel := overlay.New(
			m.helpDialog,    // Foreground (help dialog)
			bgModel,         // Background (main view)
			overlay.Center,  // X Position
			overlay.Center,  // Y Position
			0,              // X Offset
			0,              // Y Offset
		)
		return overlayModel.View()
	}

	return mainView
}

func (m model) tmuxPreviewDimensions() (int, int) {
	if m.width == 0 || m.height == 0 {
		return 0, 0
	}

	// Use the same adjusted dimensions as in View()
	availableHeight := m.height - 5 // Account for footer, titles, and top padding

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


func checkTmuxInstalled() error {
	if _, err := os.Stat("/usr/local/bin/tmux"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/tmux"); os.IsNotExist(err) {
			return fmt.Errorf("tmux is not installed. Please install tmux to use Agate.\nOn macOS: brew install tmux\nOn Ubuntu/Debian: sudo apt-get install tmux")
		}
	}
	return nil
}

func runAgent(subprocess string) error {
	if err := checkTmuxInstalled(); err != nil {
		return err
	}

	p := tea.NewProgram(initialModel(subprocess), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %v", err)
	}
	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "agate <agent>",
		Short: "A tmux-based terminal UI for AI agents",
		Long: `Agate provides a split-pane terminal interface for interacting with AI agents.

Supports any agent name (claude, amp, cn, etc.) and automatically configures
colors and settings based on the agent type.

Agate provides two interaction modes:
  Preview Mode (default): Read-only view with fast, lag-free rendering
  Attached Mode (Enter): Full tmux experience with complete control

Press Enter when focused on the right pane to attach to tmux.
Press Ctrl+Q when attached to detach back to preview.
Press ? for help once running.

Examples:
  agate claude    # Launch with Claude
  agate amp       # Launch with Amp
  agate cn        # Launch with Continue`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(args[0])
		},
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// simpleModel is a basic tea.Model that just displays static content
// Used as background for the overlay
type simpleModel struct {
	content string
}

func (s *simpleModel) Init() tea.Cmd {
	return nil
}

func (s *simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, nil
}

func (s *simpleModel) View() string {
	return s.content
}

