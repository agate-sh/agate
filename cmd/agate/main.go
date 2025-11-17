package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/internal/version"
	"agate/pkg/agents"
	"agate/pkg/app"
	"agate/pkg/telemetry"
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/gui/components"
	"agate/pkg/gui/layout"
	"agate/pkg/gui/metrics"
	"agate/pkg/gui/overlays"
	"agate/pkg/gui/panes"
	"agate/pkg/gui/theme"
	"agate/pkg/overlay"
	"agate/pkg/session"
	"agate/pkg/state"
	"agate/pkg/tmux"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
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
	helpDialog          *overlays.HelpDialog                 // Help dialog overlay
	showHelp            bool                                 // Whether help dialog is visible
	worktreeManager     *git.WorktreeManager                 // Git worktree management
	worktreeList        *overlays.WorktreeList               // Worktree list component
	worktreeConfirm     *overlays.WorktreeConfirmDialog      // Worktree deletion confirmation
	sessionConfirm      *overlays.SessionDeleteConfirmDialog // Session deletion confirmation
	showWorktreeConfirm bool                                 // Whether showing worktree deletion confirmation
	showSessionConfirm  bool                                 // Whether showing session deletion confirmation
	debugLogger         *debug.DebugLogger                   // Debug logger for development
	debugOverlay        *overlays.DebugOverlay               // Debug overlay for development
	showDebugOverlay    bool                                 // Whether showing debug overlay
	loadingState        *tmux.LoadingState                   // Loading state manager with spinner and stopwatch
	toast               *components.Toast                    // Toast notification manager
	mergeOverlay       *overlays.MergeOverlay              // Merge overlay for merging changes
	showMergeOverlay   bool                                 // Whether showing merge overlay
	agentSelector       *components.AgentSelector            // Agent selector modal
	showAgentSelector   bool                                 // Whether showing agent selector

	// New session creation flow
	chatInput             *components.ChatInput     // Chat input component for new session creation
	welcomeHeader         *components.WelcomeHeader // Header with ASCII art, version, and shortcuts
	showNewSessionInput   bool                      // true when showing new session input (no sessions or user pressed 'n')
	creatingSession       bool                      // true during session creation (shows toast)
	generatingBranchName  bool                      // true during branch name generation (shows "Generating branch name..." toast)
	generatingDescription bool                      // true while description is being generated in background

	// Panes using the new Pane interface
	repoPane        components.Pane // Agents pane (list of sessions)
	sessionViewPane components.Pane // SessionViewPane instance (header + tmux)
	changesPane     components.Pane // ChangesPane instance (file changes)
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

	// Resolve which agents to use for initialization
	// Priority: CLI argument > Config > Default ["claude", "codex"]
	var agentNames []string
	if subprocess != "" {
		// Parse comma-separated agents from CLI
		agentNames = strings.Split(subprocess, ",")
		// Trim whitespace from each agent name
		for i := range agentNames {
			agentNames[i] = strings.TrimSpace(agentNames[i])
		}
	} else if stateManager != nil {
		// Load from config (returns ["claude", "codex", "gemini"] if empty)
		agentNames = stateManager.GetSelectedAgents()
	} else {
		// Fallback if no state manager
		agentNames = []string{"claude", "codex", "gemini"}
	}

	// Convert agent names to AgentConfig objects
	agentConfigs := make([]agents.AgentConfig, 0, len(agentNames))
	for _, name := range agentNames {
		agentConfigs = append(agentConfigs, agents.GetAgentConfig(name))
	}

	// Set the first agent globally for backwards compatibility
	if len(agentConfigs) > 0 {
		app.SetCurrentAgent(agentConfigs[0])
	}

	// Create shortcut overlay using static GlobalKeys
	shortcutOverlay := common.NewShortcutOverlay(common.GlobalKeys)
	initialFocus := layout.NewSessionFocus(layout.SubPaneTmux)
	shortcutOverlay.SetFocus(initialFocus.String()) // Start with tmux pane focused
	shortcutOverlay.SetMode("preview")              // Start in preview mode

	// Initialize worktree components
	var worktreeList *overlays.WorktreeList
	if worktreeManager != nil {
		worktreeList = overlays.NewWorktreeList(worktreeManager)
	}

	// Debug logger already initialized at the beginning of initialModel
	// Initialize debug overlay
	debugOverlay := overlays.NewDebugOverlay(debugLogger)

	// Set up debug logging for git package (always enabled now)
	git.DebugLog = debug.DebugLog

	// Create shared loading state
	loadingState := tmux.NewLoadingState()

	// Initialize all panes using the new Pane interface
	repoPane := panes.NewAgentsPane(sessionManager)
	sessionViewPane := panes.NewSessionViewPane()
	changesPane := panes.NewChangesPane()

	// Initialize chat input with resolved agents
	// Use first agent as the "default" for the chat input, then set all selected agents
	chatInput := components.NewChatInput(agentConfigs[0])
	chatInput.SetSelectedAgents(agentConfigs)

	// Initialize welcome header
	welcomeHeader := components.NewWelcomeHeader()

	// Always start with new session view on launch
	showNewSessionInput := true

	m := model{
		layout:                layout.NewLayout(0, 0),  // Will be updated on first WindowSizeMsg
		sessionManager:        sessionManager,          // Session manager for coordination
		stateManager:          stateManager,            // State manager for persistence
		focus:                 layout.NewAgentsFocus(), // Always start with focus on Agents pane
		subprocess:            subprocess,
		mode:                  modePreview, // Start in preview mode
		ready:                 true,        // Ready immediately since we show new session view
		shortcutOverlay:       shortcutOverlay,
		helpDialog:            overlays.NewHelpDialog(common.GlobalKeys),
		showHelp:              false,
		worktreeManager:       worktreeManager,
		worktreeList:          worktreeList,
		showWorktreeConfirm:   false,
		showSessionConfirm:    false,
		debugLogger:           debugLogger,
		debugOverlay:          debugOverlay,
		showDebugOverlay:      false,
		loadingState:          loadingState,
		toast:                 components.NewToast(), // Toast notification manager
		chatInput:             chatInput,
		welcomeHeader:         welcomeHeader,
		showNewSessionInput:   showNewSessionInput,
		creatingSession:       false,
		generatingBranchName:  false,
		generatingDescription: false,

		// Initialize panes
		repoPane:        repoPane,
		sessionViewPane: sessionViewPane,
		changesPane:     changesPane,
	}

	// Since we start with new session input view, mark all panes as inactive
	if showNewSessionInput {
		if m.repoPane != nil {
			m.repoPane.SetActive(false)
		}
		if m.sessionViewPane != nil {
			m.sessionViewPane.SetActive(false)
		}
		if m.changesPane != nil {
			m.changesPane.SetActive(false)
		}
	}

	// Initialize Changes pane content if repo pane has items
	if m.repoPane != nil {
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok && repoPane.HasItems() {
			m.updateChangesPane()
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
	// Session view pane is active only when we're on the tmux sub-pane
	if m.sessionViewPane != nil {
		m.sessionViewPane.SetActive(targetPane.IsTmuxFocus())
	}
	if m.changesPane != nil {
		m.changesPane.SetActive(targetPane.IsGitFocus())
	}

	// Set the new focus
	m.focus = targetPane

	// Update shortcut overlay
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

		// Update ChangesPane with selected worktree/repo and get refresh command
		return m, m.updateChangesPane()
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
	return activeSession.TmuxSession()
}

// getDetachedTmuxSize returns the width and height for the tmux window in detached mode.
// For shared sessions with multiple agents, multiplies width by agent count so each pane
// gets the full contentWidth.
func (m *model) getDetachedTmuxSize(contentWidth, contentHeight int) (int, int) {
	if m.sessionManager == nil {
		return contentWidth, contentHeight
	}

	activeSession := m.sessionManager.GetActiveSession()
	if activeSession == nil {
		return contentWidth, contentHeight
	}

	// For shared sessions, multiply width by agent count
	agents := activeSession.GetOrderedAgents()
	agentCount := len(agents)
	if agentCount > 1 {
		// Each pane gets contentWidth, so total window width = contentWidth * agentCount
		return contentWidth * agentCount, contentHeight
	}

	return contentWidth, contentHeight
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

	debug.DebugLog("Got session: ID=%s, TmuxName=%s", sess.ID, sess.TmuxSession().GetSessionName())

	// Switch to this session
	m.sessionManager.SwitchToSession(sess.ID)
	debug.DebugLog("Switched to session %s", sess.ID)

	// Update global agent state
	app.SetCurrentAgent(sess.Agent())

	// Update session view pane with the session
	if m.sessionViewPane != nil {
		if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
			sessionViewPane.SetSession(sess)
			debug.DebugLog("Updated session view pane with session %s", sess.TmuxSession().GetSessionName())
		}
	}

	debug.DebugLog("Switched to session %s with agent %s", sess.ID, sess.Agent().Name)

	// Return command to immediately refresh the tmux content for the new session
	// We need to force a refresh even if the content hasn't "changed" since we're switching sessions
	if sess.TmuxSession() != nil {
		debug.DebugLog("Forcing immediate tmux content refresh for session %s", sess.TmuxSession().GetSessionName())
		return func() tea.Msg {
			// Force capture the content without checking HasUpdated
			content, err := sess.TmuxSession().CapturePaneContent()
			if err != nil {
				debug.DebugLog("ERROR capturing pane content: %v", err)
				return tmuxOutputMsg{content: ""}
			}
			debug.DebugLog("Captured %d bytes of content for session %s", len(content), sess.TmuxSession().GetSessionName())
			// Always return the content, even if it hasn't "changed"
			return tmuxOutputMsg{content: content, hasPrompt: true}
		}
	}
	debug.DebugLog("WARNING: No tmux session to refresh!")
	return nil
}

