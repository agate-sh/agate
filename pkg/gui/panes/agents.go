package panes

import (
	"agate/internal/debug"
	"agate/pkg/common"
	"agate/pkg/config"
	"agate/pkg/gui/components"
	"agate/pkg/session"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"agate/pkg/git"
	"agate/pkg/gui/theme"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AgentsPane manages the display of agent sessions
type AgentsPane struct {
	*components.BasePane
	list            list.Model
	sessionManager  *session.Manager
	groupedSessions map[string][]*session.Session // Sessions grouped by repo
	currentRepo     string
	activeWorktree  *git.WorktreeInfo
	mainWorktree    *git.WorktreeInfo
	items           []list.Item
	delegate        itemDelegate
	expandedRepos   map[string]bool // Track which repos are expanded
	lastSavedPath   string
	lastSavedBranch string
	lastSavedRepo   string
	isGitRepo       bool
	searchInput     textinput.Model // Search input for filtering by branch
	searching       bool            // Whether search input is focused
	unfilteredItems []list.Item     // Backup of items before filtering
}

// AgentListItem implements list.Item interface for agent sessions
type AgentListItem struct {
	Type      string // "repo_header", "session", "empty_message", "gap"
	RepoName  string
	RepoPath  string // Full path to repository
	Worktree  *git.WorktreeInfo
	SessionID string // Session ID
	Session   *session.Session // Session object for accessing description and timestamps
	Index     int    // Index in original repo list
}

// FilterValue implements list.Item
func (i AgentListItem) FilterValue() string {
	if i.Type == "session" && i.Worktree != nil {
		return i.Worktree.Name
	}
	return i.RepoName
}

// itemDelegate handles rendering of individual list items
type itemDelegate struct {
	currentRepo    string
	styles         *itemStyles
	expandedRepos  map[string]bool
	isActive       bool
	isMainWorktree *git.WorktreeInfo
	activeWorktree *git.WorktreeInfo
	searching      bool
}

type itemStyles struct {
	repoHeader    lipgloss.Style
	repoCurrent   lipgloss.Style
	selectedItem  lipgloss.Style
	normalItem    lipgloss.Style
	expandArrow   lipgloss.Style
	collapseArrow lipgloss.Style
	mustedText    lipgloss.Style
	innerWidth    int
	fullWidth     int
	paddingLeft   int
	paddingRight  int
}

type deletionCandidate struct {
	item  *AgentListItem
	index int
}

type deletionSelectionPlan struct {
	deletedIdx int
	aboveIdx   int
	belowIdx   int
	candidates []deletionCandidate
}

func newItemStyles() *itemStyles {
	padding := components.PaneContentHorizontalPadding()
	return &itemStyles{
		repoHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(true),
		repoCurrent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)).
			Bold(true),
		selectedItem: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.RowHighlight)).
			Foreground(lipgloss.Color(theme.TextPrimary)),
		normalItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextDescription)),
		expandArrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)),
		collapseArrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)),
		mustedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)),
		paddingLeft:  padding,
		paddingRight: padding,
	}
}

// Height implements list.ItemDelegate
func (d itemDelegate) Height() int {
	return 1 // Each item takes 1 line
}

// Spacing implements list.ItemDelegate
func (d itemDelegate) Spacing() int {
	return 0
}

