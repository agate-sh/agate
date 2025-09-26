package panes

import (
	"agate/pkg/common"
	"agate/pkg/gui/components"
	"fmt"
	"io"
	"sort"

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
	expandedRepos    map[string]bool // Track which repos are expanded
}

// WorktreeListItem implements list.Item interface for worktrees
type WorktreeListItem struct {
	Type     string // "repo_header" or "worktree"
	RepoName string
	RepoPath string // Full path to repository
	Worktree *git.WorktreeInfo
	Index    int // Index in original repo list
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
}

type itemStyles struct {
	repoHeader    lipgloss.Style
	repoCurrent   lipgloss.Style
	selectedItem  lipgloss.Style
	normalItem    lipgloss.Style
	expandArrow   lipgloss.Style
	collapseArrow lipgloss.Style
	mustedText    lipgloss.Style
	width         int
}

func newItemStyles() *itemStyles {
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
	content := ""
	width := d.styles.width
	if width == 0 {
		width = 80 // Default width
	}

	if workItem.Type == "repo_header" {
		// Repository header with expand/collapse indicator
		repoName := workItem.RepoName
		var arrow string
		if d.expandedRepos != nil && d.expandedRepos[repoName] {
			arrow = "▼ " // Expanded
		} else {
			arrow = "▶ " // Collapsed
		}

		// Build content string
		if workItem.RepoName == d.currentRepo {
			content = arrow + repoName + " (current)"
		} else {
			content = arrow + repoName
		}

		// Apply highlighting for selected items
		if selected {
			// Full row highlight with background
			content = d.styles.selectedItem.
				Width(width).
				Bold(true).
				Render(content)
		} else {
			// Normal styling without background
			arrowStyled := d.styles.mustedText.Render(arrow)
			if workItem.RepoName == d.currentRepo {
				nameStyled := d.styles.repoCurrent.Render(repoName)
				currentStyled := d.styles.mustedText.Render(" (current)")
				content = arrowStyled + nameStyled + currentStyled
			} else {
				nameStyled := d.styles.repoHeader.Render(repoName)
				content = arrowStyled + nameStyled
			}
		}
	} else if workItem.Type == "worktree" && workItem.Worktree != nil {
		// Simple worktree display - just the name
		line := "    " + workItem.Worktree.Name // Indent for hierarchy

		if selected {
			// Full row highlight with background
			content = d.styles.selectedItem.
				Width(width).
				Render(line)
		} else {
			content = d.styles.normalItem.Render(line)
		}
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
		pane.Refresh()
	}

	return pane
}

// SetSize updates the dimensions of the repo worktree pane
func (r *RepoWorktreePane) SetSize(width, height int) {
	r.BasePane.SetSize(width, height)
	r.list.SetSize(width, height)
	// Update delegate width for proper row highlighting
	r.delegate.styles.width = width
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
	case "enter":
		// Toggle repository expansion
		if len(r.items) > 0 {
			selectedItem := r.list.SelectedItem()
			if workItem, ok := selectedItem.(WorktreeListItem); ok {
				if workItem.Type == "repo_header" {
					// Toggle expansion state
					r.expandedRepos[workItem.RepoName] = !r.expandedRepos[workItem.RepoName]
					// Update the delegate with the current expandedRepos state
					r.delegate.expandedRepos = r.expandedRepos
					// Update the list's delegate
					r.list.SetDelegate(r.delegate)
					r.buildItemList()
					r.list.SetItems(r.items)
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
	if len(r.items) == 0 {
		return nil
	}

	selectedItem := r.list.SelectedItem()
	if workItem, ok := selectedItem.(WorktreeListItem); ok {
		if workItem.Type == "worktree" {
			return workItem.Worktree
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
	}

	// Update delegate's current repo and expanded state
	r.delegate.currentRepo = r.currentRepo
	r.delegate.expandedRepos = r.expandedRepos
}

// SetCurrentRepo sets the current repository
func (r *RepoWorktreePane) SetCurrentRepo(repoName string) {
	r.currentRepo = repoName
	r.delegate.currentRepo = repoName
	r.delegate.expandedRepos = r.expandedRepos
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