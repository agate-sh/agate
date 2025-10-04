package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/internal/version"
	"agate/pkg/app"
	"agate/pkg/common"
	"agate/pkg/config"
	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/layout"
	"agate/pkg/gui/overlays"
	"agate/pkg/gui/panes"
	"agate/pkg/gui/theme"
	"agate/pkg/overlay"
	"agate/pkg/session"
	"agate/pkg/tmux"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// ASCII art is embedded in the welcome overlay

// PaneBaseStyle is now defined in the layout package

type sessionMode int

const (
	modePreview sessionMode = iota // Read-only preview
)

// focusState is now defined in layout package

// Focus state constants are now defined in layout package

// String method is now in layout package

type model struct {
	layout              *layout.Layout   // Layout manager for pane dimensions
	sessionManager      *session.Manager // Session manager for all worktree/tmux coordination
	ready               bool
	focused             layout.FocusState // Current focused pane
	err                 error
	subprocess          string                               // Command to run in tmux pane
	mode                sessionMode                          // Current interaction mode
	shortcutOverlay     *common.ShortcutOverlay              // Manages contextual shortcuts
	footer              *common.Footer                       // Footer component for shortcuts
	helpDialog          *overlays.HelpDialog                 // Help dialog overlay
	showHelp            bool                                 // Whether help dialog is visible
	worktreeManager     *git.WorktreeManager                 // Git worktree management
	worktreeList        *overlays.WorktreeList               // Worktree list component
	worktreeDialog      *overlays.SessionDialog              // Worktree creation dialog
	worktreeConfirm     *overlays.WorktreeConfirmDialog      // Worktree deletion confirmation
	sessionConfirm      *overlays.SessionDeleteConfirmDialog // Session deletion confirmation
	showSessionDialog   bool                                 // Whether showing worktree creation dialog
	showWorktreeConfirm bool                                 // Whether showing worktree deletion confirmation
	showSessionConfirm  bool                                 // Whether showing session deletion confirmation
	repoDialog          *overlays.RepoDialog                 // Repository search dialog
	showRepoDialog      bool                                 // Whether showing repository dialog
	welcomeOverlay      *overlays.WelcomeOverlay             // Welcome overlay for first-time users
	showWelcomeOverlay  bool                                 // Whether showing welcome overlay
	debugLogger         *debug.DebugLogger                   // Debug logger for development
	debugOverlay        *overlays.DebugOverlay               // Debug overlay for development
	showDebugOverlay    bool                                 // Whether showing debug overlay
	loadingState        *tmux.LoadingState                   // Loading state manager with spinner and stopwatch

	// Panes using the new Pane interface
	repoPane  components.Pane // Repos & worktrees pane (will be extracted from WorktreeList)
	tmuxPane  components.Pane // Tmux terminal pane
	gitPane   components.Pane // Git file status pane
	shellPane components.Pane // Shell pane
}

func initialModel(subprocess string) model {
	// Initialize worktree manager first
	worktreeManager, err := git.NewWorktreeManager()
	if err != nil {
		// Log error but don't fail - app can still work without worktree features
		fmt.Printf("Warning: failed to initialize worktree manager: %v\n", err)
	}

	// Create session manager
	sessionManager := session.NewManager(worktreeManager)

	// Load existing sessions from persistence
	if err := sessionManager.RestoreSessions(); err != nil {
		debug.DebugLog("Failed to restore sessions on startup: %v", err)
		// Don't fail startup if session restoration fails
	}

	// No automatic main session creation - users must explicitly create agents

	// Get agent configuration based on subprocess name
	agentConfig := app.GetAgentConfig(subprocess)

	// Set the agent globally so all components can access it (for backwards compatibility)
	app.SetCurrentAgent(agentConfig)

	// Save as default agent for new sessions
	if subprocess != "" {
		if err := config.SetDefaultAgent(subprocess); err != nil {
			debug.DebugLog("Failed to save default agent: %v", err)
			// Continue without saving - not critical
		}
	}

	// Create shortcut overlay using static GlobalKeys
	shortcutOverlay := common.NewShortcutOverlay(common.GlobalKeys)
	shortcutOverlay.SetFocus(layout.FocusTmux.String()) // Start with tmux pane focused
	shortcutOverlay.SetMode("preview")                  // Start in preview mode

	// Create footer and help components
	footer := common.NewFooter()
	footer.SetShortcutOverlay(shortcutOverlay)
	footer.SetFocus(layout.FocusTmux.String()) // Start with tmux pane focused
	footer.SetMode("preview")                  // Start in preview mode

	// Initialize worktree components
	var worktreeList *overlays.WorktreeList
	if worktreeManager != nil {
		worktreeList = overlays.NewWorktreeList(worktreeManager)
	}

	// Check if welcome overlay should be shown
	welcomeShown, _ := config.GetWelcomeShownState()
	showWelcomeOverlay := !welcomeShown

	// Initialize debug logger
	debugLogger := debug.InitDebugLogger()

	// Test debug logging
	debug.DebugLog("Debug logger initialized successfully")

	// Initialize debug overlay
	debugOverlay := overlays.NewDebugOverlay(debugLogger)

	// Set up debug logging for git package (always enabled now)
	git.DebugLog = debug.DebugLog

	// Create shared loading state
	loadingState := tmux.NewLoadingState()

	// Initialize all panes using the new Pane interface
	gitPane := panes.NewGitPane()
	tmuxPane := panes.NewAgentTmuxPane(loadingState)
	shellPane := panes.NewShellTmuxPane()
	repoPane := panes.NewAgentsPane(sessionManager)

	m := model{
		layout:              layout.NewLayout(0, 0), // Will be updated on first WindowSizeMsg
		sessionManager:      sessionManager,         // Session manager for coordination
		focused:             layout.FocusTmux,       // Focus on tmux pane initially
		subprocess:          subprocess,
		mode:                modePreview, // Start in preview mode
		shortcutOverlay:     shortcutOverlay,
		footer:              footer,
		helpDialog:          overlays.NewHelpDialog(common.GlobalKeys),
		showHelp:            false,
		worktreeManager:     worktreeManager,
		worktreeList:        worktreeList,
		showSessionDialog:   false,
		showWorktreeConfirm: false,
		showSessionConfirm:  false,
		showRepoDialog:      false,
		welcomeOverlay:      overlays.NewWelcomeOverlay(),
		showWelcomeOverlay:  showWelcomeOverlay,
		debugLogger:         debugLogger,
		debugOverlay:        debugOverlay,
		showDebugOverlay:    false,
		loadingState:        loadingState,

		// Initialize panes
		repoPane:  repoPane,
		tmuxPane:  tmuxPane,
		gitPane:   gitPane,
		shellPane: shellPane,
	}

	// Initialize Git pane content if repo pane has items
	if m.repoPane != nil {
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok && repoPane.HasItems() {
			m.updateGitPane()
		}
	}

	return m
}