// Update implements list.ItemDelegate
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	workItem, ok := item.(AgentListItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	innerWidth := d.styles.innerWidth
	paddingLeft := d.styles.paddingLeft
	paddingRight := d.styles.paddingRight
	if innerWidth <= 0 {
		innerWidth = m.Width() - paddingLeft - paddingRight
		if innerWidth < 0 {
			innerWidth = 0
		}
	}
	fullWidth := innerWidth + paddingLeft + paddingRight
	leftPad := strings.Repeat(" ", paddingLeft)
	rightPad := strings.Repeat(" ", paddingRight)

	// Don't highlight when searching - only show highlight after user navigates
	highlight := selected && d.isActive && !d.searching
	if index == m.Index() && d.isActive {
		git.DebugLog("[AgentsPane.Render] index=%d, selected=%v, isActive=%v, searching=%v, highlight=%v",
			index, selected, d.isActive, d.searching, highlight)
	}

	var linePlain string
	var lineStyled string
	var hint string

	switch workItem.Type {
	case "repo_header":
		repoName := workItem.RepoName
		displayName := repoName

		linePlain = displayName

		if highlight {
			lineStyled = linePlain
		} else {
			var nameStyled string
			if repoName == d.currentRepo {
				nameStyled = d.styles.repoCurrent.Render(displayName)
			} else {
				nameStyled = d.styles.repoHeader.Render(displayName)
			}
			lineStyled = nameStyled
		}

	case "session":
		if workItem.Worktree == nil || workItem.Session == nil {
			return
		}
		// Show description + time ago instead of branch name
		description := workItem.Session.Description
		if strings.TrimSpace(description) == "" {
			description = workItem.Session.Prompt
		}
		timeAgo := common.FormatTimeAgo(workItem.Session.CreatedAt)

		leadingSpaces := ""
		timeAgoWidth := lipgloss.Width(timeAgo)

		// Calculate available width for description
		// We need minimum 2 spaces gap before timestamp
		leadingWidth := lipgloss.Width(leadingSpaces)
		minGap := 2
		availableForDesc := innerWidth - leadingWidth - minGap - timeAgoWidth

		// Truncate description if needed
		truncatedDesc := description
		if lipgloss.Width(description) > availableForDesc {
			runes := []rune(description)
			for lipgloss.Width(string(runes)+"...") > availableForDesc && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			if len(runes) > 0 {
				truncatedDesc = string(runes) + "..."
			} else {
				truncatedDesc = ""
			}
		}

		// Build linePlain with right-aligned timestamp
		descWidth := lipgloss.Width(truncatedDesc)
		paddingWidth := innerWidth - leadingWidth - descWidth - timeAgoWidth
		if paddingWidth < minGap {
			paddingWidth = minGap
		}
		padding := strings.Repeat(" ", paddingWidth)
		linePlain = leadingSpaces + truncatedDesc + padding + timeAgo

		if highlight {
			// When highlighted, just use plain text (will be styled by highlight background)
			lineStyled = linePlain
		} else {
			// Render description with TextDescription color
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
			descRendered := descStyle.Render(truncatedDesc)

			// Render timestamp with TextMuted color
			timeAgoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted))
			timeAgoRendered := timeAgoStyle.Render(timeAgo)

			// Combine: "  <description><padding><timeAgo>"
			lineStyled = leadingSpaces + descRendered + padding + timeAgoRendered
		}

	case "empty_message":
		// Show empty state message
		linePlain = "No agents"
		lineStyled = d.styles.mustedText.Render(linePlain)

	case "gap":
		// Empty line for spacing
		linePlain = ""
		lineStyled = ""

	default:
		return
	}

	// Handle hint text - only when pane is active and hovering a session item
	if hint == "" && d.isActive && highlight && workItem.Type == "session" {
		// Show delete hint
		hint = " ⌥d to delete"
	}

	contentWidth := innerWidth
	var body string
	activeHighlight := highlight
	if activeHighlight {
		bodyStyle := d.styles.selectedItem
		if workItem.Type == "repo_header" {
			bodyStyle = bodyStyle.Bold(true)
		}

		if hint != "" {
			// Use shared utility for hint rendering
			// Note: We pass paddingCount=0 because we handle padding separately in agents pane
			body = components.RenderRowWithHint(
				linePlain,
				hint,
				contentWidth,
				0, // No padding - we handle borders/padding separately
				theme.RowHighlight,
				theme.TextPrimary,
				theme.TextMuted,
			)
		} else {
			body = bodyStyle.Width(contentWidth).Render(linePlain)
		}
	} else {
		body = lipgloss.NewStyle().Width(contentWidth).Render(lineStyled)
	}

	var leftBorder string
	baseHighlightStyle := lipgloss.NewStyle()
	if activeHighlight {
		baseHighlightStyle = baseHighlightStyle.Background(lipgloss.Color(theme.RowHighlight))
	}

	// Just render padding - no cursor rune
	leftBorder = baseHighlightStyle.Render(leftPad)

	rightPadStyle := lipgloss.NewStyle()
	if activeHighlight {
		rightPadStyle = rightPadStyle.Background(lipgloss.Color(theme.RowHighlight))
	}
	rightSegment := rightPadStyle.Render(rightPad)

	content := leftBorder + body + rightSegment
	if lipgloss.Width(content) < fullWidth {
		padStyle := rightPadStyle
		if !highlight {
			padStyle = lipgloss.NewStyle()
		}
		content += padStyle.Render(strings.Repeat(" ", fullWidth-lipgloss.Width(content)))
	}

	if _, err := fmt.Fprint(w, content); err != nil {
		// Log error but continue - this is UI rendering
	}
}

// NewAgentsPane creates a new AgentsPane instance
func NewAgentsPane(sessionManager *session.Manager) *AgentsPane {
	styles := newItemStyles()

	// Create the expanded repos map that will be shared
	expandedRepos := make(map[string]bool)

	repoSelections, err := config.GetRepoSelections()
	if err != nil {
		repoSelections = map[string]config.RepoSelection{}
	}

	// Create delegate with reference to shared map
	delegate := itemDelegate{
		styles:        styles,
		expandedRepos: expandedRepos,
	}

	// Create list model with styles
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	// Create search input
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Search by description"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription))
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	ti.CharLimit = 100
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextPrimary))
	ti.Cursor.SetMode(cursor.CursorBlink)

	pane := &AgentsPane{
		BasePane:       components.NewBasePane(0, "Agents"), // Updated to Agents
		list:           l,
		sessionManager: sessionManager,
		delegate:       delegate,
		expandedRepos:  expandedRepos, // Use the same map reference
		searchInput:    ti,
		searching:      false,
	}

	// Initial refresh
	if sessionManager != nil {
		// Get worktree manager from session manager if available
		if worktreeManager := sessionManager.GetWorktreeManager(); worktreeManager != nil {
			pane.isGitRepo = worktreeManager.IsGitRepo()
			if pane.isGitRepo {
				pane.currentRepo = worktreeManager.GetRepositoryName()
			}
			if mainWorktree, err := worktreeManager.GetMainWorktreeInfo(); err == nil {
				pane.mainWorktree = mainWorktree
				if pane.isGitRepo {
					pane.activeWorktree = mainWorktree
				}
			}
		}

		pane.Refresh()

		// Handle saved selections - simplified since we're showing sessions now
		if len(repoSelections) > 0 {
			if pane.isGitRepo && pane.currentRepo != "" {
				if selection, ok := repoSelections[pane.currentRepo]; ok {
					pane.lastSavedRepo = pane.currentRepo
					pane.lastSavedPath = selection.Worktree.Path
					pane.lastSavedBranch = selection.Worktree.Branch
					pane.SelectWorktreeByRef(pane.currentRepo, selection.Worktree)
				}
			}
		}
	}

	return pane
}

// SetSize updates the dimensions of the repo worktree pane
func (r *AgentsPane) SetSize(width, height int) {
	r.BasePane.SetSize(width, height)
	fullWidth := components.PaneFullWidth(width)

	// Account for search input (1 line) + spacing (1 line)
	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	r.list.SetSize(fullWidth, listHeight)
	r.searchInput.Width = fullWidth

	// Update delegate widths so row highlighting spans the padded content
	r.delegate.styles.innerWidth = width
	r.delegate.styles.fullWidth = fullWidth
}

