package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	agateclient "agate/sdk/gen"
	"agate/tui/internal/agents"
	"agate/tui/internal/api"
	"agate/tui/internal/ui/overlays"
	"agate/tui/internal/ui/panes"
	uiagents "agate/tui/internal/ui/panes/agents"
	"agate/tui/internal/ui/theme"
	"agate/tui/internal/ws"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	topPaddingRows     = 1
	bottomSpacerRows   = 1
	paneTitleRows      = 1
	footerRows         = 1
	bottomMarginRows   = 1
	horizontalMargin   = 2
	horizontalGapWidth = 2
)

type loadState int

const (
	stateLoading loadState = iota
	stateReady
	stateError
)

type sessionSummary struct {
	ID           string
	AgentName    string
	TmuxName     string
	WorktreePath string
	Branch       string
	RepoName     string
	CreatedAt    time.Time
	LastAccessed time.Time
}

type worktreeEntry struct {
	Path     string
	Branch   string
	Commit   string
	Bare     bool
	Detached bool
	IsMain   bool
	RepoName string
	Session  *sessionSummary
	IsActive bool
}

type sectionType string

const (
	sectionMain   sectionType = "main"
	sectionLinked sectionType = "linked"
)

type displayEntry struct {
	Section  sectionType
	Worktree *worktreeEntry
}

type repoSnapshot struct {
	RepoName      string
	RepoRoot      string
	CurrentBranch string
	Worktrees     []worktreeEntry
	ActiveSession string
	DefaultAgent  string
}

type loadSuccessMsg struct {
	snapshot repoSnapshot
}

type loadErrorMsg struct {
	err error
}

type createSessionResultMsg struct {
	resp *agateclient.SessionCreate201Response
	err  error
}

type deleteSessionResultMsg struct {
	err error
}

type wsConnectedMsg struct{}

type wsErrorMsg struct {
	err error
}

type wsPtyOutputMsg struct {
	sessionID string
	data      string
}

// Model is the Bubble Tea program model for the agents-first UI slice.
type Model struct {
	client   *agateclient.APIClient
	repoPath string

	state            loadState
	agentPalette     []agents.Config
	snapshot         repoSnapshot
	entries          []displayEntry
	selectedIndex    int
	lastSelectedPath string
	err              error

	agentsPane      *uiagents.Pane
	tmuxPane        *panes.TmuxPane
	panes           []panes.Pane
	activePaneIndex int
	windowWidth     int
	windowHeight    int

	leftPaneContentWidth  int
	leftPaneContentHeight int
	leftPaneFullWidth     int
	tmuxPaneContentWidth  int
	tmuxPaneContentHeight int
	tmuxPaneFullWidth     int

	deleteOverlay tea.Model

	// WebSocket client for PTY streaming
	wsClient             *ws.Client
	wsSubscribedSessionID string
}

// NewModel constructs a model wired up to the configured API client.
func NewModel(client *agateclient.APIClient, repoPath string) Model {
	agentsPane := uiagents.NewPane(0)
	agentsPane.SetActive(true)

	tmuxPane := panes.NewTmuxPane(1)

	// Initialize WebSocket client (port 24283 from @agate/shared AGATE_SERVER_PORT)
	wsClient := ws.NewClient("ws://localhost:24283/ws")

	// Set WebSocket client on tmux pane for input forwarding
	tmuxPane.SetWSClient(wsClient)

	return Model{
		client:          client,
		repoPath:        repoPath,
		state:           stateLoading,
		agentPalette:    agents.All(),
		selectedIndex:   -1,
		agentsPane:      agentsPane,
		tmuxPane:        tmuxPane,
		panes:           []panes.Pane{agentsPane, tmuxPane},
		activePaneIndex: 0,
		wsClient:        wsClient,
	}
}

// Init kicks off loading repository data and WebSocket connection.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchRepoSnapshot(m.client, m.repoPath),
		connectWebSocket(m.wsClient),
		listenForWSEvents(m.wsClient),
	)
}

