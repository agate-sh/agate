package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agate/config"
	"agate/git"
	"agate/overlay"
	"agate/panes"
	"agate/theme"
	"agate/tmux"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

//go:embed ascii-art.txt
var asciiArt string

var (
	paneBaseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderMuted)).
		Padding(1, 2)
)

type sessionMode int

const (
	modePreview sessionMode = iota // Read-only preview
)

type focusState int

const (
	focusReposAndWorktrees focusState = iota
	focusTmux
	focusGit
	focusShell
)

// String returns the string representation of the focus state
func (f focusState) String() string {
	switch f {
	case focusReposAndWorktrees:
		return "reposAndWorktrees"
	case focusTmux:
		return "tmux"
	case focusGit:
		return "git"
	case focusShell:
		return "shell"
	default:
		return "unknown"
	}
}

type model struct {
	layout              *Layout                    // Layout manager for pane dimensions
	tmuxSession         *tmux.TmuxSession
	ready               bool
	focused             focusState                 // Current focused pane
	err                 error
	subprocess          string                     // Command to run in tmux pane
	mode                sessionMode                // Current interaction mode
	agentConfig         AgentConfig                // Configuration for the specific agent
	keyMap              *GlobalKeyMap              // Global keybindings
	shortcutOverlay     *ShortcutOverlay           // Manages contextual shortcuts
	footer              *Footer                    // Footer component for shortcuts
	helpDialog          *HelpDialog                // Help dialog overlay
	showHelp            bool                       // Whether help dialog is visible
	worktreeManager     *git.WorktreeManager       // Git worktree management
	worktreeList        *WorktreeList              // Worktree list component
	worktreeDialog      *WorktreeDialog            // Worktree creation dialog
	worktreeConfirm     *WorktreeConfirmDialog     // Worktree deletion confirmation
	showWorktreeDialog  bool                       // Whether showing worktree creation dialog
	showWorktreeConfirm bool                       // Whether showing worktree deletion confirmation
	repoDialog          *RepoDialog                // Repository search dialog
	showRepoDialog      bool                       // Whether showing repository dialog
	welcomeOverlay      *WelcomeOverlay            // Welcome overlay for first-time users
	showWelcomeOverlay  bool                       // Whether showing welcome overlay
	debugLogger         *DebugLogger               // Debug logger for development
	debugOverlay        *DebugOverlay              // Debug overlay for development
	showDebugOverlay    bool                       // Whether showing debug overlay
	loadingState        *tmux.LoadingState         // Loading state manager with spinner and stopwatch

	// Panes using the new Pane interface
	repoPane  panes.Pane // Repos & worktrees pane (will be extracted from WorktreeList)
	tmuxPane  panes.Pane // Tmux terminal pane
	gitPane   panes.Pane // Git file status pane
	shellPane panes.Pane // Shell pane
}