// SetActive sets whether the pane is focused and updates row highlighting state
func (r *AgentsPane) SetActive(active bool) {
	wasAlreadyActive := r.IsActive()
	git.DebugLog("[AgentsPane.SetActive] called with active=%v, wasAlreadyActive=%v", active, wasAlreadyActive)

	r.BasePane.SetActive(active)
	r.delegate.isActive = active
	r.list.SetDelegate(r.delegate)

	// Refresh when becoming active to pick up any new sessions
	if active {
		// Sync active worktree from session manager before refreshing
		r.SyncActiveSessionFromManager()
		r.Refresh()

		// Only enable search mode if we're transitioning from inactive to active
		// Don't re-enable search if we're already active (e.g., during navigation)
		if !wasAlreadyActive {
			git.DebugLog("[AgentsPane.SetActive] enabling search mode (was inactive)")
			r.searchInput.Focus()
			r.searching = true
			r.delegate.searching = true
			r.list.SetDelegate(r.delegate)
			// Reset list index to 0 so Down arrow starts from the first item
			r.list.Select(0)
		} else {
			git.DebugLog("[AgentsPane.SetActive] keeping search state unchanged (was already active)")
		}
	} else {
		git.DebugLog("[AgentsPane.SetActive] disabling search mode")
		// Blur search input when inactive
		r.searchInput.Blur()
		r.searching = false
		r.delegate.searching = false
		r.list.SetDelegate(r.delegate)
	}
}

// SyncActiveSessionFromManager synchronizes the pane's active worktree with the session manager's active session
func (r *AgentsPane) SyncActiveSessionFromManager() {
	git.DebugLog("[AgentsPane] SyncActiveSessionFromManager called")
	if r.sessionManager == nil {
		git.DebugLog("[AgentsPane] SyncActiveSessionFromManager: sessionManager is nil")
		return
	}

	activeSession := r.sessionManager.GetActiveSession()
	git.DebugLog("[AgentsPane] SyncActiveSessionFromManager: activeSession=%v", activeSession)
	if activeSession != nil && activeSession.Worktree() != nil {
		git.DebugLog("[AgentsPane] SyncActiveSessionFromManager: activeSession.Worktree.Path=%s, Branch=%s",
			activeSession.Worktree().Path, activeSession.Worktree().Branch)
		// Update the pane's active worktree to match the session manager
		clone := *activeSession.Worktree()
		r.activeWorktree = &clone
		if activeSession.Worktree().RepoName != "" {
			r.currentRepo = activeSession.Worktree().RepoName
		}
		// Update delegate
		r.delegate.currentRepo = r.currentRepo
		r.delegate.activeWorktree = r.activeWorktree
		git.DebugLog("[AgentsPane] SyncActiveSessionFromManager: Updated r.activeWorktree to path=%s", r.activeWorktree.Path)
	} else {
		git.DebugLog("[AgentsPane] SyncActiveSessionFromManager: No active session or worktree")
	}
}

// updateHoveredMainWorktree refreshes the delegate so hover-dependent rendering stays in sync
func (r *AgentsPane) updateHoveredMainWorktree() {
	r.list.SetDelegate(r.delegate)
}

// GetTitleStyle returns the title style for the repo worktree pane
func (r *AgentsPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if !r.IsActive() {
		// When not active, show keyboard shortcut to activate
		shortcuts = "(⌥s)"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Sessions",
		Shortcuts: shortcuts,
	}
}

// View renders the repo worktree pane content
func (r *AgentsPane) View() string {
	if r.sessionManager == nil {
		// Show placeholder message
		style := lipgloss.NewStyle().
			Width(components.PaneFullWidth(r.GetWidth())).
			Height(r.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextMuted))

		return style.Render("Session manager not available")
	}

	// Check if we need to refresh due to new sessions being available
	currentSessionCount := len(r.sessionManager.ListSessions())

	if currentSessionCount > 0 && len(r.items) == 0 {
		r.Refresh()
		r.jumpToActiveSession()
	}

	// Also check if we have sessions but still showing empty items after building
	if currentSessionCount > 0 && len(r.items) > 0 {
		// Count actual session items
		sessionItemCount := 0
		for _, item := range r.items {
			if agentItem, ok := item.(AgentListItem); ok {
				if agentItem.Type == "session" {
					sessionItemCount++
				}
			}
		}

		if sessionItemCount == 0 {
			r.Refresh()
			r.jumpToActiveSession()
		}
	}

	// Check if there are no agents
	if len(r.items) == 0 {
		style := lipgloss.NewStyle().
			Width(components.PaneFullWidth(r.GetWidth())).
			Height(r.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextMuted))

		message := "No agents yet\n\nn to create a new agent"
		return style.Render(message)
	}

	// Render search input at top
	searchInputView := r.searchInput.View()

	// Render list view
	listView := r.list.View()

	// Join search input and list vertically with spacing
	return lipgloss.JoinVertical(lipgloss.Left, searchInputView, "", listView)
}

// Update handles tea.Msg updates for the repo worktree pane
func (r *AgentsPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	var cmds []tea.Cmd

	// When searching, intercept up/down keys to exit search mode
	if r.searching {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "down", "k", "j":
				// Exit search mode and navigate
				git.DebugLog("[AgentsPane.Update] Intercepted %s key, exiting search", keyMsg.String())
				r.searching = false
				r.searchInput.Blur()
				git.DebugLog("[AgentsPane.Update] Set r.searching=false, r.searchInput.Focused()=%v", r.searchInput.Focused())

				// Find first selectable item for down, last for up
				var navCmd tea.Cmd
				if keyMsg.String() == "up" || keyMsg.String() == "k" {
					// For up, find the last selectable item
					for i := len(r.items) - 1; i >= 0; i-- {
						if r.isSelectableItem(i) {
							r.list.Select(i)
							navCmd = r.onSelectionChanged()
							break
						}
					}
				} else {
					// For down, find the first selectable item
					for i := 0; i < len(r.items); i++ {
						if r.isSelectableItem(i) {
							r.list.Select(i)
							navCmd = r.onSelectionChanged()
							break
						}
					}
				}

				// Update delegate after navigation
				r.delegate.searching = false
				r.list.SetDelegate(r.delegate)
				git.DebugLog("[AgentsPane.Update] Set delegate.searching=false, returning navCmd")

				if navCmd != nil {
					return r, navCmd
				}
				return r, nil
			}
		}

		oldValue := r.searchInput.Value()
		var searchCmd tea.Cmd
		r.searchInput, searchCmd = r.searchInput.Update(msg)
		if searchCmd != nil {
			cmds = append(cmds, searchCmd)
		}

		// If value changed, filter the list
		if r.searchInput.Value() != oldValue {
			r.filterList()
		}
	}

	// Update list
	var listCmd tea.Cmd
	r.list, listCmd = r.list.Update(msg)
	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}

	if len(cmds) == 0 {
		return r, nil
	} else if len(cmds) == 1 {
		return r, cmds[0]
	}
	return r, tea.Batch(cmds...)
}