// Update handles key events and async data loading.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipPaneUpdate := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.resizePanes(msg.Width, msg.Height)
	case tea.KeyMsg:
		// Global keys that work regardless of active pane
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Switch between panes
			m.switchPane()
			return m, nil
		case "r":
			if entry, ok := m.currentSelection(); ok {
				m.lastSelectedPath = entry.Worktree.Path
			}
			m.state = stateLoading
			m.err = nil
			return m, fetchRepoSnapshot(m.client, m.repoPath)
		}

		// Route keys to active pane
		if handled, cmd := m.handlePaneKey(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			skipPaneUpdate = true
		}
	case uiagents.CreateSessionMsg:
		if m.state == stateReady {
			repoRoot := fallback(msg.RepoPath, m.repoPath)
			agentName := fallback(m.snapshot.DefaultAgent, "claude")
			if entry, ok := m.currentSelection(); ok {
				m.lastSelectedPath = entry.Worktree.Path
			}
			m.state = stateLoading
			m.err = nil
			return m, createSessionCmd(m.client, repoRoot, m.newBranchSeed(agentName), agentName)
		}
	case uiagents.DeleteSessionMsg:
		if m.state == stateReady && msg.Worktree != nil && msg.Worktree.Session != nil {
			// Show delete confirmation overlay
			overlay := overlays.NewDeleteSessionOverlay(msg.Worktree, m.client)
			overlay.SetSize(m.windowWidth, m.windowHeight)
			m.deleteOverlay = overlay
			return m, overlay.Init()
		}
	case overlays.DeleteSessionCancelledMsg:
		// User cancelled deletion, close overlay
		m.deleteOverlay = nil
		return m, nil
	case overlays.DeleteSessionSuccessMsg:
		// Session deleted successfully, close overlay and reload
		m.deleteOverlay = nil
		m.lastSelectedPath = ""
		m.state = stateLoading
		m.err = nil
		return m, fetchRepoSnapshot(m.client, m.repoPath)
	case overlays.DeleteSessionErrorMsg:
		// Session deletion failed, show error
		m.deleteOverlay = nil
		m.state = stateError
		m.err = fmt.Errorf("failed to delete session: %s", msg.Error)
		return m, nil
	case uiagents.AttachSessionMsg:
		if m.state == stateReady && msg.Worktree != nil && msg.Worktree.Session != nil {
			// Subscribe to WebSocket for this session's PTY output
			sessionID := msg.Worktree.Session.ID
			agentName := msg.Worktree.Session.AgentName
			agentColor := m.agentColor(agentName)

			// Unsubscribe from previous session if any
			if m.wsSubscribedSessionID != "" && m.wsSubscribedSessionID != sessionID {
				_ = m.wsClient.Unsubscribe()
			}

			// Update tmux pane with session info and mark as subscribed
			m.tmuxPane.SetSession(sessionID, agentName, agentColor)
			m.tmuxPane.SetSubscribed(true)
			m.wsSubscribedSessionID = sessionID

			// Subscribe to new session and start spinner
			return m, tea.Batch(
				subscribeToSession(m.wsClient, sessionID),
				startSpinner(m.tmuxPane),
			)
		}
	case wsConnectedMsg:
		// WebSocket connected successfully
		// Continue listening for events
		cmds = append(cmds, listenForWSEvents(m.wsClient))
	case wsErrorMsg:
		// WebSocket error occurred - could trigger reconnection here
		// For now, just continue
		cmds = append(cmds, listenForWSEvents(m.wsClient))
	case wsPtyOutputMsg:
		// Received PTY output from WebSocket
		if m.tmuxPane != nil && msg.sessionID == m.wsSubscribedSessionID {
			m.tmuxPane.AppendContent(msg.data)
		}
		// Continue listening for events
		cmds = append(cmds, listenForWSEvents(m.wsClient))
	case loadSuccessMsg:
		m.state = stateReady
		m.err = nil
		m.snapshot = msg.snapshot
		m.entries = buildDisplayEntries(m.snapshot)
		m.reconcileSelection()
		m.syncAgentsPaneItems()
	case loadErrorMsg:
		m.state = stateError
		m.err = msg.err
	case createSessionResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			break
		}
		if msg.resp != nil {
			m.lastSelectedPath = filepath.Clean(msg.resp.GetWorktreePath())
		}
		return m, fetchRepoSnapshot(m.client, m.repoPath)
	case deleteSessionResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			break
		}
		return m, fetchRepoSnapshot(m.client, m.repoPath)
	}

	// Forward messages to overlay if active
	if m.deleteOverlay != nil {
		var overlayCmd tea.Cmd
		m.deleteOverlay, overlayCmd = m.deleteOverlay.Update(msg)
		if overlayCmd != nil {
			cmds = append(cmds, overlayCmd)
		}
		// Don't update panes when overlay is active
		return m, tea.Batch(cmds...)
	}

	if m.agentsPane != nil && !skipPaneUpdate {
		var paneCmd tea.Cmd
		var updated panes.Pane
		updated, paneCmd = m.agentsPane.Update(msg)
		if panePtr, ok := updated.(*uiagents.Pane); ok {
			m.agentsPane = panePtr
			if len(m.panes) > 0 {
				m.panes[0] = panePtr
			}
		}
		if paneCmd != nil {
			cmds = append(cmds, paneCmd)
		}
	}

	// Also update tmux pane (for spinner ticks)
	if m.tmuxPane != nil && !skipPaneUpdate {
		var paneCmd tea.Cmd
		var updated panes.Pane
		updated, paneCmd = m.tmuxPane.Update(msg)
		if panePtr, ok := updated.(*panes.TmuxPane); ok {
			m.tmuxPane = panePtr
			if len(m.panes) > 1 {
				m.panes[1] = panePtr
			}
		}
		if paneCmd != nil {
			cmds = append(cmds, paneCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the current UI.
func (m Model) View() string {
	padding := lipgloss.NewStyle().Padding(1, 2)

	// Base view
	var baseView string
	switch m.state {
	case stateLoading:
		baseView = padding.Render("Loading repository data…")
	case stateError:
		msg := "Failed to load data."
		if m.err != nil {
			msg = fmt.Sprintf("Failed to load data: %v", m.err)
		}
		errorStyle := padding.Copy().Foreground(lipgloss.Color("#ff5555"))
		helpStyle := padding.Copy().Foreground(lipgloss.Color("#666666")).PaddingTop(1)
		baseView = lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render(msg),
			helpStyle.Render("Press 'r' to retry or 'q' to quit."),
		)
	default:
		if m.agentsPane == nil {
			baseView = padding.Render("Agents pane not available.")
		} else {
			baseView = m.renderTwoColumnLayout()
		}
	}

	// Render overlay on top if active
	if m.deleteOverlay != nil {
		overlayView := m.deleteOverlay.View()
		return lipgloss.Place(
			m.windowWidth,
			m.windowHeight,
			lipgloss.Center,
			lipgloss.Center,
			overlayView,
			lipgloss.WithWhitespaceChars("░"),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#333333")),
		)
	}

	return baseView
}

func (m Model) renderTwoColumnLayout() string {
	// Render agents pane
	leftPaneContent := m.agentsPane.View()
	leftWrapped := lipgloss.NewStyle().
		Width(m.leftPaneContentWidth).
		MaxHeight(m.leftPaneContentHeight).
		Render(leftPaneContent)
	leftAligned := lipgloss.PlaceVertical(m.leftPaneContentHeight, lipgloss.Top, leftWrapped)
	leftWithPadding := panes.ApplyPaneContentPadding(leftAligned, m.leftPaneContentWidth)
	leftPaneWithBorder := panes.PaneBaseStyle.Render(leftWithPadding)

	leftTitleView := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.TextDescription)).
		Render("Agents")

	// Render tmux pane
	var tmuxPaneWithBorder string
	var tmuxTitleView string
	if m.tmuxPane != nil && m.tmuxPaneContentWidth > 0 {
		tmuxPaneContent := m.tmuxPane.View()
		tmuxWrapped := lipgloss.NewStyle().
			Width(m.tmuxPaneContentWidth).
			MaxHeight(m.tmuxPaneContentHeight).
			Render(tmuxPaneContent)
		tmuxAligned := lipgloss.PlaceVertical(m.tmuxPaneContentHeight, lipgloss.Top, tmuxWrapped)
		tmuxWithPadding := panes.ApplyPaneContentPadding(tmuxAligned, m.tmuxPaneContentWidth)
		tmuxPaneWithBorder = panes.PaneBaseStyle.Render(tmuxWithPadding)

		titleStyle := m.tmuxPane.GetTitleStyle()
		tmuxTitleView = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.TextDescription)).
			Render(titleStyle.Text)
	}

	// Combine panes horizontally with gap
	gap := strings.Repeat(" ", horizontalGapWidth)
	panesRow := lipgloss.JoinHorizontal(lipgloss.Top, leftPaneWithBorder, gap, tmuxPaneWithBorder)

	// Combine titles horizontally with gap
	titlesRow := lipgloss.JoinHorizontal(lipgloss.Top, leftTitleView, gap, tmuxTitleView)

	// Build complete layout
	helpLine := "tab switch pane • ↑/↓ select worktree • ↵ attach • n new session • d delete session • r reload • q quit"
	helpView := lipgloss.NewStyle().
		PaddingTop(bottomSpacerRows).
		Foreground(lipgloss.Color("#666666")).
		Render(helpLine)

	topPad := lipgloss.NewStyle().Height(topPaddingRows).Render("")
	bottomPad := lipgloss.NewStyle().Height(bottomMarginRows).Render("")

	return lipgloss.NewStyle().MarginLeft(horizontalMargin).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			topPad,
			titlesRow,
			lipgloss.NewStyle().Height(paneTitleRows).Render(""),
			panesRow,
			helpView,
			bottomPad,
		),
	)
}

