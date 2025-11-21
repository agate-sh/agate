package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/common"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"
	"agate/pkg/tmux"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/overlays"
	"agate/pkg/tui/panes"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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
	shortcutOverlay    *common.ShortcutOverlay // Manages contextual shortcuts
	loadingState       *tmux.LoadingState      // Loading state manager with spinner and stopwatch
	toast              *components.Toast       // Toast notification manager
	descriptionSpinner spinner.Model           // Spinner for description generation animation

	// Overlays
	helpDialog     *overlays.HelpDialog                 // Help dialog overlay
	debugOverlay   *overlays.DebugOverlay               // Debug overlay for development
	sessionConfirm *overlays.SessionDeleteConfirmDialog // Session deletion confirmation
	mergeOverlay   *overlays.MergeOverlay               // Merge overlay for merging changes
	agentSelector  *components.AgentSelector            // Agent selector modal

	// Managers
	debugLogger *debug.DebugLogger // Debug logger for development

	// New session creation flow
	chatInput             *components.ChatInput     // Chat input component for new session creation
	welcomeHeader         *components.WelcomeHeader // Header with ASCII art and version
	globalFooter          *components.GlobalFooter  // Global footer with keyboard shortcuts
	creatingSession       bool                      // true during session creation (shows toast)
	generatingDescription bool                      // true while description is being generated in background
	initialPrompt         string                    // Initial prompt from CLI (auto-creates session if non-empty)
	initialTermWidth      int                       // Terminal width when TUI started (for auto-created sessions)
	initialTermHeight     int                       // Terminal height when TUI started (for auto-created sessions)

	// Panes using the new Pane interface
	repoPane        components.Pane // Agents pane (list of sessions)
	sessionViewPane components.Pane // SessionViewPane instance (header + tmux)
	changesPane     components.Pane // ChangesPane instance (file changes)

	// Session switch debouncing
	pendingSessionSwitch *session.Session // Session waiting to be switched to
	switchDebounceTimer  *time.Timer      // Timer for debouncing session switches

	// Keyboard handling helpers
	pendingAltRune string // Non-alt rune to suppress after handling an alt shortcut
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

// SetOverlay sets the active overlay (mode stays unchanged)
func (m *Model) SetOverlay(overlay OverlayType) {
	m.state.ActiveOverlay = overlay
}

// ClearOverlay clears the active overlay (mode stays unchanged)
func (m *Model) ClearOverlay() {
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

// Helper methods for internal operations

// switchToPane switches focus to a target pane and updates UI state
func (m *Model) switchToPane(targetPane layout.FocusState) (*Model, tea.Cmd) {
	debug.DebugLog("[switchToPane] called with targetPane=%s, current mode=%v", targetPane.String(), m.state.Mode)
	// Update all panes' active state
	if m.repoPane != nil {
		m.repoPane.SetActive(targetPane.IsSessionsFocus())
	}
	// Session view pane is active only when we're on the tmux sub-pane
	if m.sessionViewPane != nil {
		m.sessionViewPane.SetActive(targetPane.IsTmuxFocus())
	}
	if m.changesPane != nil {
		m.changesPane.SetActive(targetPane.IsGitFocus())
	}

	// Set the new focus
	m.state.Focus = targetPane

	// Update shortcut overlay
	m.shortcutOverlay.SetFocus(m.state.Focus.String())

	// Handle chat input focus in new session mode
	var chatInputCmd tea.Cmd
	if m.state.Mode == ModeNewSession && m.chatInput != nil {
		if targetPane.IsSessionsFocus() {
			// Blur chat input when sessions pane is focused
			m.chatInput.Blur()
		} else {
			// Focus chat input when switching away from sessions pane
			chatInputCmd = m.chatInput.Focus()
		}
	}

	// Special handling for sessions pane
	if targetPane.IsSessionsFocus() && m.repoPane != nil {
		// Refresh repo pane when focusing
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			if err := repoPane.Refresh(); err != nil {
				debug.DebugLog("Failed to refresh repo pane when switching panes: %v", err)
				// UI continues to work, but log the issue for debugging
			}
		}

		// The repoPane has a search input that needs cursor blinking
		// When we switch to it, we need to trigger the initial blink
		// We'll pass the Update through to execute the pending focus command
		var repoPaneCmd tea.Cmd
		if repoPane, ok := m.repoPane.(*panes.AgentsPane); ok {
			// Call Update with a dummy message to trigger pendingFocusCmd execution
			m.repoPane, repoPaneCmd = repoPane.Update(tea.WindowSizeMsg{})
		}

		// Update ChangesPane with selected worktree/repo and get refresh command
		return m, tea.Batch(chatInputCmd, m.updateChangesPane(), repoPaneCmd)
	}

	return m, chatInputCmd
}

