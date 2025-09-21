package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agate/config"
	"agate/git"
	"agate/overlay"
	"agate/tmux"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

//go:embed ascii-art.txt
var asciiArt string

var (
	paneBaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderMuted)).
			Padding(1, 2)
	paneActiveStyle = paneBaseStyle.Copy().BorderForeground(lipgloss.Color(agateColor))
	agateColor = "#9d87ae"   // Agate purple for branding and active elements
	activeBorderGray = "250" // Brighter gray for active non-tmux pane borders
	textDescription = "250"  // Light gray for descriptions and help text
	textMuted = "240"       // Darker gray for very subtle text like file paths
	borderMuted = "240"     // Standard border color
	separatorColor = "238"  // Very dark gray for separators
	warningYellow = "220"   // Yellow for warnings/highlights
)

type sessionMode int

const (
	modePreview sessionMode = iota // Read-only preview
)

type model struct {
	width               int
	height              int
	leftContent         string
	rightOutput         string // Raw tmux pane content with ANSI codes
	tmuxSession         *tmux.TmuxSession
	ready               bool
	focused             string // "left" or "right"
	err                 error
	subprocess          string // Command to run in right pane
	mode                sessionMode // Current interaction mode
	agentConfig         AgentConfig // Configuration for the specific agent
	keyMap              *KeyMap     // Centralized keybindings
	shortcutOverlay     *ShortcutOverlay // Manages contextual shortcuts
	footer              *Footer     // Footer component for shortcuts
	helpDialog          *HelpDialog // Help dialog overlay
	showHelp            bool        // Whether help dialog is visible
	worktreeManager     *git.WorktreeManager // Git worktree management
	worktreeList        *WorktreeList // Worktree list component
	worktreeDialog      *WorktreeDialog // Worktree creation dialog
	worktreeConfirm     *WorktreeConfirmDialog // Worktree deletion confirmation
	showWorktreeDialog  bool       // Whether showing worktree creation dialog
	showWorktreeConfirm bool       // Whether showing worktree deletion confirmation
	repoDialog          *RepoDialog // Repository search dialog
	showRepoDialog      bool       // Whether showing repository dialog
	welcomeOverlay      *WelcomeOverlay // Welcome overlay for first-time users
	showWelcomeOverlay  bool       // Whether showing welcome overlay
	debugLogger         *DebugLogger // Debug logger for development
	debugOverlay        *DebugOverlay // Debug overlay for development
	showDebugOverlay    bool        // Whether showing debug overlay
}