// cycleNextAgent advances the active agent in the current session and updates the UI.
func (m *model) cycleNextAgent() tea.Cmd {
	if m.sessionManager == nil {
		return nil
	}

	activeSession := m.sessionManager.GetActiveSession()
	if activeSession == nil {
		return nil
	}

	if err := m.sessionManager.CycleActiveAgent(activeSession.ID); err != nil {
		debug.DebugLog("Failed to cycle active agent: %v", err)
		return nil
	}

	activeSession = m.sessionManager.GetActiveSession()
	if activeSession == nil {
		return nil
	}

	app.SetCurrentAgent(activeSession.Agent())

	if m.sessionViewPane != nil {
		if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
			sessionViewPane.SetSession(activeSession)
			sessionViewPane.SetActive(true)
		}
	}

	m.focus = layout.NewSessionFocus(layout.SubPaneTmux)
	if m.shortcutOverlay != nil {
		m.shortcutOverlay.SetFocus(m.focus.String())
	}

	activeAgent := activeSession.GetActiveAgent()
	var cmds []tea.Cmd

	if activeAgent != nil {
		if activeAgent.Worktree != nil {
			if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
				if !repoPane.SelectWorktreeByPath(activeAgent.Worktree.Path) {
					debug.DebugLog("cycleNextAgent: worktree %s not found in agents pane", activeAgent.Worktree.Path)
				}
			}
			if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
				changesPane.SetRepository(activeAgent.Worktree.Path)
			}
		}

		if activeSession.SharedTmux != nil {
			tmuxSession := activeSession.SharedTmux

			// Force capture immediately so the pane reflects the newly selected agent
			cmds = append(cmds, func() tea.Msg {
				content, err := tmuxSession.CapturePaneContent()
				if err != nil {
					debug.DebugLog("Failed to capture tmux content for session %s: %v", activeSession.ID, err)
					return tmuxOutputMsg{content: ""}
				}
				debug.DebugLog("Captured %d bytes of content for agent switch in session %s", len(content), activeSession.ID)
				return tmuxOutputMsg{content: content, hasPrompt: true}
			})

			// Resume normal monitoring after the forced capture
			cmds = append(cmds, waitForTmuxOutput(tmuxSession))
		}
	}

	return combineCmds(cmds...)
}

// updateChangesPane updates the Changes pane based on the currently selected worktree/repo
// and returns a command to refresh the tmux content
func (m *model) updateChangesPane() tea.Cmd {
	debug.DebugLog("===== updateChangesPane called =====")

	if m.changesPane == nil || m.repoPane == nil {
		debug.DebugLog("updateChangesPane: changesPane or repoPane is nil")
		return nil
	}

	// Cast to AgentsPane to access GetSelectedWorktree method
	repoPane, ok := m.repoPane.(*panes.AgentsPane)
	if !ok {
		debug.DebugLog("updateChangesPane: repoPane is not a AgentsPane")
		return nil
	}

	// Get the selected worktree from the repo pane
	selectedWorktree := repoPane.GetSelectedWorktree()
	if selectedWorktree == nil {
		debug.DebugLog("updateChangesPane: no selected worktree")
		return nil
	}

	repoPath := selectedWorktree.Path
	debug.DebugLog("updateChangesPane: selected worktree path=%s, branch=%s, repo=%s", repoPath, selectedWorktree.Branch, selectedWorktree.RepoName)

	// Switch to session for this worktree (this updates the agent and tmux session)
	// and get the command to refresh content
	refreshCmd := m.switchToSessionForWorktree(selectedWorktree)
	debug.DebugLog("updateChangesPane: switchToSessionForWorktree returned cmd=%v", refreshCmd != nil)

	// Cast to ChangesPane to access SetRepository method
	if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
		changesPane.SetRepository(repoPath)
		debug.DebugLog("updateChangesPane: set changes pane repository to %s", repoPath)
	}

	debug.DebugLog("===== updateChangesPane returning cmd=%v =====", refreshCmd != nil)
	return refreshCmd
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.EnterAltScreen,
		m.loadingState.TickCmd(),
	}

	if m.showNewSessionInput && m.chatInput != nil {
		if initCmd := m.chatInput.InitCmd(); initCmd != nil {
			cmds = append(cmds, initCmd)
		}
	}

	return tea.Batch(cmds...)
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

