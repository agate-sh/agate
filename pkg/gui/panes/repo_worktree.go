package panes

import (
	"agate/pkg/app"
	"agate/pkg/common"
	"agate/pkg/config"
	"agate/pkg/gui/components"
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

// RepoWorktreePane manages the display of repositories and worktrees
type RepoWorktreePane struct {
	*components.BasePane
	list             list.Model
	worktreeManager  *git.WorktreeManager
	groupedWorktrees map[string][]git.WorktreeInfo
	currentRepo      string
	activeWorktree   *git.WorktreeInfo
	mainWorktree     *git.WorktreeInfo
	items            []list.Item
	delegate         itemDelegate
	expandedRepos    map[string]bool // Track which repos are expanded
	lastSavedPath    string
	lastSavedBranch  string
	lastSavedRepo    string
	isGitRepo        bool
}

// WorktreeListItem implements list.Item interface for worktrees
type WorktreeListItem struct {
	Type       string // "repo_header" or "worktree"
	RepoName   string
	RepoPath   string // Full path to repository
	Worktree   *git.WorktreeInfo
	Index      int // Index in original repo list
	IsSelected bool
}

// FilterValue implements list.Item
func (i WorktreeListItem) FilterValue() string {
	if i.Type == "worktree" && i.Worktree != nil {
		return i.Worktree.Name
	}
	return i.RepoName
}

// itemDelegate handles rendering of individual list items
type itemDelegate struct {
	currentRepo   string
	styles        *itemStyles
	expandedRepos map[string]bool
	isActive      bool
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
	workItem, ok := item.(WorktreeListItem)
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

	switch workItem.Type {
	case "repo_header":
		repoName := workItem.RepoName
		var arrow string
		if d.expandedRepos != nil && d.expandedRepos[repoName] {
			arrow = "▼ "
		} else {
			arrow = "▶ "
		}

		linePlain = arrow + repoName
		if workItem.RepoName == d.currentRepo {
			linePlain += " (current)"
		}

		if highlight {
			lineStyled = linePlain
		} else {
			arrowStyled := d.styles.mustedText.Render(arrow)
			if workItem.RepoName == d.currentRepo {
				nameStyled := d.styles.repoCurrent.Render(repoName)
				currentStyled := d.styles.mustedText.Render(" (current)")
				lineStyled = arrowStyled + nameStyled + currentStyled
			} else {
				nameStyled := d.styles.repoHeader.Render(repoName)
				lineStyled = arrowStyled + nameStyled
			}
		}

	case "worktree":
		if workItem.Worktree == nil {
			return
		}
		label := workItem.Worktree.Branch
		if strings.TrimSpace(label) == "" {
			label = workItem.Worktree.Name
		}
		linePlain = "    " + label
		if highlight {
			lineStyled = linePlain
		} else {
			lineStyled = d.styles.normalItem.Render(linePlain)
		}

	default:
		return
	}

	hint := ""
	if highlight && workItem.Type == "worktree" && !workItem.IsSelected {
		hint = " ⏎"
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

// NewRepoWorktreePane creates a new RepoWorktreePane instance
func NewRepoWorktreePane(worktreeManager *git.WorktreeManager) *RepoWorktreePane {
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

	pane := &RepoWorktreePane{
		BasePane:        components.NewBasePane(0, "Repos & Worktrees"), // Pane index 0
		list:            l,
		worktreeManager: worktreeManager,
		delegate:        delegate,
		expandedRepos:   expandedRepos, // Use the same map reference
	}

	// Initial refresh
	if worktreeManager != nil {
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
		pane.Refresh()

		if len(repoSelections) > 0 {
			if pane.isGitRepo {
				repoName := pane.currentRepo
				if repoName != "" {
					if selection, ok := repoSelections[repoName]; ok {
						pane.lastSavedRepo = repoName
						pane.lastSavedPath = selection.Worktree.Path
						pane.lastSavedBranch = selection.Worktree.Branch
						if !pane.SelectWorktreeByRef(repoName, selection.Worktree) && pane.mainWorktree != nil {
							pane.setActiveWorktree(pane.mainWorktree)
						}
					} else if pane.mainWorktree != nil {
						pane.setActiveWorktree(pane.mainWorktree)
					}
				}
			} else {
				if firstRepo := pane.firstRepoName(); firstRepo != "" {
					if selection, ok := repoSelections[firstRepo]; ok {
						pane.lastSavedRepo = firstRepo
						pane.lastSavedPath = selection.Worktree.Path
						pane.lastSavedBranch = selection.Worktree.Branch
						pane.SelectWorktreeByRef(firstRepo, selection.Worktree)
					}
				}
			}
		} else if pane.isGitRepo && pane.mainWorktree != nil {
			// Persist default selection for current repo when none saved yet.
			pane.setActiveWorktree(pane.mainWorktree)
		}
	}

	return pane
}

// SetSize updates the dimensions of the repo worktree pane
func (r *RepoWorktreePane) SetSize(width, height int) {
	r.BasePane.SetSize(width, height)
	fullWidth := components.PaneFullWidth(width)
	r.list.SetSize(fullWidth, height)
	// Update delegate widths so row highlighting spans the padded content
	r.delegate.styles.innerWidth = width
	r.delegate.styles.fullWidth = fullWidth
}

// SetActive sets whether the pane is focused and updates row highlighting state
func (r *RepoWorktreePane) SetActive(active bool) {
	r.BasePane.SetActive(active)
	r.delegate.isActive = active
	r.list.SetDelegate(r.delegate)
}

// GetTitleStyle returns the title style for the repo worktree pane
func (r *RepoWorktreePane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if r.IsActive() {
		// When active, show the shortcut hints for repo and worktree actions
		repoHelp := common.GlobalKeys.AddRepo.Help()
		worktreeHelp := common.GlobalKeys.NewWorktree.Help()
		shortcuts = fmt.Sprintf("[%s: %s, %s: %s]", repoHelp.Key, repoHelp.Desc, worktreeHelp.Key, worktreeHelp.Desc)
	} else {
		// When not active, show pane number
		shortcuts = "[0]"
	}

	return components.TitleStyle{
		Type:      "plain",
		Color:     "",
		Text:      "Repos & Worktrees",
		Shortcuts: shortcuts,
	}
}

// View renders the repo worktree pane content
func (r *RepoWorktreePane) View() string {
	if r.worktreeManager == nil {
		// Show placeholder message
		style := lipgloss.NewStyle().
			Width(components.PaneFullWidth(r.GetWidth())).
			Height(r.GetHeight()).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(theme.TextMuted))

		return style.Render("Worktree manager not available")
	}

	return r.list.View()
}

// Update handles tea.Msg updates for the repo worktree pane
func (r *RepoWorktreePane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

// HandleKey processes keyboard input when the pane is active
func (r *RepoWorktreePane) HandleKey(key string) (handled bool, cmd tea.Cmd) {
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
		// Toggle repository expansion
		if len(r.items) > 0 {
			currentIndex := r.list.Index()
			selectedItem := r.list.SelectedItem()
			if workItem, ok := selectedItem.(WorktreeListItem); ok {
				if workItem.Type == "repo_header" {
					// Toggle expansion state
					r.expandedRepos[workItem.RepoName] = !r.expandedRepos[workItem.RepoName]
					// Update the delegate with the current expandedRepos state
					r.delegate.expandedRepos = r.expandedRepos
					// Update the list's delegate
					r.list.SetDelegate(r.delegate)
					r.rebuildListPreservingSelection(currentIndex)
				} else if workItem.Type == "worktree" && workItem.Worktree != nil {
					r.setActiveWorktree(workItem.Worktree)
					r.rebuildListPreservingSelection(currentIndex)
				}
			}
		}
		return true, nil
	default:
		return false, nil
	}
}

// MoveUp moves the selection up one item
func (r *RepoWorktreePane) MoveUp() bool {
	r.moveUp()
	return true
}

// MoveDown moves the selection down one item
func (r *RepoWorktreePane) MoveDown() bool {
	r.moveDown()
	return true
}

// moveUp navigates up to the previous visible item
func (r *RepoWorktreePane) moveUp() {
	if len(r.items) == 0 {
		return
	}

	currentIndex := r.list.Index()

	// Move up to previous item
	if currentIndex > 0 {
		r.list.Select(currentIndex - 1)
	} else {
		// Wrap to bottom
		r.list.Select(len(r.items) - 1)
	}
}

// moveDown navigates down to the next visible item
func (r *RepoWorktreePane) moveDown() {
	if len(r.items) == 0 {
		return
	}

	currentIndex := r.list.Index()

	// Move down to next item
	if currentIndex < len(r.items)-1 {
		r.list.Select(currentIndex + 1)
	} else {
		// Wrap to top
		r.list.Select(0)
	}
}

// GetSelectedWorktree returns the currently selected worktree
func (r *RepoWorktreePane) GetSelectedWorktree() *git.WorktreeInfo {
	if r.activeWorktree != nil {
		clone := *r.activeWorktree
		return &clone
	}

	if len(r.items) == 0 {
		return nil
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(WorktreeListItem); ok {
		if workItem.Type == "worktree" {
			clone := *workItem.Worktree
			return &clone
		}
	}
	return nil
}

// SelectWorktreeByRef attempts to select a worktree using the provided repo name and reference.
func (r *RepoWorktreePane) SelectWorktreeByRef(repoName string, ref config.WorktreeRef) bool {
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
func (r *RepoWorktreePane) SelectWorktreeByPath(worktreePath string) bool {
	if strings.TrimSpace(worktreePath) == "" {
		return false
	}

	target := filepath.Clean(worktreePath)

	for idx, item := range r.items {
		workItem, ok := item.(WorktreeListItem)
		if !ok || workItem.Type != "worktree" || workItem.Worktree == nil {
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

func (r *RepoWorktreePane) selectWorktreeByBranch(repoName, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}

	for idx, item := range r.items {
		workItem, ok := item.(WorktreeListItem)
		if !ok || workItem.Type != "worktree" || workItem.Worktree == nil {
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

// HasWorktrees returns whether there are any worktrees available
func (r *RepoWorktreePane) HasWorktrees() bool {
	return len(r.groupedWorktrees) > 0
}

// HasItems returns whether there are any items to display
func (r *RepoWorktreePane) HasItems() bool {
	return len(r.items) > 0
}

// Refresh refreshes the worktree list
func (r *RepoWorktreePane) Refresh() error {
	if r.worktreeManager == nil {
		return nil
	}

	groups, err := r.worktreeManager.ListWorktrees()
	if err != nil {
		return err
	}

	if mainWorktree, err := r.worktreeManager.GetMainWorktreeInfo(); err == nil {
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

	r.groupedWorktrees = groups
	r.buildItemList()

	// Update the list with new items
	r.list.SetItems(r.items)

	return nil
}

// buildItemList creates a list of items for the bubbles list
func (r *RepoWorktreePane) buildItemList() {
	r.items = nil

	// Get sorted repository names (current repo first)
	repoNames := make([]string, 0, len(r.groupedWorktrees))
	for repoName := range r.groupedWorktrees {
		repoNames = append(repoNames, repoName)
	}

	// Always include the current repository if we have a worktree manager
	// This ensures the main repo is shown even when no worktrees exist
	if r.worktreeManager != nil && r.currentRepo != "" {
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
		if r.worktreeManager != nil && repoName == r.currentRepo {
			mainRepoPath = r.worktreeManager.GetRepositoryPath()
		} else {
			mainRepoPath = repoName
		}

		r.items = append(r.items, WorktreeListItem{
			Type:     "repo_header",
			RepoName: repoName,
			RepoPath: mainRepoPath,
		})

		// Only add worktrees if repository is expanded
		if r.expandedRepos[repoName] {
			if repoName == r.currentRepo && r.mainWorktree != nil {
				mainCopy := *r.mainWorktree
				r.items = append(r.items, WorktreeListItem{
					Type:       "worktree",
					RepoName:   repoName,
					Worktree:   &mainCopy,
					IsSelected: r.isActiveWorktree(&mainCopy),
				})
			}
			// Add worktrees for this repository
			if worktrees, exists := r.groupedWorktrees[repoName]; exists {
				// Sort worktrees by name
				sort.Slice(worktrees, func(i, j int) bool {
					return worktrees[i].Name < worktrees[j].Name
				})

				for _, worktree := range worktrees {
					wt := worktree
					r.items = append(r.items, WorktreeListItem{
						Type:       "worktree",
						RepoName:   repoName,
						Worktree:   &wt,
						IsSelected: r.isActiveWorktree(&wt),
					})
				}
			}
		}
	}

	// Update delegate's current repo and expanded state
	r.delegate.currentRepo = r.currentRepo
	r.delegate.expandedRepos = r.expandedRepos
}

func (r *RepoWorktreePane) firstRepoName() string {
	for _, item := range r.items {
		if workItem, ok := item.(WorktreeListItem); ok && workItem.Type == "repo_header" {
			return workItem.RepoName
		}
	}
	return ""
}

func (r *RepoWorktreePane) isActiveWorktree(worktree *git.WorktreeInfo) bool {
	if worktree == nil || r.activeWorktree == nil {
		return false
	}
	return worktree.Path == r.activeWorktree.Path
}

func (r *RepoWorktreePane) setActiveWorktree(worktree *git.WorktreeInfo) {
	if worktree == nil {
		return
	}
	clone := *worktree
	r.activeWorktree = &clone
	if worktree.RepoName != "" {
		r.currentRepo = worktree.RepoName
	}
	r.delegate.currentRepo = r.currentRepo
	if r.expandedRepos != nil {
		r.expandedRepos[r.currentRepo] = true
	}
	r.persistSelection(r.activeWorktree)
}

func (r *RepoWorktreePane) persistSelection(worktree *git.WorktreeInfo) {
	if worktree == nil {
		return
	}

	repoKey := strings.TrimSpace(worktree.RepoName)
	if repoKey == "" && r.worktreeManager != nil {
		repoKey = strings.TrimSpace(r.worktreeManager.GetRepositoryName())
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

func (r *RepoWorktreePane) rebuildListPreservingSelection(selectedIndex int) {
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
	r.list.Select(selectedIndex)
}

// SetCurrentRepo sets the current repository
func (r *RepoWorktreePane) SetCurrentRepo(repoName string) {
	r.currentRepo = repoName
	if r.mainWorktree != nil && r.mainWorktree.RepoName == repoName {
		r.setActiveWorktree(r.mainWorktree)
	}
	r.delegate.currentRepo = repoName
	r.delegate.expandedRepos = r.expandedRepos
	r.rebuildListPreservingSelection(r.list.Index())
}

// GetPaneSpecificKeybindings returns repo worktree pane specific keybindings
func (r *RepoWorktreePane) GetPaneSpecificKeybindings() []key.Binding {
	// Repository and worktree management keybindings
	addRepo := key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "add repository"),
	)

	newWorktree := key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "new worktree"),
	)

	deleteWorktree := key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete worktree"),
	)

	return []key.Binding{addRepo, newWorktree, deleteWorktree}
}
