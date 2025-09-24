package panes

import (
	"agate/pkg/gui/components"
	"fmt"
	"io"
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
	items            []list.Item
	delegate         itemDelegate
}

// WorktreeListItem implements list.Item interface for worktrees
type WorktreeListItem struct {
	Type     string // "repo_header", "main_repo", or "worktree"
	RepoName string
	RepoPath string // Full path to repository (for main_repo type)
	Worktree *git.WorktreeInfo
	Index    int // Index in original repo list
}

// FilterValue implements list.Item
func (i WorktreeListItem) FilterValue() string {
	if i.Type == "worktree" && i.Worktree != nil {
		return i.Worktree.Name
	}
	if i.Type == "main_repo" {
		return "Main " + i.RepoName
	}
	return i.RepoName
}

// itemDelegate handles rendering of individual list items
type itemDelegate struct {
	currentRepo string
	styles      *itemStyles
}

type itemStyles struct {
	repoHeader           lipgloss.Style
	worktreeItem         lipgloss.Style
	worktreeSelectedItem lipgloss.Style
	statusClean          lipgloss.Style
	statusDirty          lipgloss.Style
	statusInfo           lipgloss.Style
	repoCurrent          lipgloss.Style
	selectedItem         lipgloss.Style
	normalItem           lipgloss.Style
}

func newItemStyles() *itemStyles {
	return &itemStyles{
		repoHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.InfoStatus)).
			MarginTop(1),
		worktreeItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextPrimary)), // White for worktree names
		worktreeSelectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.HighlightBg)). // Black text
			Background(lipgloss.Color(theme.InfoStatus)).
			Bold(true),
		statusClean: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.SuccessStatus)),
		statusDirty: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.WarningStatus)),
		statusInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)), // Gray for branch info
		repoCurrent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Selection)).
			Bold(true),
		selectedItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(theme.Selection)),
		normalItem: lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color(theme.TextDescription)),
	}
}

// Height implements list.ItemDelegate
func (d itemDelegate) Height() int {
	return 2 // Each item takes 2 lines (main + status)
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
	content := ""

	if workItem.Type == "repo_header" {
		// Repository header
		repoName := workItem.RepoName
		if repoName == d.currentRepo {
			content = d.styles.repoCurrent.Render("📁 " + repoName + " (current)")
		} else {
			content = d.styles.repoHeader.Render("📁 " + repoName)
		}
	} else if workItem.Type == "main_repo" {
		// Main repository item
		line := "Main"

		if selected {
			line = d.styles.selectedItem.Render("▶ " + line)
		} else {
			line = d.styles.normalItem.Render(line)
		}

		// Status line for main repo
		statusLine := "└── main repository"
		if selected {
			statusLine = d.styles.selectedItem.Render("  " + statusLine)
		} else {
			statusLine = d.styles.statusInfo.Render("    " + statusLine)
		}

		content = line + "\n" + statusLine
	} else if workItem.Type == "worktree" && workItem.Worktree != nil {
		// Format worktree line
		line := workItem.Worktree.Name

		// Apply theme.Selection styling
		if selected {
			line = d.styles.selectedItem.Render("▶ " + line)
		} else {
			line = d.styles.normalItem.Render(line)
		}

		// Add status line
		statusLine := d.formatStatusLine(workItem.Worktree)
		if selected {
			statusLine = d.styles.selectedItem.Render("  " + statusLine)
		} else {
			statusLine = d.styles.statusInfo.Render("    " + statusLine)
		}

		content = line + "\n" + statusLine
	}

	if _, err := fmt.Fprint(w, content); err != nil {
		// Log error but continue - this is UI rendering
		// DebugLog("Failed to write worktree content to buffer: %v", err)
	}
}

// formatStatusLine formats the status line for a worktree
func (d itemDelegate) formatStatusLine(worktree *git.WorktreeInfo) string {
	if worktree.GitStatus == nil {
		return "└── " + worktree.Branch
	}

	status := worktree.GitStatus

	// Build status display
	var parts []string

	// Branch info
	branchInfo := status.Branch
	if status.HasRemote {
		remoteBranch := fmt.Sprintf("%s/%s", status.RemoteName, status.Branch)
		branchInfo = fmt.Sprintf("%s → %s", status.Branch, remoteBranch)

		// Add ahead/behind indicators
		if status.Ahead > 0 || status.Behind > 0 {
			var indicators []string
			if status.Ahead > 0 {
				indicators = append(indicators, fmt.Sprintf("↑%d", status.Ahead))
			}
			if status.Behind > 0 {
				indicators = append(indicators, fmt.Sprintf("↓%d", status.Behind))
			}
			branchInfo = fmt.Sprintf("%s %s", branchInfo, strings.Join(indicators, " "))
		}
	} else if status.Branch != "" {
		branchInfo = fmt.Sprintf("%s (no remote)", status.Branch)
	}

	parts = append(parts, branchInfo)

	// Status indicators
	if status.IsClean {
		parts = append(parts, "[clean]")
	} else {
		var statusParts []string
		if status.Modified > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%dM", status.Modified))
		}
		if status.Staged > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%dS", status.Staged))
		}
		if status.Untracked > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%dU", status.Untracked))
		}
		if len(statusParts) > 0 {
			parts = append(parts, "["+strings.Join(statusParts, ", ")+"]")
		}
	}

	// Stash info
	if status.Stashed > 0 {
		if status.Stashed == 1 {
			parts = append(parts, "[1 stash]")
		} else {
			parts = append(parts, fmt.Sprintf("[%d stashes]", status.Stashed))
		}
	}

	return "└── " + strings.Join(parts, " ")
}