// New session creation messages
type branchNameGeneratedMsg struct {
	branchName string
	err        error
}

type sessionCreatedMsg struct {
	session *session.Session
	err     error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showAgentSelector && m.agentSelector != nil {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			if m.chatInput != nil {
				m.chatInput.SetSelectedAgents(m.agentSelector.GetSelectedAgents())
			}
			m.showAgentSelector = false
			m.agentSelector = nil
			return m, nil
		}

		var cmd tea.Cmd
		m.agentSelector, cmd = m.agentSelector.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update layout with new dimensions
		m.layout.Update(msg.Width, msg.Height)
		m.ready = true

		// Set the active pane based on current focus
		// UNLESS we're showing new session input (then all panes should be inactive)
		if m.repoPane != nil {
			if !m.showNewSessionInput {
				m.repoPane.SetActive(m.focus.IsAgentsFocus())
			}
			// Update repo pane size (always uses left dimensions)
			leftWidth, leftHeight := m.layout.GetLeftDimensions()
			m.repoPane.SetSize(leftWidth, leftHeight)
		}

		if m.sessionViewPane != nil {
			if !m.showNewSessionInput {
				m.sessionViewPane.SetActive(m.focus.IsTmuxFocus())
			}
			centerWidth, centerHeight := m.layout.GetCenterDimensions()
			m.sessionViewPane.SetSize(centerWidth, centerHeight)
		}

		if m.changesPane != nil {
			if !m.showNewSessionInput {
				m.changesPane.SetActive(m.focus.IsGitFocus())
			}
			rightWidth, rightHeight := m.layout.GetRightDimensions()
			m.changesPane.SetSize(rightWidth, rightHeight)
		}

		// Update component sizes
		m.helpDialog.SetSize(msg.Width, msg.Height)

		// Update debug overlay size
		if m.debugOverlay != nil {
			m.debugOverlay.SetSize(msg.Width, msg.Height)
		}

		// Update Changes pane content after all components are sized
		// This ensures the worktree list has proper dimensions and selection
		m.updateChangesPane()

	case tmuxSessionStartedMsg:
		// Store the session (msg.session is now a *session.Session)
		activeSession := msg.session

		// Set the current agent based on the session's agent
		app.SetCurrentAgent(activeSession.Agent())

		// Initialize session view pane with the session
		if m.sessionViewPane != nil {
			if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				sessionViewPane.SetSession(activeSession)
			}
		}

		// Start loading timer for stopwatch
		m.loadingState.Start()

		// Update Changes pane now that the app is fully initialized
		// This ensures the worktree list has been sized and has a selection
		m.updateChangesPane()

		// Set initial tmux session size using layout
		if m.ready && activeSession.TmuxSession() != nil {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				// For shared sessions, multiply width by agent count so each pane gets full width
				tmuxWidth, tmuxHeight := m.getDetachedTmuxSize(contentWidth, contentHeight)
				if err := activeSession.TmuxSession().SetDetachedSize(tmuxWidth, tmuxHeight); err != nil {
					debug.DebugLog("Failed to set tmux session initial size to %dx%d: %v", tmuxWidth, tmuxHeight, err)
					// Continue - tmux will use default size
				}
			}
		}

		// Start monitoring tmux output and set up loading timeout
		return m, tea.Batch(
			waitForTmuxOutput(activeSession.TmuxSession()),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return loadingTimeoutMsg{}
			}),
		)

	case tmuxOutputMsg:
		// Update session view pane tmux content
		if msg.content != "" {
			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetTmuxContent(msg.content)
				}
			}
			// Clear loading timer when we have meaningful content
			if m.loadingState.IsLoading() && strings.TrimSpace(msg.content) != "" {
				m.loadingState.Stop()
			}

			// On first real output, ensure Changes pane is initialized
			m.updateChangesPane()
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
		// TODO: This handler is part of the old session dialog system and should be removed
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
			if activeSession != nil {
				if activeSession.Worktree() != nil {
					debug.DebugLog("[tmuxDetachedMsg] Active session before switchToPane: path=%s, branch=%s",
						activeSession.Worktree().Path, activeSession.Worktree().Branch)
				}
			} else {
				debug.DebugLog("[tmuxDetachedMsg] No active session or worktree")
			}
		}

		// Left content is now handled by WorktreeList directly
		// ASCII art will be displayed by WorktreeList

		// Update shortcut overlay back to preview mode
		m.shortcutOverlay.SetMode("preview")

		debug.DebugLog("[tmuxDetachedMsg] About to call switchToPane(FocusAgents)")
		// Return focus to the agents pane (which will automatically jump to the active agent's row)
		m, _ = m.switchToPane(layout.NewAgentsFocus())
		debug.DebugLog("[tmuxDetachedMsg] Returned from switchToPane(FocusAgents)")

		// Immediately resize the tmux session to current window dimensions
		if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil && m.ready {
			if contentWidth, contentHeight := m.layout.GetTmuxDimensions(); contentWidth > 0 && contentHeight > 0 {
				// For shared sessions, multiply width by agent count so each pane gets full width
				tmuxWidth, tmuxHeight := m.getDetachedTmuxSize(contentWidth, contentHeight)
				if err := currentTmux.SetDetachedSize(tmuxWidth, tmuxHeight); err != nil {
					debug.DebugLog("Failed to resize tmux session after detaching to %dx%d: %v", tmuxWidth, tmuxHeight, err)
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

	// Worktree dialog messages (TODO: This is part of the old system, may need refactoring)
	case overlays.WorktreeCreatedMsg:
		var cmds []tea.Cmd

		// Worktree created successfully - start tmux session
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
				m.sessionManager.SwitchToSession(newSession.ID)

				// Update agent based on new session
				app.SetCurrentAgent(newSession.Agent())

				// Update the agents pane to select the new worktree
				if agentsPane, ok := m.repoPane.(*panes.AgentsPane); ok {
					agentsPane.Refresh()
					agentsPane.SelectWorktreeByPath(msg.Worktree.Path)
				}

				// Refresh worktree list and update changes pane
				if m.worktreeList != nil {
					if err := m.worktreeList.Refresh(); err != nil {
						debug.DebugLog("Failed to refresh worktree list after creating worktree: %v", err)
					}
					// Now updateChangesPane will use the correct selected worktree
					m.updateChangesPane()
				}

				// Update session view pane with new session
				if m.sessionViewPane != nil {
					if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
						sessionViewPane.SetSession(newSession)
					}
				}

				// Switch focus to session pane
				m.focus = layout.NewSessionFocus(layout.SubPaneTmux)
				// Update shortcut overlay focus
				m.shortcutOverlay.SetFocus(m.focus.String())

				// Start monitoring the new session
				if newSession.TmuxSession() != nil {
					cmds = append(cmds, waitForTmuxOutput(newSession.TmuxSession()))
				}
			} else {
				debug.DebugLog("Failed to create session for worktree: %v", err)
			}
		}
		return m, combineCmds(cmds...)

	case overlays.WorktreeInitializationCompleteMsg:
		// Initialization complete - auto-attach (TODO: Part of old system)

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
			if newSession != nil && newSession.TmuxSession() != nil && m.focus.IsTmuxFocus() {
				// Clear screen first
				fmt.Print("\033[2J\033[H")
				// Block directly in Update like Claude Squad
				detachCh, err := newSession.TmuxSession().Attach()
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
		// TODO: Part of old system, handle error appropriately
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

	case panes.SessionSwitchedMsg:
		// Session was auto-switched by navigation in agents pane
		// Hide new session input and show the 3-pane layout
		if msg.Session != nil {
			m.showNewSessionInput = false

			// Ensure session pane reflects the new active session
			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetSession(msg.Session)
				}
			}

			// Force a tmux content refresh for the newly selected agent
			if msg.Session.TmuxSession() != nil {
				return m, waitForTmuxOutput(msg.Session.TmuxSession())
			}
		}
		return m, nil

	case panes.AttachToSessionMsg:
		// User wants to attach to a tmux session from the agents pane
		if msg.Session != nil && msg.Session.TmuxSession != nil {
			if m.sessionManager != nil {
				m.sessionManager.SwitchToSession(msg.Session.ID)
				app.SetCurrentAgent(msg.Session.Agent())
			}

			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetSession(msg.Session)
				}
			}
			m.focus = layout.NewSessionFocus(layout.SubPaneTmux)
			m.shortcutOverlay.SetFocus(m.focus.String())
			m.shortcutOverlay.SetMode("attached")

			detachCh, err := msg.Session.TmuxSession().Attach()
			if err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			<-detachCh
			return m.Update(tmuxDetachedMsg{})
		}
		return m, nil

	case overlays.SessionDialogCancelledMsg:
		// TODO: Old dialog system, remove
		return m, nil

	case components.OpenAgentSelectorMsg:
		// Open agent selector with current selections from chat input
		if m.chatInput != nil {
			m.agentSelector = components.NewAgentSelector(m.chatInput.GetSelectedAgents())
			m.agentSelector.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
			m.showAgentSelector = true

			var cmds []tea.Cmd
			if initCmd := m.agentSelector.InitCmd(); initCmd != nil {
				cmds = append(cmds, initCmd)
			}
			cmds = append(cmds, textinput.Blink)

			return m, tea.Batch(cmds...)
		}
		return m, nil

	case panes.GitRefreshMsg:
		// Changes pane needs to refresh after discard or other operations
		if m.changesPane != nil {
			var cmd tea.Cmd
			m.changesPane, cmd = m.changesPane.Update(msg)
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
			// Update Changes pane after deletion
			m.updateChangesPane()
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
		// Update Changes pane
		m.updateChangesPane()

		// Show success toast
		if msg.Session != nil && m.toast != nil {
			branchName := "session"
			if msg.Session.Worktree != nil {
				branchName = msg.Session.Worktree().Branch
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

		// Update SessionViewPane's spinner if needed
		if m.sessionViewPane != nil {
			if _, cmd := m.sessionViewPane.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update merge overlay's spinner
		if m.showMergeOverlay && m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update chat input spinner if generating
		if m.showNewSessionInput && m.chatInput != nil {
			if cmd := m.chatInput.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
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
		// Pass time ticks to merge overlay for elapsed time updates
		if m.showMergeOverlay && m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}
		return m, nil

	case overlays.MergeSuccessMsg:
		// Merge succeeded - show success toast and close overlay
		m.showMergeOverlay = false
		m.mergeOverlay = nil

		// Show success toast with green checkmark
		checkmarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.SuccessStatus))
		checkmark := checkmarkStyle.Render("✓")
		message := fmt.Sprintf("%s Merged %s to main", checkmark, msg.SHA)
		toastCmd := m.toast.Show(message, 0)

		// Refresh changes pane to show updated status
		if m.changesPane != nil {
			if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
				changesPane.Refresh()
			}
		}

		return m, toastCmd

	case overlays.MergeMessageGeneratedMsg, overlays.FileDiscardedMsg:
		// Pass overlay-specific messages to merge overlay
		if m.showMergeOverlay && m.mergeOverlay != nil {
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}
		return m, nil

	case overlays.MergeErrorMsg:
		// Merge failed - show error toast and keep overlay open
		toastCmd := m.toast.Show(fmt.Sprintf("✗ Failed to merge: %s", msg.Err.Error()), 0)
		return m, toastCmd

	case branchNameGeneratedMsg:
		// Branch name generation completed
		m.generatingBranchName = false

		if msg.err != nil {
			// Generation failed - use random fallback
			debug.DebugLog("Branch name generation failed: %v, using random name", msg.err)
			msg.branchName = session.GenerateRandomBranchName()
		}

		// Now start session creation
		m.creatingSession = true
		prompt := m.chatInput.GetValue()
		agents := m.chatInput.GetSelectedAgents()
		agentNames := make([]string, 0, len(agents))
		for _, agent := range agents {
			agentNames = append(agentNames, agent.Name)
		}

		// Build agent list for toast message
		var agentNameList []string
		for _, agent := range agents {
			agentNameList = append(agentNameList, "@"+agent.Name)
		}
		toastMsg := "Creating " + strings.Join(agentNameList, ", ") + " agent(s)..."
		toastCmd := m.toast.Show(toastMsg, 0)

		createCmd := func() tea.Msg {
			newSession, err := m.sessionManager.CreateSession(prompt, msg.branchName, agentNames)
			return sessionCreatedMsg{session: newSession, err: err}
		}

		return m, tea.Batch(toastCmd, createCmd)

	case sessionCreatedMsg:
		// Session creation completed
		m.creatingSession = false

		if msg.err != nil {
			// Session creation failed - show error
			toastCmd := m.toast.Show(fmt.Sprintf("Failed to create session: %v", msg.err), 0)
			return m, toastCmd
		}

		// Session created successfully!
		m.showNewSessionInput = false
		m.chatInput.Reset()

		// Make the new session the active session immediately so the rest of the UI
		// (changes pane, tmux monitor, agent badges, etc.) points at the right data.
		if m.sessionManager != nil && msg.session != nil {
			if _, err := m.sessionManager.SwitchToSession(msg.session.ID); err != nil {
				debug.DebugLog("Failed to switch to newly created session %s: %v", msg.session.ID, err)
			}
		}
		app.SetCurrentAgent(msg.session.Agent())

		// Update session view pane
		if m.sessionViewPane != nil {
			if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				sessionView.SetSession(msg.session)
			}
		}

		var cmds []tea.Cmd

		// Set tmux content from shared session
		activeAgent := msg.session.GetActiveAgent()
		if activeAgent != nil && msg.session.SharedTmux != nil {
			if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				content, _ := msg.session.SharedTmux.CapturePaneContent()
				sessionView.SetTmuxContent(content)
			}

			// Ensure the agents pane selects this worktree so focus/changes pane stay in sync
			if m.repoPane != nil && activeAgent.Worktree != nil {
				if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
					if err := repoPane.Refresh(); err != nil {
						debug.DebugLog("Failed to refresh agents pane after session creation: %v", err)
					}
					if !repoPane.SelectWorktreeByPath(activeAgent.Worktree.Path) {
						debug.DebugLog("Agents pane could not find worktree %s right after session creation", activeAgent.Worktree.Path)
					}
				}
			}

			// Update changes pane
			if m.changesPane != nil && activeAgent.Worktree != nil {
				if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
					changesPane.SetRepository(activeAgent.Worktree.Path)
				}
			}

			// Force an immediate tmux capture to populate the pane, then start monitoring
			cmds = append(cmds, func() tea.Msg {
				if msg.session.SharedTmux == nil {
					return tmuxOutputMsg{content: ""}
				}
				content, err := msg.session.SharedTmux.CapturePaneContent()
				if err != nil {
					debug.DebugLog("ERROR capturing pane content for new session %s: %v", msg.session.ID, err)
					return tmuxOutputMsg{content: ""}
				}
				debug.DebugLog("Captured %d bytes of content for new session %s", len(content), msg.session.SharedTmux.GetSessionName())
				return tmuxOutputMsg{content: content, hasPrompt: true}
			})

			cmds = append(cmds, waitForTmuxOutput(msg.session.SharedTmux))
		}

		// Persist the selected agents to config for next session
		if m.chatInput != nil && m.stateManager != nil {
			selectedAgents := m.chatInput.GetSelectedAgents()
			agentNames := make([]string, 0, len(selectedAgents))
			for _, agent := range selectedAgents {
				agentNames = append(agentNames, agent.Name)
			}
			if err := m.stateManager.SetSelectedAgents(agentNames); err != nil {
				debug.DebugLog("Failed to persist selected agents: %v", err)
				// Continue - not critical
			} else {
				debug.DebugLog("Persisted selected agents: %v", agentNames)
			}
		}

		// Start async description generation
		m.generatingDescription = true
		prompt := msg.session.Prompt
		defaultAgent := agents.GetAgentConfig(m.subprocess)
		descCmd := session.GenerateSessionDescription(prompt, defaultAgent, msg.session, m.sessionManager)

		cmds = append(cmds, descCmd)

		return m, combineCmds(cmds...)

	case session.SessionDescriptionGeneratedMsg:
		// Description generation completed (async)
		m.generatingDescription = false

		if msg.Error != nil {
			debug.DebugLog("Description generation failed: %v", msg.Error)
			// Continue - session is functional without description
			return m, nil
		}

		// Description was already updated in the session manager by the generation function
		// Just refresh the session view
		if m.sessionViewPane != nil {
			activeSession := m.sessionManager.GetActiveSession()
			if activeSession != nil && activeSession.ID == msg.SessionID {
				if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionView.SetSession(activeSession)
				}
			}
		}

		return m, nil

	case tea.KeyMsg:
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

		// TODO: Old worktree dialog handler removed

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

		// Handle merge overlay input
		if m.showMergeOverlay && m.mergeOverlay != nil {
			// Check for esc to close overlay
			if msg.String() == "esc" {
				m.showMergeOverlay = false
				m.mergeOverlay = nil
				return m, nil
			}

			var cmd tea.Cmd
			model, cmd := m.mergeOverlay.Update(msg)
			m.mergeOverlay = model.(*overlays.MergeOverlay)
			return m, cmd
		}

		// Handle chat input when showing new session input
		if m.showNewSessionInput && m.chatInput != nil {
			// Check for pane navigation keys - these should NOT be consumed by chat input
			isNavKey := key.Matches(msg, common.GlobalKeys.AgentsPane) ||
				key.Matches(msg, common.GlobalKeys.SessionPane) ||
				key.Matches(msg, common.GlobalKeys.ChangesPane) ||
				key.Matches(msg, common.GlobalKeys.Keybindings) ||
				key.Matches(msg, common.GlobalKeys.Quit)

			// Check if agents pane is active - if so, skip chat input and let pane handle it
			agentsPaneActive := m.focus.IsAgentsFocus() && m.repoPane != nil && m.repoPane.IsActive()

			if !isNavKey && !agentsPaneActive {
				// Not a navigation key - handle as chat input
				if msg.Type == tea.KeyEnter && !msg.Alt {
					// User pressed Enter - start session creation
					prompt := m.chatInput.GetValue()
					agents := m.chatInput.GetSelectedAgents()

					if strings.TrimSpace(prompt) == "" {
						// Empty prompt - show error toast
						toastCmd := m.toast.Show("Please enter a prompt", 0)
						return m, toastCmd
					}

					// Start branch name generation (show toast)
					m.generatingBranchName = true
					toastCmd := m.toast.Show("Generating branch name...", 0)
					branchNameCmd := func() tea.Msg {
						branchName, err := session.GenerateBranchNameFromPrompt(prompt, agents[0])
						return branchNameGeneratedMsg{branchName: branchName, err: err}
					}
					return m, tea.Batch(toastCmd, branchNameCmd)
				} else if msg.Type == tea.KeyEscape {
					// Cancel new session input
					if len(m.sessionManager.ListSessions()) > 0 {
						m.showNewSessionInput = false
						m.chatInput.Reset()
					}
					return m, nil
				} else {
					// Update chat input
					cmd := m.chatInput.Update(msg)
					return m, cmd
				}
			}
			// If it IS a nav key, fall through to global handlers below
		}

		// Handle preview mode - navigation and mode switches only
		switch {
		case msg.String() == "enter":
			// Enter key handling - delegate to the active pane
			switch {
			case m.focus.IsAgentsFocus():
				// Let the repo pane handle enter key for toggling repo expansion only
				if m.repoPane != nil {
					handled, cmd := m.repoPane.HandleKey("enter")
					if handled {
						return m, cmd
					}
				}
			case m.focus.IsGitFocus():
				// Let the changes pane handle enter key for opening files
				if m.changesPane != nil {
					handled, cmd := m.changesPane.HandleKey("enter")
					if handled {
						return m, cmd
					}
				}
			case m.focus.IsTmuxFocus():
				// Enter key attaches to agent tmux session when tmux pane is focused
				if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
					// Update UI to show attached mode
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

		case msg.String() == "alt+d":
			// 'alt+d' key handling - delegate to the agents pane for session deletion
			if m.focus.IsAgentsFocus() && m.repoPane != nil {
				handled, cmd := m.repoPane.HandleKey("alt+d")
				if handled {
					return m, cmd
				}
			}
			// Also handle 'alt+d' in changes pane for discarding files
			if m.focus.IsGitFocus() && m.changesPane != nil {
				handled, cmd := m.changesPane.HandleKey("alt+d")
				if handled {
					// Refresh changes pane after discard
					m.updateChangesPane()
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

		case key.Matches(msg, common.GlobalKeys.DebugOverlay):
			// Show debug overlay
			m.showDebugOverlay = true
			return m, nil

		case key.Matches(msg, common.GlobalKeys.NewSession):
			// Step 5.5: 'n' key shows new chat input interface
			m.showNewSessionInput = true
			var focusCmd tea.Cmd
			if m.chatInput != nil {
				m.chatInput.Reset()
				focusCmd = m.chatInput.Focus()
			}
			// Mark all panes as inactive when entering new session view
			if m.repoPane != nil {
				m.repoPane.SetActive(false)
			}
			if m.sessionViewPane != nil {
				m.sessionViewPane.SetActive(false)
			}
			if m.changesPane != nil {
				m.changesPane.SetActive(false)
			}
			return m, combineCmds(focusCmd)

		case key.Matches(msg, common.GlobalKeys.AttachAgent):
			// Attach to agent tmux session (global shortcut)
			if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
				m.shortcutOverlay.SetMode("attached")
				detachCh, err := currentTmux.Attach()
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
				<-detachCh
				return m.Update(tmuxDetachedMsg{})
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.MergeChanges):
			// Show merge overlay (global shortcut)
			activeSession := m.sessionManager.GetActiveSession()
			if activeSession != nil && activeSession.Worktree() != nil {
				// Check if there are any changes to merge
				fileStatus := git.GetFileStatuses(activeSession.Worktree().Path)
				if fileStatus == nil || fileStatus.IsClean {
					// No changes to merge - show toast
					toastCmd := m.toast.Show("No changes to merge", 0)
					return m, toastCmd
				}

				// There are changes - show merge overlay
				m.mergeOverlay = overlays.NewMergeOverlay(activeSession)
				m.mergeOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
				m.showMergeOverlay = true
				initCmd := m.mergeOverlay.Init()
				return m, initCmd
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Up):
			// Navigate up in focused pane
			switch {
			case m.focus.IsAgentsFocus():
				if m.repoPane != nil {
					// Use HandleKey to ensure search exit logic runs
					handled, cmd := m.repoPane.HandleKey("up")
					if handled {
						// Update changes pane after navigation
						return m, tea.Batch(cmd, m.updateChangesPane())
					}
				}
			case m.focus.IsGitFocus():
				if m.changesPane != nil {
					handled, cmd := m.changesPane.HandleKey(msg.String())
					if handled {
						return m, cmd
					}
					return m, nil
				}
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.Down):
			// Navigate down in focused pane
			switch {
			case m.focus.IsAgentsFocus():
				if m.repoPane != nil {
					// Use HandleKey to ensure search exit logic runs
					handled, cmd := m.repoPane.HandleKey("down")
					if handled {
						// Update changes pane after navigation
						return m, tea.Batch(cmd, m.updateChangesPane())
					}
				}
			case m.focus.IsGitFocus():
				if m.changesPane != nil {
					handled, cmd := m.changesPane.HandleKey(msg.String())
					if handled {
						return m, cmd
					}
					return m, nil
				}
			}
			return m, nil

			// OpenInEditor is now handled by GitPane's HandleKey method directly
			// No separate global keybinding needed

		case key.Matches(msg, common.GlobalKeys.AgentsPane):
			// Alt+S jumps to sessions pane
			return m.switchToPane(layout.NewAgentsFocus())

		case key.Matches(msg, common.GlobalKeys.SessionPane):
			// Alt+A jumps to agents pane (if session exists)
			if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
				m.showNewSessionInput = false
				return m.switchToPane(layout.NewSessionFocus(layout.SubPaneTmux))
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.ChangesPane):
			// Alt+C jumps to changes pane (if session exists)
			if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
				m.showNewSessionInput = false
				return m.switchToPane(layout.NewSessionFocus(layout.SubPaneGit))
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.NextAgent):
			// Tab toggles sub-panes in changes pane, or cycles agents in session pane
			// Check git focus first since it's also PaneTypeSession
			if m.focus.IsGitFocus() && m.changesPane != nil {
				handled, cmd := m.changesPane.HandleKey("tab")
				if handled {
					return m, cmd
				}
			} else if m.focus.PaneType == layout.PaneTypeSession {
				// Cycle to next agent in session pane
				return m, m.cycleNextAgent()
			}
			return m, nil

		case key.Matches(msg, common.GlobalKeys.PrevAgent):
			// Shift+Tab cycles to previous agent (only on session pane)
			if m.focus.PaneType == layout.PaneTypeSession {
				// TODO: Implement backwards agent cycling
				// For now, do nothing
			}
			return m, nil

		default:
			// If agents pane is focused, try HandleKey first for navigation/special keys
			// then fall back to Update for search input
			if m.focus.IsAgentsFocus() && m.repoPane != nil && m.repoPane.IsActive() {
				// Try HandleKey first (handles navigation and search exit logic)
				keyStr := msg.String()
				debug.DebugLog("main.Update default: keyStr=%q, calling HandleKey", keyStr)
				handled, cmd := m.repoPane.HandleKey(keyStr)
				debug.DebugLog("main.Update default: HandleKey returned handled=%v", handled)
				if handled {
					return m, cmd
				}
				// If not handled by HandleKey, pass to Update for search input
				m.repoPane, cmd = m.repoPane.Update(msg)
				return m, cmd
			}
			// Handle other key combinations if needed
		}

	case tea.MouseMsg:
		// Allow changes pane to react to mouse events regardless of focus
		if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
			if handled, cmd := changesPane.HandleMouse(msg); handled {
				return m, cmd
			}
		}

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

	if m.sessionManager == nil {
		return "No session manager"
	}

	var panesWithPadding string
	var (
		newSessionPaneLeft   int
		newSessionPaneTop    int
		newSessionPaneWidth  int
		newSessionPaneHeight int
		hasNewSessionPane    bool
	)

	// Step 5.4: Handle different UI states

	// State 1: Showing new session input (no sessions or user pressed 'n')
	if m.showNewSessionInput {
		// Calculate layout dimensions
		chromeHeight := layout.TopPaddingRows + layout.BottomSpacerRows + layout.PaneTitleRows + layout.BottomMarginRows
		availableHeight := m.layout.GetHeight() - chromeHeight
		leftWidth, leftHeight := m.layout.GetLeftDimensions()
		frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
		contentPaddingWidth := components.PaneContentHorizontalPadding() * 2

		// Render agents pane on left (may be empty or show existing sessions)
		// When in new session input view, no panes should be highlighted/active
		agentsContent := m.repoPane.View()
		agentsStyle := components.PaneBaseStyle
		// Don't highlight agents pane in new session input view

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
			Height(availableHeight - 1).
			Render(agentsContentAligned)

		// Render center pane with header (ASCII art + version + shortcuts) + chat input
		totalHorizontalMargins := layout.HorizontalMargin*2 + layout.HorizontalGapWidth*2
		usableWidth := m.layout.GetWidth() - totalHorizontalMargins
		centerWidth := usableWidth - leftWidth - contentPaddingWidth - frameHeight

		// Render header with ASCII art, version, and shortcuts
		if m.welcomeHeader != nil {
			m.welcomeHeader.SetWidth(centerWidth - 4)
		}
		headerView := ""
		if m.welcomeHeader != nil {
			headerView = m.welcomeHeader.View()
		}

		// Render chat input centered in middle
		if m.chatInput != nil {
			m.chatInput.SetWidth(min(80, centerWidth-8))
		}
		chatInputView := ""
		if m.chatInput != nil {
			chatInputView = m.chatInput.View()
		}

		// Combine header + spacing + chat input
		var centerParts []string
		centerParts = append(centerParts, headerView)
		centerParts = append(centerParts, "")
		centerParts = append(centerParts, "")
		centerParts = append(centerParts, chatInputView)

		centerContent := strings.Join(centerParts, "\n")

		// Center the content vertically and horizontally
		placeholderContentHeight := availableHeight - frameHeight
		centeredContent := lipgloss.Place(
			centerWidth,
			placeholderContentHeight,
			lipgloss.Center,
			lipgloss.Center,
			centerContent,
		)

		// Render center pane with border (highlighted since it's the active/focused pane)
		centerPaneStyle := components.PaneBaseStyle.
			BorderForeground(lipgloss.Color(theme.BorderActive))
		centerPane := centerPaneStyle.
			Height(availableHeight - 1).
			Render(centeredContent)

		// Render pane titles with padding
		leftTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.repoPane))
		// For new session input, show "New Session" as the title
		titleText := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Render("New Session")
		shortcut := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)).
			Render("(⌥n)")
		centerTitle := lipgloss.NewStyle().PaddingLeft(1).Render(titleText + " " + shortcut)

		// Join titles with panes vertically (title above pane)
		leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitle, agentsPane)
		centerWithTitle := lipgloss.JoinVertical(lipgloss.Left, centerTitle, centerPane)

		// Track bounds of the new session pane so overlays can align to it later
		hasNewSessionPane = true
		leftBlockWidth := lipgloss.Width(leftWithTitle)
		centerBlockWidth := lipgloss.Width(centerWithTitle)
		newSessionPaneLeft = layout.HorizontalMargin + leftBlockWidth + layout.HorizontalGapWidth
		newSessionPaneTop = layout.TopPaddingRows + layout.PaneTitleRows
		newSessionPaneWidth = centerBlockWidth
		newSessionPaneHeight = lipgloss.Height(centerPane)

		// Join agents pane and center pane (right pane hidden)
		gap := lipgloss.NewStyle().Width(layout.HorizontalGapWidth).Render("")
		panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, centerWithTitle)

		// Add padding
		panesWithPadding = lipgloss.NewStyle().
			PaddingTop(layout.TopPaddingRows).
			PaddingBottom(layout.BottomSpacerRows).
			PaddingLeft(layout.HorizontalMargin).
			PaddingRight(layout.HorizontalMargin).
			Render(panes)

	} else {
		// State 2: Active session - always render 3-pane layout: Agents | Session | Changes

		// Render pane titles with padding
		leftTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.repoPane))
		centerTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.sessionViewPane))
		rightTitle := lipgloss.NewStyle().PaddingLeft(1).Render(m.renderPaneTitle(m.changesPane))

		// Render pane content
		agentsContent := m.repoPane.View()
		sessionContent := m.sessionViewPane.View()
		changesContent := m.changesPane.View()

		defaultPadding := components.PaneContentVerticalPadding()

		leftPadTop, leftPadBottom := defaultPadding, defaultPadding
		if m.repoPane != nil {
			leftPadTop, leftPadBottom = m.repoPane.GetChromePadding()
		}

		centerPadTop, centerPadBottom := defaultPadding, defaultPadding
		if m.sessionViewPane != nil {
			centerPadTop, centerPadBottom = m.sessionViewPane.GetChromePadding()
		}

		rightPadTop, rightPadBottom := defaultPadding, defaultPadding
		if m.changesPane != nil {
			rightPadTop, rightPadBottom = m.changesPane.GetChromePadding()
		}

		// Use layout's RenderPanes to render the 3-column layout
		leftPane, tmuxPane, gitPane := m.layout.RenderPanes(
			layout.PaneRenderParams{
				Content:       agentsContent,
				PaddingTop:    leftPadTop,
				PaddingBottom: leftPadBottom,
			},
			layout.PaneRenderParams{
				Content:       sessionContent,
				PaddingTop:    centerPadTop,
				PaddingBottom: centerPadBottom,
			},
			layout.PaneRenderParams{
				Content:       changesContent,
				PaddingTop:    rightPadTop,
				PaddingBottom: rightPadBottom,
			},
			m.focus,
			false, // isLoading - handled by SessionViewPane internally
			nil,   // loadingState - not needed since SessionViewPane handles it
		)

		// Join titles with panes vertically (title above pane)
		leftWithTitle := lipgloss.JoinVertical(lipgloss.Left, leftTitle, leftPane)
		tmuxWithTitle := lipgloss.JoinVertical(lipgloss.Left, centerTitle, tmuxPane)
		gitWithTitle := lipgloss.JoinVertical(lipgloss.Left, rightTitle, gitPane)

		// Join the three panes horizontally
		gap := lipgloss.NewStyle().Width(layout.HorizontalGapWidth).Render("")
		panes := lipgloss.JoinHorizontal(lipgloss.Top, leftWithTitle, gap, tmuxWithTitle, gap, gitWithTitle)

		// Add padding
		panesWithPadding = lipgloss.NewStyle().
			PaddingTop(layout.TopPaddingRows).
			PaddingBottom(layout.BottomSpacerRows).
			PaddingLeft(layout.HorizontalMargin).
			PaddingRight(layout.HorizontalMargin).
			Render(panes)
	}

	// Add bottom margin
	var bottomComponents []string
	bottomComponents = append(bottomComponents, panesWithPadding)
	for i := 0; i < layout.BottomMarginRows; i++ {
		bottomComponents = append(bottomComponents, "")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, bottomComponents...)

	// Overlay toast notifications if visible
	if m.toast != nil && m.toast.IsVisible() {
		mainView = m.toast.PlaceOverlay(mainView, m.layout.GetWidth(), m.layout.GetHeight())
	}

	// If help dialog is visible, overlay it
	if m.showHelp {
		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.helpDialog.View(), mainView, true, true))
	}

	// If debug overlay is visible, overlay it (high priority)
	if m.showDebugOverlay && m.debugOverlay != nil {
		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.debugOverlay.View(), mainView, true, true))
	}

	// TODO: Old worktree dialog removed, replaced with chat input

	// If worktree deletion confirmation is visible, overlay it
	if m.showWorktreeConfirm && m.worktreeConfirm != nil {
		// Update dialog size
		m.worktreeConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.worktreeConfirm.View(), mainView, true, true))
	}

	// If session deletion confirmation is visible, overlay it
	if m.showSessionConfirm && m.sessionConfirm != nil {
		// Update dialog size
		m.sessionConfirm.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.sessionConfirm.View(), mainView, true, true))
	}

	// If merge overlay is visible, overlay it
	if m.showMergeOverlay && m.mergeOverlay != nil {
		// Update dialog size
		m.mergeOverlay.SetSize(m.layout.GetWidth(), m.layout.GetHeight())

		// Use Claude Squad's overlay implementation
		return zone.Scan(overlay.PlaceOverlay(0, 0, m.mergeOverlay.View(), mainView, true, true))
	}

	// If agent selector is visible, overlay it
	if m.showAgentSelector && m.agentSelector != nil {
		selectorWidth := m.layout.GetWidth()
		selectorHeight := m.layout.GetHeight()

		if m.showNewSessionInput && hasNewSessionPane && newSessionPaneWidth > 0 && newSessionPaneHeight > 0 {
			selectorWidth = newSessionPaneWidth
			selectorHeight = newSessionPaneHeight
		}

		m.agentSelector.SetSize(selectorWidth, selectorHeight)
		selectorView := m.agentSelector.View()

		if m.showNewSessionInput && hasNewSessionPane && newSessionPaneWidth > 0 && newSessionPaneHeight > 0 {
			overlayWidth := lipgloss.Width(selectorView)
			overlayHeight := lipgloss.Height(selectorView)

			overlayX := newSessionPaneLeft
			if overlayWidth < newSessionPaneWidth {
				overlayX += (newSessionPaneWidth - overlayWidth) / 2
			}

			overlayY := newSessionPaneTop
			if overlayHeight < newSessionPaneHeight {
				overlayY += (newSessionPaneHeight - overlayHeight) / 2
			}

			return zone.Scan(overlay.PlaceOverlay(overlayX, overlayY, selectorView, mainView, true, false))
		}

		// Default to centering on the entire screen
		return zone.Scan(overlay.PlaceOverlay(0, 0, selectorView, mainView, true, true))
	}

	// Render toast notifications (always rendered last, on top of everything)
	// Toasts do NOT dim the background and do NOT block interaction
	if m.toast != nil && m.toast.IsVisible() {
		return zone.Scan(m.toast.PlaceOverlay(mainView, m.layout.GetWidth(), m.layout.GetHeight()))
	}

	return zone.Scan(mainView)
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

	// Track TUI started
	telemetry.TrackTUIStarted()

	p := tea.NewProgram(initialModel(subprocess), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %v", err)
	}
	return nil
}

