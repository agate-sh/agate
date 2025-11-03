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
	"agate/pkg/gui/icons"
	"agate/pkg/gui/layout"
	"agate/pkg/gui/overlays"
	"agate/pkg/gui/panes"
	"agate/pkg/gui/theme"
	"agate/pkg/overlay"
	"agate/pkg/session"
	"agate/pkg/state"
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
	stateManager        *state.Manager   // Thread-safe state manager
	ready               bool
	focus               layout.FocusState // Hierarchical focus state
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
	toast               *components.Toast                    // Toast notification manager
	commitOverlay       *overlays.CommitOverlay              // Commit overlay for creating commits
	showCommitOverlay   bool                                 // Whether showing commit overlay

	// Panes using the new Pane interface
	repoPane components.Pane // Repos & worktrees pane (will be extracted from WorktreeList)
	tmuxPane components.Pane // Tmux terminal pane
	gitPane  components.Pane // Git file status pane
}

func initialModel(subprocess string) model {
	// Initialize debug logger FIRST so all subsequent logs are captured
	debugLogger := debug.InitDebugLogger()
	debug.DebugLog("Debug logger initialized successfully")

	// Initialize state manager (thread-safe state persistence)
	stateManager, err := state.NewManager()
	if err != nil {
		fmt.Printf("ERROR: failed to initialize state manager: %v\n", err)
		fmt.Printf("Agate will run without session persistence. Please check ~/.agate permissions.\n")
		debug.DebugLog("ERROR: StateManager initialization failed: %v", err)
		// Continue with nil stateManager - session manager will handle it gracefully
	} else {
		debug.DebugLog("StateManager initialized successfully")
	}

	// Initialize worktree manager
	worktreeManager, err := git.NewWorktreeManager()
	if err != nil {
		// Log error but don't fail - app can still work without worktree features
		fmt.Printf("Warning: failed to initialize worktree manager: %v\n", err)
	}

	// Create session manager with state manager
	sessionManager := session.NewManager(worktreeManager, stateManager)

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
	if subprocess != "" && stateManager != nil {
		if err := stateManager.SetDefaultAgent(subprocess); err != nil {
			debug.DebugLog("Failed to save default agent: %v", err)
			// Continue without saving - not critical
		}
	}

	// Create shortcut overlay using static GlobalKeys
	shortcutOverlay := common.NewShortcutOverlay(common.GlobalKeys)
	initialFocus := layout.NewSessionFocus(0, layout.SubPaneTmux)
	shortcutOverlay.SetFocus(initialFocus.String()) // Start with tmux pane focused
	shortcutOverlay.SetMode("preview")              // Start in preview mode

	// Create footer and help components
	footer := common.NewFooter()
	footer.SetShortcutOverlay(shortcutOverlay)
	footer.SetFocus(initialFocus.String()) // Start with tmux pane focused
	footer.SetMode("preview")              // Start in preview mode

	// Initialize worktree components
	var worktreeList *overlays.WorktreeList
	if worktreeManager != nil {
		worktreeList = overlays.NewWorktreeList(worktreeManager)
	}

	// Check if welcome overlay should be shown
	var showWelcomeOverlay bool
	if stateManager != nil {
		var welcomeShown bool
		stateManager.ReadUI(func(ui *config.UIState) error {
			welcomeShown = ui.Welcome.Shown
			return nil
		})
		showWelcomeOverlay = !welcomeShown
	}

	// Debug logger already initialized at the beginning of initialModel
	// Initialize debug overlay
	debugOverlay := overlays.NewDebugOverlay(debugLogger)

	// Set up debug logging for git package (always enabled now)
	git.DebugLog = debug.DebugLog

	// Create shared loading state
	loadingState := tmux.NewLoadingState()

	// Initialize all panes using the new Pane interface
	gitPane := panes.NewGitPane()
	tmuxPane := panes.NewAgentTmuxPane(loadingState)
	repoPane := panes.NewAgentsPane(sessionManager)

	m := model{
		layout:              layout.NewLayout(0, 0),  // Will be updated on first WindowSizeMsg
		sessionManager:      sessionManager,          // Session manager for coordination
		stateManager:        stateManager,            // State manager for persistence
		focus:               layout.NewAgentsFocus(), // Always start with focus on Agents pane
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
		toast:               components.NewToast(), // Toast notification manager

		// Initialize panes
		repoPane: repoPane,
		tmuxPane: tmuxPane,
		gitPane:  gitPane,
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
		m.repoPane.SetActive(targetPane.IsAgentsFocus())
	}
	// Tmux pane is active whenever we're on ANY session sub-pane (shows agent badge)
	if m.tmuxPane != nil {
		m.tmuxPane.SetActive(targetPane.PaneType == layout.PaneTypeSession)
	}
	if m.gitPane != nil {
		m.gitPane.SetActive(targetPane.IsGitFocus())
	}

	// Set the new focus
	m.focus = targetPane

	// Update footer and shortcut overlay
	m.footer.SetFocus(m.focus.String())
	m.shortcutOverlay.SetFocus(m.focus.String())

	// Special handling for repos & worktrees pane
	if targetPane.IsAgentsFocus() && m.repoPane != nil {
		// Refresh repo pane when focusing
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			if err := repoPane.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh repo pane when switching panes: %v", err)
				// UI continues to work, but log the issue for debugging
			}
		}

		// Update GitPane with selected worktree/repo and get refresh command
		return m, m.updateGitPane()
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