// switchToPane handles switching focus to a specific pane with all necessary updates
func (m model) switchToPane(targetPane layout.FocusState) (model, tea.Cmd) {
	// Update all panes' active state
	if m.repoPane != nil {
		m.repoPane.SetActive(targetPane == layout.FocusAgents)
	}
	if m.tmuxPane != nil {
		m.tmuxPane.SetActive(targetPane == layout.FocusTmux)
	}
	if m.gitPane != nil {
		m.gitPane.SetActive(targetPane == layout.FocusGit)
	}
	if m.shellPane != nil {
		m.shellPane.SetActive(targetPane == layout.FocusShell)
	}

	// Set the new focus
	m.focused = targetPane

	// Update footer and shortcut overlay
	m.footer.SetFocus(m.focused.String())
	m.shortcutOverlay.SetFocus(m.focused.String())

	// Special handling for repos & worktrees pane
	if targetPane == layout.FocusAgents && m.repoPane != nil {
		// Refresh repo pane when focusing
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			if err := repoPane.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh repo pane when switching panes: %v", err)
				// UI continues to work, but log the issue for debugging
			}
		}

		// Update GitPane with selected worktree/repo
		m.updateGitPane()
	}

	return m, nil
}

// getCurrentTmuxSession returns the active tmux session from the session manager
func (m *model) getCurrentTmuxSession() *tmux.TmuxSession {
	if m.sessionManager == nil {
		return nil
	}
	activeSession := m.sessionManager.GetActiveSession()
	if activeSession == nil {
		return nil
	}
	return activeSession.TmuxSession
}

// getCurrentShellTmuxSession returns the active shell tmux session from the session manager
func (m *model) getCurrentShellTmuxSession() *tmux.TmuxSession {
	if m.sessionManager == nil {
		return nil
	}
	activeSession := m.sessionManager.GetActiveSession()
	if activeSession == nil {
		return nil
	}
	return activeSession.ShellTmuxSession
}

// switchToSessionForWorktree switches to the session associated with the given worktree
func (m *model) switchToSessionForWorktree(worktree *git.WorktreeInfo) {
	if m.sessionManager == nil || worktree == nil {
		return
	}

	// Get or create session for this worktree
	sess, err := m.sessionManager.GetOrCreateSession(worktree, m.subprocess)
	if err != nil {
		debug.DebugLog("Failed to get/create session for worktree %s: %v", worktree.Path, err)
		return
	}

	// Switch to this session
	m.sessionManager.SwitchToSession(sess.WorktreeKey)

	// Update global agent state
	app.SetCurrentAgent(sess.Agent)

	// Update tmux pane with the session
	if m.tmuxPane != nil {
		if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
			tmuxPane.SetSession(sess.TmuxSession)
		}
	}

	// Update shell pane with the session
	if m.shellPane != nil {
		if shellPane, ok := m.shellPane.(*panes.ShellTmuxPane); ok {
			shellPane.SetSession(sess.ShellTmuxSession)
		}
	}

	debug.DebugLog("Switched to session %s with agent %s", sess.ID, sess.Agent.Name)
}