// NewRepoWorktreePane creates a new RepoWorktreePane instance
func NewRepoWorktreePane(worktreeManager *git.WorktreeManager) *RepoWorktreePane {
	styles := newItemStyles()

	// Create delegate
	delegate := itemDelegate{
		styles: styles,
	}

	// Create list model with styles
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	pane := &RepoWorktreePane{
		BasePane:        components.NewBasePane(0, "Repos & Worktrees"), // Pane index 0
		list:            l,
		worktreeManager: worktreeManager,
		delegate:        delegate,
	}

	// Initial refresh
	if worktreeManager != nil {
		pane.Refresh()
	}

	return pane
}

// SetSize updates the dimensions of the repo worktree pane
func (r *RepoWorktreePane) SetSize(width, height int) {
	r.BasePane.SetSize(width, height)
	r.list.SetSize(width, height)
}

// GetTitleStyle returns the title style for the repo worktree pane
func (r *RepoWorktreePane) GetTitleStyle() components.TitleStyle {
	shortcuts := ""
	if r.IsActive() {
		// When active, show the shortcut hints for repo and worktree actions
		shortcuts = "[r: add repo, w: add worktree]"
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
			Width(r.GetWidth()).
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

// moveUp navigates up, skipping repo headers
func (r *RepoWorktreePane) moveUp() {
	if len(r.items) == 0 {
		return
	}

	currentIndex := r.list.Index()

	for i := currentIndex - 1; i >= 0; i-- {
		if item, ok := r.items[i].(WorktreeListItem); ok {
			if item.Type != "repo_header" {
				r.list.Select(i)
				return
			}
		}
	}

	// If we didn't find a valid item above, wrap to the bottom
	for i := len(r.items) - 1; i >= 0; i-- {
		if item, ok := r.items[i].(WorktreeListItem); ok {
			if item.Type != "repo_header" {
				r.list.Select(i)
				return
			}
		}
	}
}

// moveDown navigates down, skipping repo headers
func (r *RepoWorktreePane) moveDown() {
	if len(r.items) == 0 {
		return
	}

	currentIndex := r.list.Index()

	for i := currentIndex + 1; i < len(r.items); i++ {
		if item, ok := r.items[i].(WorktreeListItem); ok {
			if item.Type != "repo_header" {
				r.list.Select(i)
				return
			}
		}
	}

	// If we didn't find a valid item below, wrap to the top
	for i := 0; i < len(r.items); i++ {
		if item, ok := r.items[i].(WorktreeListItem); ok {
			if item.Type != "repo_header" {
				r.list.Select(i)
				return
			}
		}
	}
}

// GetSelectedWorktree returns the currently selected worktree
func (r *RepoWorktreePane) GetSelectedWorktree() *git.WorktreeInfo {
	if len(r.items) == 0 {
		return nil
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(WorktreeListItem); ok {
		if workItem.Type == "worktree" {
			return workItem.Worktree
		}
		// For "main_repo" type, we need to find the main repo path
		if workItem.Type == "main_repo" && workItem.RepoPath != "" {
			// Create a synthetic WorktreeInfo for the main repo
			return &git.WorktreeInfo{
				Name:   "main",
				Path:   workItem.RepoPath,
				Branch: "main", // Default branch name
			}
		}
	}
	return nil
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

	for _, repoName := range repoNames {
		// Add repository header
		r.items = append(r.items, WorktreeListItem{
			Type:     "repo_header",
			RepoName: repoName,
		})

		// Add main repository item
		var mainRepoPath string
		if r.worktreeManager != nil && repoName == r.currentRepo {
			mainRepoPath = r.worktreeManager.GetRepositoryPath()
		} else {
			// For other repos, derive the path from the repo name
			// The worktree structure puts repos under the worktree base directory
			mainRepoPath = repoName
		}

		r.items = append(r.items, WorktreeListItem{
			Type:     "main_repo",
			RepoName: repoName,
			RepoPath: mainRepoPath,
		})

		// Add worktrees for this repository
		if worktrees, exists := r.groupedWorktrees[repoName]; exists {
			// Sort worktrees by name
			sort.Slice(worktrees, func(i, j int) bool {
				return worktrees[i].Name < worktrees[j].Name
			})

			for _, worktree := range worktrees {
				r.items = append(r.items, WorktreeListItem{
					Type:     "worktree",
					RepoName: repoName,
					Worktree: &worktree,
				})
			}
		}
	}

	// Update delegate's current repo
	r.delegate.currentRepo = r.currentRepo
}

// SetCurrentRepo sets the current repository
func (r *RepoWorktreePane) SetCurrentRepo(repoName string) {
	r.currentRepo = repoName
	r.delegate.currentRepo = repoName
	r.buildItemList()
	r.list.SetItems(r.items)
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