func (m *Model) resizePanes(width, height int) {
	if width <= 0 || height <= 0 || m.agentsPane == nil {
		return
	}

	metrics := calculateAgentsPaneMetrics(width, height)
	if metrics.contentWidth <= 0 || metrics.contentHeight <= 0 {
		return
	}

	m.leftPaneContentWidth = metrics.contentWidth
	m.leftPaneContentHeight = metrics.contentHeight
	m.leftPaneFullWidth = metrics.fullWidth

	m.agentsPane.SetSize(metrics.contentWidth, metrics.contentHeight)

	// Calculate tmux pane dimensions (remaining space after agents pane)
	if m.tmuxPane != nil {
		tmuxMetrics := calculateTmuxPaneMetrics(width, height, m.leftPaneFullWidth)
		m.tmuxPaneContentWidth = tmuxMetrics.contentWidth
		m.tmuxPaneContentHeight = tmuxMetrics.contentHeight
		m.tmuxPaneFullWidth = tmuxMetrics.fullWidth
		m.tmuxPane.SetSize(tmuxMetrics.contentWidth, tmuxMetrics.contentHeight)
	}
}

func (m *Model) handlePaneKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.state != stateReady {
		return false, nil
	}

	// Route to active pane
	switch m.activePaneIndex {
	case 0: // Agents pane
		if m.agentsPane == nil {
			return false, nil
		}
		handled, cmd := m.agentsPane.HandleKey(msg.String())
		if handled {
			m.syncSelectionFromPane()
		}
		return handled, cmd
	case 1: // Tmux pane
		if m.tmuxPane == nil {
			return false, nil
		}
		return m.tmuxPane.HandleKey(msg.String())
	default:
		return false, nil
	}
}

