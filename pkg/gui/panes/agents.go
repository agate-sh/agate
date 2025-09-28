package panes

import (
	"agate/pkg/app"
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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
}

// AgentListItem implements list.Item interface for agent sessions
type AgentListItem struct {
	Type         string // "repo_header", "section_header", "main_session", "linked_session", "empty_message"
	RepoName     string
	RepoPath     string // Full path to repository
	Worktree     *git.WorktreeInfo
	Index        int // Index in original repo list
	IsSelected   bool
	SectionTitle string // For section headers: "Main worktree" or "Linked worktrees"
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

	highlight := selected && d.isActive

	var linePlain string
	var lineStyled string
	var hint string

	switch workItem.Type {
	case "repo_header":
		repoName := workItem.RepoName
		var arrow string
		if d.expandedRepos != nil && d.expandedRepos[repoName] {
			arrow = "▼"
		} else {
			arrow = "▶"
		}

		// Calculate spacing to push arrow to the right
		leftContentWidth := lipgloss.Width(repoName)
		arrowWidth := lipgloss.Width(arrow)
		availableSpace := innerWidth - leftContentWidth - arrowWidth
		if availableSpace < 1 {
			availableSpace = 1
		}
		spacing := strings.Repeat(" ", availableSpace)

		linePlain = repoName + spacing + arrow

		if highlight {
			lineStyled = linePlain
		} else {
			arrowStyled := d.styles.repoCurrent.Render(arrow)
			if workItem.RepoName == d.currentRepo {
				nameStyled := d.styles.repoCurrent.Render(repoName)
				spacingStyled := strings.Repeat(" ", availableSpace)
				lineStyled = nameStyled + spacingStyled + arrowStyled
			} else {
				nameStyled := d.styles.repoHeader.Render(repoName)
				spacingStyled := strings.Repeat(" ", availableSpace)
				lineStyled = nameStyled + spacingStyled + arrowStyled
			}
		}

	case "section_header":
		// Section headers with Nerd Font icons
		var icon string
		if workItem.SectionTitle == "Main worktree" {
			icon = "\uf07b" // Nerd Font folder icon
		} else if workItem.SectionTitle == "Linked worktrees" {
			icon = "\uf0c1" // Nerd Font link icon
		}
		iconStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted)).Render(icon)
		sectionStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription)).Render(workItem.SectionTitle)
		lineStyled = iconStyled + "  " + sectionStyled

	case "gap":
		// Empty line for spacing
		linePlain = ""
		lineStyled = ""

	case "main_session":
		if workItem.Worktree == nil {
			return
		}
		label := workItem.Worktree.Branch
		if strings.TrimSpace(label) == "" {
			label = workItem.Worktree.Name
		}
		branchIcon := "\ue0a0" // Nerd Font git branch icon
		linePlain = "   " + branchIcon + "  " + label
		if highlight {
			lineStyled = linePlain
		} else {
			lineStyled = d.styles.normalItem.Render(linePlain)
		}

	case "linked_session":
		if workItem.Worktree == nil {
			return
		}
		label := workItem.Worktree.Branch
		if strings.TrimSpace(label) == "" {
			label = workItem.Worktree.Name
		}
		branchIcon := "\ue0a0" // Nerd Font git branch icon
		linePlain = "   " + branchIcon + "  " + label
		if highlight {
			lineStyled = linePlain
		} else {
			lineStyled = d.styles.normalItem.Render(linePlain)
		}

	case "empty_message":
		// Show empty state message
		if workItem.SectionTitle == "main" {
			linePlain = "   No main session"
		} else if workItem.SectionTitle == "linked" {
			linePlain = "   No agents - n to create"
		} else {
			linePlain = "   No agents - n to create"
		}
		lineStyled = d.styles.mustedText.Render(linePlain)

		// Add the hint about creating a new agent for linked worktrees
		if highlight && workItem.SectionTitle == "linked" {
			hint = " n to create a new agent"
		}

	default:
		return
	}

	// Handle hint text - only when pane is active and hovering a session item
	if hint == "" && d.isActive && highlight && (workItem.Type == "main_session" || workItem.Type == "linked_session") {
		// Use the same logic as the orange bar - workItem.IsSelected indicates active/selected
		if workItem.IsSelected {
			// Hovering a row that is already selected (has orange bar) - show "enter to open"
			hint = " ↵ to open"
			if workItem.Type == "linked_session" {
				hint = " ↵ to open, D to delete"
			}
		} else {
			// Hovering a row that is not selected (no orange bar) - show "enter to select"
			hint = " ↵ to select"
			if workItem.Type == "linked_session" {
				hint = " ↵ to select, D to delete"
			}
		}
	}

	contentWidth := innerWidth
	var body string
	activeHighlight := highlight || workItem.IsSelected
	if activeHighlight {
		bodyStyle := d.styles.selectedItem
		if workItem.Type == "repo_header" {
			bodyStyle = bodyStyle.Bold(true)
		}

		if hint != "" {
			hintStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(theme.RowHighlight)).
				Foreground(lipgloss.Color(theme.TextMuted))
			hintRendered := hintStyle.Render(hint)
			hintWidth := lipgloss.Width(hintRendered)
			available := contentWidth - hintWidth
			if available < 0 {
				available = 0
			}
			bodySegment := bodyStyle.Width(available).Render(linePlain)
			body = bodySegment + hintRendered
			if lipgloss.Width(body) < contentWidth {
				body += bodyStyle.Width(contentWidth - lipgloss.Width(body)).Render("")
			}
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

	if workItem.IsSelected && paddingLeft > 0 {
		cursorRune := components.BlinkingCursor.Frames[0]
		if len(cursorRune) == 0 {
			cursorRune = "█"
		}
		agentColor := app.GetCurrentAgentColor()
		cursorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(agentColor)).
			Background(lipgloss.Color(theme.RowHighlight)).
			Render(cursorRune)
		remaining := paddingLeft - 1
		if remaining < 0 {
			remaining = 0
		}
		leftBorder = cursorStyle
		if remaining > 0 {
			leftBorder += baseHighlightStyle.Render(strings.Repeat(" ", remaining))
		}
	} else {
		leftBorder = baseHighlightStyle.Render(leftPad)
	}

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

	pane := &AgentsPane{
		BasePane:       components.NewBasePane(0, "Agents"), // Updated to Agents
		list:           l,
		sessionManager: sessionManager,
		delegate:       delegate,
		expandedRepos:  expandedRepos, // Use the same map reference
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
	r.list.SetSize(fullWidth, height)
	// Update delegate widths so row highlighting spans the padded content
	r.delegate.styles.innerWidth = width
	r.delegate.styles.fullWidth = fullWidth
}