func initialModel(subprocess string) model {
	// Get agent configuration based on subprocess name
	agentConfig := GetAgentConfig(subprocess)

	// Create keybindings and shortcut overlay
	keyMap := NewKeyMap()
	shortcutOverlay := NewShortcutOverlay(keyMap)
	shortcutOverlay.SetFocus("right") // Start with right pane focused
	shortcutOverlay.SetMode("preview") // Start in preview mode

	// Create footer and help components
	footer := NewFooter()
	footer.SetAgentConfig(agentConfig)
	footer.SetShortcutOverlay(shortcutOverlay)
	footer.SetFocus("right") // Start with right pane focused
	footer.SetMode("preview") // Start in preview mode

	// Initialize worktree manager
	worktreeManager, err := git.NewWorktreeManager()
	if err != nil {
		// Log error but don't fail - app can still work without worktree features
		fmt.Printf("Warning: failed to initialize worktree manager: %v\n", err)
	}

	// Initialize worktree components
	var worktreeList *WorktreeList
	if worktreeManager != nil {
		worktreeList = NewWorktreeList(worktreeManager)
	}

	// Check if welcome overlay should be shown
	welcomeShown, _ := config.GetWelcomeShownState()
	showWelcomeOverlay := !welcomeShown

	// Initialize debug logger
	debugLogger := InitDebugLogger()

	// Test debug logging
	DebugLog("Debug logger initialized successfully")

	// Initialize debug overlay
	debugOverlay := NewDebugOverlay(debugLogger)

	// Set up debug logging for git package (always enabled now)
	git.DebugLog = DebugLog

	return model{
		focused:             "right", // Focus on right pane for subprocess interaction
		leftContent:         "", // No longer using ASCII art
		subprocess:          subprocess,
		mode:               modePreview, // Start in preview mode
		agentConfig:        agentConfig,
		keyMap:             keyMap,
		shortcutOverlay:    shortcutOverlay,
		footer:             footer,
		helpDialog:         NewHelpDialog(keyMap),
		showHelp:           false,
		worktreeManager:    worktreeManager,
		worktreeList:       worktreeList,
		showWorktreeDialog: false,
		showWorktreeConfirm: false,
		showRepoDialog:     false,
		welcomeOverlay:     NewWelcomeOverlay(),
		showWelcomeOverlay: showWelcomeOverlay,
		debugLogger:        debugLogger,
		debugOverlay:       debugOverlay,
		showDebugOverlay:   false,
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


type tmuxDetachedMsg struct{}

type autoAttachMsg struct{}

type initializationCompleteMsg struct{}

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


		// Update debug overlay size
		if m.debugOverlay != nil {
			m.debugOverlay.SetSize(msg.Width, msg.Height)
		}

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
			Foreground(lipgloss.Color(agateColor))
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


	case autoAttachMsg:
		// Auto-attach to the tmux session after it's ready
		if m.tmuxSession != nil && m.focused == "right" {
			// Use blocking attachment
			return m, func() tea.Msg {
				detachCh, err := m.tmuxSession.Attach()
				if err != nil {
					return errMsg{err}
				}
				// Block until detachment
				<-detachCh
				return tmuxDetachedMsg{}
			}
		}
		return m, nil

	case initializationCompleteMsg:
		// Close the worktree dialog and auto-attach
		m.showWorktreeDialog = false
		m.worktreeDialog = nil

		// Auto-attach to the tmux session
		if m.tmuxSession != nil && m.focused == "right" {
			return m, tea.Batch(
				tea.ClearScreen,
				func() tea.Msg {
					detachCh, err := m.tmuxSession.Attach()
					if err != nil {
						return errMsg{err}
					}
					// Block until detachment
					<-detachCh
					return tmuxDetachedMsg{}
				},
			)
		}
		return m, tea.ClearScreen

	case tmuxDetachedMsg:
		m.mode = modePreview
		asciiStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(agateColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii

		// Update footer back to preview mode
		m.footer.SetMode("preview")
		m.shortcutOverlay.SetMode("preview")

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
			Foreground(lipgloss.Color(agateColor))
		styledAscii := asciiStyle.Render(asciiArt)
		m.leftContent = styledAscii + fmt.Sprintf("\n\nError: %v", msg.error)

	// Worktree dialog messages
	case WorktreeCreatedMsg:
		// Worktree created successfully - start tmux session but keep dialog open
		// Refresh the worktree list
		if m.worktreeList != nil {
			m.worktreeList.Refresh()
		}

		// Create and switch to new tmux session for the worktree
		if msg.Worktree != nil && m.subprocess != "" {
			// Kill the existing tmux session if any
			if m.tmuxSession != nil {
				_ = m.tmuxSession.Kill()
				m.tmuxSession = nil
			}

			// Create new tmux session for the worktree
			sessionName := fmt.Sprintf("%s_%s", filepath.Base(msg.Worktree.Path), msg.Worktree.Branch)
			newSession := tmux.NewTmuxSession(sessionName, m.subprocess)

			// Start the new session in the worktree directory
			if err := newSession.Start(msg.Worktree.Path); err == nil {
				m.tmuxSession = newSession
				// Switch focus to right pane
				m.focused = "right"
				// Update footer focus
				m.footer.SetFocus("right")
				m.shortcutOverlay.SetFocus("right")

				// Start monitoring the new session
				return m, waitForTmuxOutput(newSession)
			}
		}
		return m, nil

	case WorktreeInitializationCompleteMsg:
		// Initialization complete - close dialog and auto-attach
		m.showWorktreeDialog = false
		m.worktreeDialog = nil

		// Auto-attach to the tmux session
		if m.tmuxSession != nil && m.focused == "right" {
			return m, tea.Batch(
				tea.ClearScreen,
				func() tea.Msg {
					detachCh, err := m.tmuxSession.Attach()
					if err != nil {
						return errMsg{err}
					}
					// Block until detachment
					<-detachCh
					return tmuxDetachedMsg{}
				},
			)
		}
		return m, tea.ClearScreen

	case WorktreeCreationErrorMsg:
		// Worktree creation failed - dialog will handle error display
		return m, nil

	case WorktreeDialogCancelledMsg:
		// Dialog cancelled
		m.showWorktreeDialog = false
		m.worktreeDialog = nil
		return m, nil

	case WorktreeDeletedMsg:
		// Worktree deleted successfully
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		if m.worktreeList != nil {
			m.worktreeList.Refresh()
		}
		return m, nil

	case WorktreeDeletionErrorMsg:
		// Worktree deletion failed
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		m.err = fmt.Errorf("failed to delete worktree: %s", msg.Error)
		return m, nil

	case WorktreeDeleteCancelledMsg:
		// Deletion cancelled
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		return m, nil

	case DebugOverlayClosedMsg:
		// Debug overlay closed
		m.showDebugOverlay = false
		return m, nil

	// Repository dialog messages
	case RepoAddedMsg:
		// Repository was successfully added
		m.showRepoDialog = false
		m.repoDialog = nil

		// Add to persistent config
		if err := config.AddRepository(msg.Path); err != nil {
			m.err = fmt.Errorf("failed to save repository: %v", err)
		} else {
			// Refresh the worktree list to include the new repo
			if m.worktreeList != nil {
				m.worktreeList.Refresh()
			}
		}
		return m, nil

	case RepoDialogCancelledMsg:
		// Repository dialog cancelled
		m.showRepoDialog = false
		m.repoDialog = nil
		return m, nil

	case tea.KeyMsg:
		// If welcome overlay is visible, any key closes it
		if m.showWelcomeOverlay {
			m.showWelcomeOverlay = false
			// Mark welcome as shown so it doesn't appear again
			_ = config.SetWelcomeShown(true)
			return m, nil
		}

		// If help dialog is visible, any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Handle debug overlay input (highest priority after welcome overlay)
		if m.showDebugOverlay && m.debugOverlay != nil {
			var cmd tea.Cmd
			overlay, cmd := m.debugOverlay.Update(msg)
			m.debugOverlay = overlay
			return m, cmd
		}

		// Handle worktree dialog input
		if m.showWorktreeDialog && m.worktreeDialog != nil {
			var cmd tea.Cmd
			model, cmd := m.worktreeDialog.Update(msg)
			m.worktreeDialog = model.(*WorktreeDialog)
			return m, cmd
		}

		// Handle worktree confirm dialog input
		if m.showWorktreeConfirm && m.worktreeConfirm != nil {
			var cmd tea.Cmd
			model, cmd := m.worktreeConfirm.Update(msg)
			m.worktreeConfirm = model.(*WorktreeConfirmDialog)
			return m, cmd
		}

		// Handle repo dialog input
		if m.showRepoDialog && m.repoDialog != nil {
			var cmd tea.Cmd
			model, cmd := m.repoDialog.Update(msg)
			m.repoDialog = model.(*RepoDialog)
			return m, cmd
		}

		// Handle preview mode - navigation and mode switches only
		switch {
		case key.Matches(msg, m.keyMap.AttachTmux):
			// Enter key on left pane opens/switches to worktree or main repo session
			if m.focused == "left" && m.worktreeList != nil {
				selectedItem := m.worktreeList.GetSelectedItem()
				if selectedItem != nil && m.subprocess != "" {
					var sessionPath string
					var sessionName string

					// Handle different types of selections
					if selectedItem.Type == "worktree" && selectedItem.Worktree != nil {
						// Selected a worktree
						sessionPath = selectedItem.Worktree.Path
						sessionName = fmt.Sprintf("%s_%s", filepath.Base(sessionPath), selectedItem.Worktree.Branch)
					} else if selectedItem.Type == "main_repo" {
						// Selected the main repository
						sessionPath = selectedItem.RepoPath
						sessionName = fmt.Sprintf("%s_main", filepath.Base(sessionPath))
					} else {
						// Invalid selection
						return m, nil
					}

					// Kill the existing tmux session if any
					if m.tmuxSession != nil {
						_ = m.tmuxSession.Kill()
						m.tmuxSession = nil
					}

					// Create new tmux session for the selected path
					newSession := tmux.NewTmuxSession(sessionName, m.subprocess)

					// Start the new session in the selected directory
					if err := newSession.Start(sessionPath); err == nil {
						m.tmuxSession = newSession
						// Switch focus to right pane for immediate interaction
						m.focused = "right"
						// Update footer focus
						m.footer.SetFocus("right")
						m.shortcutOverlay.SetFocus("right")
						// Start monitoring the new session and clear screen
						return m, tea.Batch(
							waitForTmuxOutput(newSession),
							tea.ClearScreen,
						)
					}
				}
			}
			// Enter key attaches to full tmux when right pane is focused
			if m.focused == "right" && m.tmuxSession != nil {
				// Use blocking attachment like Claude Squad - don't return to event loop during attachment
				return m, func() tea.Msg {
					detachCh, err := m.tmuxSession.Attach()
					if err != nil {
						return errMsg{err}
					}
					// Block until detachment like Claude Squad does
					<-detachCh
					return tmuxDetachedMsg{}
				}
			}

		case key.Matches(msg, m.keyMap.Quit):
			// Quit available from both panes
			if m.tmuxSession != nil {
				m.tmuxSession.Kill()
			}
			// Close debug panel and log file
			if m.debugLogger != nil {
				m.debugLogger.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Help):
			// Show help dialog
			m.showHelp = true
			return m, nil

		case key.Matches(msg, m.keyMap.FocusRight):
			// Switch to right pane (tmux pane)
			m.focused = "right"
			m.footer.SetFocus(m.focused)
			m.shortcutOverlay.SetFocus(m.focused)
			return m, nil

		case key.Matches(msg, m.keyMap.FocusLeft):
			// Switch focus to left pane (worktree pane)
			m.focused = "left"
			m.footer.SetFocus(m.focused)
			m.shortcutOverlay.SetFocus(m.focused)
			if m.worktreeList != nil {
				// Refresh worktree list when focusing
				m.worktreeList.Refresh()
			}
			return m, nil

		case key.Matches(msg, m.keyMap.AddRepo):
			// Add new repository using fzf search
			DebugLog("Creating new repo dialog...")
			m.repoDialog = NewRepoDialog()
			m.showRepoDialog = true
			// Initialize the repo dialog to start the repository discovery
			initCmd := m.repoDialog.Init()
			return m, initCmd

		case key.Matches(msg, m.keyMap.DebugOverlay):
			// Show debug overlay
			m.showDebugOverlay = true
			return m, nil

		case key.Matches(msg, m.keyMap.NewWorktree):
			// Create new worktree (available from both panes)
			if m.worktreeManager != nil {
				m.worktreeDialog = NewWorktreeDialog(m.worktreeManager, m.agentConfig)
				m.showWorktreeDialog = true
				return m, nil
			}

		case key.Matches(msg, m.keyMap.DeleteWorktree):
			// Delete worktree (when left pane focused)
			if m.focused == "left" && m.worktreeList != nil {
				selected := m.worktreeList.GetSelected()
				if selected != nil {
					m.worktreeConfirm = NewWorktreeConfirmDialog(selected, m.worktreeManager)
					m.showWorktreeConfirm = true
					return m, nil
				}
			}

		case key.Matches(msg, m.keyMap.Up):
			// Navigate up in worktree list
			if m.focused == "left" && m.worktreeList != nil {
				m.worktreeList.MoveUp()
				return m, nil
			}

		case key.Matches(msg, m.keyMap.Down):
			// Navigate down in worktree list
			if m.focused == "left" && m.worktreeList != nil {
				m.worktreeList.MoveDown()
				return m, nil
			}

		default:
			// Handle other key combinations
			switch msg.String() {
			}
		}

	case tea.MouseMsg:
		// Handle mouse events in preview mode only when right pane is focused
		if m.mode == modePreview && m.focused == "right" && m.tmuxSession != nil {
			switch msg.Type {
			case tea.MouseWheelUp:
				// Enter copy mode and scroll up
				m.tmuxSession.SendScrollUp()
			case tea.MouseWheelDown:
				// Scroll down (or exit copy mode if at bottom)
				m.tmuxSession.SendScrollDown()
			}
			// Trigger content refresh after scroll
			return m, waitForTmuxOutput(m.tmuxSession)
		}
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Reserve space for proper border rendering, footer, titles, top padding, and debug panel
	// Subtract 5 from height (1 for top, 1 for bottom of terminal, 1 for footer, 1 for titles, 1 for top padding)
	availableHeight := m.height - 5

	// Calculate the actual frame sizes to be precise
	frameWidth := paneBaseStyle.GetHorizontalFrameSize()

	// We need space for both panes' frames plus a small buffer for the right edge
	totalFrameWidth := frameWidth * 2 + 4 // 2 panes + 4 char buffer for right border

	// Available width for actual content across both panes
	availableContentWidth := m.width - totalFrameWidth

	// Split content 40/60, then add frame back to each pane
	leftContentWidth := int(float64(availableContentWidth) * 0.4)
	rightContentWidth := availableContentWidth - leftContentWidth

	leftWidth := leftContentWidth + frameWidth
	rightWidth := rightContentWidth + frameWidth

	// Create pane titles with index and company name
	// Create title strings with proper focus styling
	var leftTitle, rightTitle string

	// Left pane is always Repos & Worktrees now
	leftPaneTitle := "Repos & Worktrees"

	// Define consistent number style that always stays light gray
	numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Agent title with colored rectangle background
	agentTitleStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(m.agentConfig.BorderColor)).
		Foreground(lipgloss.Color("255")). // White text
		Padding(0, 1). // Add horizontal padding inside the rectangle
		Bold(true)

	// Create the rectangle style for the agent title
	agentTitleBox := agentTitleStyle.Copy().
		MarginRight(1) // Add space between box and shortcut

	if m.focused == "left" {
		// Left pane focused: title turns white and bold, show shortcuts in white too
		focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true) // White and bold
		shortcutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true) // White for shortcuts when focused
		leftTitle = focusStyle.Render(leftPaneTitle) + " " + shortcutStyle.Render("[r to add repo, w to add worktree]")
		// Right pane unfocused: agent name in colored box, number outside in gray
		rightTitle = agentTitleBox.Render(m.agentConfig.CompanyName) + numberStyle.Render("[0]")
	} else {
		// Right pane focused: agent name in colored box, show shortcut text instead of [0]
		shortcutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.agentConfig.BorderColor)).Bold(true)
		rightTitle = agentTitleBox.Render(m.agentConfig.CompanyName) + shortcutStyle.Render("[↵ attach to tmux]")
		// Left pane unfocused: title white and bold, number stays gray
		leftTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Render(leftPaneTitle) + " " +
					numberStyle.Render("[1]")
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

	// Left pane always shows worktree list now
	var leftPaneContent string
	if m.worktreeList != nil {
		// Update worktree list size and render
		m.worktreeList.SetSize(leftContentWidth, availableHeight-frameWidth)
		leftPaneContent = m.worktreeList.View()
	} else {
		// Fallback if worktree manager failed to initialize
		leftPaneContent = "Worktree manager not available"
	}

	// Render pane content WITHOUT titles (titles will be separate)
	leftContent := leftStyle.Copy().
		Width(leftWidth).
		Height(availableHeight).
		Render(leftPaneContent)

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
	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	bottomComponents = append(bottomComponents, m.footer.View())

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)

	// If welcome overlay is visible, overlay it (highest priority)
	if m.showWelcomeOverlay {
		// Update overlay size
		m.welcomeOverlay.SetSize(m.width, m.height)
		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.welcomeOverlay.View(), mainView, true, true)
	}

	// If help dialog is visible, overlay it
	if m.showHelp {
		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.helpDialog.View(), mainView, true, true)
	}

	// If debug overlay is visible, overlay it (high priority)
	if m.showDebugOverlay && m.debugOverlay != nil {
		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.debugOverlay.View(), mainView, true, true)
	}

	// If worktree creation dialog is visible, overlay it
	if m.showWorktreeDialog && m.worktreeDialog != nil {
		// Update dialog size
		m.worktreeDialog.SetSize(m.width, m.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeDialog.View(), mainView, true, true)
	}

	// If repository dialog is visible, overlay it
	if m.showRepoDialog && m.repoDialog != nil {
		// Update dialog size
		m.repoDialog.SetSize(m.width, m.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.repoDialog.View(), mainView, true, true)
	}

	// If worktree deletion confirmation is visible, overlay it
	if m.showWorktreeConfirm && m.worktreeConfirm != nil {
		// Update dialog size
		m.worktreeConfirm.SetSize(m.width, m.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeConfirm.View(), mainView, true, true)
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
	rightContentWidth := availableContentWidth - int(float64(availableContentWidth) * 0.4)

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