func (m *Model) syncSelectionFromPane() {
	if m.agentsPane == nil {
		m.selectedIndex = -1
		return
	}
	item, ok := m.agentsPane.SelectedItem()
	if !ok || item.EntryIndex < 0 || item.EntryIndex >= len(m.entries) {
		m.selectedIndex = -1
		return
	}

	m.selectedIndex = item.EntryIndex
	if tw := m.entries[m.selectedIndex].Worktree; tw != nil {
		m.lastSelectedPath = tw.Path
	} else {
		m.lastSelectedPath = ""
	}
}

func (m *Model) currentSelection() (*displayEntry, bool) {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.entries) {
		return nil, false
	}
	entry := &m.entries[m.selectedIndex]
	if entry.Worktree == nil {
		return nil, false
	}
	return entry, true
}

func (m *Model) reconcileSelection() {
	if len(m.entries) == 0 {
		m.selectedIndex = -1
		m.lastSelectedPath = ""
		return
	}

	desiredPath := strings.TrimSpace(m.lastSelectedPath)
	if desiredPath == "" && m.snapshot.ActiveSession != "" {
		if entry := findEntryBySession(m.entries, m.snapshot.ActiveSession); entry != nil && entry.Worktree != nil {
			desiredPath = entry.Worktree.Path
		}
	}

	if desiredPath != "" {
		if idx := findEntryIndexByPath(m.entries, desiredPath); idx >= 0 {
			m.selectedIndex = idx
		} else {
			m.selectedIndex = clampIndex(m.selectedIndex, len(m.entries))
		}
	} else {
		m.selectedIndex = clampIndex(m.selectedIndex, len(m.entries))
	}

	if m.selectedIndex >= 0 && m.selectedIndex < len(m.entries) && m.entries[m.selectedIndex].Worktree != nil {
		m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
	} else if len(m.entries) > 0 {
		m.selectedIndex = 0
		m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
	} else {
		m.selectedIndex = -1
		m.lastSelectedPath = ""
	}
}