// updateGitPane updates the Git pane based on the currently selected worktree/repo
func (m *model) updateGitPane() {
	if m.gitPane == nil || m.repoPane == nil {
		debug.DebugLog("updateGitPane: gitPane or repoPane is nil")
		return
	}

	// Cast to AgentsPane to access GetSelectedWorktree method
	repoPane, ok := m.repoPane.(*panes.AgentsPane)
	if !ok {
		debug.DebugLog("updateGitPane: repoPane is not a AgentsPane")
		return
	}

	// Get the selected worktree from the repo pane
	selectedWorktree := repoPane.GetSelectedWorktree()
	if selectedWorktree == nil {
		debug.DebugLog("updateGitPane: no selected worktree")
		return
	}

	repoPath := selectedWorktree.Path
	debug.DebugLog("updateGitPane: setting repository to: %s", repoPath)

	// Switch to session for this worktree (this updates the agent and tmux session)
	m.switchToSessionForWorktree(selectedWorktree)

	// Cast to GitPane to access SetRepository method
	if gitPane, ok := m.gitPane.(*panes.GitPane); ok {
		gitPane.SetRepository(repoPath)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		startInitialMainSession(m.sessionManager, m.subprocess),
		tea.EnterAltScreen,
		m.loadingState.TickCmd(),
	)
}

