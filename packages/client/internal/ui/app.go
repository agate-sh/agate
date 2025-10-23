package ui

import (
	"context"
	"fmt"
	"os"
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

type worktree struct {
	Path     string
	Branch   string
	Commit   string
	Bare     bool
	Detached bool
}

type loadSuccessMsg struct {
	repoName     string
	repoRoot     string
	currentBranch string
	worktrees    []worktree
}

type loadErrorMsg struct {
	err error
}

// Model is the Bubble Tea program model for the agents-first UI slice.
type Model struct {
	client   *agateclient.APIClient
	repoPath string

	state          loadState
	repoName       string
	repoRoot       string
	currentBranch  string
	agents         []agents.Config
	selectedAgent  int
	worktrees      []worktree
	selectedTree   int
	err            error
}

// NewModel constructs a model wired up to the configured API client.
func NewModel(client *agateclient.APIClient, repoPath string) Model {
	return Model{
		client:  client,
		repoPath: repoPath,
		state:   stateLoading,
		agents:  agents.All(),
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
			if m.state == stateReady && m.selectedTree > 0 {
				m.selectedTree--
			}
		case "down":
			if m.state == stateReady && m.selectedTree < len(m.worktrees)-1 {
				m.selectedTree++
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
			m.state = stateLoading
			m.err = nil
			return m, fetchRepoSnapshot(m.client, m.repoPath)
		}
	case loadSuccessMsg:
		m.state = stateReady
		m.err = nil
		m.repoName = msg.repoName
		m.repoRoot = msg.repoRoot
		m.currentBranch = msg.currentBranch
		m.worktrees = msg.worktrees
		if len(m.worktrees) == 0 {
			m.selectedTree = 0
		} else if m.selectedTree >= len(m.worktrees) {
			m.selectedTree = len(m.worktrees) - 1
		}
	case loadErrorMsg:
		m.state = stateError
		m.err = msg.err
	}

	return m, nil
}

// View renders the current UI.
func (m Model) View() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9d87ae"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	listStyle := lipgloss.NewStyle().PaddingLeft(2)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).PaddingTop(1)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).PaddingTop(1)

	b.WriteString(headerStyle.Render("Agate — Agents Panel Preview"))
	b.WriteString("\n\n")

	switch m.state {
	case stateLoading:
		b.WriteString("Loading repository data…")
	case stateError:
		b.WriteString(errorStyle.Render(fmt.Sprintf("Failed to load data: %v", m.err)))
		b.WriteString(helpStyle.Render("\nPress 'r' to retry or 'q' to quit."))
	default:
		b.WriteString(sectionStyle.Render("Repository"))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Name: %s", fallback(m.repoName, "unknown"))))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Root: %s", fallback(m.repoRoot, m.repoPath))))
		b.WriteString("\n")
		b.WriteString(listStyle.Render(fmt.Sprintf("Current Branch: %s", fallback(m.currentBranch, "n/a"))))
		b.WriteString("\n\n")

		b.WriteString(sectionStyle.Render("Agents"))
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
		if len(m.worktrees) == 0 {
			b.WriteString(listStyle.Render("No worktrees found. Create a session to initialize one."))
			b.WriteString("\n")
		} else {
			for idx, wt := range m.worktrees {
				prefix := " "
				if idx == m.selectedTree {
					prefix = "•"
				}
				line := fmt.Sprintf("%s %-20s %s", prefix, wt.Branch, wt.Path)
				if wt.Detached {
					line += " (detached)"
				}
				b.WriteString(listStyle.Render(line))
				b.WriteString("\n")
			}
		}

		b.WriteString(helpStyle.Render("\n←/→ select agent • ↑/↓ select worktree • r reload • q quit"))
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

		worktrees := make([]worktree, 0, len(workResp.GetWorktrees()))
		for _, wt := range workResp.GetWorktrees() {
			worktrees = append(worktrees, worktree{
				Path:     wt.GetPath(),
				Branch:   wt.GetBranch(),
				Commit:   wt.GetCommit(),
				Bare:     wt.GetBare(),
				Detached: wt.GetDetached(),
			})
		}

		return loadSuccessMsg{
			repoName:      repoInfo.GetRepoName(),
			repoRoot:      repoInfo.GetRepoRoot(),
			currentBranch: repoInfo.GetCurrentBranch(),
			worktrees:     worktrees,
		}
	}
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