func (m *Model) syncAgentsPaneItems() {
	if m.agentsPane == nil {
		return
	}

	state := buildAgentsPaneState(m.snapshot, m.entries, m.selectedIndex, m.agentColor)
	m.agentsPane.SetState(state)
	m.agentsPane.SelectByEntryIndex(m.selectedIndex)
	m.syncSelectionFromPane()
	m.resizePanes(m.windowWidth, m.windowHeight)
}

func buildAgentsPaneState(snapshot repoSnapshot, entries []displayEntry, selectedIndex int, colorForAgent func(string) string) uiagents.PaneState {
	repoName := fallback(snapshot.RepoName, "unknown")
	repoPath := fallback(snapshot.RepoRoot, "")

	repoState := uiagents.RepoState{
		Name:            repoName,
		Path:            repoPath,
		IsCurrentRepo:   true,
		MainEmptyHint:   "No main session",
		LinkedEmptyHint: "No linked sessions. Press n to create one.",
	}

	mainIndices := indicesForSection(entries, sectionMain)
	if len(mainIndices) > 0 {
		if info := toWorktreeInfo(entries[mainIndices[0]], mainIndices[0], repoName, colorForAgent); info != nil {
			repoState.MainWorktree = info
		}
	}

	linkedIndices := indicesForSection(entries, sectionLinked)
	for _, idx := range linkedIndices {
		if info := toWorktreeInfo(entries[idx], idx, repoName, colorForAgent); info != nil {
			repoState.LinkedWorktrees = append(repoState.LinkedWorktrees, info)
		}
	}

	return uiagents.PaneState{
		Repos:              []uiagents.RepoState{repoState},
		SelectedEntryIndex: selectedIndex,
	}
}

func toWorktreeInfo(entry displayEntry, idx int, repoName string, colorForAgent func(string) string) *uiagents.WorktreeInfo {
	if entry.Worktree == nil {
		return nil
	}

	wt := entry.Worktree
	name := filepath.Base(wt.Path)
	info := &uiagents.WorktreeInfo{
		RepoName:   repoName,
		Name:       name,
		Path:       wt.Path,
		Branch:     wt.Branch,
		IsMain:     wt.IsMain,
		IsActive:   wt.IsActive,
		EntryIndex: idx,
	}

	if wt.Session != nil {
		session := toPaneSession(wt.Session, colorForAgent)
		info.Session = session
		if session != nil {
			info.AgentColor = session.AgentColor
		}
	}

	return info
}

