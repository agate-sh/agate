package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agate/client/internal/agents"
	"agate/client/internal/api"
	agateclient "agate/sdk/gen"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// Model is the Bubble Tea program model for the agents-first UI slice.
type Model struct {
	client   *agateclient.APIClient
	repoPath string

	state            loadState
	agents           []agents.Config
	selectedAgent    int
	snapshot         repoSnapshot
	entries          []displayEntry
	selectedIndex    int
	lastSelectedPath string
	err              error
}

// NewModel constructs a model wired up to the configured API client.
func NewModel(client *agateclient.APIClient, repoPath string) Model {
	return Model{
		client:        client,
		repoPath:      repoPath,
		state:         stateLoading,
		agents:        agents.All(),
		selectedIndex: -1,
	}
}

// Init kicks off loading repository data.
func (m Model) Init() tea.Cmd {
	return fetchRepoSnapshot(m.client, m.repoPath)
}

// Update handles key events and async data loading.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.state == stateReady && len(m.entries) > 0 && m.selectedIndex > 0 {
				m.selectedIndex--
				m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
			}
		case "down":
			if m.state == stateReady && len(m.entries) > 0 && m.selectedIndex < len(m.entries)-1 {
				m.selectedIndex++
				m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
			}
		case "left":
			if m.selectedAgent > 0 {
				m.selectedAgent--
			}
		case "right":
			if m.selectedAgent < len(m.agents)-1 {
				m.selectedAgent++
			}
		case "r":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.entries) {
				m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
			}
			m.state = stateLoading
			m.err = nil
			return m, fetchRepoSnapshot(m.client, m.repoPath)
		case "n":
			if m.state == stateReady && len(m.agents) > 0 {
				agent := m.agents[m.selectedAgent]
				repoRoot := fallback(m.snapshot.RepoRoot, m.repoPath)
				branchSeed := m.newBranchSeed(agent.Name)
				if m.selectedIndex >= 0 && m.selectedIndex < len(m.entries) {
					m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
				}
				m.state = stateLoading
				m.err = nil
				return m, createSessionCmd(m.client, repoRoot, branchSeed, agent.Name)
			}
		case "d":
			if m.state == stateReady && m.selectedIndex >= 0 && m.selectedIndex < len(m.entries) {
				entry := m.entries[m.selectedIndex]
				if entry.Worktree != nil && entry.Worktree.Session != nil {
					sessionID := entry.Worktree.Session.ID
					m.lastSelectedPath = ""
					m.state = stateLoading
					m.err = nil
					return m, deleteSessionCmd(m.client, sessionID)
				}
			}
		}
	case loadSuccessMsg:
		m.state = stateReady
		m.err = nil
		m.snapshot = msg.snapshot
		if idx := m.agentIndexByName(m.snapshot.DefaultAgent); idx >= 0 {
			m.selectedAgent = idx
		}
		m.entries = buildDisplayEntries(m.snapshot)

		if len(m.entries) == 0 {
			m.selectedIndex = -1
			m.lastSelectedPath = ""
		} else {
			desiredPath := m.lastSelectedPath
			if desiredPath == "" && m.snapshot.ActiveSession != "" {
				if entry := findEntryBySession(m.entries, m.snapshot.ActiveSession); entry != nil {
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
			if m.selectedIndex == -1 {
				m.selectedIndex = 0
			}
			m.lastSelectedPath = m.entries[m.selectedIndex].Worktree.Path
		}
	case loadErrorMsg:
		m.state = stateError
		m.err = msg.err
	case createSessionResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		if msg.resp != nil {
			m.lastSelectedPath = filepath.Clean(msg.resp.GetWorktreePath())
		}
		return m, fetchRepoSnapshot(m.client, m.repoPath)
	case deleteSessionResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		return m, fetchRepoSnapshot(m.client, m.repoPath)
	}

	return m, nil
}