// switchToSessionForWorktree switches to the session associated with the given worktree
// and returns a command to immediately refresh the tmux content
func (m *model) switchToSessionForWorktree(worktree *git.WorktreeInfo) tea.Cmd {
	debug.DebugLog("switchToSessionForWorktree called for worktree: %s (branch: %s)", worktree.Path, worktree.Branch)

	if m.sessionManager == nil || worktree == nil {
		debug.DebugLog("switchToSessionForWorktree: sessionManager or worktree is nil")
		return nil
	}

	// Get or create session for this worktree
	sess, err := m.sessionManager.GetOrCreateSession(worktree, m.subprocess)
	if err != nil {
		debug.DebugLog("Failed to get/create session for worktree %s: %v", worktree.Path, err)
		return nil
	}

	debug.DebugLog("Got session: ID=%s, WorktreeKey=%s, TmuxName=%s", sess.ID, sess.WorktreeKey, sess.TmuxSession.GetSessionName())

	// Switch to this session
	m.sessionManager.SwitchToSession(sess.WorktreeKey)
	debug.DebugLog("Switched to session %s", sess.WorktreeKey)

	// Update global agent state
	app.SetCurrentAgent(sess.Agent)

	// Update tmux pane with the session
	if m.tmuxPane != nil {
		if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
			tmuxPane.SetSession(sess.TmuxSession)
			debug.DebugLog("Updated tmux pane with session %s", sess.TmuxSession.GetSessionName())
		}
	}

	debug.DebugLog("Switched to session %s with agent %s", sess.ID, sess.Agent.Name)

	// Return command to immediately refresh the tmux content for the new session
	// We need to force a refresh even if the content hasn't "changed" since we're switching sessions
	if sess.TmuxSession != nil {
		debug.DebugLog("Forcing immediate tmux content refresh for session %s", sess.TmuxSession.GetSessionName())
		return func() tea.Msg {
			// Force capture the content without checking HasUpdated
			content, err := sess.TmuxSession.CapturePaneContent()
			if err != nil {
				debug.DebugLog("ERROR capturing pane content: %v", err)
				return tmuxOutputMsg{content: ""}
			}
			debug.DebugLog("Captured %d bytes of content for session %s", len(content), sess.TmuxSession.GetSessionName())
			// Always return the content, even if it hasn't "changed"
			return tmuxOutputMsg{content: content, hasPrompt: true}
		}
	}
	debug.DebugLog("WARNING: No tmux session to refresh!")
	return nil
}