func toPaneSession(summary *sessionSummary, colorForAgent func(string) string) *uiagents.SessionInfo {
	if summary == nil {
		return nil
	}

	return &uiagents.SessionInfo{
		ID:         summary.ID,
		AgentName:  summary.AgentName,
		AgentColor: colorForAgent(summary.AgentName),
		TmuxName:   summary.TmuxName,
	}
}

type agentsPaneMetrics struct {
	contentWidth  int
	contentHeight int
	fullWidth     int
}

func calculateAgentsPaneMetrics(totalWidth, totalHeight int) agentsPaneMetrics {
	metrics := agentsPaneMetrics{}
	if totalWidth <= 0 || totalHeight <= 0 {
		return metrics
	}

	chromeHeight := topPaddingRows + bottomSpacerRows + paneTitleRows + footerRows + bottomMarginRows
	availableHeight := totalHeight - chromeHeight
	frameHeight := panes.PaneBaseStyle.GetVerticalFrameSize()
	minPaneHeight := frameHeight + 1
	if availableHeight < minPaneHeight {
		availableHeight = minPaneHeight
	}
	contentHeight := availableHeight - frameHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	totalHorizontalMargins := horizontalMargin*2 + horizontalGapWidth*2
	usableWidth := totalWidth - totalHorizontalMargins
	if usableWidth < 0 {
		usableWidth = 0
	}

	frameWidth := panes.PaneBaseStyle.GetHorizontalFrameSize()
	contentPaddingWidth := panes.PaneContentHorizontalPadding() * 2
	// Calculate chrome for 2 panes (agents + tmux) not 3
	totalChromeWidth := (frameWidth + contentPaddingWidth) * 2
	availableContentWidth := usableWidth - totalChromeWidth
	if availableContentWidth < 0 {
		availableContentWidth = 0
	}

	// Agents pane should be ~30% of content width
	leftContentWidth := int(float64(availableContentWidth) * 0.30)
	const minContentWidth = 30
	if leftContentWidth < minContentWidth {
		if availableContentWidth >= minContentWidth {
			leftContentWidth = minContentWidth
		} else {
			leftContentWidth = availableContentWidth
		}
	}

	metrics.contentWidth = leftContentWidth
	metrics.contentHeight = contentHeight
	metrics.fullWidth = panes.PaneFullWidth(leftContentWidth)

	return metrics
}

type tmuxPaneMetrics struct {
	contentWidth  int
	contentHeight int
	fullWidth     int
}

func calculateTmuxPaneMetrics(totalWidth, totalHeight int, leftPaneFullWidth int) tmuxPaneMetrics {
	metrics := tmuxPaneMetrics{}
	if totalWidth <= 0 || totalHeight <= 0 {
		return metrics
	}

	// Calculate content height (same as agents pane)
	chromeHeight := topPaddingRows + bottomSpacerRows + paneTitleRows + footerRows + bottomMarginRows
	availableHeight := totalHeight - chromeHeight
	frameHeight := panes.PaneBaseStyle.GetVerticalFrameSize()
	minPaneHeight := frameHeight + 1
	if availableHeight < minPaneHeight {
		availableHeight = minPaneHeight
	}
	contentHeight := availableHeight - frameHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Calculate tmux pane width (remaining space after agents pane)
	totalHorizontalMargins := horizontalMargin * 2
	usableWidth := totalWidth - totalHorizontalMargins
	if usableWidth < 0 {
		usableWidth = 0
	}

	// Remaining width = usable width - left pane full width - horizontal gap
	remainingWidth := usableWidth - leftPaneFullWidth - horizontalGapWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	// Subtract frame and padding from remaining width to get content width
	frameWidth := panes.PaneBaseStyle.GetHorizontalFrameSize()
	contentPaddingWidth := panes.PaneContentHorizontalPadding() * 2
	tmuxContentWidth := remainingWidth - frameWidth - contentPaddingWidth
	if tmuxContentWidth < 0 {
		tmuxContentWidth = 0
	}

	metrics.contentWidth = tmuxContentWidth
	metrics.contentHeight = contentHeight
	metrics.fullWidth = panes.PaneFullWidth(tmuxContentWidth)

	return metrics
}

