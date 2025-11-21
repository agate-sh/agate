package main

import (
	_ "embed"
	"fmt"
	"os"

	"agate/internal/debug"
	"agate/internal/version"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"
	"agate/pkg/telemetry"
	"agate/pkg/tmux"
	"agate/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/term"
)

// ASCII art is embedded in the welcome overlay

// PaneBaseStyle is now defined in the layout package

// managers holds the backend components that can be created before TUI launch
type managers struct {
	sessionManager  *session.Manager
	stateManager    *state.Manager
	worktreeManager *git.WorktreeManager
	debugLogger     *debug.DebugLogger
}

func checkTmuxInstalled() error {
	if !tmux.IsTmuxInstalled() {
		installCmd := tmux.GetInstallCommand()
		return fmt.Errorf("tmux is not installed. Please install tmux to use Agate.\n%s", installCmd)
	}
	return nil
}

// createManagers initializes all backend managers
func createManagers(subprocess string) (*managers, error) {
	// Initialize debug logger FIRST so all subsequent logs are captured
	debugLogger := debug.InitDebugLogger()

	// Initialize state manager (thread-safe state persistence)
	stateManager, err := state.NewManager()
	if err != nil {
		fmt.Printf("ERROR: failed to initialize state manager: %v\n", err)
		fmt.Printf("Agate will run without session persistence. Please check ~/.agate permissions.\n")
		debug.DebugLog("ERROR: StateManager initialization failed: %v", err)
		// Continue with nil stateManager - session manager will handle it gracefully
	}

	// Initialize worktree manager
	worktreeManager, err := git.NewWorktreeManager()
	if err != nil {
		// Log error but don't fail - app can still work without worktree features
		fmt.Printf("Warning: failed to initialize worktree manager: %v\n", err)
	}

	// Create session manager with state manager
	sessionManager := session.NewManager(worktreeManager, stateManager)

	// Load existing sessions from persistence
	if err := sessionManager.RestoreSessions(); err != nil {
		debug.DebugLog("Failed to restore sessions on startup: %v", err)
		// Don't fail startup if session restoration fails
	}

	// Set up debug logging for git package (always enabled now)
	git.DebugLog = debug.DebugLog

	return &managers{
		sessionManager:  sessionManager,
		stateManager:    stateManager,
		worktreeManager: worktreeManager,
		debugLogger:     debugLogger,
	}, nil
}

func runAgent(subprocess string) error {
	return runAgentWithPrompt(subprocess, "")
}

func runAgentWithPrompt(subprocess string, initialPrompt string) error {
	if err := checkTmuxInstalled(); err != nil {
		return err
	}

	// Track TUI started
	telemetry.TrackTUIStarted()

	// Get actual terminal size before creating Bubble Tea program
	// This ensures we create sessions with the correct dimensions from the start
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to defaults if we can't detect terminal size
		width, height = 200, 50
	}

	// Create managers first (always needed)
	mgrs, err := createManagers(subprocess)
	if err != nil {
		return fmt.Errorf("failed to initialize managers: %v", err)
	}

	// If we have an initial prompt, create the session BEFORE launching TUI
	// This prevents empty panes from showing
	var preCreatedSession *session.Session
	if initialPrompt != "" {
		// Start spinner
		stopSpinner := startSpinner("Starting agents...")

		// Generate branch name
		repoName := ""
		if mgrs.worktreeManager != nil {
			repoName = mgrs.worktreeManager.GetRepositoryName()
		}
		branchName := session.GenerateBranchNameFromPrompt(repoName)

		// Resolve agents
		agentNames := ResolveAgentNames(subprocess, mgrs.stateManager)

		// Create session synchronously
		preCreatedSession, err = mgrs.sessionManager.CreateSessionWithSize(initialPrompt, branchName, agentNames, width, height)

		// Stop spinner and clear line
		stopSpinner()

		if err != nil {
			return fmt.Errorf("failed to create session: %v", err)
		}
	}

	model := tui.NewModelWithConfig(tui.InitConfig{
		Subprocess:        subprocess,
		InitialPrompt:     initialPrompt,
		TermWidth:         width,
		TermHeight:        height,
		DebugLogger:       mgrs.debugLogger,
		StateManager:      mgrs.stateManager,
		WorktreeManager:   mgrs.worktreeManager,
		SessionManager:    mgrs.sessionManager,
		PreCreatedSession: preCreatedSession,
	})

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %v", err)
	}
	return nil
}


func main() {
	zone.NewGlobal()

	// Initialize analytics (disabled for dev builds)
	apiKey := os.Getenv("POSTHOG_API_KEY")
	if err := telemetry.Init(apiKey, version.Short()); err != nil {
		debug.DebugLog("Failed to initialize analytics: %v", err)
		// Continue - app works without analytics
	}
	defer telemetry.Close()

	// Get distinct ID and identify user with environment properties
	if distinctID, err := telemetry.GetDistinctId(); err == nil {
		telemetry.SetDistinctId(distinctID)
		envProps := telemetry.GetEnvironmentProperties(version.Short())
		if err := telemetry.Identify(distinctID, envProps); err != nil {
			debug.DebugLog("Failed to identify user: %v", err)
		}
	}

	// Execute CLI
	ExecuteCLI()
}