// HandleKey processes keyboard input when the pane is active
func (r *AgentsPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	if !r.IsActive() {
		return false, nil
	}

	// When searching, up/down exits search mode and navigates the list
	if r.searching {
		switch key {
		case "up", "k", "down", "j":
			// Exit search mode and navigate
			git.DebugLog("[AgentsPane.HandleKey] up/down pressed, exiting search mode")
			r.searching = false
			r.searchInput.Blur()
			git.DebugLog("[AgentsPane.HandleKey] r.searching=%v, r.searchInput.Focused()=%v", r.searching, r.searchInput.Focused())

			// Find first selectable item for down, last for up
			var cmd tea.Cmd
			if key == "up" || key == "k" {
				// For up, find the last selectable item
				for i := len(r.items) - 1; i >= 0; i-- {
					if r.isSelectableItem(i) {
						r.list.Select(i)
						cmd = r.onSelectionChanged()
						break
					}
				}
			} else {
				// For down, find the first selectable item
				for i := 0; i < len(r.items); i++ {
					if r.isSelectableItem(i) {
						r.list.Select(i)
						cmd = r.onSelectionChanged()
						break
					}
				}
			}

			// Update delegate after navigation (which may have updated other fields)
			r.delegate.searching = false
			r.list.SetDelegate(r.delegate)
			git.DebugLog("[AgentsPane.HandleKey] after SetDelegate, r.delegate.searching=%v", r.delegate.searching)
			return true, cmd
		case "esc":
			// Clear filter and exit search mode
			r.clearFilter()
			r.searching = false
			r.delegate.searching = false
			r.list.SetDelegate(r.delegate)
			r.searchInput.Blur()
			return true, nil
		case "alt+d":
			// Exit search mode and trigger deletion
			r.searching = false
			r.delegate.searching = false
			r.list.SetDelegate(r.delegate)
			r.searchInput.Blur()
			// Fall through to normal alt+d handling below
		default:
			// Return false so typing keys go to Update method for search input
			return false, nil
		}
	}

	// When not searching, normal navigation
	switch key {
	case "up", "k":
		r.MoveUp()
		return true, nil
	case "down", "j":
		r.MoveDown()
		return true, nil
	case "alt+d":
		// Delete selected session
		if len(r.items) > 0 {
			selectedItem := r.list.SelectedItem()
			if workItem, ok := selectedItem.(AgentListItem); ok && workItem.Type == "session" && workItem.Worktree != nil {
				if r.sessionManager != nil {
					if session := r.sessionManager.GetSessionForWorktree(workItem.Worktree); session != nil {
						// Return a command to trigger the delete confirmation dialog
						return true, func() tea.Msg {
							return DeleteSessionRequestMsg{Session: session}
						}
					}
				}
			}
		}
		return true, nil
	default:
		return false, nil
	}
}

// MoveUp moves the selection up one item (implements Pane interface)
func (r *AgentsPane) MoveUp() bool {
	r.moveUp()
	return true
}

// MoveDown moves the selection down one item (implements Pane interface)
func (r *AgentsPane) MoveDown() bool {
	r.moveDown()
	return true
}

// MoveUpWithCmd moves the selection up and returns a command if session was switched
func (r *AgentsPane) MoveUpWithCmd() (bool, tea.Cmd) {
	cmd := r.moveUp()
	return true, cmd
}

// MoveDownWithCmd moves the selection down and returns a command if session was switched
func (r *AgentsPane) MoveDownWithCmd() (bool, tea.Cmd) {
	cmd := r.moveDown()
	return true, cmd
}

// moveUp navigates up to the previous selectable item
func (r *AgentsPane) moveUp() tea.Cmd {
	if len(r.items) == 0 {
		return nil
	}

	currentIndex := r.list.Index()
	startIndex := currentIndex

	for {
		// Move up to previous item
		if currentIndex > 0 {
			currentIndex--
		} else {
			// Wrap to bottom
			currentIndex = len(r.items) - 1
		}

		// Check if we've wrapped around completely
		if currentIndex == startIndex {
			break
		}

		// Check if this item is selectable
		if r.isSelectableItem(currentIndex) {
			r.list.Select(currentIndex)
			return r.onSelectionChanged()
		}
	}
	return nil
}

// moveDown navigates down to the next selectable item
func (r *AgentsPane) moveDown() tea.Cmd {
	if len(r.items) == 0 {
		return nil
	}

	currentIndex := r.list.Index()
	startIndex := currentIndex

	for {
		// Move down to next item
		if currentIndex < len(r.items)-1 {
			currentIndex++
		} else {
			// Wrap to top
			currentIndex = 0
		}

		// Check if we've wrapped around completely
		if currentIndex == startIndex {
			break
		}

		// Check if this item is selectable
		if r.isSelectableItem(currentIndex) {
			r.list.Select(currentIndex)
			return r.onSelectionChanged()
		}
	}
	return nil
}