// SetActive sets whether the pane is focused and updates row highlighting state
func (r *AgentsPane) SetActive(active bool) {
	r.BasePane.SetActive(active)
	r.delegate.isActive = active
	r.list.SetDelegate(r.delegate)

	// Refresh when becoming active to pick up any new sessions
	if active {
		r.Refresh()
		// Jump to active session if one exists
		r.jumpToActiveSession()
	}
}

// GetTitleStyle returns the title style for the repo worktree pane
func (r *AgentsPane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if r.IsActive() {
		// When active, format shortcuts like the footer (without brackets)
		repoHelp := common.GlobalKeys.AddRepo.Help()
		sessionHelp := common.GlobalKeys.NewWorktree.Help()
		shortcuts = fmt.Sprintf("%s %s • %s %s", repoHelp.Key, repoHelp.Desc, sessionHelp.Key, sessionHelp.Desc)
	} else {
		// When not active, show pane number
		shortcuts = "(0)"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Agents",
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
		// Count actual session items (not headers/gaps)
		sessionItemCount := 0
		for _, item := range r.items {
			if agentItem, ok := item.(AgentListItem); ok {
				if agentItem.Type == "main_session" || agentItem.Type == "linked_session" {
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

	return r.list.View()
}

// Update handles tea.Msg updates for the repo worktree pane
func (r *AgentsPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

// HandleKey processes keyboard input when the pane is active
func (r *AgentsPane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
	if !r.IsActive() {
		return false, nil
	}

	switch key {
	case "up", "k":
		r.MoveUp()
		return true, nil
	case "down", "j":
		r.MoveDown()
		return true, nil
	case "enter":
		// Toggle repository expansion or attach to tmux session
		if len(r.items) > 0 {
			currentIndex := r.list.Index()
			selectedItem := r.list.SelectedItem()
			if workItem, ok := selectedItem.(AgentListItem); ok {
				if workItem.Type == "repo_header" {
					// Toggle expansion state
					r.expandedRepos[workItem.RepoName] = !r.expandedRepos[workItem.RepoName]
					// Update the delegate with the current expandedRepos state
					r.delegate.expandedRepos = r.expandedRepos
					// Update the list's delegate
					r.list.SetDelegate(r.delegate)
					r.rebuildListPreservingSelection(currentIndex)
				} else if (workItem.Type == "main_session" || workItem.Type == "linked_session") && workItem.Worktree != nil {
					// Check if this worktree is already selected
					if r.isActiveWorktree(workItem.Worktree) {
						// Already selected - attach to tmux session
						if r.sessionManager != nil {
							if session := r.sessionManager.GetSessionForWorktree(workItem.Worktree); session != nil && session.TmuxSession != nil {
								// Return a command that triggers tmux attachment
								return true, func() tea.Msg {
									return AttachToSessionMsg{Session: session}
								}
							}
						}
					} else {
						// Not selected yet - select it
						r.setActiveWorktree(workItem.Worktree)
						r.rebuildListPreservingSelection(currentIndex)
					}
				}
			}
		}
		return true, nil
	case "D":
		// Delete selected session
		if len(r.items) > 0 {
			selectedItem := r.list.SelectedItem()
			if workItem, ok := selectedItem.(AgentListItem); ok && workItem.Type == "linked_session" && workItem.Worktree != nil {
				// Only allow deletion of linked worktree sessions, not main sessions
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

// MoveUp moves the selection up one item
func (r *AgentsPane) MoveUp() bool {
	r.moveUp()
	return true
}

// MoveDown moves the selection down one item
func (r *AgentsPane) MoveDown() bool {
	r.moveDown()
	return true
}

// moveUp navigates up to the previous selectable item
func (r *AgentsPane) moveUp() {
	if len(r.items) == 0 {
		return
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
			return
		}
	}
}

// moveDown navigates down to the next selectable item
func (r *AgentsPane) moveDown() {
	if len(r.items) == 0 {
		return
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
			return
		}
	}
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
		if workItem.Type == "session" {
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
	if strings.TrimSpace(worktreePath) == "" {
		return false
	}

	target := filepath.Clean(worktreePath)

	for idx, item := range r.items {
		workItem, ok := item.(AgentListItem)
		if !ok || workItem.Type != "session" || workItem.Worktree == nil {
			continue
		}

		if filepath.Clean(workItem.Worktree.Path) == target {
			r.setActiveWorktree(workItem.Worktree)
			r.rebuildListPreservingSelection(idx)
			return true
		}
	}

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
		if sess.Worktree != nil && sess.Worktree.RepoName != "" {
			r.groupedSessions[sess.Worktree.RepoName] = append(r.groupedSessions[sess.Worktree.RepoName], sess)
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

	// Update the list with new items
	r.list.SetItems(r.items)

	return nil
}

// buildItemList creates a list of items for the bubbles list
func (r *AgentsPane) buildItemList() {
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

	// Initialize expanded state for first repo if needed
	if len(r.expandedRepos) == 0 && len(repoNames) > 0 {
		r.expandedRepos[repoNames[0]] = true
	}

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

		// Only add sessions if repository is expanded
		if r.expandedRepos[repoName] {
			// Get main and linked sessions for this repository
			mainSession := r.sessionManager.GetMainSession(repoName)
			linkedSessions := r.sessionManager.GetLinkedSessions(repoName)

			// Add "Main worktree" section header
			r.items = append(r.items, AgentListItem{
				Type:         "section_header",
				RepoName:     repoName,
				SectionTitle: "Main worktree",
			})

			// Add main worktree session
			if mainSession != nil && mainSession.Worktree != nil {
				worktreeCopy := *mainSession.Worktree
				r.items = append(r.items, AgentListItem{
					Type:       "main_session",
					RepoName:   repoName,
					Worktree:   &worktreeCopy,
					IsSelected: r.isActiveWorktree(&worktreeCopy),
				})
			} else {
				// Show placeholder if no main session
				r.items = append(r.items, AgentListItem{
					Type:         "empty_message",
					RepoName:     repoName,
					SectionTitle: "main",
				})
			}

			// Add gap between sections
			r.items = append(r.items, AgentListItem{
				Type:     "gap",
				RepoName: repoName,
			})

			// Add "Linked worktrees" section header
			r.items = append(r.items, AgentListItem{
				Type:         "section_header",
				RepoName:     repoName,
				SectionTitle: "Linked worktrees",
			})

			// Add linked worktree sessions
			if len(linkedSessions) > 0 {
				// Sort linked sessions by branch name
				sort.Slice(linkedSessions, func(i, j int) bool {
					if linkedSessions[i].Worktree != nil && linkedSessions[j].Worktree != nil {
						return linkedSessions[i].Worktree.Branch < linkedSessions[j].Worktree.Branch
					}
					return linkedSessions[i].Name < linkedSessions[j].Name
				})

				for _, sess := range linkedSessions {
					if sess.Worktree != nil {
						worktreeCopy := *sess.Worktree
						r.items = append(r.items, AgentListItem{
							Type:       "linked_session",
							RepoName:   repoName,
							Worktree:   &worktreeCopy,
							IsSelected: r.isActiveWorktree(&worktreeCopy),
						})
					}
				}
			} else {
				// No linked worktrees - add placeholder
				r.items = append(r.items, AgentListItem{
					Type:         "empty_message",
					RepoName:     repoName,
					SectionTitle: "linked",
				})
			}
		}
	}

	// Update delegate's current repo, expanded state, and main worktree
	r.delegate.currentRepo = r.currentRepo
	r.delegate.expandedRepos = r.expandedRepos
	r.delegate.isMainWorktree = r.mainWorktree
	r.delegate.activeWorktree = r.activeWorktree
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
		return false
	}
	return worktree.Path == r.activeWorktree.Path
}

func (r *AgentsPane) setActiveWorktree(worktree *git.WorktreeInfo) {
	if worktree == nil {
		return
	}
	clone := *worktree
	r.activeWorktree = &clone
	if worktree.RepoName != "" {
		r.currentRepo = worktree.RepoName
	}
	r.delegate.currentRepo = r.currentRepo
	r.delegate.isMainWorktree = r.mainWorktree
	r.delegate.activeWorktree = r.activeWorktree
	if r.expandedRepos != nil {
		r.expandedRepos[r.currentRepo] = true
	}
	r.persistSelection(r.activeWorktree)
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

	// Gap items, section headers, and "No agents - n to create" messages are not selectable
	if item.Type == "gap" || item.Type == "empty_message" || item.Type == "section_header" {
		return false
	}

	return true
}

// jumpToActiveSession selects the currently active session in the list
func (r *AgentsPane) jumpToActiveSession() {
	if r.sessionManager == nil {
		return
	}

	activeSession := r.sessionManager.GetActiveSession()
	if activeSession == nil || activeSession.Worktree == nil {
		return
	}

	// Find the active session in the list
	for idx, item := range r.items {
		if workItem, ok := item.(AgentListItem); ok {
			if (workItem.Type == "main_session" || workItem.Type == "linked_session") &&
				workItem.Worktree != nil &&
				workItem.Worktree.Path == activeSession.Worktree.Path {
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
		common.GlobalKeys.AddRepo,
		common.GlobalKeys.NewWorktree,
		common.GlobalKeys.DeleteWorktree,
	}
}

// DeleteSessionRequestMsg is sent when the user requests to delete a session
type DeleteSessionRequestMsg struct {
	Session *session.Session
}

// AttachToSessionMsg is sent when the user wants to attach to a tmux session
type AttachToSessionMsg struct {
	Session *session.Session
}