// updateGitPane updates the Git pane based on the currently selected worktree/repo
// and returns a command to refresh the tmux content
func (m *model) updateGitPane() tea.Cmd {
	debug.DebugLog("===== updateGitPane called =====")

	if m.gitPane == nil || m.repoPane == nil {
		debug.DebugLog("updateGitPane: gitPane or repoPane is nil")
		return nil
	}

	// Cast to AgentsPane to access GetSelectedWorktree method
	repoPane, ok := m.repoPane.(*panes.AgentsPane)
	if !ok {
		debug.DebugLog("updateGitPane: repoPane is not a AgentsPane")
		return nil
	}

	// Get the selected worktree from the repo pane
	selectedWorktree := repoPane.GetSelectedWorktree()
	if selectedWorktree == nil {
		debug.DebugLog("updateGitPane: no selected worktree")
		return nil
	}

	repoPath := selectedWorktree.Path
	debug.DebugLog("updateGitPane: selected worktree path=%s, branch=%s, repo=%s", repoPath, selectedWorktree.Branch, selectedWorktree.RepoName)

	// Switch to session for this worktree (this updates the agent and tmux session)
	// and get the command to refresh content
	refreshCmd := m.switchToSessionForWorktree(selectedWorktree)
	debug.DebugLog("updateGitPane: switchToSessionForWorktree returned cmd=%v", refreshCmd != nil)

	// Cast to GitPane to access SetRepository method
	if gitPane, ok := m.gitPane.(*panes.GitPane); ok {
		gitPane.SetRepository(repoPath)
		debug.DebugLog("updateGitPane: set git pane repository to %s", repoPath)
	}

	debug.DebugLog("===== updateGitPane returning cmd=%v =====", refreshCmd != nil)
	return refreshCmd
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.loadingState.TickCmd(),
	)
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

		// Calculate grid layout based on pinned sessions
		if m.sessionManager != nil {
			pinnedSessions := m.sessionManager.GetPinnedSessions()
			m.layout.CalculateGridLayout(len(pinnedSessions))
		}

		// Set the active pane based on current focus
		if m.repoPane != nil {
			m.repoPane.SetActive(m.focus.IsAgentsFocus())
			// Update repo pane size (always uses left dimensions)
			leftWidth, leftHeight := m.layout.GetLeftDimensions()
			m.repoPane.SetSize(leftWidth, leftHeight)
		}

		// Update component sizes
		m.footer.SetSize(msg.Width, 1)
		m.helpDialog.SetSize(msg.Width, msg.Height)

		// Update debug overlay size
		if m.debugOverlay != nil {
			m.debugOverlay.SetSize(msg.Width, msg.Height)
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
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.focus.IsTmuxFocus() {
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
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.focus.IsTmuxFocus() {
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
		debug.DebugLog("[tmuxDetachedMsg] ===== DETACHED FROM TMUX =====")
		if m.sessionManager != nil {
			activeSession := m.sessionManager.GetActiveSession()
			if activeSession != nil && activeSession.Worktree != nil {
				debug.DebugLog("[tmuxDetachedMsg] Active session before switchToPane: path=%s, branch=%s",
					activeSession.Worktree.Path, activeSession.Worktree.Branch)
			} else {
				debug.DebugLog("[tmuxDetachedMsg] No active session or worktree")
			}
		}

		// Left content is now handled by WorktreeList directly
		// ASCII art will be displayed by WorktreeList

		// Update footer back to preview mode
		m.footer.SetMode("preview")
		m.shortcutOverlay.SetMode("preview")

		debug.DebugLog("[tmuxDetachedMsg] About to call switchToPane(FocusAgents)")
		// Return focus to the agents pane (which will automatically jump to the active agent's row)
		m, _ = m.switchToPane(layout.NewAgentsFocus())
		debug.DebugLog("[tmuxDetachedMsg] Returned from switchToPane(FocusAgents)")

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
		// Create and switch to new session for the worktree FIRST
		if msg.Worktree != nil {
			// Use the agent name from the message (selected by user in dialog)
			agentName := msg.AgentName
			if agentName == "" {
				agentName = m.subprocess // Fallback to subprocess if not provided
			}
			// Create or get session for this worktree using session manager
			newSession, err := m.sessionManager.GetOrCreateSession(msg.Worktree, agentName)
			if err == nil {
				// Switch to the new session FIRST
				m.sessionManager.SwitchToSession(newSession.WorktreeKey)

				// Update agent based on new session
				app.SetCurrentAgent(newSession.Agent)

				// Update the agents pane to select the new worktree
				if agentsPane, ok := m.repoPane.(*panes.AgentsPane); ok {
					agentsPane.Refresh()
					agentsPane.SelectWorktreeByPath(msg.Worktree.Path)
				}

				// Refresh worktree list and update git pane
				if m.worktreeList != nil {
					if err := m.worktreeList.Refresh(); err != nil {
						debug.DebugLog("Failed to refresh worktree list after creating worktree: %v", err)
					}
					// Now updateGitPane will use the correct selected worktree
					m.updateGitPane()
				}

				// Update tmux pane with new session
				if m.tmuxPane != nil {
					if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
						tmuxPane.SetSession(newSession.TmuxSession)
					}
				}

				// Switch focus to tmux pane
				m.focus = layout.NewSessionFocus(0, layout.SubPaneTmux)
				// Update footer focus
				m.footer.SetFocus(m.focus.String())
				m.shortcutOverlay.SetFocus(m.focus.String())

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

		// Refresh the agents pane to show the new session AND select it
		if agentsPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			agentsPane.Refresh()
			if msg.Worktree != nil {
				// Select the newly created worktree so it becomes the active one
				agentsPane.SelectWorktreeByPath(msg.Worktree.Path)
				debug.DebugLog("[WorktreeInitializationCompleteMsg] Selected worktree: %s", msg.Worktree.Path)
			}
		}

		// Auto-attach to the newly created session's tmux
		if msg.Worktree != nil {
			newSession := m.sessionManager.GetSessionForWorktree(msg.Worktree)
			if newSession != nil && newSession.TmuxSession != nil && m.focus.IsTmuxFocus() {
				// Clear screen first
				fmt.Print("\033[2J\033[H")
				// Block directly in Update like Claude Squad
				detachCh, err := newSession.TmuxSession.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				// Block until detachment
				<-detachCh
				// Process detached message immediately
				return m.Update(tmuxDetachedMsg{})
			}
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

	case panes.PinErrorMsg:
		// Show error toast for pin failure (max 4 limit)
		if msg.Error != nil {
			cmd := m.toast.Show(msg.Error.Error(), 0)
			return m, cmd
		}
		return m, nil

	case panes.AttachToSessionMsg:
		// User wants to attach to a tmux session from the agents pane
		if msg.Session != nil && msg.Session.TmuxSession != nil {
			if m.sessionManager != nil {
				m.sessionManager.SwitchToSession(msg.Session.WorktreeKey)
				app.SetCurrentAgent(msg.Session.Agent)
			}

			if tmuxPane, ok := m.tmuxPane.(*panes.AgentTmuxPane); ok {
				tmuxPane.SetSession(msg.Session.TmuxSession)
			}
			m.focus = layout.NewSessionFocus(0, layout.SubPaneTmux)
			m.footer.SetFocus(m.focus.String())
			m.shortcutOverlay.SetFocus(m.focus.String())
			m.footer.SetMode("attached")
			m.shortcutOverlay.SetMode("attached")

			detachCh, err := msg.Session.TmuxSession.Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			<-detachCh
			return m.Update(tmuxDetachedMsg{})
		}
		return m, nil

	case overlays.SessionDialogCancelledMsg:
		// Dialog cancelled
		m.showSessionDialog = false
		m.worktreeDialog = nil
		return m, nil

	case panes.GitRefreshMsg:
		// Git pane needs to refresh after discard or other operations
		if m.gitPane != nil {
			var cmd tea.Cmd
			m.gitPane, cmd = m.gitPane.Update(msg)
			return m, cmd
		}
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
		// Session deleted successfully (already deleted by overlay)
		m.showSessionConfirm = false
		m.sessionConfirm = nil

		// Refresh UI to reflect deletion
		if m.repoPane != nil {
			if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
				var snapshot []panes.AgentListItem
				if msg.Session != nil {
					snapshot = repoPane.SnapshotItems()
				}

				if err := repoPane.Refresh(); err != nil {
					debug.DebugLog("Failed to refresh repo pane after session deletion: %v", err)
				}

				// Select the next appropriate session with priority:
				// 1. Session above, 2. Session below, 3. Main worktree
				if msg.Session != nil {
					repoPane.SelectSessionAfterDeletion(msg.Session, snapshot)
				}
			}
		}
		// Update Git pane
		m.updateGitPane()

		// Show success toast
		if msg.Session != nil && m.toast != nil {
			branchName := "session"
			if msg.Session.Worktree != nil {
				branchName = msg.Session.Worktree.Branch
			}
			// Create styled message with green checkmark
			checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
			checkmark := checkmarkStyle.Render("✓")
			message := checkmark + " Deleted " + branchName
			toastCmd := m.toast.Show(message, 0)
			return m, toastCmd
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
		if m.stateManager != nil {
			if err := m.stateManager.AddRepository(msg.Path); err != nil {
				m.err = fmt.Errorf("failed to save repository: %v", err)
			}
		}
		if m.err == nil {
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

		// Update commit overlay's spinner
		if m.showCommitOverlay && m.commitOverlay != nil {
			model, cmd := m.commitOverlay.Update(msg)
			m.commitOverlay = model.(*overlays.CommitOverlay)
			if cmd != nil {
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

	case components.ToastTickMsg:
		// Handle toast timer updates
		if m.toast != nil {
			return m, m.toast.Update(msg)
		}
		return m, nil

	case time.Time:
		// Pass time ticks to commit overlay for elapsed time updates
		if m.showCommitOverlay && m.commitOverlay != nil {
			model, cmd := m.commitOverlay.Update(msg)
			m.commitOverlay = model.(*overlays.CommitOverlay)
			return m, cmd
		}
		return m, nil

	case overlays.CommitSuccessMsg:
		// Commit succeeded - show success toast and close overlay
		m.showCommitOverlay = false
		m.commitOverlay = nil

		// Show success toast with green checkmark
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		checkmark := checkmarkStyle.Render("✓")
		message := fmt.Sprintf("%s Created commit %s", checkmark, msg.SHA)
		toastCmd := m.toast.Show(message, 0)

		// Refresh git pane to show updated status
		if m.gitPane != nil {
			if gitPane, ok := m.gitPane.(*panes.GitPane); ok {
				gitPane.Refresh()
			}
		}

		return m, toastCmd

	case overlays.CommitMessageGeneratedMsg, overlays.FileDiscardedMsg:
		// Pass overlay-specific messages to commit overlay
		if m.showCommitOverlay && m.commitOverlay != nil {
			model, cmd := m.commitOverlay.Update(msg)
			m.commitOverlay = model.(*overlays.CommitOverlay)
			return m, cmd
		}
		return m, nil

	case overlays.CommitErrorMsg:
		// Commit failed - show error toast and keep overlay open
		toastCmd := m.toast.Show(fmt.Sprintf("✗ Failed to create commit: %s", msg.Err.Error()), 0)
		return m, toastCmd

	case tea.KeyMsg:
		// If welcome overlay is visible, any key closes it
		if m.showWelcomeOverlay {
			m.showWelcomeOverlay = false
			// Mark welcome as shown so it doesn't appear again
			if m.stateManager != nil {
				m.stateManager.UpdateUI(func(ui *config.UIState) error {
					ui.Welcome.Shown = true
					return nil
				})
			}
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

		// Handle commit overlay input
		if m.showCommitOverlay && m.commitOverlay != nil {
			// Check for esc to close overlay
			if msg.String() == "esc" {
				m.showCommitOverlay = false
				m.commitOverlay = nil
				return m, nil
			}

			var cmd tea.Cmd
			model, cmd := m.commitOverlay.Update(msg)
			m.commitOverlay = model.(*overlays.CommitOverlay)
			return m, cmd
		}

		// Handle preview mode - navigation and mode switches only
		switch {
		case msg.String() == "enter":
			// Enter key handling - delegate to the active pane
			switch {
			case m.focus.IsAgentsFocus():
				// Let the repo pane handle enter key for toggling expansion
				if m.repoPane != nil {
					debug.DebugLog("Enter key pressed in Agents pane")
					handled, cmd := m.repoPane.HandleKey("enter")
					debug.DebugLog("Agents pane HandleKey returned: handled=%v, cmd=%v", handled, cmd != nil)
					if handled {
						// Update Git pane and get refresh command for tmux content
						refreshCmd := m.updateGitPane()
						debug.DebugLog("After updateGitPane: refreshCmd=%v", refreshCmd != nil)
						// Combine the pane's command with the refresh command
						combinedCmd := combineCmds(cmd, refreshCmd)
						debug.DebugLog("Returning combined command: %v", combinedCmd != nil)
						return m, combinedCmd
					}
				}
			case m.focus.IsGitFocus():
				// Let the git pane handle enter key for opening files
				if m.gitPane != nil {
					handled, cmd := m.gitPane.HandleKey("enter")
					if handled {
						return m, cmd
					}
				}
			case m.focus.IsTmuxFocus():
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
			}
			// Enter key now handles attachment for tmux pane

		case msg.String() == "d":
			// 'd' key handling - delegate to the agents pane for session deletion
			if m.focus.IsAgentsFocus() && m.repoPane != nil {
				handled, cmd := m.repoPane.HandleKey("d")
				if handled {
					return m, cmd
				}
			}
			// Also handle 'd' in git pane for discarding files
			if m.focus.IsGitFocus() && m.gitPane != nil {
				handled, cmd := m.gitPane.HandleKey("d")
				if handled {
					// Refresh git pane after discard
					m.updateGitPane()
					return m, cmd
				}
			}

		case key.Matches(msg, common.GlobalKeys.Quit):
			// Persist sessions before quitting
			if m.sessionManager != nil {
				debug.DebugLog("Quit: Persisting sessions before exit")
				if err := m.sessionManager.PersistSessions(); err != nil {
					debug.DebugLog("Quit: Failed to persist sessions: %v", err)
				} else {
					debug.DebugLog("Quit: Successfully persisted sessions before exit")
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

		case key.Matches(msg, common.GlobalKeys.NewSession):
			// Create new worktree (available from both panes)
			if m.sessionManager != nil {
				// Get default agent from state or use current subprocess
				var defaultAgent string
				if m.stateManager != nil {
					defaultAgent = m.stateManager.GetDefaultAgent()
				}
				if defaultAgent == "" {
					// Fallback to current subprocess if no default set
					defaultAgent = m.subprocess
				}

				var worktreeManager *git.WorktreeManager
				var repoName string
				if pane, ok := m.repoPane.(*panes.AgentsPane); ok {
					repoName = pane.GetHoveredRepoName()
				}

				var err error
				if repoName != "" {
					worktreeManager, err = m.sessionManager.GetWorktreeManagerForRepo(repoName)
					if err != nil {
						debug.DebugLog("NewSession: failed to resolve worktree manager for repo %s: %v", repoName, err)
					}
				}

				if worktreeManager == nil {
					worktreeManager = m.sessionManager.GetWorktreeManager()
				}

				if worktreeManager != nil {
					m.worktreeManager = worktreeManager
					m.worktreeDialog = overlays.NewSessionDialog(worktreeManager, defaultAgent)
					m.showSessionDialog = true
					return m, nil
				}
			}

		case key.Matches(msg, common.GlobalKeys.AttachAgent):
			// Attach to agent tmux session (global shortcut)
			if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
				m.footer.SetMode("attached")
				m.shortcutOverlay.SetMode("attached")
				detachCh, err := currentTmux.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				<-detachCh
				return m.Update(tmuxDetachedMsg{})
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Commit):
			// Show commit overlay (global shortcut)
			activeSession := m.sessionManager.GetActiveSession()
			if activeSession != nil && activeSession.Worktree != nil {
				// Check if there are any changes to commit
				fileStatus := git.GetFileStatuses(activeSession.Worktree.Path)
				if fileStatus == nil || fileStatus.IsClean {
					// No changes to commit - show toast
					toastCmd := m.toast.Show("No changes to commit", 0)
					return m, toastCmd
				}

				// There are changes - show commit overlay
				m.commitOverlay = overlays.NewCommitOverlay(activeSession)
				m.commitOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
				m.showCommitOverlay = true
				initCmd := m.commitOverlay.Init()
				return m, initCmd
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Up):
			// Navigate up in focused pane
			switch {
			case m.focus.IsAgentsFocus():
				if m.repoPane != nil {
					m.repoPane.MoveUp()
				}
			case m.focus.IsGitFocus():
				if m.gitPane != nil {
					m.gitPane.MoveUp()
					return m, nil
				}
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Down):
			// Navigate down in focused pane
			switch {
			case m.focus.IsAgentsFocus():
				if m.repoPane != nil {
					m.repoPane.MoveDown()
				}
			case m.focus.IsGitFocus():
				if m.gitPane != nil {
					m.gitPane.MoveDown()
					return m, nil
				}
			}
			return m, nil

			// OpenInEditor is now handled by GitPane's HandleKey method directly
			// No separate global keybinding needed

		case common.IsNextSubPaneKey(msg):
			// Shift+Tab toggles within session sub-panes: Tmux ↔ Git
			if m.focus.PaneType == layout.PaneTypeSession {
				var nextSubPane layout.SubPane
				switch m.focus.SessionSubPane {
				case layout.SubPaneTmux:
					nextSubPane = layout.SubPaneGit
				case layout.SubPaneGit:
					nextSubPane = layout.SubPaneTmux
				}
				return m.switchToPane(layout.NewSessionFocus(m.focus.SessionIndex, nextSubPane))
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.TabNextPane):
			// Tab key cycles through: Agents → Session[0] → Session[1] → ... → Agents
			if m.sessionManager == nil {
				return m, nil
			}

			pinnedSessions := m.sessionManager.GetPinnedSessions()
			numSessions := len(pinnedSessions)

			if numSessions == 0 {
				// No pinned sessions, stay on Agents
				return m, nil
			}

			if m.focus.IsAgentsFocus() {
				// Switch from Agents to first Session
				return m.switchToPane(layout.NewSessionFocus(0, layout.SubPaneTmux))
			} else if m.focus.PaneType == layout.PaneTypeSession {
				// Currently on a session, cycle to next or wrap to Agents
				nextSessionIndex := m.focus.SessionIndex + 1
				if nextSessionIndex >= numSessions {
					// Wrap back to Agents
					return m.switchToPane(layout.NewAgentsFocus())
				} else {
					// Go to next session
					return m.switchToPane(layout.NewSessionFocus(nextSessionIndex, layout.SubPaneTmux))
				}
			}
			return m, nil

		default:
			// Handle other key combinations if needed
		}

	case tea.MouseMsg:
		// Handle mouse events when right pane is focused
		if currentTmux := m.getCurrentTmuxSession(); m.focus.IsTmuxFocus() && currentTmux != nil {
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

// parseAndStyleShortcuts parses shortcut strings and applies default bubbles styling
func (m model) parseAndStyleShortcuts(shortcuts string) string {
	// Use the shortcut component's ParseAndRenderShortcuts function with default variant
	return components.ParseAndRenderShortcuts(shortcuts, components.ShortcutDefault, "")
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Get pinned sessions for grid rendering
	var panesWithPadding string

	if m.sessionManager == nil {
		return "No session manager"
	}

	pinnedSessions := m.sessionManager.GetPinnedSessions()

	if len(pinnedSessions) == 0 {
		// No pinned sessions - render agents pane + placeholder
		agentsContent := m.repoPane.View()

		// Calculate layout dimensions
		chromeHeight := layout.TopPaddingRows + layout.BottomSpacerRows + layout.PaneTitleRows + layout.FooterRows + layout.BottomMarginRows
		availableHeight := m.layout.GetHeight() - chromeHeight
		leftWidth, leftHeight := m.layout.GetLeftDimensions()
		frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
		contentPaddingWidth := components.PaneContentHorizontalPadding() * 2

		// Render agents pane
		agentsStyle := components.PaneBaseStyle
		if m.focus.IsAgentsFocus() {
			agentsStyle = agentsStyle.BorderForeground(lipgloss.Color(theme.BorderActive))
		}

		leftFullWidth := leftWidth + contentPaddingWidth

		if lipgloss.Width(agentsContent) < leftFullWidth {
			agentsContent = components.ApplyPaneContentPadding(agentsContent, leftWidth)
		}

		agentsWrapped := lipgloss.NewStyle().
			Width(leftFullWidth).
			MaxHeight(leftHeight).
			Render(agentsContent)
		agentsContentAligned := lipgloss.PlaceVertical(leftHeight, lipgloss.Top, agentsWrapped)
		agentsPane := agentsStyle.
			Height(availableHeight).
			Render(agentsContentAligned)

		// Create placeholder pane for right section
		totalHorizontalMargins := layout.HorizontalMargin*2 + layout.HorizontalGapWidth
		usableWidth := m.layout.GetWidth() - totalHorizontalMargins
		rightWidth := usableWidth - leftWidth - contentPaddingWidth - frameHeight

		placeholderText := "No pinned sessions\n\nPress ↵ on a session in the Agents pane to pin it"
		placeholderContentHeight := availableHeight - frameHeight
		placeholderContent := lipgloss.Place(
			rightWidth,
			placeholderContentHeight,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted)).Render(placeholderText),
		)

		// Render placeholder with border (same height as agents pane)
		placeholder := components.PaneBaseStyle.
			Height(availableHeight).
			Render(placeholderContent)

		// Join agents pane and placeholder
		gap := lipgloss.NewStyle().Width(layout.HorizontalGapWidth).Render("")
		panes := lipgloss.JoinHorizontal(lipgloss.Top, agentsPane, gap, placeholder)

		// Add padding
		panesWithPadding = lipgloss.NewStyle().
			PaddingTop(layout.TopPaddingRows).
			PaddingBottom(layout.BottomSpacerRows).
			PaddingLeft(layout.HorizontalMargin).
			PaddingRight(layout.HorizontalMargin).
			Render(panes)
	} else {
		// Calculate grid layout
		m.layout.CalculateGridLayout(len(pinnedSessions))

		// Build SessionPaneContent for each pinned session
		sessionContents := make([]layout.SessionPaneContent, len(pinnedSessions))
		for i, sess := range pinnedSessions {
			branchName := "Session"
			if sess.Worktree != nil && sess.Worktree.Branch != "" {
				branchName = sess.Worktree.Branch
			}

			// Determine the grid cell and size tmux panes accordingly
			contentWidth := 0
			contentHeight := 0
			if cell := m.layout.GetGridCell(i); cell != nil {
				tabsForSizing := []components.Tab{
					{Name: icons.GetGitRepo() + " " + branchName},
					{Name: "Changes"},
				}
				sizingPane := components.NewTabbedPane(tabsForSizing)
				sizingPane.SetSize(cell.Width, cell.Height)
				contentWidth, contentHeight = sizingPane.ContentSize()
			}

			if contentWidth > 0 && contentHeight > 0 {
				if sess.TmuxSession != nil {
					if err := sess.TmuxSession.SetDetachedSize(contentWidth, contentHeight); err != nil {
						debug.DebugLog("Failed to resize tmux session %s to %dx%d: %v", sess.ID, contentWidth, contentHeight, err)
					}
				}
			}

			// Get actual tmux content (after ensuring size matches the grid cell)
			tmuxContent := ""
			if sess.TmuxSession != nil {
				captured, err := sess.TmuxSession.CapturePaneContent()
				if err == nil {
					tmuxContent = captured
				}
			}

			// Get actual git files
			gitContent := ""
			changeCount := 0
			if sess.Worktree != nil {
				fileStatus := git.GetFileStatuses(sess.Worktree.Path)
				if fileStatus == nil || fileStatus.IsClean {
					gitContent = "No changes"
					changeCount = 0
				} else {
					var gitLines []string
					for _, file := range fileStatus.Files {
						gitLines = append(gitLines, file.FilePath)
					}
					gitContent = strings.Join(gitLines, "\n")
					changeCount = len(fileStatus.Files)
				}
			}

			// Determine active tab based on focus
			activeTab := 0 // Default to Tmux
			if m.focus.PaneType == layout.PaneTypeSession && m.focus.SessionIndex == i {
				switch m.focus.SessionSubPane {
				case layout.SubPaneTmux:
					activeTab = 0
				case layout.SubPaneGit:
					activeTab = 1
				}
			}

			sessionContents[i] = layout.SessionPaneContent{
				TmuxContent:  tmuxContent,
				GitContent:   gitContent,
				BranchName:   branchName,
				AgentColor:   app.GetCurrentAgentColor(),
				IsFocused:    m.focus.PaneType == layout.PaneTypeSession && m.focus.SessionIndex == i,
				ActiveTab:    activeTab,
				ChangesCount: changeCount,
			}
		}

		// Render grid panes
		panesWithPadding = m.layout.RenderGridPanes(m.repoPane.View(), sessionContents, m.focus)
	}

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

	// If commit overlay is visible, overlay it
	if m.showCommitOverlay && m.commitOverlay != nil {
		// Update dialog size
		m.commitOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return overlay.PlaceOverlay(0, 0, m.commitOverlay.View(), mainView, true, true)
	}

	// Render toast notifications (always rendered last, on top of everything)
	// Toasts do NOT dim the background and do NOT block interaction
	if m.toast != nil && m.toast.IsVisible() {
		return m.toast.PlaceOverlay(mainView, m.layout.GetWidth(), m.layout.GetHeight())
	}

	return mainView
}

func checkTmuxInstalled() error {
	if !tmux.IsTmuxInstalled() {
		installCmd := tmux.GetInstallCommand()
		return fmt.Errorf("tmux is not installed. Please install tmux to use Agate.\n%s", installCmd)
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