// Start launches the Bubble Tea program.
func Start() error {
	client, err := api.New()
	if err != nil {
		return err
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining repo path: %w", err)
	}

	model := NewModel(client, repoPath)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func createSessionCmd(client *agateclient.APIClient, repoRoot, branchName, agentName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := agateclient.SessionCreateRequest{
			Dir:        repoRoot,
			BranchName: branchName,
			AgentName:  agentName,
		}

		resp, err := api.CreateSession(ctx, client, req)
		if err != nil {
			return createSessionResultMsg{err: err}
		}
		return createSessionResultMsg{resp: resp}
	}
}

func deleteSessionCmd(client *agateclient.APIClient, sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := api.DeleteSession(ctx, client, sessionID); err != nil {
			return deleteSessionResultMsg{err: err}
		}
		return deleteSessionResultMsg{}
	}
}

func fetchRepoSnapshot(client *agateclient.APIClient, repoPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		repoInfo, _, err := client.DefaultAPI.GitRepoInfo(ctx).Dir(repoPath).Execute()
		if err != nil {
			return loadErrorMsg{err: fmt.Errorf("git repo info: %w", err)}
		}

		sessionResp, err := api.ListSessions(ctx, client)
		if err != nil {
			return loadErrorMsg{err: err}
		}

		// Only create worktree entries for sessions (don't show all git worktrees)
		worktrees := make([]worktreeEntry, 0, len(sessionResp.GetSessions()))

		for _, sess := range sessionResp.GetSessions() {
			createdAt, _ := time.Parse(time.RFC3339, sess.GetCreatedAt())
			lastAccessed, _ := time.Parse(time.RFC3339, sess.GetLastAccessed())
			summary := &sessionSummary{
				ID:           sess.GetId(),
				AgentName:    sess.GetAgentName(),
				TmuxName:     sess.GetTmuxName(),
				WorktreePath: sess.GetWorktreePath(),
				Branch:       sess.GetBranch(),
				RepoName:     sess.GetRepoName(),
				CreatedAt:    createdAt,
				LastAccessed: lastAccessed,
			}

			// Create worktree entry for this session
			isMain := !strings.Contains(sess.GetWorktreePath(), "/.agate/worktrees/")
			entry := worktreeEntry{
				Path:     filepath.Clean(sess.GetWorktreePath()),
				Branch:   sess.GetBranch(),
				Commit:   "",
				Bare:     false,
				Detached: false,
				IsMain:   isMain,
				RepoName: sess.GetRepoName(),
				Session:  summary,
			}
			worktrees = append(worktrees, entry)
		}

		for i := range worktrees {
			if worktrees[i].Session != nil && sessionResp.GetActiveSession() == worktrees[i].Session.ID {
				worktrees[i].IsActive = true
			}
		}

		return loadSuccessMsg{
			snapshot: repoSnapshot{
				RepoName:      repoInfo.GetRepoName(),
				RepoRoot:      repoInfo.GetRepoRoot(),
				CurrentBranch: repoInfo.GetCurrentBranch(),
				Worktrees:     worktrees,
				ActiveSession: sessionResp.GetActiveSession(),
				DefaultAgent:  sessionResp.GetDefaultAgent(),
			},
		}
	}
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func humanizePath(path string) string {
	if path == "" {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(path, home) {
			return strings.Replace(path, home, "~", 1)
		}
	}
	return path
}

func (m Model) agentColor(agentName string) string {
	for _, cfg := range m.agentPalette {
		if strings.EqualFold(cfg.Name, agentName) {
			return cfg.BorderColor
		}
	}
	return "#9d87ae"
}

func indicesForSection(entries []displayEntry, section sectionType) []int {
	out := make([]int, 0, len(entries))
	for idx, entry := range entries {
		if entry.Section == section && entry.Worktree != nil {
			out = append(out, idx)
		}
	}
	return out
}

