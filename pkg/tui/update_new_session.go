package tui

import (
	"strings"

	"agate/pkg/common"
	"agate/pkg/session"
	"agate/pkg/tui/components"
	"agate/pkg/tui/layout"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateNewSession handles messages in ModeNewSession (new session creation)
func (m *Model) updateNewSession(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleNewSessionKeys(msg)

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

	return m, nil
}

// handleNewSessionKeys handles keyboard input in new session mode
func (m *Model) handleNewSessionKeys(msg tea.KeyMsg) (*Model, tea.Cmd) {
	// Check for pane navigation keys - these should NOT be consumed by chat input
	isNavKey := key.Matches(msg, common.GlobalKeys.AgentsPane) ||
		key.Matches(msg, common.GlobalKeys.SessionPane) ||
		key.Matches(msg, common.GlobalKeys.ChangesPane) ||
		key.Matches(msg, common.GlobalKeys.Keybindings) ||
		key.Matches(msg, common.GlobalKeys.Quit)

	// Check if agents pane is active - if so, skip chat input and let pane handle it
	agentsPaneActive := m.state.Focus.IsAgentsFocus() && m.repoPane != nil && m.repoPane.IsActive()

	if !isNavKey && !agentsPaneActive {
		// Not a navigation key - handle as chat input
		if msg.Type == tea.KeyEnter && !msg.Alt {
			// User pressed Enter - start session creation
			prompt := m.chatInput.GetValue()

			if strings.TrimSpace(prompt) == "" {
				// Empty prompt - show error toast
				toastCmd := m.toast.Show("Please enter a prompt", 0)
				return m, toastCmd
			}

			// Generate deterministic branch name instantly
			repoName := m.worktreeManager.GetRepositoryName()
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
		} else {
			// Update chat input
			cmd := m.chatInput.Update(msg)
			return m, cmd
		}
	}

	// Handle navigation keys
	switch {
	case key.Matches(msg, common.GlobalKeys.AgentsPane):
		// Alt+S jumps to sessions pane
		return m.switchToPane(layout.NewAgentsFocus())

	case key.Matches(msg, common.GlobalKeys.SessionPane):
		// Alt+A jumps to agents pane (if session exists)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewSessionFocus(layout.SubPaneTmux))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.ChangesPane):
		// Alt+C jumps to changes pane (if session exists)
		if m.sessionManager != nil && m.sessionManager.GetActiveSession() != nil {
			m.SetMode(ModeSession)
			return m.switchToPane(layout.NewSessionFocus(layout.SubPaneGit))
		}
		return m, nil

	case key.Matches(msg, common.GlobalKeys.Keybindings):
		// Show help dialog
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
	}

	return m, nil
}
