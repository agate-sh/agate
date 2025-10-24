package agents

import (
	"fmt"
	"io"
	"strings"

	"agate/tui/internal/ui/components"
	"agate/tui/internal/ui/icons"
	"agate/tui/internal/ui/panes"
	"agate/tui/internal/ui/theme"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SessionInfo captures the metadata needed for rendering sessions.
type SessionInfo struct {
	ID         string
	AgentName  string
	AgentColor string
	TmuxName   string
}

// WorktreeInfo mirrors the data required for a worktree row.
type WorktreeInfo struct {
	RepoName   string
	Name       string
	Path       string
	Branch     string
	IsMain     bool
	IsActive   bool
	AgentColor string
	EntryIndex int
	Session    *SessionInfo
}

// RepoState bundles rendered state per repository.
type RepoState struct {
	Name            string
	Path            string
	IsCurrentRepo   bool
	MainWorktree    *WorktreeInfo
	LinkedWorktrees []*WorktreeInfo
	LinkedEmptyHint string
	MainEmptyHint   string
}

// PaneState describes the data required to render the pane.
type PaneState struct {
	Repos              []RepoState
	SelectedEntryIndex int
}

// AgentListItem represents a single row in the list model.
type AgentListItem struct {
	Type         string
	RepoName     string
	RepoPath     string
	SectionTitle string
	Worktree     *WorktreeInfo
	EntryIndex   int
	IsSelected   bool
}

// AgentColor returns the highlight color for the worktree.
func (i AgentListItem) AgentColor() string {
	if i.Worktree != nil {
		if strings.TrimSpace(i.Worktree.AgentColor) != "" {
			return i.Worktree.AgentColor
		}
		if i.Worktree.Session != nil && strings.TrimSpace(i.Worktree.Session.AgentColor) != "" {
			return i.Worktree.Session.AgentColor
		}
	}
	return theme.AgateColor
}

// FilterValue satisfies the list.Item interface.
func (i AgentListItem) FilterValue() string {
	if i.Worktree != nil {
		if strings.TrimSpace(i.Worktree.Branch) != "" {
			return i.Worktree.Branch
		}
		if strings.TrimSpace(i.Worktree.Name) != "" {
			return i.Worktree.Name
		}
		if strings.TrimSpace(i.Worktree.Path) != "" {
			return i.Worktree.Path
		}
	}
	return i.RepoName
}

// Pane renders the legacy agents pane experience.
type Pane struct {
	*panes.BasePane

	list          list.Model
	delegate      itemDelegate
	items         []AgentListItem
	expandedRepos map[string]bool
	state         PaneState
}

// NewPane constructs an agents pane.
func NewPane(index int) *Pane {
	styles := newItemStyles()
	expanded := make(map[string]bool)

	delegate := itemDelegate{
		styles:        styles,
		expandedRepos: expanded,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.KeyMap.NextPage.SetEnabled(false)
	l.KeyMap.PrevPage.SetEnabled(false)
	l.KeyMap.GoToStart.SetEnabled(false)
	l.KeyMap.GoToEnd.SetEnabled(false)
	l.KeyMap.Filter.SetEnabled(false)
	l.KeyMap.ClearFilter.SetEnabled(false)
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	return &Pane{
		BasePane:      panes.NewBasePane(index, "Agents"),
		list:          l,
		delegate:      delegate,
		items:         nil,
		expandedRepos: expanded,
		state:         PaneState{},
	}
}

// SetState applies a new rendering state to the pane.
func (p *Pane) SetState(state PaneState) {
	p.state = state
	p.state.SelectedEntryIndex = state.SelectedEntryIndex

	currentRepo := ""
	newExpanded := make(map[string]bool)
	for _, repo := range state.Repos {
		if repo.IsCurrentRepo && currentRepo == "" {
			currentRepo = repo.Name
		}
		if expanded, ok := p.expandedRepos[repo.Name]; ok {
			newExpanded[repo.Name] = expanded
		} else {
			newExpanded[repo.Name] = true
		}
	}
	if currentRepo == "" && len(state.Repos) > 0 {
		currentRepo = state.Repos[0].Name
	}
	p.expandedRepos = newExpanded
	p.delegate.currentRepo = currentRepo
	p.delegate.expandedRepos = p.expandedRepos
	p.delegate.isActive = p.IsActive()

	p.buildItems()
	p.refreshList()
	p.ensureSelection(state.SelectedEntryIndex)
}

// SetSize updates pane dimensions (width/height are content dimensions, not including padding/border).
func (p *Pane) SetSize(width, height int) {
	p.BasePane.SetSize(width, height)

	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	// The list should render at the full width including padding
	// since app.go will apply padding around the list content
	fullWidth := panes.PaneFullWidth(width)

	p.list.SetSize(fullWidth, height)
	p.delegate.styles.innerWidth = width
	p.delegate.styles.fullWidth = fullWidth
	p.delegate.styles.paddingLeft = panes.PaneContentHorizontalPadding()
	p.delegate.styles.paddingRight = panes.PaneContentHorizontalPadding()
	p.list.SetDelegate(p.delegate)
}

// SetActive flips the focus state.
func (p *Pane) SetActive(active bool) {
	p.BasePane.SetActive(active)
	p.delegate.isActive = active
	p.list.SetDelegate(p.delegate)
}

// View renders the pane content (without border - app.go handles that).
func (p *Pane) View() string {
	return p.list.View()
}

// Update forwards messages to the list model.
func (p *Pane) Update(msg tea.Msg) (panes.Pane, tea.Cmd) {
	// Don't forward keyboard messages to the list - we handle all keys in HandleKey
	if _, ok := msg.(tea.KeyMsg); ok {
		return p, nil
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// HandleKey processes navigation keys.
func (p *Pane) HandleKey(key string) (bool, tea.Cmd) {
	if !p.IsActive() {
		return false, nil
	}

	switch key {
	case "up", "k":
		p.moveSelection(-1)
		return true, nil
	case "down", "j":
		p.moveSelection(1)
		return true, nil
	case "enter":
		return p.handleEnter()
	case "n":
		return p.handleNewSession()
	case "d":
		return p.handleDelete()
	default:
		return false, nil
	}
}

func (p *Pane) handleEnter() (bool, tea.Cmd) {
	if len(p.items) == 0 {
		return false, nil
	}
	idx := p.list.Index()
	if idx < 0 || idx >= len(p.items) {
		return false, nil
	}

	item := p.items[idx]
	switch item.Type {
	case "repo_header":
		current := p.expandedRepos[item.RepoName]
		p.expandedRepos[item.RepoName] = !current
		p.delegate.expandedRepos = p.expandedRepos
		p.buildItems()
		p.refreshList()
		p.ensureSelection(p.state.SelectedEntryIndex)
		return true, nil
	case "main_session", "linked_session":
		if item.Worktree == nil {
			return false, nil
		}
		// If this worktree is already selected/active, emit attach message
		// Otherwise, just update the selection (no message needed)
		if item.IsSelected {
			return true, func() tea.Msg {
				return AttachSessionMsg{Worktree: item.Worktree}
			}
		}
		// Not selected yet - selection will be updated by the pane state
		return true, nil
	default:
		return false, nil
	}
}

func (p *Pane) handleNewSession() (bool, tea.Cmd) {
	if len(p.items) == 0 {
		return false, nil
	}
	idx := p.list.Index()
	if idx < 0 || idx >= len(p.items) {
		return false, nil
	}

	item := p.items[idx]
	return true, func() tea.Msg {
		return CreateSessionMsg{
			RepoName: item.RepoName,
			RepoPath: item.RepoPath,
		}
	}
}

func (p *Pane) handleDelete() (bool, tea.Cmd) {
	if len(p.items) == 0 {
		return false, nil
	}
	idx := p.list.Index()
	if idx < 0 || idx >= len(p.items) {
		return false, nil
	}

	item := p.items[idx]
	// Only allow deletion of linked sessions (not main sessions)
	if item.Type == "linked_session" && item.Worktree != nil && item.Worktree.Session != nil {
		return true, func() tea.Msg {
			return DeleteSessionMsg{Worktree: item.Worktree}
		}
	}
	return false, nil
}

// MoveUp moves selection upward.
func (p *Pane) MoveUp() bool {
	return p.moveSelection(-1)
}

// MoveDown moves selection downward.
func (p *Pane) MoveDown() bool {
	return p.moveSelection(1)
}

func (p *Pane) moveSelection(delta int) bool {
	if len(p.items) == 0 {
		return false
	}

	idx := p.list.Index()
	if idx < 0 || idx >= len(p.items) {
		if first := p.firstSelectableIndex(); first >= 0 {
			p.list.Select(first)
			p.state.SelectedEntryIndex = p.items[first].EntryIndex
			return true
		}
		return false
	}

	next := idx
	for {
		next += delta
		if next < 0 || next >= len(p.items) {
			return false
		}
		if p.isSelectableItem(next) {
			p.list.Select(next)
			p.state.SelectedEntryIndex = p.items[next].EntryIndex
			return next != idx
		}
	}
}

func (p *Pane) firstSelectableIndex() int {
	for idx := range p.items {
		if p.isSelectableItem(idx) {
			return idx
		}
	}
	return -1
}

func (p *Pane) isSelectableItem(index int) bool {
	if index < 0 || index >= len(p.items) {
		return false
	}
	item := p.items[index]
	return item.Type == "main_session" || item.Type == "linked_session"
}

func (p *Pane) ensureSelection(entryIndex int) {
	if len(p.items) == 0 {
		p.list.Select(-1)
		return
	}

	if entryIndex >= 0 && p.SelectByEntryIndex(entryIndex) {
		return
	}

	if first := p.firstSelectableIndex(); first >= 0 {
		p.list.Select(first)
		p.state.SelectedEntryIndex = p.items[first].EntryIndex
	}
}

func (p *Pane) refreshList() {
	listItems := make([]list.Item, len(p.items))
	for i := range p.items {
		listItems[i] = p.items[i]
	}
	p.list.SetItems(listItems)
}

func (p *Pane) buildItems() {
	var items []AgentListItem

	for repoIdx, repo := range p.state.Repos {
		if repoIdx > 0 {
			items = append(items, AgentListItem{Type: "gap"})
		}

		items = append(items, AgentListItem{
			Type:     "repo_header",
			RepoName: repo.Name,
			RepoPath: repo.Path,
		})

		if !p.expandedRepos[repo.Name] {
			continue
		}

		items = append(items, AgentListItem{
			Type:         "section_header",
			RepoName:     repo.Name,
			SectionTitle: "Main worktree",
		})

		if repo.MainWorktree != nil {
			items = append(items, AgentListItem{
				Type:       "main_session",
				RepoName:   repo.Name,
				RepoPath:   repo.Path,
				Worktree:   repo.MainWorktree,
				EntryIndex: repo.MainWorktree.EntryIndex,
				IsSelected: repo.MainWorktree.IsActive,
			})
		} else {
			items = append(items, AgentListItem{
				Type:         "empty_message",
				RepoName:     repo.Name,
				SectionTitle: "main",
			})
		}

		items = append(items, AgentListItem{Type: "gap"})

		items = append(items, AgentListItem{
			Type:         "section_header",
			RepoName:     repo.Name,
			SectionTitle: "Linked worktrees",
		})

		if len(repo.LinkedWorktrees) == 0 {
			items = append(items, AgentListItem{
				Type:         "empty_message",
				RepoName:     repo.Name,
				SectionTitle: "linked",
			})
		} else {
			for _, wt := range repo.LinkedWorktrees {
				items = append(items, AgentListItem{
					Type:       "linked_session",
					RepoName:   repo.Name,
					RepoPath:   repo.Path,
					Worktree:   wt,
					EntryIndex: wt.EntryIndex,
					IsSelected: wt.IsActive,
				})
			}
		}
	}

	p.items = items
	perPage := len(p.items)
	if perPage <= 0 {
		perPage = 1
	}
	p.list.Paginator.PerPage = perPage
	p.list.Paginator.Page = 0
	p.list.Paginator.SetTotalPages(len(p.items))
}

// SelectedItem returns the currently selected session item.
func (p *Pane) SelectedItem() (*AgentListItem, bool) {
	if len(p.items) == 0 {
		return nil, false
	}
	if idx := p.list.Index(); idx >= 0 && idx < len(p.items) {
		item := p.items[idx]
		if p.isSelectableItem(idx) {
			copied := item
			return &copied, true
		}
	}
	return nil, false
}

// SelectByEntryIndex focuses the row matching the entry index.
func (p *Pane) SelectByEntryIndex(entryIndex int) bool {
	if entryIndex < 0 {
		return false
	}
	for idx, item := range p.items {
		if item.EntryIndex == entryIndex && p.isSelectableItem(idx) {
			p.list.Select(idx)
			p.state.SelectedEntryIndex = entryIndex
			return true
		}
	}
	return false
}

// itemDelegate recreates the exact rendering behaviour from the Go client.
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
	mutedText     lipgloss.Style
	innerWidth    int
	fullWidth     int
	paddingLeft   int
	paddingRight  int
}

func newItemStyles() *itemStyles {
	padding := panes.PaneContentHorizontalPadding()
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
		mutedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)),
		paddingLeft:  padding,
		paddingRight: padding,
	}
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	workItem, ok := listItem.(AgentListItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	innerWidth := d.styles.innerWidth
	fullWidth := d.styles.fullWidth
	if innerWidth <= 0 || fullWidth <= 0 {
		// Fallback: calculate from list model width
		// m.Width() is the fullWidth we set in SetSize
		fullWidth = m.Width()
		padding := d.styles.paddingLeft + d.styles.paddingRight
		innerWidth = fullWidth - padding
		if innerWidth < 0 {
			innerWidth = 0
		}
	}
	leftPad := strings.Repeat(" ", d.styles.paddingLeft)
	rightPad := strings.Repeat(" ", d.styles.paddingRight)

	highlight := selected && d.isActive

	var linePlain string
	var lineStyled string
	var hint string

	var hoveredRepo string
	var hoveredRepoExpanded bool
	if hovered := m.SelectedItem(); hovered != nil {
		if hoveredItem, ok := hovered.(AgentListItem); ok {
			hoveredRepo = hoveredItem.RepoName
			if d.expandedRepos != nil {
				hoveredRepoExpanded = d.expandedRepos[hoveredRepo]
			}
		}
	}

	switch workItem.Type {
	case "repo_header":
		repoName := workItem.RepoName
		arrow := "▶"
		if d.expandedRepos != nil && d.expandedRepos[repoName] {
			arrow = "▼"
		}

		showShortcut := repoName == hoveredRepo && !hoveredRepoExpanded
		shortcutText := "n new agent"
		shortcutStyled := ""
		if showShortcut {
			shortcutStyled = components.RenderShortcut("n", "new agent", components.ShortcutDefault, "")
		}

		leftWidth := lipgloss.Width(repoName)
		arrowWidth := lipgloss.Width(arrow)
		hintWidth := 0
		gapWidth := 0
		safetyMargin := 1
		if showShortcut {
			hintWidth = lipgloss.Width(shortcutText)
			gapWidth = 1
			safetyMargin = 2  // Styled shortcuts have some ANSI overhead
		}

		available := innerWidth - leftWidth - arrowWidth - hintWidth - gapWidth - safetyMargin
		if available < 1 {
			available = 1
		}
		spacing := strings.Repeat(" ", available)

		linePlain = repoName + spacing
		if showShortcut {
			linePlain += shortcutText + " "
		}
		linePlain += arrow

		if highlight {
			lineStyled = linePlain
		} else {
			spacingStyled := strings.Repeat(" ", available)
			arrowStyled := d.styles.repoCurrent.Render(arrow)
			var nameStyled string
			if repoName == d.currentRepo {
				nameStyled = d.styles.repoCurrent.Render(repoName)
			} else {
				nameStyled = d.styles.repoHeader.Render(repoName)
			}
			if showShortcut {
				lineStyled = nameStyled + spacingStyled + shortcutStyled + " " + arrowStyled
			} else {
				lineStyled = nameStyled + spacingStyled + arrowStyled
			}
		}

	case "section_header":
		iconGlyph := ""
		switch workItem.SectionTitle {
		case "Main worktree":
			iconGlyph = icons.Home.Get()
		case "Linked worktrees":
			iconGlyph = icons.Link.Get()
		}

		iconStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextMuted)).Render(iconGlyph)
		sectionStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.TextDescription)).Render(workItem.SectionTitle)

		if workItem.SectionTitle == "Linked worktrees" && hoveredRepo == workItem.RepoName {
			hintText := components.RenderShortcut("n", "new agent", components.ShortcutDefault, "")
			leftContent := iconGlyph + "  " + workItem.SectionTitle
			leftWidth := lipgloss.Width(leftContent)
			hintWidth := lipgloss.Width(hintText)
			// Subtract small safety margin for ANSI overhead
			available := innerWidth - leftWidth - hintWidth - 2
			if available < 1 {
				available = 1
			}
			spacing := strings.Repeat(" ", available)
			lineStyled = iconStyled + "  " + sectionStyled + spacing + hintText
		} else {
			if iconGlyph != "" {
				lineStyled = iconStyled + "  " + sectionStyled
			} else {
				lineStyled = sectionStyled
			}
		}

	case "gap":
		linePlain = ""
		lineStyled = ""

	case "main_session", "linked_session":
		if workItem.Worktree == nil {
			return
		}

		// Calculate hint early so we can account for it in truncation
		tempHint := ""
		if d.isActive && highlight {
			if workItem.Type == "linked_session" {
				tempHint = " d to delete"
			}
		}

		label := strings.TrimSpace(workItem.Worktree.Branch)
		if label == "" {
			label = strings.TrimSpace(workItem.Worktree.Name)
		}
		if label == "" {
			label = strings.TrimSpace(workItem.Worktree.Path)
		}
		if label == "" {
			label = "(unknown worktree)"
		}

		// Remove any newlines or carriage returns
		label = strings.ReplaceAll(label, "\n", " ")
		label = strings.ReplaceAll(label, "\r", " ")

		// Build icon prefix and calculate its actual width
		iconRune := icons.GitRepo.NerdFont
		if workItem.IsSelected {
			iconRune = icons.Ready.NerdFont
		}
		iconPrefix := "   " + iconRune + "  "
		iconWidth := lipgloss.Width(iconPrefix)

		// Truncate label to fit available space (accounting for icon and hint)
		availableForLabel := innerWidth - iconWidth
		if tempHint != "" {
			availableForLabel -= lipgloss.Width(tempHint)
			// Safety margin for ANSI codes in both hint and highlight styling
			availableForLabel -= 3
		}
		if availableForLabel < 0 {
			availableForLabel = 0
		}
		if lipgloss.Width(label) > availableForLabel {
			runes := []rune(label)
			for lipgloss.Width(string(runes)) > availableForLabel-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			if availableForLabel >= 3 {
				label = string(runes) + "..."
			} else {
				label = string(runes)
			}
		}

		linePlain = iconPrefix + label
		if highlight {
			lineStyled = linePlain
		} else {
			lineStyled = d.styles.normalItem.Render(linePlain)
		}

	case "empty_message":
		message := "   No agents"
		if strings.EqualFold(workItem.SectionTitle, "main") {
			message = "   No main session"
		} else if strings.EqualFold(workItem.SectionTitle, "linked") {
			message = "   No linked sessions"
		}
		lineStyled = d.styles.mutedText.Render(message)

	default:
		return
	}

	if hint == "" && d.isActive && highlight && (workItem.Type == "main_session" || workItem.Type == "linked_session") {
		if workItem.Type == "linked_session" {
			hint = " d to delete"
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

		// Build plain text row with hint if present
		// Note: linePlain was already truncated early to account for hint
		plainRow := linePlain
		if hint != "" {
			// Just append hint with minimal padding to avoid wrapping
			currentWidth := lipgloss.Width(linePlain)
			hintWidth := lipgloss.Width(hint)
			// Pad to fill remaining space (leave safety margin for ANSI codes)
			paddingNeeded := contentWidth - currentWidth - hintWidth - 3
			if paddingNeeded > 0 {
				plainRow += strings.Repeat(" ", paddingNeeded)
			}
			plainRow += hint
		}

		// Render with highlight style
		body = bodyStyle.Render(plainRow)
	} else {
		body = lipgloss.NewStyle().Width(contentWidth).Render(lineStyled)
	}

	baseStyle := lipgloss.NewStyle()
	if activeHighlight {
		baseStyle = baseStyle.Background(lipgloss.Color(theme.RowHighlight))
	}

	leftSegment := baseStyle.Render(leftPad)
	rightStyle := lipgloss.NewStyle()
	if activeHighlight {
		rightStyle = rightStyle.Background(lipgloss.Color(theme.RowHighlight))
	}
	rightSegment := rightStyle.Render(rightPad)

	content := leftSegment + body + rightSegment
	if lipgloss.Width(content) < fullWidth {
		padding := rightStyle
		if !highlight {
			padding = lipgloss.NewStyle()
		}
		content += padding.Render(strings.Repeat(" ", fullWidth-lipgloss.Width(content)))
	}

	fmt.Fprint(w, content)
}

// CreateSessionMsg is sent when the user requests to create a new session.
type CreateSessionMsg struct {
	RepoName string
	RepoPath string
}

// DeleteSessionMsg is sent when the user requests to delete a session.
type DeleteSessionMsg struct {
	Worktree *WorktreeInfo
}

// AttachSessionMsg is sent when the user wants to attach to a session.
type AttachSessionMsg struct {
	Worktree *WorktreeInfo
}