func buildDisplayEntries(snapshot repoSnapshot) []displayEntry {
	if len(snapshot.Worktrees) == 0 {
		return nil
	}

	mainList := make([]*worktreeEntry, 0)
	linkedList := make([]*worktreeEntry, 0)

	for i := range snapshot.Worktrees {
		wt := &snapshot.Worktrees[i]
		if wt.Session != nil && snapshot.ActiveSession == wt.Session.ID {
			wt.IsActive = true
		}
		if wt.IsMain {
			mainList = append(mainList, wt)
		} else {
			linkedList = append(linkedList, wt)
		}
	}

	sort.SliceStable(mainList, func(i, j int) bool {
		return strings.ToLower(mainList[i].Branch) < strings.ToLower(mainList[j].Branch)
	})
	sort.SliceStable(linkedList, func(i, j int) bool {
		left := strings.ToLower(linkedList[i].Branch)
		right := strings.ToLower(linkedList[j].Branch)
		if left == right {
			return linkedList[i].Path < linkedList[j].Path
		}
		return left < right
	})

	entries := make([]displayEntry, 0, len(snapshot.Worktrees))
	for _, wt := range mainList {
		entries = append(entries, displayEntry{
			Section:  sectionMain,
			Worktree: wt,
		})
	}
	for _, wt := range linkedList {
		entries = append(entries, displayEntry{
			Section:  sectionLinked,
			Worktree: wt,
		})
	}
	return entries
}

func findEntryBySession(entries []displayEntry, sessionID string) *displayEntry {
	if sessionID == "" {
		return nil
	}
	for i := range entries {
		entry := &entries[i]
		if entry.Worktree != nil && entry.Worktree.Session != nil && entry.Worktree.Session.ID == sessionID {
			return entry
		}
	}
	return nil
}

func findEntryIndexByPath(entries []displayEntry, targetPath string) int {
	if targetPath == "" {
		return -1
	}
	cleanTarget := filepath.Clean(targetPath)
	for idx, entry := range entries {
		if entry.Worktree == nil {
			continue
		}
		if filepath.Clean(entry.Worktree.Path) == cleanTarget {
			return idx
		}
	}
	return -1
}

func clampIndex(current, length int) int {
	if length == 0 {
		return -1
	}
	if current < 0 {
		return 0
	}
	if current >= length {
		return length - 1
	}
	return current
}

func (m Model) newBranchSeed(agentName string) string {
	agentSlug := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(agentName)), " ", "-")
	if agentSlug == "" {
		agentSlug = "agent"
	}
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("feat/%s-%s", agentSlug, timestamp)
}

// connectWebSocket connects to the WebSocket server
func connectWebSocket(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.ConnectWithRetry(3); err != nil {
			return wsErrorMsg{err: err}
		}
		return wsConnectedMsg{}
	}
}

// listenForWSEvents listens for WebSocket events and converts them to Bubble Tea messages
func listenForWSEvents(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		// Block waiting for next event from WebSocket
		event := <-client.Events()

		switch event.Type {
		case ws.EventPtyOutput:
			return wsPtyOutputMsg{
				sessionID: event.SessionID,
				data:      event.Data,
			}
		default:
			// Unknown event type, continue listening
			return nil
		}
	}
}

// subscribeToSession subscribes to a session's PTY output
func subscribeToSession(client *ws.Client, sessionID string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Subscribe(sessionID); err != nil {
			return wsErrorMsg{err: err}
		}
		return nil
	}
}

// startSpinner starts the spinner for the tmux pane loading state
func startSpinner(pane *panes.TmuxPane) tea.Cmd {
	if pane == nil {
		return nil
	}
	return pane.TickCmd()
}

// switchPane switches focus between panes
func (m *Model) switchPane() {
	m.activePaneIndex = (m.activePaneIndex + 1) % len(m.panes)
	for i, pane := range m.panes {
		pane.SetActive(i == m.activePaneIndex)
	}
}