func main() {
	zone.NewGlobal()

	// Initialize analytics (disabled for dev builds)
	apiKey := os.Getenv("POSTHOG_API_KEY")
	if err := telemetry.Init(apiKey, version.Short()); err != nil {
		debug.DebugLog("Failed to initialize analytics: %v", err)
		// Continue - app works without analytics
	}
	defer telemetry.Close()

	// Get distinct ID and identify user with environment properties
	if distinctID, err := telemetry.GetDistinctId(); err == nil {
		telemetry.SetDistinctId(distinctID)
		envProps := telemetry.GetEnvironmentProperties(version.Short())
		if err := telemetry.Identify(distinctID, envProps); err != nil {
			debug.DebugLog("Failed to identify user: %v", err)
		}
	}

	var showVersion bool
	var agentsFlag string

	var rootCmd = &cobra.Command{
		Use:   "agate",
		Short: "A tmux-based terminal UI for AI agents",
		Long: `Agate provides a split-pane terminal interface for interacting with AI agents.

Supports any agent name (claude, codex, gemini, etc.) and automatically configures
colors and settings based on the agent type.

Agate provides two interaction modes:
  Preview Mode (default): Read-only view with fast, lag-free rendering
  Attached Mode (a): Full tmux experience with complete control

Press 'a' when focused on the right pane to attach to tmux.
Press Ctrl+Q when attached to detach back to preview.
Press ? for help once running.

Examples:
  agate                                           # Launch with last selected agents (defaults to claude,codex)
  agate -a claude                                 # Launch with Claude
  agate --agents claude,codex                     # Launch with Claude and Codex
  agate --agents claude,codex "add a new feature" # Create session with prompt and attach directly`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(version.Short())
				return nil
			}

			// If a prompt is provided, create session directly and attach
			if len(args) > 0 {
				prompt := args[0]
				return newSessionFromCLI(agentsFlag, prompt)
			}

			// Otherwise, launch TUI
			return runAgent(agentsFlag)
		},
	}

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	rootCmd.Flags().StringVarP(&agentsFlag, "agents", "a", "", "Comma-separated list of agents (e.g., claude,codex)")

	// Metrics subcommand
	var metricsSessionID string
	var metricsCmd = &cobra.Command{
		Use:   "metrics",
		Short: "View live metrics for a session",
		Long:  `Display a live metrics view for the specified session ID.`,
		RunE: func(_ *cobra.Command, args []string) error {
			if metricsSessionID == "" {
				return fmt.Errorf("--session-id is required")
			}

			// Initialize state manager to load session
			stateManager, err := state.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize state manager: %w", err)
			}

			// Load sessions from state
			sessionMappings := stateManager.GetSessionMappings()
			persistedSession, exists := sessionMappings[metricsSessionID]
			if !exists {
				return fmt.Errorf("session not found: %s", metricsSessionID)
			}

			// Reconstruct session object from persisted state
			sess := &session.Session{
				ID:             persistedSession.ID,
				Prompt:         persistedSession.Prompt,
				Description:    persistedSession.Description,
				BranchBaseName: persistedSession.BranchBaseName,
				CreatedAt:      persistedSession.CreatedAt,
				LastAccessed:   persistedSession.LastAccessed,
			}

			// Run metrics TUI
			return metrics.Run(sess)
		},
	}
	metricsCmd.Flags().StringVar(&metricsSessionID, "session-id", "", "Session ID to view metrics for (required)")

	rootCmd.AddCommand(metricsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