// enterNewSessionMode switches UI into new session mode and focuses chat input.
func (m *Model) enterNewSessionMode(resetInput bool) tea.Cmd {
	m.SetMode(ModeNewSession)

	if m.sessionManager != nil {
		m.sessionManager.ClearActiveSession()
	}

	// Update pane focus and shortcut overlay
	m.state.Focus = layout.NewChatInputFocus()
	m.shortcutOverlay.SetFocus(m.state.Focus.String())

	// New session view shouldn't highlight any of the panes on the main UI
	if m.repoPane != nil {
		m.repoPane.SetActive(false)
	}
	if m.sessionViewPane != nil {
		m.sessionViewPane.SetActive(false)
	}
	if m.changesPane != nil {
		m.changesPane.SetActive(false)
	}

	if m.chatInput == nil {
		return nil
	}

	if resetInput {
		m.chatInput.Reset()
	}
	return m.chatInput.Focus()
}

// startSuppressingAltRune remembers the rune produced by an alt shortcut so we
// can suppress the duplicate non-alt key the terminal might emit afterwards.
func (m *Model) startSuppressingAltRune(msg tea.KeyMsg) {
	key := extractAltRune(msg.String())
	if key == "" || utf8.RuneCountInString(key) != 1 {
		return
	}
	m.pendingAltRune = key
}

// consumePendingAltRune returns true if the current key should be ignored
// because it's the duplicate rune emitted after an alt shortcut.
func (m *Model) consumePendingAltRune(msg tea.KeyMsg) bool {
	if m.pendingAltRune == "" {
		return false
	}
	// Any new alt combo cancels the pending rune suppression.
	if msg.Alt {
		m.pendingAltRune = ""
		return false
	}

	if msg.Type == tea.KeyRunes &&
		utf8.RuneCountInString(msg.String()) == 1 &&
		msg.String() == m.pendingAltRune {
		debug.DebugLog("[Model] Consuming stray key %q following alt shortcut", msg.String())
		m.pendingAltRune = ""
		return true
	}

	// Not the expected key - clear so we don't suppress unrelated keys later.
	m.pendingAltRune = ""
	return false
}

func extractAltRune(keyStr string) string {
	for _, prefix := range []string{"alt+", "opt+"} {
		if strings.HasPrefix(keyStr, prefix) {
			return keyStr[len(prefix):]
		}
	}
	return ""
}

// getCurrentTmuxSession returns the active tmux session from the session manager
func (m *Model) getCurrentTmuxSession() *tmux.TmuxSession {
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
func (m *Model) getDetachedTmuxSize(contentWidth, contentHeight int) (int, int) {
	if m.sessionManager == nil {
		return contentWidth, contentHeight
	}

	activeSession := m.sessionManager.GetActiveSession()
	if activeSession == nil || activeSession.SharedTmux == nil {
		return contentWidth, contentHeight
	}

	// For shared sessions with multiple agents, each agent gets a vertical tmux pane
	// The total tmux window width = contentWidth * agentCount
	agentCount := len(activeSession.Agents)
	if agentCount <= 1 {
		return contentWidth, contentHeight
	}

	return contentWidth * agentCount, contentHeight
}

// switchToSessionForWorktree switches to the session associated with a worktree
func (m *Model) switchToSessionForWorktree(worktree *git.WorktreeInfo) tea.Cmd {
	if m.sessionManager == nil || worktree == nil {
		return nil
	}

	// Find session with matching worktree
	sessions := m.sessionManager.ListSessions()
	for _, sess := range sessions {
		if sess.Worktree() != nil && sess.Worktree().Path == worktree.Path {
			// Found matching session - switch to it
			if _, err := m.sessionManager.SwitchToSession(sess.ID); err != nil {
				debug.DebugLog("Failed to switch to session %s for worktree %s: %v", sess.ID, worktree.Path, err)
				return nil
			}

			// Set the current agent based on the session's agent
			app.SetCurrentAgent(sess.Agent())

			// Update session view pane with the switched session
			if m.sessionViewPane != nil {
				if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
					sessionViewPane.SetSession(sess)
				}
			}

			// Return command to refresh tmux content
			if sess.TmuxSession() != nil {
				return waitForTmuxOutput(sess.TmuxSession())
			}
			return nil
		}
	}

	// No session found for this worktree
	debug.DebugLog("No session found for worktree %s", worktree.Path)
	return nil
}

