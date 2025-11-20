package tui

import (
	"agate/internal/debug"
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/overlays"
	"agate/pkg/tmux"
)

// sessionMode represents the interaction mode for a session
type sessionMode int

const (
	modePreview sessionMode = iota // Read-only preview
)

// Model represents the TUI model
type Model struct {
	// State machine (replaces 7 boolean flags)
	state UIState

	// Layout and managers
	layout         *layout.Layout   // Layout manager for pane dimensions
	sessionManager *session.Manager // Session manager for all worktree/tmux coordination
	stateManager   *state.Manager   // Thread-safe state manager
	ready          bool
	err            error

	// Session configuration
	subprocess string      // Command to run in tmux pane
	mode       sessionMode // Current interaction mode

	// UI components
	shortcutOverlay *common.ShortcutOverlay // Manages contextual shortcuts
	loadingState    *tmux.LoadingState      // Loading state manager with spinner and stopwatch
	toast           *components.Toast       // Toast notification manager

	// Overlays
	helpDialog    *overlays.HelpDialog                 // Help dialog overlay
	debugOverlay  *overlays.DebugOverlay               // Debug overlay for development
	worktreeList  *overlays.WorktreeList               // Worktree list component
	sessionConfirm *overlays.SessionDeleteConfirmDialog // Session deletion confirmation
	mergeOverlay  *overlays.MergeOverlay               // Merge overlay for merging changes
	agentSelector *components.AgentSelector            // Agent selector modal

	// Managers
	worktreeManager *git.WorktreeManager // Git worktree management
	debugLogger     *debug.DebugLogger   // Debug logger for development

	// New session creation flow
	chatInput             *components.ChatInput     // Chat input component for new session creation
	welcomeHeader         *components.WelcomeHeader // Header with ASCII art, version, and shortcuts
	creatingSession       bool                      // true during session creation (shows toast)
	generatingDescription bool                      // true while description is being generated in background
	initialPrompt         string                    // Initial prompt from CLI (auto-creates session if non-empty)
	initialTermWidth      int                       // Terminal width when TUI started (for auto-created sessions)
	initialTermHeight     int                       // Terminal height when TUI started (for auto-created sessions)

	// Panes using the new Pane interface
	repoPane        components.Pane // Agents pane (list of sessions)
	sessionViewPane components.Pane // SessionViewPane instance (header + tmux)
	changesPane     components.Pane // ChangesPane instance (file changes)
}

// New creates a new TUI model with default state
func New() *Model {
	return &Model{
		state: UIState{
			Mode:          ModeSession,
			ActiveOverlay: NoOverlay,
			Focus:         layout.FocusState{}, // Will be initialized properly later
		},
	}
}

// State returns the current UI state
func (m *Model) State() *UIState {
	return &m.state
}

// SetMode sets the current UI mode
func (m *Model) SetMode(mode UIMode) {
	m.state.Mode = mode
}

// SetOverlay sets the active overlay (and switches to overlay mode)
func (m *Model) SetOverlay(overlay OverlayType) {
	if overlay != NoOverlay {
		m.state.Mode = ModeOverlay
	} else {
		m.state.Mode = ModeSession
	}
	m.state.ActiveOverlay = overlay
}

// ClearOverlay clears the active overlay and returns to session mode
func (m *Model) ClearOverlay() {
	m.state.Mode = ModeSession
	m.state.ActiveOverlay = NoOverlay
}

// ShowNewSessionInput returns true if showing new session input
func (m *Model) ShowNewSessionInput() bool {
	return m.state.InNewSession()
}

// ShowHelp returns true if help overlay is visible
func (m *Model) ShowHelp() bool {
	return m.state.HasOverlay(HelpOverlay)
}

// ShowDebugOverlay returns true if debug overlay is visible
func (m *Model) ShowDebugOverlay() bool {
	return m.state.HasOverlay(DebugOverlay)
}

// ShowSessionConfirm returns true if session confirmation is visible
func (m *Model) ShowSessionConfirm() bool {
	return m.state.HasOverlay(SessionDeleteOverlay)
}

// ShowMergeOverlay returns true if merge overlay is visible
func (m *Model) ShowMergeOverlay() bool {
	return m.state.HasOverlay(MergeOverlay)
}

// ShowAgentSelector returns true if agent selector is visible
func (m *Model) ShowAgentSelector() bool {
	return m.state.HasOverlay(AgentSelectorOverlay)
}
