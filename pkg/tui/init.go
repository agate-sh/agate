package tui

import (
	"fmt"
	"strings"

	"agate/internal/debug"
	"agate/pkg/agents"
	"agate/pkg/app"
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"
	"agate/pkg/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// InitConfig holds configuration for initializing the TUI
type InitConfig struct {
	Subprocess        string
	InitialPrompt     string
	TermWidth         int
	TermHeight        int
	DebugLogger       *debug.DebugLogger
	StateManager      *state.Manager
	WorktreeManager   *git.WorktreeManager
	SessionManager    *session.Manager
	PreCreatedSession *session.Session
}

// NewModelWithConfig creates a new TUI model with the given configuration
func NewModelWithConfig(cfg InitConfig) *Model {
	// Use passed-in managers if provided, otherwise create new ones
	var debugLogger = cfg.DebugLogger
	var stateManager = cfg.StateManager
	var worktreeManager = cfg.WorktreeManager
	var sessionManager = cfg.SessionManager

	if debugLogger == nil {
		debugLogger = debug.InitDebugLogger()
	}

	if stateManager == nil {
		var err error
		stateManager, err = state.NewManager()
		if err != nil {
			fmt.Printf("ERROR: failed to initialize state manager: %v\n", err)
			fmt.Printf("Agate will run without session persistence. Please check ~/.agate permissions.\n")
			debug.DebugLog("ERROR: StateManager initialization failed: %v", err)
		}
	}

	if worktreeManager == nil {
		var err error
		worktreeManager, err = git.NewWorktreeManager()
		if err != nil {
			fmt.Printf("Warning: failed to initialize worktree manager: %v\n", err)
		}
	}

	if sessionManager == nil {
		sessionManager = session.NewManager(worktreeManager, stateManager)
		if err := sessionManager.RestoreSessions(); err != nil {
			debug.DebugLog("Failed to restore sessions on startup: %v", err)
		}
	}

	// Set up debug logging for git package
	git.DebugLog = debug.DebugLog

	// Resolve which agents to use for initialization
	agentNames := resolveAgentNames(cfg.Subprocess, stateManager)

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
	shortcutOverlay.SetFocus(initialFocus.String())
	shortcutOverlay.SetMode("preview")

	// Initialize worktree components
	var worktreeList *overlays.WorktreeList
	if worktreeManager != nil {
		worktreeList = overlays.NewWorktreeList(worktreeManager)
	}

	// Initialize debug overlay
	debugOverlay := overlays.NewDebugOverlay(debugLogger)

	// Create shared loading state
	loadingState := tmux.NewLoadingState()

	// Initialize all panes using the new Pane interface
	repoPane := panes.NewAgentsPane(sessionManager)
	sessionViewPane := panes.NewSessionViewPane()
	changesPane := panes.NewChangesPane()

	// Initialize chat input with resolved agents
	chatInput := components.NewChatInput(agentConfigs[0])
	chatInput.SetSelectedAgents(agentConfigs)

	// Initialize welcome header
	welcomeHeader := components.NewWelcomeHeader()

	// Show new session input only if we don't have a pre-created session
	showNewSessionInput := cfg.PreCreatedSession == nil && cfg.InitialPrompt == ""

	// Use provided terminal dimensions, or fallback to 0x0 (will be updated by WindowSizeMsg)
	layoutWidth := cfg.TermWidth
	layoutHeight := cfg.TermHeight
	if layoutWidth == 0 || layoutHeight == 0 {
		layoutWidth, layoutHeight = 0, 0
	}

	// Determine initial mode
	initialMode := ModeSession
	if showNewSessionInput {
		initialMode = ModeNewSession
	}

	m := &Model{
		state: UIState{
			Mode:          initialMode,
			ActiveOverlay: NoOverlay,
			Focus:         layout.NewAgentsFocus(), // Always start with focus on Agents pane
		},
		layout:                layout.NewLayout(layoutWidth, layoutHeight),
		sessionManager:        sessionManager,
		stateManager:          stateManager,
		ready:                 true,
		subprocess:            cfg.Subprocess,
		mode:                  modePreview,
		shortcutOverlay:       shortcutOverlay,
		loadingState:          loadingState,
		toast:                 components.NewToast(),
		helpDialog:            overlays.NewHelpDialog(common.GlobalKeys),
		debugOverlay:          debugOverlay,
		worktreeList:          worktreeList,
		sessionConfirm:        nil,
		mergeOverlay:          nil,
		agentSelector:         nil,
		worktreeManager:       worktreeManager,
		debugLogger:           debugLogger,
		chatInput:             chatInput,
		welcomeHeader:         welcomeHeader,
		creatingSession:       false,
		generatingDescription: false,
		initialPrompt:         cfg.InitialPrompt,
		initialTermWidth:      cfg.TermWidth,
		initialTermHeight:     cfg.TermHeight,
		repoPane:              repoPane,
		sessionViewPane:       sessionViewPane,
		changesPane:           changesPane,
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

	// If we have a pre-created session, set it up now
	if cfg.PreCreatedSession != nil {
		// Set as active session
		if m.sessionManager != nil {
			if _, err := m.sessionManager.SwitchToSession(cfg.PreCreatedSession.ID); err != nil {
				debug.DebugLog("Failed to switch to pre-created session %s: %v", cfg.PreCreatedSession.ID, err)
			}
		}
		app.SetCurrentAgent(cfg.PreCreatedSession.Agent())

		// Update session view pane
		if m.sessionViewPane != nil {
			if sessionView, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
				sessionView.SetSession(cfg.PreCreatedSession)
			}
		}

		// Refresh repo pane to show the new session
		if m.repoPane != nil {
			if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
				if err := repoPane.Refresh(); err != nil {
					debug.DebugLog("Failed to refresh repo pane with pre-created session: %v", err)
				}
			}
		}

		// Update changes pane
		m.updateChangesPane()

		// Set panes to active since we're showing the 3-pane layout
		if m.repoPane != nil {
			m.repoPane.SetActive(true)
		}
		if m.sessionViewPane != nil {
			m.sessionViewPane.SetActive(false) // Agents pane is initially focused
		}
		if m.changesPane != nil {
			m.changesPane.SetActive(false)
		}
	}

	return m
}