func initialModel(subprocess string) model {
	// Get agent configuration based on subprocess name
	agentConfig := GetAgentConfig(subprocess)

	// Create keybindings and shortcut overlay
	keyMap := NewGlobalKeyMap()
	shortcutOverlay := NewShortcutOverlay(keyMap)
	shortcutOverlay.SetFocus(focusTmux.String()) // Start with tmux pane focused
	shortcutOverlay.SetMode("preview")           // Start in preview mode

	// Create footer and help components
	footer := NewFooter()
	footer.SetAgentConfig(agentConfig)
	footer.SetShortcutOverlay(shortcutOverlay)
	footer.SetFocus(focusTmux.String()) // Start with tmux pane focused
	footer.SetMode("preview")           // Start in preview mode

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

	// Initialize all panes using the new Pane interface
	gitPane := panes.NewGitPane()
	tmuxPane := panes.NewTmuxPane(panes.AgentConfig{
		CompanyName: agentConfig.CompanyName,
		BorderColor: agentConfig.BorderColor,
	}, tmux.NewLoadingState())
	shellPane := panes.NewShellPane()
	repoPane := panes.NewRepoWorktreePane(worktreeManager)

	m := model{
		layout:              NewLayout(0, 0), // Will be updated on first WindowSizeMsg
		focused:             focusTmux,       // Focus on tmux pane initially
		subprocess:          subprocess,
		mode:                modePreview, // Start in preview mode
		agentConfig:         agentConfig,
		keyMap:              keyMap,
		shortcutOverlay:     shortcutOverlay,
		footer:              footer,
		helpDialog:          NewHelpDialog(keyMap),
		showHelp:            false,
		worktreeManager:     worktreeManager,
		worktreeList:        worktreeList,
		showWorktreeDialog:  false,
		showWorktreeConfirm: false,
		showRepoDialog:      false,
		welcomeOverlay:      NewWelcomeOverlay(),
		showWelcomeOverlay:  showWelcomeOverlay,
		debugLogger:         debugLogger,
		debugOverlay:        debugOverlay,
		showDebugOverlay:    false,
		loadingState:        tmux.NewLoadingState(),

		// Initialize panes
		repoPane:  repoPane,
		tmuxPane:  tmuxPane,
		gitPane:   gitPane,
		shellPane: shellPane,
	}

	// Initialize Git pane content if repo pane has items
	if m.repoPane != nil {
		if repoPane, ok := m.repoPane.(*panes.RepoWorktreePane); ok && repoPane.HasItems() {
			m.updateGitPane()
		}
	}

	return m
}

// switchToPane handles switching focus to a specific pane with all necessary updates
func (m model) switchToPane(targetPane focusState) (model, tea.Cmd) {
	// Update all panes' active state
	if m.repoPane != nil {
		m.repoPane.SetActive(targetPane == focusReposAndWorktrees)
	}
	if m.tmuxPane != nil {
		m.tmuxPane.SetActive(targetPane == focusTmux)
	}
	if m.gitPane != nil {
		m.gitPane.SetActive(targetPane == focusGit)
	}
	if m.shellPane != nil {
		m.shellPane.SetActive(targetPane == focusShell)
	}

	// Set the new focus
	m.focused = targetPane

	// Update footer and shortcut overlay
	m.footer.SetFocus(m.focused.String())
	m.shortcutOverlay.SetFocus(m.focused.String())

	// Special handling for repos & worktrees pane
	if targetPane == focusReposAndWorktrees && m.repoPane != nil {
		// Refresh repo pane when focusing
		if repoPane, ok := m.repoPane.(*panes.RepoWorktreePane); ok {
			if err := repoPane.Refresh(); err != nil {
				DebugLog("Failed to refresh repo pane when switching panes: %v", err)
				// UI continues to work, but log the issue for debugging
			}
		}

		// Update GitPane with selected worktree/repo
		m.updateGitPane()
	}

	return m, nil
}