func startInitialMainSession(sessionMgr *session.Manager, agentName string) tea.Cmd {
	return func() tea.Msg {
		// Only create main session if we're in a git repository
		if sessionMgr == nil || sessionMgr.GetWorktreeManager() == nil {
			return nil
		}

		worktreeManager := sessionMgr.GetWorktreeManager()
		if !worktreeManager.IsGitRepo() {
			return nil
		}

		// Get the main worktree info
		mainWorktree, err := worktreeManager.GetMainWorktreeInfo()
		if err != nil {
			debug.DebugLog("Failed to get main worktree info: %v", err)
			return nil
		}

		// Check if main session already exists
		existingSession := sessionMgr.GetMainSession(mainWorktree.RepoName)
		if existingSession != nil {
			debug.DebugLog("Main session already exists for repo: %s", mainWorktree.RepoName)
			// Switch to existing main session
			sessionMgr.SwitchToSession(existingSession.WorktreeKey)
			return tmuxSessionStartedMsg{session: existingSession}
		}

		// Create new main session
		sess, err := sessionMgr.GetOrCreateSession(mainWorktree, agentName)
		if err != nil {
			debug.DebugLog("Failed to create main session: %v", err)
			return errMsg{err}
		}

		// Set as active session
		sessionMgr.SwitchToSession(sess.WorktreeKey)

		debug.DebugLog("Created main session for repo: %s", mainWorktree.RepoName)
		return tmuxSessionStartedMsg{session: sess}
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
	session *session.Session
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

		// Set the active pane based on current focus
		if m.repoPane != nil {
			m.repoPane.SetActive(m.focused == layout.FocusAgents)
		}
		if m.tmuxPane != nil {
			m.tmuxPane.SetActive(m.focused == layout.FocusTmux)
		}
		if m.gitPane != nil {
			m.gitPane.SetActive(m.focused == layout.FocusGit)
		}
		if m.shellPane != nil {
			m.shellPane.SetActive(m.focused == layout.FocusShell)
		}

		// Update component sizes
		m.footer.SetSize(msg.Width, 1)
		m.helpDialog.SetSize(msg.Width, msg.Height)

		// Update debug overlay size
		if m.debugOverlay != nil {
			m.debugOverlay.SetSize(msg.Width, msg.Height)
		}

		// Update tmux session size if it exists
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := currentTmux.SetDetachedSize(contentWidth, contentHeight); err != nil {
					debug.DebugLog("Failed to resize tmux session to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - terminal will work with previous size
				}
			}
		}

		// Update repo pane size
		if m.repoPane != nil {
			leftWidth, leftHeight := m.layout.GetLeftDimensions()
			m.repoPane.SetSize(leftWidth, leftHeight)
		}

		// Update TmuxPane size
		if m.tmuxPane != nil {
			tmuxWidth, tmuxHeight := m.layout.GetTmuxDimensions()
			m.tmuxPane.SetSize(tmuxWidth, tmuxHeight)
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
		// Store the session (msg.session is now a *session.Session)
		activeSession := msg.session

		// Set the current agent based on the session's agent
		app.SetCurrentAgent(activeSession.Agent)

		// Initialize loading state for tmux pane
		if m.tmuxPane != nil {
			if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
				tmuxPane.SetLoading(true)
				tmuxPane.SetSession(activeSession.TmuxSession)
			}
		}

		// Initialize shell pane with session
		if m.shellPane != nil {
			if shellPane, ok := m.shellPane.(*panes.ShellTmuxPane); ok {
				shellPane.SetSession(activeSession.ShellTmuxSession)
			}
		}

		// Start loading timer for stopwatch
		m.loadingState.Start()

		// Update Git pane now that the app is fully initialized
		// This ensures the worktree list has been sized and has a selection
		m.updateGitPane()

		// Set initial tmux session size using layout
		if m.ready && activeSession.TmuxSession != nil {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := activeSession.TmuxSession.SetDetachedSize(contentWidth, contentHeight); err != nil {
					debug.DebugLog("Failed to set tmux session initial size to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - tmux will use default size
				}
			}
		}

		// Start monitoring tmux output and set up loading timeout
		return m, tea.Batch(
			waitForTmuxOutput(activeSession.TmuxSession),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			}),
		)

	case tmuxOutputMsg:
		// Update tmux pane content
		if msg.content != "" {
			if m.tmuxPane != nil {
				if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
					tmuxPane.SetContent(msg.content)
					// Stop loading when we have meaningful content (not just whitespace)
					if strings.TrimSpace(msg.content) != "" {
						tmuxPane.SetLoading(false)
					}
				}
			}
			// Clear loading timer when we have meaningful content
			if m.loadingState.IsLoading() && strings.TrimSpace(msg.content) != "" {
				m.loadingState.Stop()
			}

			// On first real output, ensure Git pane is initialized
			m.updateGitPane()
		}

		// Continue monitoring (increased frequency for better responsiveness)
		return m, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
			if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
				return waitForTmuxOutput(currentTmux)()
			}
			return nil
		})

	case autoAttachMsg:
		// Auto-attach to the tmux session after it's ready
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.focused == layout.FocusTmux {
			// Block directly in Update like Claude Squad
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, nil

	case initializationCompleteMsg:
		// Close the worktree dialog and auto-attach
		m.showSessionDialog = false
		m.worktreeDialog = nil

		// Auto-attach to the tmux session
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.focused == layout.FocusTmux {
			// Clear screen first
			fmt.Print("\033[2J\033[H")
			// Block directly in Update like Claude Squad
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, tea.ClearScreen

	case tmuxDetachedMsg:
		// Left content is now handled by WorktreeList directly
		// ASCII art will be displayed by WorktreeList

		// Update footer back to preview mode
		m.footer.SetMode("preview")
		m.shortcutOverlay.SetMode("preview")

		// Immediately resize the tmux session to current window dimensions
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.ready {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				if err := currentTmux.SetDetachedSize(contentWidth, contentHeight); err != nil {
					debug.DebugLog("Failed to resize tmux session after detaching to %dx%d: %v", contentWidth, contentHeight, err)
					// Continue - terminal will work with current size
				}
			}
		}

		// Resume monitoring and trigger window size recalculation
		var monitorCmd tea.Cmd
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
			monitorCmd = waitForTmuxOutput(currentTmux)
		}
		return m, tea.Batch(
			monitorCmd,
			tea.WindowSize(), // Trigger complete UI layout recalculation
		)

	case errMsg:
		m.err = msg.error
		// Left content error will be displayed by WorktreeList directly
		// Error: msg.error can be handled by WorktreeList if needed

	// Worktree dialog messages
	case overlays.WorktreeCreatedMsg:
		var cmds []tea.Cmd
		if m.showSessionDialog && m.worktreeDialog != nil {
			var dialogCmd tea.Cmd
			var dialogModel tea.Model
			dialogModel, dialogCmd = m.worktreeDialog.Update(msg)
			m.worktreeDialog = dialogModel.(*overlays.SessionDialog)
			cmds = append(cmds, dialogCmd)
		}

		// Worktree created successfully - start tmux session but keep dialog open
		// Refresh the worktree list
		if m.worktreeList != nil {
			if err := m.worktreeList.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh worktree list after creating worktree: %v", err)
				// Continue showing success message, but log the refresh failure
			}
			// Update Git pane after refresh
			m.updateGitPane()
		}

		// Create and switch to new session for the worktree
		if msg.Worktree != nil {
			// Use the agent name from the message (selected by user in dialog)
			agentName := msg.AgentName
			if agentName == "" {
				agentName = m.subprocess // Fallback to subprocess if not provided
			}
			// Create or get session for this worktree using session manager
			newSession, err := m.sessionManager.GetOrCreateSession(msg.Worktree, agentName)
			if err == nil {
				// Switch to the new session
				m.sessionManager.SwitchToSession(newSession.WorktreeKey)

				// Update agent based on new session
				app.SetCurrentAgent(newSession.Agent)

				// Update tmux pane with new session
				if m.tmuxPane != nil {
					if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
						tmuxPane.SetSession(newSession.TmuxSession)
					}
				}

				// Update shell pane with new session
				if m.shellPane != nil {
					if shellPane, ok := m.shellPane.(*panes.ShellTmuxPane); ok {
						shellPane.SetSession(newSession.ShellTmuxSession)
					}
				}

				// Switch focus to tmux pane
				m.focused = layout.FocusTmux
				// Update footer focus
				m.footer.SetFocus(layout.FocusTmux.String())
				m.shortcutOverlay.SetFocus(layout.FocusTmux.String())

				// Start monitoring the new session
				if newSession.TmuxSession != nil {
					cmds = append(cmds, waitForTmuxOutput(newSession.TmuxSession))
				}
			} else {
				debug.DebugLog("Failed to create session for worktree: %v", err)
			}
		}
		return m, combineCmds(cmds...)

	case overlays.WorktreeInitializationCompleteMsg:
		// Initialization complete - close dialog and auto-attach
		m.showSessionDialog = false
		m.worktreeDialog = nil

		// Refresh the agents pane to show the new session
		if agentsPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			agentsPane.Refresh()
		}

		// Auto-attach to the tmux session
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.focused == layout.FocusTmux {
			// Clear screen first
			fmt.Print("\033[2J\033[H")
			// Block directly in Update like Claude Squad
			detachCh, err := currentTmux.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, tea.ClearScreen

	case overlays.WorktreeCreationErrorMsg:
		if m.showSessionDialog && m.worktreeDialog != nil {
			model, cmd := m.worktreeDialog.Update(msg)
			m.worktreeDialog = model.(*overlays.SessionDialog)
			return m, cmd
		}
		return m, nil

	case panes.DeleteSessionRequestMsg:
		// User wants to delete a session - show confirmation dialog
		if msg.Session != nil {
			m.showSessionConfirm = true
			m.sessionConfirm = overlays.NewSessionDeleteConfirmDialog(msg.Session, m.sessionManager)
			if m.sessionConfirm != nil {
				m.sessionConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
			}
		}
		return m, nil

	case panes.AttachToSessionMsg:
		// User wants to attach to a tmux session from the agents pane
		if msg.Session != nil && msg.Session.TmuxSession != nil {
			// Switch to this session first
			if m.sessionManager != nil {
				m.sessionManager.SwitchToSession(msg.Session.WorktreeKey)

				// Update agent based on session
				app.SetCurrentAgent(msg.Session.Agent)

				// Update tmux pane with the session
				if m.tmuxPane != nil {
					if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
						tmuxPane.SetSession(msg.Session.TmuxSession)
					}
				}

				// Update shell pane with the session
				if m.shellPane != nil {
					if shellPane, ok := m.shellPane.(*panes.ShellTmuxPane); ok {
						shellPane.SetSession(msg.Session.ShellTmuxSession)
					}
				}
			}

			// Switch focus to tmux pane
			m.focused = layout.FocusTmux
			m.footer.SetFocus(layout.FocusTmux.String())
			m.shortcutOverlay.SetFocus(layout.FocusTmux.String())

			// Update UI to show attached mode
			m.footer.SetMode("attached")
			m.shortcutOverlay.SetMode("attached")

			// Attach to the tmux session
			detachCh, err := msg.Session.TmuxSession.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			// Block until detachment like Claude Squad does
			<-detachCh
			// Process detached message immediately
			return m.Update(tmuxDetachedMsg{})
		}
		return m, nil

	case overlays.SessionDialogCancelledMsg:
		// Dialog cancelled
		m.showSessionDialog = false
		m.worktreeDialog = nil
		return m, nil

	case overlays.WorktreeDeletedMsg:
		// Worktree deleted successfully
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		if m.worktreeList != nil {
			if err := m.worktreeList.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh worktree list after deletion: %v", err)
				// UI will still show deletion success, but log refresh failure
			}
			// Update Git pane after deletion
			m.updateGitPane()
		}
		return m, nil

	case overlays.WorktreeDeletionErrorMsg:
		// Worktree deletion failed
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		m.err = fmt.Errorf("failed to delete worktree: %s", msg.Error)
		return m, nil

	case overlays.WorktreeDeleteCancelledMsg:
		// Deletion cancelled
		m.showWorktreeConfirm = false
		m.worktreeConfirm = nil
		return m, nil

	case overlays.SessionDeletedMsg:
		// Session deleted successfully
		m.showSessionConfirm = false
		m.sessionConfirm = nil

		// Remove session from session manager
		if m.sessionManager != nil && msg.Session != nil {
			err := m.sessionManager.DeleteSession(msg.Session.WorktreeKey)
			if err != nil {
				debug.DebugLog("Failed to delete session: %v", err)
				m.err = fmt.Errorf("failed to delete session: %v", err)
			} else {
				// Refresh worktree list
				if m.repoPane != nil {
					if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
						if err := repoPane.Refresh(); err != nil {
							debug.DebugLog("Failed to refresh repo pane after session deletion: %v", err)
						}
					}
				}
				// Update Git pane
				m.updateGitPane()
			}
		}
		return m, nil

	case overlays.SessionDeletionErrorMsg:
		// Session deletion failed
		m.showSessionConfirm = false
		m.sessionConfirm = nil
		m.err = fmt.Errorf("failed to delete session: %s", msg.Error)
		return m, nil

	case overlays.SessionDeleteCancelledMsg:
		// Session deletion cancelled
		m.showSessionConfirm = false
		m.sessionConfirm = nil
		return m, nil

	case overlays.DebugOverlayClosedMsg:
		// Debug overlay closed
		m.showDebugOverlay = false
		return m, nil

	// Repository dialog messages
	case overlays.RepoAddedMsg:
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
					debug.DebugLog("Failed to refresh worktree list after adding repository: %v", err)
					// Repository was saved successfully, but UI refresh failed
				}
				// Update Git pane after adding repository
				m.updateGitPane()
			}
		}
		return m, nil

	case overlays.RepoDialogCancelledMsg:
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

		// Update TmuxPane's spinner (which now shares the same LoadingState)
		if m.tmuxPane != nil {
			if _, cmd := m.tmuxPane.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		if m.showSessionDialog && m.worktreeDialog != nil {
			var dialogCmd tea.Cmd
			var dialogModel tea.Model
			dialogModel, dialogCmd = m.worktreeDialog.Update(msg)
			m.worktreeDialog = dialogModel.(*overlays.SessionDialog)
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
		if m.showSessionDialog && m.worktreeDialog != nil {
			var cmd tea.Cmd
			model, cmd := m.worktreeDialog.Update(msg)
			m.worktreeDialog = model.(*overlays.SessionDialog)
			return m, cmd
		}

		// Handle worktree confirm dialog input
		if m.showWorktreeConfirm && m.worktreeConfirm != nil {
			var cmd tea.Cmd
			model, cmd := m.worktreeConfirm.Update(msg)
			m.worktreeConfirm = model.(*overlays.WorktreeConfirmDialog)
			return m, cmd
		}

		// Handle session confirm dialog input
		if m.showSessionConfirm && m.sessionConfirm != nil {
			var cmd tea.Cmd
			model, cmd := m.sessionConfirm.Update(msg)
			m.sessionConfirm = model.(*overlays.SessionDeleteConfirmDialog)
			return m, cmd
		}

		// Handle repo dialog input
		if m.showRepoDialog && m.repoDialog != nil {
			var cmd tea.Cmd
			model, cmd := m.repoDialog.Update(msg)
			m.repoDialog = model.(*overlays.RepoDialog)
			return m, cmd
		}

		// Handle preview mode - navigation and mode switches only
		switch {
		case msg.String() == "enter":
			// Enter key handling - delegate to the active pane
			switch m.focused {
			case layout.FocusAgents:
				// Let the repo pane handle enter key for toggling expansion
				if m.repoPane != nil {
					handled, cmd := m.repoPane.HandleKey("enter")
					if handled {
						m.updateGitPane() // Update Git pane when selection/expansion changes
						return m, cmd
					}
				}
			case layout.FocusGit:
				// Let the git pane handle enter key for opening files
				if m.gitPane != nil {
					handled, cmd := m.gitPane.HandleKey("enter")
					if handled {
						return m, cmd
					}
				}
			case layout.FocusTmux:
				// Enter key attaches to agent tmux session when tmux pane is focused
				if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
					// Update UI to show attached mode
					m.footer.SetMode("attached")
					m.shortcutOverlay.SetMode("attached")
					// Block directly in Update like Claude Squad - don't return to event loop during attachment
					detachCh, err := currentTmux.Attach()
					if err != nil {
						return m, func() tea.Msg { return errMsg{err} }
					}
					// Block until detachment like Claude Squad does
					<-detachCh
					// Process detached message immediately
					return m.Update(tmuxDetachedMsg{})
				}
			case layout.FocusShell:
				// Enter key attaches to shell tmux session when shell pane is focused
				if currentShellTmux := m.getCurrentShellTmuxSession(); currentShellTmux != nil {
					// Update UI to show attached mode
					m.footer.SetMode("attached")
					m.shortcutOverlay.SetMode("attached")
					// Block directly in Update like Claude Squad - don't return to event loop during attachment
					detachCh, err := currentShellTmux.Attach()
					if err != nil {
						return m, func() tea.Msg { return errMsg{err} }
					}
					// Block until detachment like Claude Squad does
					<-detachCh
					// Process detached message immediately
					return m.Update(tmuxDetachedMsg{})
				}
			}
			// Enter key now handles attachment for tmux and shell panes

		case key.Matches(msg, common.GlobalKeys.Quit):
			// Quit available from both panes - clean up all sessions
			if m.sessionManager != nil {
				for _, session := range m.sessionManager.ListSessions() {
					if session.TmuxSession != nil {
						if err := session.TmuxSession.Kill(); err != nil {
							debug.DebugLog("Failed to kill tmux session %s on quit: %v", session.ID, err)
							// Continue with quit regardless
						}
					}
					if session.ShellTmuxSession != nil {
						if err := session.ShellTmuxSession.Kill(); err != nil {
							debug.DebugLog("Failed to kill shell tmux session %s on quit: %v", session.ID, err)
							// Continue with quit regardless
						}
					}
				}
			}
			// Close debug panel and log file
			if m.debugLogger != nil {
				m.debugLogger.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, common.GlobalKeys.Keybindings):
			// Show help dialog
			m.showHelp = true
			return m, nil

		case key.Matches(msg, common.GlobalKeys.AddRepo):
			// Add new repository using fzf search
			debug.DebugLog("Creating new repo dialog...")
			m.repoDialog = overlays.NewRepoDialog()
			m.showRepoDialog = true
			// Initialize the repo dialog to start the repository discovery
			initCmd := m.repoDialog.Init()
			return m, initCmd

		case key.Matches(msg, common.GlobalKeys.DebugOverlay):
			// Show debug overlay
			m.showDebugOverlay = true
			return m, nil

		case key.Matches(msg, common.GlobalKeys.NewWorktree):
			// Create new worktree (available from both panes)
			if m.worktreeManager != nil {
				// Get default agent from config or use current subprocess
				defaultAgent, _ := config.GetDefaultAgent()
				if defaultAgent == "" {
					// Fallback to current subprocess if no default set
					defaultAgent = m.subprocess
				}
				m.worktreeDialog = overlays.NewSessionDialog(m.worktreeManager, defaultAgent)
				m.showSessionDialog = true
				return m, nil
			}

		case key.Matches(msg, common.GlobalKeys.DeleteWorktree):
			// Delete worktree (when left pane focused)
			if m.focused == layout.FocusAgents && m.worktreeList != nil {
				selected := m.worktreeList.GetSelected()
				if selected != nil {
					m.worktreeConfirm = overlays.NewWorktreeConfirmDialog(selected, m.worktreeManager)
					m.showWorktreeConfirm = true
					return m, nil
				}
			}

		case key.Matches(msg, common.GlobalKeys.DeleteSession):
			// Delete entire session (when repos pane focused and session active)
			if m.focused == layout.FocusAgents && m.sessionManager != nil {
				if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
					selected := repoPane.GetSelectedWorktree()
					if selected != nil {
						// Check if this is the main worktree (can't be deleted)
						isMainWorktree := false
						if m.worktreeManager != nil {
							if mainWorktree, err := m.worktreeManager.GetMainWorktreeInfo(); err == nil {
								isMainWorktree = selected.Path == mainWorktree.Path
							}
						}

						if isMainWorktree {
							// Don't allow deletion of main worktree
							return m, nil
						}

						// Get the session for this worktree (may be nil)
						sess := m.sessionManager.GetSessionForWorktree(selected)

						// Show delete confirmation dialog (can handle both session+worktree or just worktree)
						m.sessionConfirm = overlays.NewSessionDeleteConfirmDialog(sess, m.sessionManager)
						if sess == nil {
							// Pass the worktree info for worktree-only deletion
							m.sessionConfirm.SetWorktreeInfo(selected, m.worktreeManager)
						}
						m.showSessionConfirm = true
						return m, nil
					}
				}
			}

		case key.Matches(msg, common.GlobalKeys.Up):
			// Navigate up in focused pane
			switch m.focused {
			case layout.FocusAgents:
				if m.repoPane != nil {
					m.repoPane.MoveUp()
				}
			case layout.FocusGit:
				if m.gitPane != nil {
					m.gitPane.MoveUp()
					return m, nil
				}
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Down):
			// Navigate down in focused pane
			switch m.focused {
			case layout.FocusAgents:
				if m.repoPane != nil {
					m.repoPane.MoveDown()
				}
			case layout.FocusGit:
				if m.gitPane != nil {
					m.gitPane.MoveDown()
					return m, nil
				}
			}
			return m, nil

		// OpenInEditor is now handled by GitPane's HandleKey method directly
		// No separate global keybinding needed

		case key.Matches(msg, common.GlobalKeys.FocusPaneRepos):
			// Switch to repos & worktrees pane (0)
			return m.switchToPane(layout.FocusAgents)

		case key.Matches(msg, common.GlobalKeys.FocusPaneTmux):
			// Switch to tmux pane (1)
			return m.switchToPane(layout.FocusTmux)

		case key.Matches(msg, common.GlobalKeys.FocusPaneGit):
			// Switch to git pane (2)
			return m.switchToPane(layout.FocusGit)

		case key.Matches(msg, common.GlobalKeys.FocusPaneShell):
			// Switch to shell pane (3)
			return m.switchToPane(layout.FocusShell)

		case key.Matches(msg, common.GlobalKeys.AttachTmux):
			// Attach to agent tmux session (global shortcut 'a')
			if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
				// Update UI to show attached mode
				m.footer.SetMode("attached")
				m.shortcutOverlay.SetMode("attached")
				// Block directly in Update like Claude Squad - don't return to event loop during attachment
				detachCh, err := currentTmux.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				// Block until detachment like Claude Squad does
				<-detachCh
				// Process detached message immediately
				return m.Update(tmuxDetachedMsg{})
			}

		case key.Matches(msg, common.GlobalKeys.AttachShell):
			// Attach to shell tmux session (global shortcut 's')
			if currentShellTmux := m.getCurrentShellTmuxSession(); currentShellTmux != nil {
				// Update UI to show attached mode
				m.footer.SetMode("attached")
				m.shortcutOverlay.SetMode("attached")
				// Block directly in Update like Claude Squad - don't return to event loop during attachment
				detachCh, err := currentShellTmux.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				// Block until detachment like Claude Squad does
				<-detachCh
				// Process detached message immediately
				return m.Update(tmuxDetachedMsg{})
			}

		default:
			// Handle other key combinations if needed
		}

	case tea.MouseMsg:
		// Handle mouse events when right pane is focused
		if currentTmux := m.getCurrentTmuxSession(); m.focused == layout.FocusTmux && currentTmux != nil {
			switch msg.Action {
			case tea.MouseActionPress:
				if msg.Button == tea.MouseButtonWheelUp {
					// Enter copy mode and scroll up
					if err := currentTmux.SendScrollUp(); err != nil {
						debug.DebugLog("Failed to send scroll up to tmux session: %v", err)
					}
				} else if msg.Button == tea.MouseButtonWheelDown {
					// Scroll down (or exit copy mode if at bottom)
					if err := currentTmux.SendScrollDown(); err != nil {
						debug.DebugLog("Failed to send scroll down to tmux session: %v", err)
					}
				}
			}
			// Trigger content refresh after scroll
			return m, waitForTmuxOutput(currentTmux)
		}
	}

	return m, nil
}

