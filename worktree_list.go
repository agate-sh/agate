package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"agate/git"
	"agate/icons"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
			Foreground(lipgloss.Color(infoStatus)).
			MarginTop(1),
		worktreeItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(textPrimary)), // White for worktree names
		worktreeSelectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(highlightBg)). // Black text
			Background(lipgloss.Color(infoStatus)).
			Bold(true),
		statusClean: lipgloss.NewStyle().
			Foreground(lipgloss.Color(successStatus)),
		statusDirty: lipgloss.NewStyle().
			Foreground(lipgloss.Color(warningStatus)),
		statusInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(textMuted)), // Gray for branch info
		repoCurrent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(selection)).
			Bold(true),
		selectedItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(selection)),
		normalItem: lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color(textDescription)),
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
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(WorktreeListItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	var content string

	switch item.Type {
	case "repo_header":
		// Repository header with icon
		gitIcon := icons.GetGitRepo()
		repoDisplay := gitIcon + " " + item.RepoName
		if item.RepoName == d.currentRepo {
			repoDisplay += " (current)"
			content = d.styles.repoCurrent.Render("┌─ " + repoDisplay + " ")
		} else {
			content = d.styles.repoHeader.Render("┌─ " + repoDisplay + " ")
		}

	case "main_repo":
		// Main repository entry with folder icon and path on same line
		folderIcon := icons.GetFolder()

		// Calculate available width for the path (rough estimate)
		// Account for icon, "Main", spaces, and padding
		availableWidth := 35 // Reduced to make room for path on same line
		truncatedPath := truncatePathFromLeft(item.RepoPath, availableWidth)

		// Put icon, "Main", and path all on the same line with proper spacing
		mainDisplay := folderIcon + "  Main " + d.styles.statusInfo.Render("("+truncatedPath+")")

		// Apply selection styling
		if selected {
			line := d.styles.selectedItem.Render("▶ " + mainDisplay)
			content = line + "\n" // Single line, but preserve the two-line structure for consistency
		} else {
			line := d.styles.normalItem.Render(mainDisplay)
			content = line + "\n" // Single line, but preserve the two-line structure for consistency
		}

	case "worktree":
		// Worktree item
		if item.Worktree == nil {
			return
		}

		// Format worktree line
		line := item.Worktree.Name

		// Apply selection styling
		if selected {
			line = d.styles.selectedItem.Render("▶ " + line)
		} else {
			line = d.styles.normalItem.Render(line)
		}

		// Add status line
		statusLine := d.formatStatusLine(item.Worktree)
		if selected {
			statusLine = d.styles.selectedItem.Render("  " + statusLine)
		} else {
			statusLine = d.styles.statusInfo.Render("    " + statusLine)
		}

		content = line + "\n" + statusLine
	}

	if _, err := fmt.Fprint(w, content); err != nil {
		// Log error but continue - this is UI rendering
		DebugLog("Failed to write worktree content to buffer: %v", err)
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

// WorktreeList manages the display and navigation of Git worktrees using bubbles list
type WorktreeList struct {
	list             list.Model
	worktreeManager  *git.WorktreeManager
	groupedWorktrees map[string][]git.WorktreeInfo
	currentRepo      string
	items            []list.Item
	delegate         itemDelegate
}

// NewWorktreeList creates a new WorktreeList instance
func NewWorktreeList(worktreeManager *git.WorktreeManager) *WorktreeList {
	styles := newItemStyles()

	// Create delegate
	delegate := itemDelegate{
		styles: styles,
	}

	// Get current repository name for prioritization
	if worktreeManager != nil {
		delegate.currentRepo = worktreeManager.GetRepositoryName()
	}

	// Create list model
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Worktrees"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()

	// Configure styles for the list itself
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(infoStatus))

	wl := &WorktreeList{
		list:            l,
		worktreeManager: worktreeManager,
		currentRepo:     delegate.currentRepo,
		delegate:        delegate,
	}

	// Load worktrees
	if err := wl.Refresh(); err != nil {
		DebugLog("Failed to refresh worktree list during initialization: %v", err)
		// Continue with empty list - UI will show "no repositories found"
	}

	return wl
}

// Refresh reloads the worktree list from the filesystem
func (wl *WorktreeList) Refresh() error {
	if wl.worktreeManager == nil {
		return fmt.Errorf("worktree manager not initialized")
	}

	groups, err := wl.worktreeManager.ListWorktrees()
	if err != nil {
		return err
	}

	wl.groupedWorktrees = groups
	wl.buildItemList()

	// Update the list with new items
	wl.list.SetItems(wl.items)

	return nil
}

// buildItemList creates a list of items for the bubbles list
func (wl *WorktreeList) buildItemList() {
	wl.items = nil

	// Get sorted repository names (current repo first)
	repoNames := make([]string, 0, len(wl.groupedWorktrees))
	for repoName := range wl.groupedWorktrees {
		repoNames = append(repoNames, repoName)
	}

	// Sort: current repo first, then alphabetical
	sort.Slice(repoNames, func(i, j int) bool {
		if repoNames[i] == wl.currentRepo {
			return true
		}
		if repoNames[j] == wl.currentRepo {
			return false
		}
		return repoNames[i] < repoNames[j]
	})

	// Build item list
	for _, repoName := range repoNames {
		worktrees := wl.groupedWorktrees[repoName]

		// Sort worktrees by creation time (newest first)
		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].CreatedAt.After(worktrees[j].CreatedAt)
		})

		// Add repository header (but skip it for selection)
		wl.items = append(wl.items, WorktreeListItem{
			Type:     "repo_header",
			RepoName: repoName,
		})

		// Add Main repository entry (selectable)
		var repoPath string
		if wl.worktreeManager != nil && repoName == wl.currentRepo {
			repoPath = wl.worktreeManager.GetRepositoryPath()
		} else {
			// For other repos, we'll need to derive the path somehow
			// For now, just use the repo name as a placeholder
			repoPath = repoName
		}

		wl.items = append(wl.items, WorktreeListItem{
			Type:     "main_repo",
			RepoName: repoName,
			RepoPath: repoPath,
		})

		// Add worktrees
		for i := range worktrees {
			wl.items = append(wl.items, WorktreeListItem{
				Type:     "worktree",
				RepoName: repoName,
				Worktree: &worktrees[i],
				Index:    i,
			})
		}
	}

	// Skip repo headers when setting initial selection
	if len(wl.items) > 0 {
		for i, item := range wl.items {
			if listItem, ok := item.(WorktreeListItem); ok && (listItem.Type == "worktree" || listItem.Type == "main_repo") {
				wl.list.Select(i)
				break
			}
		}
	}
}