// updateGitPane updates the Git pane based on the currently selected worktree/repo
func (m *model) updateGitPane() {
	if m.gitPane == nil || m.repoPane == nil {
		DebugLog("updateGitPane: gitPane or repoPane is nil")
		return
	}

	// Cast to RepoWorktreePane to access GetSelectedWorktree method
	repoPane, ok := m.repoPane.(*panes.RepoWorktreePane)
	if !ok {
		DebugLog("updateGitPane: repoPane is not a RepoWorktreePane")
		return
	}

	// Get the selected worktree from the repo pane
	selectedWorktree := repoPane.GetSelectedWorktree()
	if selectedWorktree == nil {
		DebugLog("updateGitPane: no selected worktree")
		return
	}

	repoPath := selectedWorktree.Path
	DebugLog("updateGitPane: setting repository to: %s", repoPath)

	// Cast to GitPane to access SetRepository method
	if gitPane, ok := m.gitPane.(*panes.GitPane); ok {
		gitPane.SetRepository(repoPath)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		startTmuxSession(m.subprocess),
		tea.EnterAltScreen,
		m.loadingState.TickCmd(),
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

func combineCmds(cmds ...tea.Cmd) tea.Cmd {
	filtered := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			filtered = append(filtered, cmd)
		}
	}

	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return tea.Batch(filtered...)
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

type errMsg struct {
	error
}

type loadingTimeoutMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update layout with new dimensions
		m.layout.Update(msg.Width, msg.Height)
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
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := m.tmuxSession.SetDetachedSize(contentWidth, contentHeight); err != nil {
					DebugLog("Failed to resize tmux session to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - terminal will work with previous size
				}
			}
		}

		// Update repo pane size
		if m.repoPane != nil {
			leftWidth, leftHeight := m.layout.GetLeftDimensions()
			m.repoPane.SetSize(leftWidth, leftHeight)
		}

		// Update GitPane size
		if m.gitPane != nil {
			gitWidth, gitHeight := m.layout.GetGitDimensions()
			m.gitPane.SetSize(gitWidth, gitHeight)
		}

		// Update Git pane content after all components are sized
		// This ensures the worktree list has proper dimensions and selection
		m.updateGitPane()

	case tmuxSessionStartedMsg:
		m.tmuxSession = msg.session

		// Initialize loading state for tmux pane
		if m.tmuxPane != nil {
			if tmuxPane, ok := m.tmuxPane.(*panes.TmuxPane); ok {
				tmuxPane.SetLoading(true)
			}
		}

		// Start loading timer for stopwatch
		m.loadingState.Start()

		// Update Git pane now that the app is fully initialized
		// This ensures the worktree list has been sized and has a selection
		m.updateGitPane()

		// Set initial tmux session size using layout
		if m.ready {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := m.tmuxSession.SetDetachedSize(contentWidth, contentHeight); err != nil {
					DebugLog("Failed to set tmux session initial size to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - tmux will use default size
				}
			}
		}

		// Start monitoring tmux output and set up loading timeout
		return m, tea.Batch(
			waitForTmuxOutput(m.tmuxSession),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			}),
		)

	case tmuxOutputMsg:
		// Update tmux pane content
		if msg.content != "" {
			if m.tmuxPane != nil {
				if tmuxPane, ok := m.tmuxPane.(*panes.TmuxPane); ok {
					tmuxPane.SetContent(msg.content)
					tmuxPane.SetLoading(false)
				}
			}
			// Clear loading timer once we have content
			if m.loadingState.IsLoading() && strings.TrimSpace(msg.content) != "" {
				m.loadingState.Stop()
			}

			// On first real output, ensure Git pane is initialized
			m.updateGitPane()
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
		if m.tmuxSession != nil && m.focused == focusTmux {
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
		if m.tmuxSession != nil && m.focused == focusTmux {
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
		// Left content is now handled by WorktreeList directly
		// ASCII art will be displayed by WorktreeList

		// Update footer back to preview mode
		m.footer.SetMode("preview")
		m.shortcutOverlay.SetMode("preview")

		// Immediately resize the tmux session to current window dimensions
		if m.tmuxSession != nil && m.ready {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := m.tmuxSession.SetDetachedSize(contentWidth, contentHeight); err != nil {
					DebugLog("Failed to resize tmux session after detaching to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - terminal will work with current size
				}
			}
		}

		// Resume monitoring and trigger window size recalculation
		return m, tea.Batch(
			waitForTmuxOutput(m.tmuxSession),
			tea.WindowSize(), // Trigger complete UI layout recalculation
		)

	case errMsg:
		m.err = msg.error
		// Left content error will be displayed by WorktreeList directly
		// Error: msg.error can be handled by WorktreeList if needed

	// Worktree dialog messages
	case WorktreeCreatedMsg:
		var cmds []tea.Cmd
		if m.showWorktreeDialog && m.worktreeDialog != nil {
			var dialogCmd tea.Cmd
			var dialogModel tea.Model
			dialogModel, dialogCmd = m.worktreeDialog.Update(msg)
			m.worktreeDialog = dialogModel.(*WorktreeDialog)
			cmds = append(cmds, dialogCmd)
		}

		// Worktree created successfully - start tmux session but keep dialog open
		// Refresh the worktree list
		if m.worktreeList != nil {
			if err := m.worktreeList.Refresh(); err != nil {
				DebugLog("Failed to refresh worktree list after creating worktree: %v", err)
				// Continue showing success message, but log the refresh failure
			}
			// Update Git pane after refresh
			m.updateGitPane()
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
				// Switch focus to tmux pane
				m.focused = focusTmux
				// Update footer focus
				m.footer.SetFocus(focusTmux.String())
				m.shortcutOverlay.SetFocus(focusTmux.String())

				// Start monitoring the new session
				cmds = append(cmds, waitForTmuxOutput(newSession))
			}
		}
		return m, combineCmds(cmds...)

	case WorktreeInitializationCompleteMsg:
		// Initialization complete - close dialog and auto-attach
		m.showWorktreeDialog = false
		m.worktreeDialog = nil

		// Auto-attach to the tmux session
		if m.tmuxSession != nil && m.focused == focusTmux {
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
		if m.showWorktreeDialog && m.worktreeDialog != nil {
			model, cmd := m.worktreeDialog.Update(msg)
			m.worktreeDialog = model.(*WorktreeDialog)
			return m, cmd
		}
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
			if err := m.worktreeList.Refresh(); err != nil {
				DebugLog("Failed to refresh worktree list after deletion: %v", err)
				// UI will still show deletion success, but log refresh failure
			}
			// Update Git pane after deletion
			m.updateGitPane()
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
				if err := m.worktreeList.Refresh(); err != nil {
					DebugLog("Failed to refresh worktree list after adding repository: %v", err)
					// Repository was saved successfully, but UI refresh failed
				}
				// Update Git pane after adding repository
				m.updateGitPane()
			}
		}
		return m, nil

	case RepoDialogCancelledMsg:
		// Repository dialog cancelled
		m.showRepoDialog = false
		m.repoDialog = nil
		return m, nil

	case loadingTimeoutMsg:
		// After 3 seconds of loading, start periodic updates for stopwatch
		if m.loadingState.IsLoading() {
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			})
		}
		return m, nil

	case spinner.TickMsg:
		var cmds []tea.Cmd
		if cmd := m.loadingState.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

		if m.showWorktreeDialog && m.worktreeDialog != nil {
			var dialogCmd tea.Cmd
			var dialogModel tea.Model
			dialogModel, dialogCmd = m.worktreeDialog.Update(msg)
			m.worktreeDialog = dialogModel.(*WorktreeDialog)
			if dialogCmd != nil {
				cmds = append(cmds, dialogCmd)
			}
		}

		return m, combineCmds(cmds...)

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
		case key.Matches(msg, m.keyMap.AttachTmux) && m.focused != focusGit:
			// Enter key on left pane opens/switches to worktree or main repo session
			// Only handle if not focused on git pane (let OpenInEditor handle git pane)
			if m.focused == focusReposAndWorktrees && m.worktreeList != nil {
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
						// Switch focus to tmux pane for immediate interaction
						m.focused = focusTmux
						// Update footer focus
						m.footer.SetFocus(focusTmux.String())
						m.shortcutOverlay.SetFocus(focusTmux.String())
						// Start monitoring the new session and clear screen
						return m, tea.Batch(
							waitForTmuxOutput(newSession),
							tea.ClearScreen,
						)
					}
				}
			}
			// Enter key attaches to full tmux when tmux pane is focused
			if m.focused == focusTmux && m.tmuxSession != nil {
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
			// Fall through to other handlers if Enter wasn't handled above
			// This allows the git pane to handle Enter for opening files

		case key.Matches(msg, m.keyMap.Quit):
			// Quit available from both panes
			if m.tmuxSession != nil {
				if err := m.tmuxSession.Kill(); err != nil {
					DebugLog("Failed to kill tmux session on quit: %v", err)
					// Continue with quit regardless
				}
			}
			// Close debug panel and log file
			if m.debugLogger != nil {
				m.debugLogger.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Keybindings):
			// Show help dialog
			m.showHelp = true
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
			if m.focused == focusReposAndWorktrees && m.worktreeList != nil {
				selected := m.worktreeList.GetSelected()
				if selected != nil {
					m.worktreeConfirm = NewWorktreeConfirmDialog(selected, m.worktreeManager)
					m.showWorktreeConfirm = true
					return m, nil
				}
			}

		case key.Matches(msg, m.keyMap.Up):
			// Navigate up in focused pane
			switch m.focused {
			case focusReposAndWorktrees:
				if m.repoPane != nil {
					m.repoPane.MoveUp()
					m.updateGitPane() // Update Git pane when selection changes
				}
			case focusGit:
				if m.gitPane != nil {
					m.gitPane.MoveUp()
					return m, nil
				}
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Down):
			// Navigate down in focused pane
			switch m.focused {
			case focusReposAndWorktrees:
				if m.repoPane != nil {
					m.repoPane.MoveDown()
					m.updateGitPane() // Update Git pane when selection changes
				}
			case focusGit:
				if m.gitPane != nil {
					m.gitPane.MoveDown()
					return m, nil
				}
			}
			return m, nil

		// OpenInEditor is now handled by GitPane's HandleKey method directly
		// No separate global keybinding needed

		case key.Matches(msg, m.keyMap.FocusPaneRepos):
			// Switch to repos & worktrees pane (0)
			return m.switchToPane(focusReposAndWorktrees)

		case key.Matches(msg, m.keyMap.FocusPaneTmux):
			// Switch to tmux pane (1)
			return m.switchToPane(focusTmux)

		case key.Matches(msg, m.keyMap.FocusPaneGit):
			// Switch to git pane (2)
			return m.switchToPane(focusGit)

		case key.Matches(msg, m.keyMap.FocusPaneShell):
			// Switch to shell pane (3)
			return m.switchToPane(focusShell)

		default:
			// Handle other key combinations if needed
		}

	case tea.MouseMsg:
		// Handle mouse events in preview mode only when right pane is focused
		if m.mode == modePreview && m.focused == focusTmux && m.tmuxSession != nil {
			switch msg.Action {
			case tea.MouseActionPress:
				if msg.Button == tea.MouseButtonWheelUp {
					// Enter copy mode and scroll up
					if err := m.tmuxSession.SendScrollUp(); err != nil {
						DebugLog("Failed to send scroll up to tmux session: %v", err)
					}
				} else if msg.Button == tea.MouseButtonWheelDown {
					// Scroll down (or exit copy mode if at bottom)
					if err := m.tmuxSession.SendScrollDown(); err != nil {
						DebugLog("Failed to send scroll down to tmux session: %v", err)
					}
				}
			}
			// Trigger content refresh after scroll
			return m, waitForTmuxOutput(m.tmuxSession)
		}
	}

	return m, nil
}