// Init initializes the TUI model and returns initial commands
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Check if we have an active session (from pre-creation via cfg.PreCreatedSession)
	hasActiveSession := m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil

	// If no initial prompt was provided, ALWAYS start in new session mode
	// This ensures running just `./agate` shows the welcome screen with chat input
	if m.initialPrompt == "" {
		m.SetMode(ModeNewSession)
		if m.chatInput != nil {
			if initCmd := m.chatInput.InitCmd(); initCmd != nil {
				cmds = append(cmds, initCmd)
			}
		}
	}

	// Auto-create session if initial prompt was provided but no session exists
	if m.initialPrompt != "" && !hasActiveSession {
		// Generate deterministic branch name
		repoName := ""
		if m.worktreeManager != nil {
			repoName = m.worktreeManager.GetRepositoryName()
		}
		branchName := session.GenerateBranchNameFromPrompt(repoName)

		// Trigger session creation
		cmds = append(cmds, func() tea.Msg {
			return branchNameGeneratedMsg{
				branchName: branchName,
				prompt:     m.initialPrompt,
			}
		})
	}

	// If we have a pre-created session (from main.go when running `./agate "prompt"`),
	// trigger sessionCreatedMsg to start monitoring.
	// This ensures both pre-created and TUI-created sessions share the same code path.
	if hasActiveSession && m.initialPrompt != "" {
		activeSession := m.sessionManager.GetActiveSession()
		if activeSession != nil {
			debug.DebugLog("[ISSUE1-PRECREATED] Triggering sessionCreatedMsg for pre-created session %s", activeSession.ID)
			cmds = append(cmds, func() tea.Msg {
				return sessionCreatedMsg{session: activeSession, err: nil}
			})
		}
	}

	return tea.Batch(cmds...)
}

// resolveAgentNames determines which agents to use based on subprocess and state
func resolveAgentNames(subprocess string, stateManager *state.Manager) []string {
	// If subprocess is specified, use it
	if subprocess != "" {
		// Split by comma for multiple agents
		names := make([]string, 0)
		parts := strings.Split(subprocess, ",")
		for _, name := range parts {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			return names
		}
	}

	// Otherwise, try to get from state manager
	if stateManager != nil {
		selectedAgents := stateManager.GetSelectedAgents()
		if len(selectedAgents) > 0 {
			return selectedAgents
		}
	}

	// Fall back to default agent
	return []string{"claude"}
}