// onSelectionChanged is called when the selection changes and automatically switches to the selected session
func (r *AgentsPane) onSelectionChanged() tea.Cmd {
	if r.sessionManager == nil {
		return nil
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(AgentListItem); ok {
		if workItem.Type == "session" && workItem.Worktree != nil && workItem.SessionID != "" {
			// Switch to the selected session
			if sess, err := r.sessionManager.SwitchToSession(workItem.SessionID); err != nil {
				debug.DebugLog("Failed to auto-switch to session %s: %v", workItem.SessionID, err)
			} else {
				// Update the pane to reflect the new active session
				r.setActiveWorktree(workItem.Worktree)

				// Send message to notify that session was switched
				return func() tea.Msg {
					return SessionSwitchedMsg{Session: sess}
				}
			}
		}
	}
	return nil
}

// GetSelectedWorktree returns the currently selected worktree
func (r *AgentsPane) GetSelectedWorktree() *git.WorktreeInfo {
	if r.activeWorktree != nil {
		clone := *r.activeWorktree
		return &clone
	}

	if len(r.items) == 0 {
		return nil
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(AgentListItem); ok {
		if workItem.Type == "session" && workItem.Worktree != nil {
			clone := *workItem.Worktree
			return &clone
		}
	}
	return nil
}

// SelectWorktreeByRef attempts to select a worktree using the provided repo name and reference.
func (r *AgentsPane) SelectWorktreeByRef(repoName string, ref config.WorktreeRef) bool {
	repoName = strings.TrimSpace(repoName)
	if repoName != "" && r.expandedRepos != nil {
		r.expandedRepos[repoName] = true
		r.rebuildListPreservingSelection(r.list.Index())
	}

	if strings.TrimSpace(ref.Path) != "" {
		if r.SelectWorktreeByPath(ref.Path) {
			return true
		}
	}

	if strings.TrimSpace(ref.Branch) != "" {
		return r.selectWorktreeByBranch(repoName, ref.Branch)
	}

	return false
}

// SelectWorktreeByPath attempts to select the worktree matching the provided path.
func (r *AgentsPane) SelectWorktreeByPath(worktreePath string) bool {
	git.DebugLog("[AgentsPane] SelectWorktreeByPath called with path: %s", worktreePath)
	if strings.TrimSpace(worktreePath) == "" {
		git.DebugLog("[AgentsPane] SelectWorktreeByPath: empty path, returning false")
		return false
	}

	target := filepath.Clean(worktreePath)
	git.DebugLog("[AgentsPane] SelectWorktreeByPath: cleaned target path: %s", target)
	git.DebugLog("[AgentsPane] SelectWorktreeByPath: searching through %d items", len(r.items))

	for idx, item := range r.items {
		workItem, ok := item.(AgentListItem)
		if !ok || workItem.Type != "session" || workItem.Worktree == nil {
			continue
		}

		itemPath := filepath.Clean(workItem.Worktree.Path)
		git.DebugLog("[AgentsPane] SelectWorktreeByPath: checking item %d: type=%s, path=%s", idx, workItem.Type, itemPath)
		if itemPath == target {
			git.DebugLog("[AgentsPane] SelectWorktreeByPath: MATCH FOUND at index %d", idx)
			r.setActiveWorktree(workItem.Worktree)
			r.rebuildListPreservingSelection(idx)
			return true
		}
	}

	git.DebugLog("[AgentsPane] SelectWorktreeByPath: NO MATCH FOUND")
	return false
}

func (r *AgentsPane) selectWorktreeByBranch(repoName, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}

	for idx, item := range r.items {
		workItem, ok := item.(AgentListItem)
		if !ok || workItem.Type != "session" || workItem.Worktree == nil {
			continue
		}
		if repoName != "" && workItem.RepoName != repoName {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(workItem.Worktree.Branch), branch) {
			r.setActiveWorktree(workItem.Worktree)
			r.rebuildListPreservingSelection(idx)
			return true
		}
	}

	return false
}

// GetHoveredRepoName returns the repository name of the currently selected list item.
func (r *AgentsPane) GetHoveredRepoName() string {
	if len(r.items) == 0 {
		return ""
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(AgentListItem); ok {
		return workItem.RepoName
	}
	return ""
}

// SnapshotItems returns a copy of the current list items as AgentListItems.
func (r *AgentsPane) SnapshotItems() []AgentListItem {
	snapshot := make([]AgentListItem, 0, len(r.items))
	for _, item := range r.items {
		if workItem, ok := item.(AgentListItem); ok {
			snapshot = append(snapshot, workItem)
		}
	}
	return snapshot
}

// SelectSessionAfterDeletion selects the appropriate session after deletion with priority:
// 1. Session above if available
// 2. Session below if available
func (r *AgentsPane) SelectSessionAfterDeletion(deletedSession *session.Session, previousItems []AgentListItem) bool {
	if deletedSession == nil || deletedSession.Worktree() == nil {
		return false
	}

	deletedPath := filepath.Clean(deletedSession.Worktree().Path)
	deletedWasActive := r.isActiveWorktree(deletedSession.Worktree())

	git.DebugLog("[AgentsPane] SelectSessionAfterDeletion: deleted=%s, wasActive=%v", deletedPath, deletedWasActive)

	searchItems := previousItems
	if len(searchItems) == 0 {
		searchItems = r.SnapshotItems()
	}

	plan, found := determineDeletionSelectionPlan(deletedSession, searchItems)
	if !found {
		git.DebugLog("[AgentsPane] SelectSessionAfterDeletion: deleted item not found in previous list")
		return false
	}

	git.DebugLog("[AgentsPane] SelectSessionAfterDeletion: deletedIdx=%d, aboveIdx=%d, belowIdx=%d",
		plan.deletedIdx, plan.aboveIdx, plan.belowIdx)

	for _, candidate := range plan.candidates {
		if candidate.item == nil || candidate.item.Worktree == nil {
			continue
		}
		if r.selectSessionCandidate(candidate.item.Worktree, deletedWasActive) {
			git.DebugLog("[AgentsPane] SelectSessionAfterDeletion: selected candidate at idx %d", candidate.index)
			return true
		}
	}

	git.DebugLog("[AgentsPane] SelectSessionAfterDeletion: no selectable candidates")
	return false
}

func (r *AgentsPane) selectSessionCandidate(candidate *git.WorktreeInfo, makeActive bool) bool {
	if candidate == nil {
		return false
	}

	idx := r.findSessionIndexByPath(candidate.Path, candidate.RepoName)
	if idx == -1 {
		return false
	}

	workItem, ok := r.items[idx].(AgentListItem)
	if !ok || workItem.Worktree == nil {
		return false
	}

	if makeActive {
		r.setActiveWorktree(workItem.Worktree)
		if r.sessionManager != nil {
			if _, err := r.sessionManager.SwitchToSession(worktreeKey(workItem.Worktree)); err != nil {
				git.DebugLog("[AgentsPane] selectSessionCandidate: failed to switch active session: %v", err)
			}
		}
	}

	r.rebuildListPreservingSelection(idx)
	return true
}

func (r *AgentsPane) findSessionIndexByPath(targetPath, repoName string) int {
	if strings.TrimSpace(targetPath) == "" {
		return -1
	}

	cleanTarget := filepath.Clean(targetPath)
	for idx, item := range r.items {
		workItem, ok := item.(AgentListItem)
		if !ok || workItem.Worktree == nil {
			continue
		}
		if workItem.Type != "session" {
			continue
		}
		if repoName != "" && workItem.RepoName != repoName {
			continue
		}
		if filepath.Clean(workItem.Worktree.Path) == cleanTarget {
			return idx
		}
	}
	return -1
}

func worktreeKey(worktree *git.WorktreeInfo) string {
	if worktree == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", worktree.RepoName, worktree.Path)
}

func determineDeletionSelectionPlan(deletedSession *session.Session, items []AgentListItem) (deletionSelectionPlan, bool) {
	plan := deletionSelectionPlan{
		deletedIdx: -1,
		aboveIdx:   -1,
		belowIdx:   -1,
	}

	if deletedSession == nil || deletedSession.Worktree() == nil {
		return plan, false
	}

	deletedPath := filepath.Clean(deletedSession.Worktree().Path)
	deletedRepoName := deletedSession.Worktree().RepoName

	for idx := range items {
		item := &items[idx]
		if item.Worktree == nil {
			continue
		}
		if item.Type != "session" {
			continue
		}
		if filepath.Clean(item.Worktree.Path) == deletedPath {
			plan.deletedIdx = idx
			break
		}
	}

	if plan.deletedIdx == -1 {
		return plan, false
	}

	for idx := plan.deletedIdx - 1; idx >= 0; idx-- {
		candidate := &items[idx]
		if candidate.RepoName != deletedRepoName {
			continue
		}
		if candidate.Type == "session" && candidate.Worktree != nil {
			plan.aboveIdx = idx
			break
		}
	}

	for idx := plan.deletedIdx + 1; idx < len(items); idx++ {
		candidate := &items[idx]
		if candidate.RepoName != deletedRepoName {
			continue
		}
		if candidate.Type == "session" && candidate.Worktree != nil {
			plan.belowIdx = idx
			break
		}
	}

	preferBelowFirst := false

	appendCandidate := func(idx int) {
		if idx == -1 {
			return
		}
		candidate := &items[idx]
		if candidate.Worktree == nil {
			return
		}
		plan.candidates = append(plan.candidates, deletionCandidate{item: candidate, index: idx})
	}

	if preferBelowFirst {
		appendCandidate(plan.belowIdx)
		appendCandidate(plan.aboveIdx)
	} else {
		appendCandidate(plan.aboveIdx)
		appendCandidate(plan.belowIdx)
	}

	return plan, true
}

// HasWorktrees returns whether there are any sessions available
func (r *AgentsPane) HasWorktrees() bool {
	return len(r.groupedSessions) > 0
}

// HasItems returns whether there are any items to display
func (r *AgentsPane) HasItems() bool {
	return len(r.items) > 0
}

// Refresh refreshes the session list
func (r *AgentsPane) Refresh() error {
	if r.sessionManager == nil {
		return nil
	}

	// Get all sessions from session manager (now includes both main and linked)
	sessions := r.sessionManager.ListSessions()

	// Group sessions by repository
	r.groupedSessions = make(map[string][]*session.Session)
	for _, sess := range sessions {
		if sess.Worktree() != nil && sess.Worktree().RepoName != "" {
			r.groupedSessions[sess.Worktree().RepoName] = append(r.groupedSessions[sess.Worktree().RepoName], sess)
		}
	}

	// Update main worktree info if available
	if worktreeManager := r.sessionManager.GetWorktreeManager(); worktreeManager != nil {
		if mainWorktree, err := worktreeManager.GetMainWorktreeInfo(); err == nil {
			r.mainWorktree = mainWorktree
			if r.isGitRepo {
				if r.currentRepo == "" {
					r.currentRepo = mainWorktree.RepoName
				}
				if r.activeWorktree == nil || (r.activeWorktree != nil && r.activeWorktree.Path == mainWorktree.Path) {
					r.activeWorktree = mainWorktree
				}
			}
		}
	}

	r.buildItemList()

	// Update the list with new items and delegate
	r.list.SetItems(r.items)
	r.list.SetDelegate(r.delegate)
	if !r.isSelectableItem(r.list.Index()) {
		r.moveDown()
	}

	return nil
}

// filterList filters the list based on the search input value
func (r *AgentsPane) filterList() {
	query := strings.ToLower(strings.TrimSpace(r.searchInput.Value()))

	// If query is empty, show everything
	if query == "" {
		r.clearFilter()
		return
	}

	// Backup unfiltered items if not already backed up
	if r.unfilteredItems == nil {
		r.unfilteredItems = make([]list.Item, len(r.items))
		copy(r.unfilteredItems, r.items)
	}

	// Filter sessions by branch name
	filtered := []list.Item{}
	repoHasMatches := false
	var pendingRepoHeader *AgentListItem

	for _, item := range r.unfilteredItems {
		agentItem, ok := item.(AgentListItem)
		if !ok {
			continue
		}

		switch agentItem.Type {
		case "repo_header":
			// If previous repo had matches, add its header
			if pendingRepoHeader != nil && repoHasMatches {
				filtered = append(filtered, *pendingRepoHeader)
			}
			// Store this repo header to add later if it has matches
			pendingRepoHeader = &agentItem
			repoHasMatches = false

		case "session":
			if agentItem.Worktree != nil {
				branchName := strings.ToLower(agentItem.Worktree.Branch)
				if strings.Contains(branchName, query) {
					// This session matches - mark repo as having matches
					repoHasMatches = true
				}
			}

		case "empty_message", "gap":
			// Skip these in filtered view
			continue
		}
	}

	// Add final repo header if it had matches
	if pendingRepoHeader != nil && repoHasMatches {
		filtered = append(filtered, *pendingRepoHeader)
	}

	// Now do a second pass to add matching sessions
	for _, item := range r.unfilteredItems {
		agentItem, ok := item.(AgentListItem)
		if !ok {
			continue
		}

		if agentItem.Type == "session" && agentItem.Worktree != nil {
			branchName := strings.ToLower(agentItem.Worktree.Branch)
			if strings.Contains(branchName, query) {
				filtered = append(filtered, agentItem)
			}
		}
	}

	r.items = filtered
	r.list.SetItems(r.items)

	// Select first selectable item
	if len(r.items) > 0 {
		for i := range r.items {
			if r.isSelectableItem(i) {
				r.list.Select(i)
				break
			}
		}
	}
}

// clearFilter clears the search filter and restores unfiltered items
func (r *AgentsPane) clearFilter() {
	r.searchInput.SetValue("")
	if r.unfilteredItems != nil {
		r.items = r.unfilteredItems
		r.unfilteredItems = nil
		r.list.SetItems(r.items)
		// Try to restore selection to active session
		r.jumpToActiveSession()
	}
}

// buildItemList creates a list of items for the bubbles list
func (r *AgentsPane) buildItemList() {
	git.DebugLog("[AgentsPane] buildItemList called")
	if r.activeWorktree != nil {
		git.DebugLog("[AgentsPane] buildItemList: r.activeWorktree.Path=%s, Branch=%s", r.activeWorktree.Path, r.activeWorktree.Branch)
	} else {
		git.DebugLog("[AgentsPane] buildItemList: r.activeWorktree is nil")
	}
	r.items = nil

	// Get sorted repository names (current repo first)
	repoNames := make([]string, 0, len(r.groupedSessions))
	for repoName := range r.groupedSessions {
		repoNames = append(repoNames, repoName)
	}

	// Always include the current repository if we have a session manager
	// This ensures the main repo is shown even when no sessions exist
	if r.sessionManager != nil && r.currentRepo != "" {
		hasCurrentRepo := false
		for _, name := range repoNames {
			if name == r.currentRepo {
				hasCurrentRepo = true
				break
			}
		}
		if !hasCurrentRepo {
			repoNames = append(repoNames, r.currentRepo)
		}
	}

	// Sort repositories, putting current repo first
	sort.Slice(repoNames, func(i, j int) bool {
		if repoNames[i] == r.currentRepo {
			return true
		}
		if repoNames[j] == r.currentRepo {
			return false
		}
		return repoNames[i] < repoNames[j]
	})

	for _, repoName := range repoNames {
		// Add repository header
		var mainRepoPath string
		if r.sessionManager != nil && repoName == r.currentRepo {
			if worktreeManager := r.sessionManager.GetWorktreeManager(); worktreeManager != nil {
				mainRepoPath = worktreeManager.GetRepositoryPath()
			} else {
				mainRepoPath = repoName
			}
		} else {
			mainRepoPath = repoName
		}

		r.items = append(r.items, AgentListItem{
			Type:     "repo_header",
			RepoName: repoName,
			RepoPath: mainRepoPath,
		})

		// Always show all sessions (no collapse/expand)
		sessions := r.groupedSessions[repoName]

		if len(sessions) > 0 {
			// Sort sessions by CreatedAt in descending order (newest first)
			sort.Slice(sessions, func(i, j int) bool {
				return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
			})

			for _, sess := range sessions {
				if sess.Worktree() != nil {
					worktreeCopy := *sess.Worktree()
					r.items = append(r.items, AgentListItem{
						Type:      "session",
						RepoName:  repoName,
						Worktree:  &worktreeCopy,
						SessionID: sess.ID,
						Session:   sess,
					})
				}
			}
		} else {
			// No sessions - add placeholder
			r.items = append(r.items, AgentListItem{
				Type:     "empty_message",
				RepoName: repoName,
			})
		}

		// Add gap after repo for visual separation
		r.items = append(r.items, AgentListItem{
			Type:     "gap",
			RepoName: repoName,
		})
	}

	// Update delegate's current repo, expanded state, and main worktree
	r.delegate.currentRepo = r.currentRepo
	r.delegate.expandedRepos = r.expandedRepos
	r.delegate.isMainWorktree = r.mainWorktree
	r.delegate.activeWorktree = r.activeWorktree
	// Don't overwrite delegate.searching - it should only be set by SetActive or HandleKey
	// r.delegate.searching = r.searching
}

func (r *AgentsPane) firstRepoName() string {
	for _, item := range r.items {
		if workItem, ok := item.(AgentListItem); ok && workItem.Type == "repo_header" {
			return workItem.RepoName
		}
	}
	return ""
}

func (r *AgentsPane) isActiveWorktree(worktree *git.WorktreeInfo) bool {
	if worktree == nil || r.activeWorktree == nil {
		git.DebugLog("[AgentsPane] isActiveWorktree: worktree=%v, r.activeWorktree=%v -> false", worktree, r.activeWorktree)
		return false
	}
	result := worktree.Path == r.activeWorktree.Path
	git.DebugLog("[AgentsPane] isActiveWorktree: comparing '%s' == '%s' -> %v", worktree.Path, r.activeWorktree.Path, result)
	return result
}

func (r *AgentsPane) setActiveWorktree(worktree *git.WorktreeInfo) {
	git.DebugLog("[AgentsPane] setActiveWorktree called with worktree: %v", worktree)
	if worktree == nil {
		git.DebugLog("[AgentsPane] setActiveWorktree: worktree is nil, returning")
		return
	}
	git.DebugLog("[AgentsPane] setActiveWorktree: setting to path=%s, branch=%s, r.searching=%v, r.delegate.searching=%v",
		worktree.Path, worktree.Branch, r.searching, r.delegate.searching)
	clone := *worktree
	r.activeWorktree = &clone
	if worktree.RepoName != "" {
		r.currentRepo = worktree.RepoName
	}
	r.delegate.currentRepo = r.currentRepo
	r.delegate.isMainWorktree = r.mainWorktree
	r.delegate.activeWorktree = r.activeWorktree
	// Don't overwrite delegate.searching - it may have been explicitly set to false during navigation
	// r.delegate.searching = r.searching
	git.DebugLog("[AgentsPane] setActiveWorktree: keeping delegate.searching=%v (not copying from r.searching=%v)",
		r.delegate.searching, r.searching)
	r.list.SetDelegate(r.delegate)
	if r.expandedRepos != nil {
		r.expandedRepos[r.currentRepo] = true
	}
	r.persistSelection(r.activeWorktree)
	git.DebugLog("[AgentsPane] setActiveWorktree: completed, r.activeWorktree.Path=%s", r.activeWorktree.Path)
}

func (r *AgentsPane) persistSelection(worktree *git.WorktreeInfo) {
	if worktree == nil {
		return
	}

	repoKey := strings.TrimSpace(worktree.RepoName)
	if repoKey == "" && r.sessionManager != nil {
		if worktreeManager := r.sessionManager.GetWorktreeManager(); worktreeManager != nil {
			repoKey = strings.TrimSpace(worktreeManager.GetRepositoryName())
		}
	}
	if repoKey == "" {
		return
	}

	samePath := false
	if r.lastSavedRepo == repoKey && r.lastSavedPath != "" && worktree.Path != "" {
		samePath = filepath.Clean(r.lastSavedPath) == filepath.Clean(worktree.Path)
	}

	if samePath && r.lastSavedBranch == worktree.Branch {
		return
	}

	worktreeRef := config.WorktreeRef{
		Path:   worktree.Path,
		Branch: worktree.Branch,
	}

	if err := config.SetLastWorktreeForRepo(repoKey, worktreeRef); err != nil {
		git.DebugLog("failed to persist workspace selection: %v", err)
		return
	}

	r.lastSavedRepo = repoKey
	r.lastSavedPath = worktree.Path
	r.lastSavedBranch = worktree.Branch
}

// isSelectableItem returns whether an item at the given index can be selected
func (r *AgentsPane) isSelectableItem(index int) bool {
	if index < 0 || index >= len(r.items) {
		return false
	}

	item, ok := r.items[index].(AgentListItem)
	if !ok {
		return false
	}

	// Only session items are selectable
	if item.Type == "session" {
		return true
	}

	return false
}

// jumpToActiveSession selects the currently active session in the list
func (r *AgentsPane) jumpToActiveSession() {
	if r.sessionManager == nil {
		return
	}

	activeSession := r.sessionManager.GetActiveSession()
	if activeSession == nil || activeSession.Worktree() == nil {
		return
	}

	// Find the active session in the list
	for idx, item := range r.items {
		if workItem, ok := item.(AgentListItem); ok {
			if workItem.Type == "session" &&
				workItem.Worktree != nil &&
				workItem.Worktree.Path == activeSession.Worktree().Path {
				// Ensure this item is selectable
				if r.isSelectableItem(idx) {
					r.list.Select(idx)
					return
				}
			}
		}
	}
}

func (r *AgentsPane) rebuildListPreservingSelection(selectedIndex int) {
	r.buildItemList()
	r.list.SetItems(r.items)
	if len(r.items) == 0 {
		return
	}
	if selectedIndex < 0 {
		selectedIndex = 0
	}
	if selectedIndex >= len(r.items) {
		selectedIndex = len(r.items) - 1
	}

	// Ensure we select a selectable item
	if !r.isSelectableItem(selectedIndex) {
		// Find the next selectable item
		for i := selectedIndex; i < len(r.items); i++ {
			if r.isSelectableItem(i) {
				selectedIndex = i
				break
			}
		}
		// If not found forward, search backward
		if !r.isSelectableItem(selectedIndex) {
			for i := selectedIndex; i >= 0; i-- {
				if r.isSelectableItem(i) {
					selectedIndex = i
					break
				}
			}
		}
	}

	r.list.Select(selectedIndex)
}

// SetCurrentRepo sets the current repository
func (r *AgentsPane) SetCurrentRepo(repoName string) {
	r.currentRepo = repoName
	if r.mainWorktree != nil && r.mainWorktree.RepoName == repoName {
		r.setActiveWorktree(r.mainWorktree)
	}
	r.delegate.currentRepo = repoName
	r.delegate.expandedRepos = r.expandedRepos
	r.delegate.isMainWorktree = r.mainWorktree
	r.delegate.activeWorktree = r.activeWorktree
	r.rebuildListPreservingSelection(r.list.Index())
}

// GetPaneSpecificKeybindings returns repo worktree pane specific keybindings
func (r *AgentsPane) GetPaneSpecificKeybindings() []key.Binding {
	// Use the global keybindings to ensure consistency
	return []key.Binding{
		common.GlobalKeys.NewSession,
	}
}

// DeleteSessionRequestMsg is sent when the user requests to delete a session
type DeleteSessionRequestMsg struct {
	Session *session.Session
}

// SessionSwitchedMsg is sent when the selection changes and a session is auto-switched
type SessionSwitchedMsg struct {
	Session *session.Session
}

// AttachToSessionMsg is sent when the user wants to attach to a tmux session
type AttachToSessionMsg struct {
	Session *session.Session
}