// renderLeftPaneTitle renders the title for the left pane (repos & worktrees)
func (m model) renderLeftPaneTitle() string {
	if m.focused == focusReposAndWorktrees {
		focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary)).Bold(true)
		return focusStyle.Render("Repos & Worktrees") + " " +
			focusStyle.Render("[r: add repo, w: add worktree]")
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted))
	return titleStyle.Render("Repos & Worktrees") + numberStyle.Render(" [0]")
}

// renderPaneTitle renders a title using the pane's GetTitleStyle method
func (m model) renderPaneTitle(pane panes.Pane) string {
	if pane == nil {
		return ""
	}

	titleStyle := pane.GetTitleStyle()

	// Style the text based on the title type
	var styledText string
	if titleStyle.Type == "badge" {
		// Badge style (like tmux pane) with colored background
		badgeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(titleStyle.Color)).
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Padding(0, 1).
			Bold(true)
		styledText = badgeStyle.Render(titleStyle.Text)
	} else {
		// Plain style
		var textStyle lipgloss.Style
		if pane.IsActive() {
			textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary)).Bold(true)
		} else {
			textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
		}
		styledText = textStyle.Render(titleStyle.Text)
	}

	// Add shortcuts with appropriate styling
	if titleStyle.Shortcuts != "" {
		var shortcutStyle lipgloss.Style
		if titleStyle.Type == "badge" && titleStyle.Color != "" {
			// For badge titles, style shortcuts with the badge color
			shortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(titleStyle.Color)).
				Bold(true)
		} else if pane.IsActive() {
			// For active plain titles, use primary text color
			shortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextPrimary)).
				Bold(true)
		} else {
			// For inactive titles, use muted color
			shortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted))
		}
		return styledText + " " + shortcutStyle.Render(titleStyle.Shortcuts)
	}

	return styledText
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render titles using the new Pane interface
	leftTitle := m.renderLeftPaneTitle()
	tmuxTitle := m.renderPaneTitle(m.tmuxPane)
	gitTitle := m.renderPaneTitle(m.gitPane)
	shellTitle := m.renderPaneTitle(m.shellPane)

	// Render pane content using the new Pane interface
	leftPaneContent := m.repoPane.View()

	// Render all panes using the layout system
	leftPane, tmuxPane, gitPane, shellPane := m.layout.RenderPanes(
		leftPaneContent,
		m.tmuxPane.View(),
		m.gitPane.View(),
		m.shellPane.View(),
		m.focused,
		m.agentConfig.BorderColor, // Pass the agent's color
		false, // isLoading - handled by tmux pane internally now
		m.loadingState,
		m.agentConfig.CompanyName,
	)

	// Add padding to titles
	leftTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(leftTitle)
	tmuxTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(tmuxTitle)
	gitTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(gitTitle)
	shellTitleWithPadding := lipgloss.NewStyle().PaddingLeft(1).Render(shellTitle)

	// Add titles above panes
	leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitleWithPadding, leftPane)
	tmuxWithTitle := lipgloss.JoinVertical(lipgloss.Left, tmuxTitleWithPadding, tmuxPane)
	gitWithTitle := lipgloss.JoinVertical(lipgloss.Left, gitTitleWithPadding, gitPane)
	shellWithTitle := lipgloss.JoinVertical(lipgloss.Left, shellTitleWithPadding, shellPane)

	// Stack the right panes vertically
	rightPanes := lipgloss.JoinVertical(lipgloss.Top, gitWithTitle, shellWithTitle)

	gap := strings.Repeat(" ", horizontalGapWidth)
	// Join all panes horizontally with consistent gutters
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, tmuxWithTitle, gap, rightPanes)

	// Add top/bottom padding and outer horizontal margins
	panesWithPadding := lipgloss.NewStyle().
		PaddingTop(topPaddingRows).
		PaddingBottom(bottomSpacerRows).
		PaddingLeft(horizontalMargin).
		PaddingRight(horizontalMargin).
		Render(panes)

	// Add footer at the bottom
	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	bottomComponents = append(bottomComponents, m.footer.View())
	for i := 0; i < bottomMarginRows; i++ {
		bottomComponents = append(bottomComponents, "")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)

	// If welcome overlay is visible, overlay it (highest priority)
	if m.showWelcomeOverlay {
		// Update overlay size
		m.welcomeOverlay.SetSize(m.layout.width, m.layout.height)
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
		m.worktreeDialog.SetSize(m.layout.width, m.layout.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeDialog.View(), mainView, true, true)
	}

	// If repository dialog is visible, overlay it
	if m.showRepoDialog && m.repoDialog != nil {
		// Update dialog size
		m.repoDialog.SetSize(m.layout.width, m.layout.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.repoDialog.View(), mainView, true, true)
	}

	// If worktree deletion confirmation is visible, overlay it
	if m.showWorktreeConfirm && m.worktreeConfirm != nil {
		// Update dialog size
		m.worktreeConfirm.SetSize(m.layout.width, m.layout.height)

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeConfirm.View(), mainView, true, true)
	}

	return mainView
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
		RunE: func(_ *cobra.Command, args []string) error {
			return runAgent(args[0])
		},
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