// View renders the current UI.
func (m Model) View() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9d87ae"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	listStyle := lipgloss.NewStyle().PaddingLeft(2)
	subsectionStyle := listStyle.PaddingLeft(0).Bold(true)
	itemStyle := lipgloss.NewStyle().PaddingLeft(4)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).PaddingTop(1)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).PaddingTop(1)
	emptyStyle := listStyle.Foreground(lipgloss.Color("#555555"))

	b.WriteString(headerStyle.Render("Agate — Agents Panel Preview"))
	b.WriteString("\n\n")

	switch m.state {
	case stateLoading:
		b.WriteString("Loading repository data…")
	case stateError:
		b.WriteString(errorStyle.Render(fmt.Sprintf("Failed to load data: %v", m.err)))
		b.WriteString(helpStyle.Render("\nPress 'r' to retry or 'q' to quit."))
	default:
		snapshot := m.snapshot

		b.WriteString(sectionStyle.Render("Repository"))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Name: %s", fallback(snapshot.RepoName, "unknown"))))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Root: %s", fallback(snapshot.RepoRoot, m.repoPath))))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Current Branch: %s", fallback(snapshot.CurrentBranch, "n/a"))))
		if snapshot.ActiveSession != "" {
			if entry := findEntryBySession(m.entries, snapshot.ActiveSession); entry != nil && entry.Worktree != nil && entry.Worktree.Session != nil {
				b.WriteString("\n")
				active := entry.Worktree
				b.WriteString(listStyle.Render(fmt.Sprintf("Active Agent: %s on %s", active.Session.AgentName, fallback(active.Branch, humanizePath(active.Path)))))
			}
		}
		b.WriteString("\n\n")

		b.WriteString(sectionStyle.Render("Available Agents"))
		b.WriteString("\n")
		for idx, agent := range m.agents {
			prefix := " "
			if idx == m.selectedAgent {
				prefix = "›"
			}
			line := fmt.Sprintf("%s %-12s (%s)", prefix, agent.CompanyName, agent.Name)
			agentStyle := listStyle
			if idx == m.selectedAgent {
				agentStyle = agentStyle.Foreground(lipgloss.Color(agent.BorderColor)).Bold(true)
			}
			b.WriteString(agentStyle.Render(line))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("Worktrees"))
		b.WriteString("\n")
		if len(m.entries) == 0 {
			b.WriteString(emptyStyle.Render("No worktrees found. Create an agent session to initialize one."))
			b.WriteString("\n")
		} else {
			mainIdx := indicesForSection(m.entries, sectionMain)
			linkedIdx := indicesForSection(m.entries, sectionLinked)

			renderSection := func(title string, indices []int, emptyMessage string) {
				b.WriteString(subsectionStyle.Render(title))
				b.WriteString("\n")
				if len(indices) == 0 {
					b.WriteString(emptyStyle.Render("   " + emptyMessage))
					b.WriteString("\n")
					return
				}
				for _, idx := range indices {
					entry := m.entries[idx]
					if entry.Worktree == nil {
						continue
					}
					wt := entry.Worktree
					prefix := " "
					if wt.IsActive {
						prefix = "•"
					}
					if idx == m.selectedIndex {
						prefix = "▸"
					}
					branchLabel := fallback(wt.Branch, filepath.Base(wt.Path))
					if wt.Detached {
						branchLabel += " (detached)"
					}

					sessionLabel := "no agent"
					color := "#9d87ae"
					if wt.Session != nil {
						sessionLabel = wt.Session.AgentName
						color = m.agentColor(wt.Session.AgentName)
						if wt.IsActive {
							sessionLabel += " (active)"
						}
					} else if entry.Section == sectionLinked {
						sessionLabel = "empty worktree"
					}

					line := fmt.Sprintf("%s %-20s — %s", prefix, branchLabel, sessionLabel)
					lineStyle := itemStyle

					if wt.IsActive {
						lineStyle = lineStyle.Foreground(lipgloss.Color(color)).Bold(true)
					}
					if idx == m.selectedIndex {
						lineStyle = lineStyle.Foreground(lipgloss.Color(color)).Bold(true)
					}

					b.WriteString(lineStyle.Render(line))
					b.WriteString("\n")

					if idx == m.selectedIndex {
						b.WriteString(itemStyle.PaddingLeft(2).Render(humanizePath(wt.Path)))
						b.WriteString("\n")
						if wt.Session != nil && wt.Session.TmuxName != "" {
							b.WriteString(itemStyle.PaddingLeft(2).Foreground(lipgloss.Color("#777777")).Render(fmt.Sprintf("tmux: %s", wt.Session.TmuxName)))
							b.WriteString("\n")
						}
					}
				}
			}

			renderSection("Main worktree", mainIdx, "No main worktree sessions.")
			b.WriteString("\n")
			renderSection("Linked worktrees", linkedIdx, "No linked sessions yet. Create one with 'n'.")
		}

		b.WriteString(helpStyle.Render("\n←/→ select agent • ↑/↓ select worktree • n new session • d delete session • r reload • q quit"))
	}

	return b.String()
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
	program := tea.NewProgram(model)
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

		workResp, _, err := client.DefaultAPI.GitWorktreesList(ctx).RepoPath(repoInfo.GetRepoRoot()).Execute()
		if err != nil {
			return loadErrorMsg{err: fmt.Errorf("git worktrees: %w", err)}
		}

		sessionResp, err := api.ListSessions(ctx, client)
		if err != nil {
			return loadErrorMsg{err: err}
		}

		sessionByPath := map[string]*sessionSummary{}
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
			sessionByPath[filepath.Clean(summary.WorktreePath)] = summary
		}

		worktrees := make([]worktreeEntry, 0, len(workResp.GetWorktrees())+len(sessionByPath))
		for _, wt := range workResp.GetWorktrees() {
			path := filepath.Clean(wt.GetPath())
			entry := worktreeEntry{
				Path:     path,
				Branch:   wt.GetBranch(),
				Commit:   wt.GetCommit(),
				Bare:     wt.GetBare(),
				Detached: wt.GetDetached(),
				IsMain:   wt.GetIsMain(),
			}
			if summary, ok := sessionByPath[path]; ok {
				entry.Session = summary
				delete(sessionByPath, path)
			}
			worktrees = append(worktrees, entry)
		}

		// Include persisted sessions whose worktrees are missing from git output (stale or detached)
		for _, summary := range sessionByPath {
			entry := worktreeEntry{
				Path:     filepath.Clean(summary.WorktreePath),
				Branch:   summary.Branch,
				Commit:   "",
				Bare:     false,
				Detached: false,
				IsMain:   !strings.Contains(summary.WorktreePath, "/.agate/worktrees/"),
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
	for _, cfg := range m.agents {
		if strings.EqualFold(cfg.Name, agentName) {
			return cfg.BorderColor
		}
	}
	return "#9d87ae"
}

func (m Model) agentIndexByName(agentName string) int {
	for idx, cfg := range m.agents {
		if strings.EqualFold(cfg.Name, agentName) {
			return idx
		}
	}
	return -1
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