// cycleNextAgent cycles to the next agent in a shared session
func (m *Model) cycleNextAgent() tea.Cmd {
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

	m.state.Focus = layout.NewAgentsFocus(layout.SubPaneTmux)
	if m.shortcutOverlay != nil {
		m.shortcutOverlay.SetFocus(m.state.Focus.String())
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
				if repo, err := m.sessionManager.GetRepository(); err == nil {
					changesPane.SetRepositoryAndPath(repo, activeAgent.Worktree.Path)
				}
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
				return tmuxOutputMsg{content: content}
			})

			// Resume normal monitoring after the forced capture
			cmds = append(cmds, waitForTmuxOutput(tmuxSession))
		}
	}

	return combineCmds(cmds...)
}

// updateChangesPane updates the changes pane based on the selected worktree
func (m *Model) updateChangesPane() tea.Cmd {
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

	// Cast to ChangesPane to access SetRepositoryAndPath method
	if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
		if repo, err := m.sessionManager.GetRepository(); err == nil {
			changesPane.SetRepositoryAndPath(repo, repoPath)
			debug.DebugLog("updateChangesPane: set changes pane worktree to %s", repoPath)
		}
	}

	debug.DebugLog("===== updateChangesPane returning cmd=%v =====", refreshCmd != nil)
	return refreshCmd
}

// waitForTmuxOutput creates a command that waits for tmux output
func waitForTmuxOutput(session *tmux.TmuxSession) tea.Cmd {
	return func() tea.Msg {
		// Capture tmux pane content with ANSI codes preserved
		content, err := session.CapturePaneContent()
		if err != nil {
			debug.DebugLog("[waitForTmuxOutput] ERROR capturing: %v", err)
			return tmuxOutputMsg{content: ""}
		}

		// Check if output has changed
		hasUpdated := session.HasUpdated()
		if !hasUpdated {
			// debug.DebugLog("[waitForTmuxOutput] No update, content_len=%d", len(content))
			return tmuxOutputMsg{content: ""}
		}

		debug.DebugLog("[waitForTmuxOutput] Content UPDATED! len=%d, preview=%q", len(content), truncateString(content, 100))
		// Return the raw content with ANSI codes
		return tmuxOutputMsg{content: content}
	}
}

// switchToSessionUI updates all UI components when switching to a new session
// This is the single source of truth for session UI updates
// Uses debouncing to prevent rapid session switches during fast navigation
func (m *Model) switchToSessionUI(sess *session.Session) tea.Cmd {
	if sess == nil {
		return nil
	}

	// Cancel any pending session switch
	if m.switchDebounceTimer != nil {
		m.switchDebounceTimer.Stop()
	}

	// Store the session to switch to
	m.pendingSessionSwitch = sess

	// Start a 100ms timer - if no new navigation happens, we'll switch
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return debouncedSessionSwitchMsg{session: sess}
	}
}

// performSessionSwitch actually performs the UI updates after debounce delay
func (m *Model) performSessionSwitch(sess *session.Session) tea.Cmd {
	if sess == nil {
		return nil
	}

	// Only switch if this is still the pending session (not superseded by another navigation)
	if m.pendingSessionSwitch != sess {
		debug.DebugLog("[performSessionSwitch] Skipping switch to %s (superseded)", sess.ID)
		return nil
	}

	debug.DebugLog("[performSessionSwitch] Switching UI to session: %s", sess.ID)

	// Clear pending switch
	m.pendingSessionSwitch = nil

	// Update session view pane header
	if m.sessionViewPane != nil {
		if sessionViewPane, ok := m.sessionViewPane.(*panes.SessionViewPane); ok {
			sessionViewPane.SetSession(sess)
			debug.DebugLog("[performSessionSwitch] Updated session view pane header")
		}
	}

	// Update changes pane to show the new session's worktree
	if m.changesPane != nil && sess.Worktree() != nil {
		if changesPane, ok := m.changesPane.(*panes.ChangesPane); ok {
			if repo, err := m.sessionManager.GetRepository(); err == nil {
				changesPane.SetRepositoryAndPath(repo, sess.Worktree().Path)
				debug.DebugLog("[performSessionSwitch] Updated changes pane for worktree: %s", sess.Worktree().Path)
			}
		}
	}

	// Force a tmux content refresh for the newly selected session
	if currentTmux := m.getCurrentTmuxSession(); currentTmux != nil {
		debug.DebugLog("[performSessionSwitch] Refreshing tmux content")
		return waitForTmuxOutput(currentTmux)
	}

	debug.DebugLog("[performSessionSwitch] No tmux session to refresh")
	return nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// combineCmds combines multiple commands into one, filtering out nils
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