// SetSize updates the dimensions of the worktree list
func (wl *WorktreeList) SetSize(width, height int) {
	wl.list.SetWidth(width)
	wl.list.SetHeight(height)
}

// Update handles tea.Msg updates for the worktree list
func (wl *WorktreeList) Update(msg tea.Msg) (*WorktreeList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle custom navigation to skip repo headers
		switch msg.String() {
		case "up", "k":
			wl.moveUp()
			return wl, nil
		case "down", "j":
			wl.moveDown()
			return wl, nil
		default:
			// Let the list handle other keys
			var cmd tea.Cmd
			wl.list, cmd = wl.list.Update(msg)
			return wl, cmd
		}
	default:
		var cmd tea.Cmd
		wl.list, cmd = wl.list.Update(msg)
		return wl, cmd
	}
}

// moveUp moves selection up, skipping repo headers
func (wl *WorktreeList) moveUp() {
	currentIndex := wl.list.Index()
	if currentIndex > 0 {
		currentIndex--
		// Skip repository headers
		for currentIndex > 0 {
			if item, ok := wl.items[currentIndex].(WorktreeListItem); ok {
				if item.Type == "worktree" || item.Type == "main_repo" {
					break
				}
			}
			currentIndex--
		}
		wl.list.Select(currentIndex)
	}
}

// moveDown moves selection down, skipping repo headers
func (wl *WorktreeList) moveDown() {
	currentIndex := wl.list.Index()
	if currentIndex < len(wl.items)-1 {
		currentIndex++
		// Skip repository headers
		for currentIndex < len(wl.items)-1 {
			if item, ok := wl.items[currentIndex].(WorktreeListItem); ok {
				if item.Type == "worktree" || item.Type == "main_repo" {
					break
				}
			}
			currentIndex++
		}
		wl.list.Select(currentIndex)
	}
}

// MoveUp moves the selection up (for compatibility)
func (wl *WorktreeList) MoveUp() {
	wl.moveUp()
}

// MoveDown moves the selection down (for compatibility)
func (wl *WorktreeList) MoveDown() {
	wl.moveDown()
}

// GetSelected returns the currently selected worktree
func (wl *WorktreeList) GetSelected() *git.WorktreeInfo {
	selectedItem := wl.list.SelectedItem()
	if selectedItem != nil {
		if item, ok := selectedItem.(WorktreeListItem); ok && item.Type == "worktree" {
			return item.Worktree
		}
	}
	return nil
}

// GetSelectedItem returns the currently selected item (worktree or main repo)
func (wl *WorktreeList) GetSelectedItem() *WorktreeListItem {
	selectedItem := wl.list.SelectedItem()
	if selectedItem != nil {
		if item, ok := selectedItem.(WorktreeListItem); ok && (item.Type == "worktree" || item.Type == "main_repo") {
			return &item
		}
	}
	return nil
}

// View renders the worktree list
func (wl *WorktreeList) View() string {
	if wl.worktreeManager == nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(textMuted)).
			Render("Worktree manager not available")
	}

	if len(wl.items) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(textMuted)).
			Render("No repositories found\n\nPress 'r' to add a repository")
	}

	return wl.list.View()
}

// HasWorktrees returns true if there are any worktrees
func (wl *WorktreeList) HasWorktrees() bool {
	for _, item := range wl.items {
		if listItem, ok := item.(WorktreeListItem); ok && listItem.Type == "worktree" {
			return true
		}
	}
	return false
}

// HasItems returns true if there are any selectable items (worktrees or main repos)
func (wl *WorktreeList) HasItems() bool {
	for _, item := range wl.items {
		if listItem, ok := item.(WorktreeListItem); ok && (listItem.Type == "worktree" || listItem.Type == "main_repo") {
			return true
		}
	}
	return false
}

// GetWorktreeCount returns the total number of worktrees
func (wl *WorktreeList) GetWorktreeCount() int {
	count := 0
	for _, item := range wl.items {
		if listItem, ok := item.(WorktreeListItem); ok && listItem.Type == "worktree" {
			count++
		}
	}
	return count
}