// renderPaneTitle renders a title using the pane's GetTitleStyle method
func (m model) renderPaneTitle(pane components.Pane) string {
	if pane == nil {
		return ""
	}

	titleStyle := pane.GetTitleStyle()

	// Style the text based on the title type
	var styledText string
	if titleStyle.Type == "badge" {
		// Badge style (like tmux pane) with colored background
		var backgroundColor string
		if pane.IsActive() {
			// When active, use the agent's brand color
			backgroundColor = titleStyle.Color
		} else {
			// When inactive, use very muted color to blend into background
			backgroundColor = theme.SeparatorColor
		}

		badgeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(backgroundColor)).
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
		if pane.IsActive() {
			// When active, put formatted shortcuts in parentheses
			formattedShortcuts := m.parseAndStyleShortcuts(titleStyle.Shortcuts)

			// Style the parentheses consistently
			parenStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted))

			leftParen := parenStyle.Render("(")
			rightParen := parenStyle.Render(")")
			return styledText + " " + leftParen + formattedShortcuts + rightParen
		} else {
			// When inactive, show pane number in parentheses
			shortcutStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.TextMuted))
			return styledText + " " + shortcutStyle.Render(titleStyle.Shortcuts)
		}
	}

	return styledText
}

// parseAndStyleShortcuts parses shortcut strings and applies footer-like styling
func (m model) parseAndStyleShortcuts(shortcuts string) string {
	// Split shortcuts by bullet separator
	parts := strings.Split(shortcuts, " • ")
	var styledParts []string

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextPrimary)).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.TextDescription))
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SeparatorColor))

	for i, part := range parts {
		if i > 0 {
			styledParts = append(styledParts, separatorStyle.Render(" • "))
		}

		// Split each part into key and description (by first space)
		tokens := strings.SplitN(strings.TrimSpace(part), " ", 2)
		if len(tokens) >= 2 {
			key := tokens[0]
			desc := tokens[1]
			styledPart := keyStyle.Render(key) + " " + descStyle.Render(desc)
			styledParts = append(styledParts, styledPart)
		} else {
			// If we can't parse properly, just style the whole thing as a key
			styledParts = append(styledParts, keyStyle.Render(part))
		}
	}

	return strings.Join(styledParts, "")
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render titles using the new Pane interface
	leftTitle := m.renderPaneTitle(m.repoPane)
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
		false, // isLoading - handled by tmux pane internally now
		m.loadingState,
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

	gap := strings.Repeat(" ", layout.HorizontalGapWidth)
	// Join all panes horizontally with consistent gutters
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, tmuxWithTitle, gap, rightPanes)

	// Add top/bottom padding and outer horizontal margins
	panesWithPadding := lipgloss.NewStyle().
		PaddingTop(layout.TopPaddingRows).
		PaddingBottom(layout.BottomSpacerRows).
		PaddingLeft(layout.HorizontalMargin).
		PaddingRight(layout.HorizontalMargin).
		Render(panes)

	// Add footer at the bottom
	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	bottomComponents = append(bottomComponents, m.footer.View())
	for i := 0; i < layout.BottomMarginRows; i++ {
		bottomComponents = append(bottomComponents, "")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)

	// If welcome overlay is visible, overlay it (highest priority)
	if m.showWelcomeOverlay {
		// Update overlay size
		m.welcomeOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
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
	if m.showSessionDialog && m.worktreeDialog != nil {
		// Update dialog size
		m.worktreeDialog.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeDialog.View(), mainView, true, true)
	}

	// If repository dialog is visible, overlay it
	if m.showRepoDialog && m.repoDialog != nil {
		// Update dialog size
		m.repoDialog.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.repoDialog.View(), mainView, true, true)
	}

	// If worktree deletion confirmation is visible, overlay it
	if m.showWorktreeConfirm && m.worktreeConfirm != nil {
		// Update dialog size
		m.worktreeConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.worktreeConfirm.View(), mainView, true, true)
	}

	// If session deletion confirmation is visible, overlay it
	if m.showSessionConfirm && m.sessionConfirm != nil {
		// Update dialog size
		m.sessionConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.sessionConfirm.View(), mainView, true, true)
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
	var showVersion bool

	var rootCmd = &cobra.Command{
		Use:   "agate <agent>",
		Short: "A tmux-based terminal UI for AI agents",
		Long: `Agate provides a split-pane terminal interface for interacting with AI agents.

Supports any agent name (claude, amp, cn, etc.) and automatically configures
colors and settings based on the agent type.

Agate provides two interaction modes:
  Preview Mode (default): Read-only view with fast, lag-free rendering
  Attached Mode (a): Full tmux experience with complete control

Press 'a' when focused on the right pane to attach to tmux.
Press Ctrl+Q when attached to detach back to preview.
Press ? for help once running.

Examples:
  agate claude    # Launch with Claude
  agate amp       # Launch with Amp
  agate cn        # Launch with Continue`,
		Args: cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(version.Short())
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("exactly one agent name is required")
			}
			return runAgent(args[0])
		},
	}

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
