package tui

import (
	"fmt"
	"strings"

	"agate/internal/debug"
	"agate/pkg/common"
	"agate/pkg/session"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"
	"agate/pkg/tui/panes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateNewSession handles messages in ModeNewSession (new session creation)
func (m *Model) updateNewSession(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleNewSessionKeys(msg)

	case panes.SessionSwitchedMsg:
		// Session was auto-switched by navigation in agents pane
		// Switch to session mode and schedule a debounced UI update
		if msg.Session != nil {
			m.SetMode(ModeSession)
			return m, m.switchToSessionUI(msg.Session)
		}
		return m, nil

	case debouncedSessionSwitchMsg:
		// Debounce timer expired - perform the actual UI switch
		return m, m.performSessionSwitch(msg.session)

	case components.OpenAgentSelectorMsg:
		// Open agent selector with current selections from chat input
		if m.chatInput != nil {
			m.agentSelector = components.NewAgentSelector(m.chatInput.GetSelectedAgents())
			m.agentSelector.SetSize(m.layout.GetWidth(), m.layout.GetHeight())
			m.SetOverlay(AgentSelectorOverlay)

			var cmds []tea.Cmd
			if initCmd := m.agentSelector.InitCmd(); initCmd != nil {
				cmds = append(cmds, initCmd)
			}

			return m, combineCmds(cmds...)
		}
		return m, nil

	case components.AgentsSelectedMsg:
		// User selected agents from the selector
		if m.chatInput != nil {
			m.chatInput.SetSelectedAgents(msg.Agents)
			m.ClearOverlay()
		}
		return m, nil
	}

	// Pass all other messages (including cursor blink) to the appropriate component
	msgType := fmt.Sprintf("%T", msg)
	if msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg" {
		debug.DebugLog("[UPDATE_NEW_SESSION] Received blink message, IsSessionsFocus=%v, chatInput=%v",
			m.state.Focus.IsSessionsFocus(), m.chatInput != nil)
	}

	if !m.state.Focus.IsSessionsFocus() && m.chatInput != nil {
		// Chat input is focused - pass messages to it
		cmd := m.chatInput.Update(msg)
		if cmd != nil && (msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg") {
			debug.DebugLog("[UPDATE_NEW_SESSION] Passing blink message to chatInput, got command back")
		}
		return m, cmd
	}

	if m.state.Focus.IsSessionsFocus() && m.repoPane != nil && m.repoPane.IsActive() {
		// Sessions pane is focused - pass non-keyboard messages (like blink) to it
		if msgType == "cursor.BlinkMsg" || msgType == "cursor.blinkMsg" {
			debug.DebugLog("[UPDATE_NEW_SESSION] Passing blink message to repoPane")
		}
		var cmd tea.Cmd
		m.repoPane, cmd = m.repoPane.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleNewSessionKeys handles keyboard input in new session mode
func (m *Model) handleNewSessionKeys(msg tea.KeyMsg) (*Model, tea.Cmd) {
	debug.DebugLog("[handleNewSessionKeys] ENTRY: key=%q", msg.String())

	if m.consumePendingAltRune(msg) {
		debug.DebugLog("[handleNewSessionKeys] Consumed stray key %q after alt shortcut", msg.String())
		return m, nil
	}

	// Handle global shortcuts first - these work regardless of which pane is focused
	switch {
	case key.Matches(msg, common.GlobalKeys.AgentsPane):
		// Alt+S in new session mode should focus sessions pane for browsing
		debug.DebugLog("[handleNewSessionKeys] AgentsPane key matched, switching focus to sessions pane")
		m.startSuppressingAltRune(msg)
		m.state.Focus = layout.NewSessionsFocus()
		m.shortcutOverlay.SetFocus(m.state.Focus.String())
		// Blur chat input when switching away
		if m.chatInput != nil {
			debug.DebugLog("[handleNewSessionKeys] Blurring chat input before switching to sessions pane")
			m.chatInput.Blur()
			debug.DebugLog("[handleNewSessionKeys] Chat input blurred")
		}
		// Activate sessions pane for search
		if m.repoPane != nil {
			debug.DebugLog("[handleNewSessionKeys] Activating sessions pane")
			m.repoPane.SetActive(true)
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.SessionPane):
		// Alt+A jumps to agents pane (if session exists)
		m.startSuppressingAltRune(msg)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewAgentsFocus(layout.SubPaneTmux))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.ChangesPane):
		// Alt+C jumps to changes pane (if session exists)
		m.startSuppressingAltRune(msg)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewAgentsFocus(layout.SubPaneGit))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.Keybindings):
		// Show help dialog
		m.startSuppressingAltRune(msg)
		m.SetOverlay(HelpOverlay)
		return m, nil

	case key.Matches(msg, common.GlobalKeys.Quit):
		// Persist sessions before quitting
		if m.sessionManager != nil {
			if err := m.sessionManager.PersistSessions(); err != nil {
				// Log error but don't fail quit
			}
		}

		// Close debug panel and log file
		if m.debugLogger != nil {
			m.debugLogger.Close()
		}
		return m, tea.Quit

	case key.Matches(msg, common.GlobalKeys.NewSession):
		// Already in new session mode - ensure chat input is focused
		m.startSuppressingAltRune(msg)
		focusCmd := m.enterNewSessionMode(false)
		return m, focusCmd
	}

	// Check if sessions pane is active - if so, route ALL remaining keys to it
	sessionsPaneActive := m.state.Focus.IsSessionsFocus() && m.repoPane != nil && m.repoPane.IsActive()
	if sessionsPaneActive {
		// Sessions pane is active - send all keys to it
		var cmd tea.Cmd
		m.repoPane, cmd = m.repoPane.Update(msg)
		return m, tea.Batch(cmd, m.updateChangesPane())
	}

	// Sessions pane not active - handle chat input
	if msg.Type == tea.KeyEnter && !msg.Alt {
		// User pressed Enter - start session creation
		prompt := m.chatInput.GetValue()

		if strings.TrimSpace(prompt) == "" {
			// Empty prompt - show error toast
			toastCmd := m.toast.Show("Please enter a prompt", 0)
			return m, toastCmd
		}

		// Generate deterministic branch name instantly
		repoName := ""
		if repo, err := m.sessionManager.GetRepository(); err == nil {
			repoName = repo.GetRepositoryName()
		}
		branchName := session.GenerateBranchNameFromPrompt(repoName)

		// Start session creation immediately by sending message
		return m, func() tea.Msg {
			return branchNameGeneratedMsg{branchName: branchName}
		}
	} else if msg.Type == tea.KeyEscape {
		// Cancel new session input
		if len(m.sessionManager.ListSessions()) > 0 {
			m.SetMode(ModeSession)
			m.chatInput.Reset()
		}
		return m, nil
	}

	// Update chat input with other keys
	cmd := m.chatInput.Update(msg)
	return m, cmd
}
