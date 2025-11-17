package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/agents"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"

	"golang.org/x/term"
)

// newSessionFromCLI creates a new session from CLI arguments and attaches directly to tmux
func newSessionFromCLI(agentsFlag string, prompt string) error {
	if err := checkTmuxInstalled(); err != nil {
		return err
	}

	// Initialize debug logger
	debugLogger := debug.InitDebugLogger()
	defer debugLogger.Close()

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		fmt.Printf("Warning: failed to initialize state manager: %v\n", err)
		fmt.Printf("Session will not be persisted.\n")
	}

	// Initialize worktree manager
	worktreeManager, err := git.NewWorktreeManager()
	if err != nil {
		return fmt.Errorf("failed to initialize worktree manager: %v", err)
	}

	// Create session manager
	sessionManager := session.NewManager(worktreeManager, stateManager)

	// Determine which agents to use
	var agentNames []string
	if agentsFlag != "" {
		agentNames = strings.Split(agentsFlag, ",")
		for i := range agentNames {
			agentNames[i] = strings.TrimSpace(agentNames[i])
		}
	} else if stateManager != nil {
		agentNames = stateManager.GetSelectedAgents()
	} else {
		agentNames = []string{"claude", "codex"}
	}

	if len(agentNames) == 0 {
		agentNames = []string{"claude", "codex"}
	}

	// Convert to agent configs
	agentConfigs := make([]agents.AgentConfig, 0, len(agentNames))
	for _, name := range agentNames {
		agentConfigs = append(agentConfigs, agents.GetAgentConfig(name))
	}

	// Start spinner
	stopSpinner := startSpinner("Starting agents...")
	defer stopSpinner()

	// Detect terminal size
	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		debug.DebugLog("Warning: failed to detect terminal size: %v, using defaults", err)
		termWidth = 200
		termHeight = 50
	}
	debug.DebugLog("Detected terminal size: %dx%d", termWidth, termHeight)

	// Generate branch name from prompt
	branchName, err := session.GenerateBranchNameFromPrompt(prompt, agentConfigs[0])
	if err != nil {
		debug.DebugLog("Branch name generation failed: %v, using random name", err)
		branchName = session.GenerateRandomBranchName()
	}

	// Create session with terminal dimensions
	sess, err := sessionManager.CreateSessionWithSize(prompt, branchName, agentNames, termWidth, termHeight)
	if err != nil {
		stopSpinner()
		return fmt.Errorf("failed to create session: %v", err)
	}

	// Wait for session to be ready (poll until hasPrompt is true)
	if sess.SharedTmux != nil {
		for {
			_, err := sess.SharedTmux.CapturePaneContent()
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Call HasUpdated to check hasPrompt (we don't care about updated flag here)
			_, hasPrompt := sess.SharedTmux.HasUpdated()

			if hasPrompt {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Stop spinner and clear line
	stopSpinner()

	// Attach to tmux session using exec (proper terminal handoff for CLI mode)
	if sess.SharedTmux != nil {
		attachCmd := sess.SharedTmux.AttachCommand()
		if attachCmd == nil {
			return fmt.Errorf("failed to get attach command")
		}

		// Set stdin, stdout, stderr to terminal
		attachCmd.Stdin = os.Stdin
		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr

		// Run the attach command
		if err := attachCmd.Run(); err != nil {
			return fmt.Errorf("failed to attach to tmux session: %v", err)
		}
	}

	return nil
}

// startSpinner starts a terminal spinner and returns a function to stop it
func startSpinner(message string) func() {
	// Agate purple color
	purple := "\033[38;2;157;135;174m"
	reset := "\033[0m"

	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	done := make(chan bool, 1)

	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r%s%s%s %s", purple, spinnerChars[i%len(spinnerChars)], reset, message)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return func() {
		done <- true
		fmt.Print("\r\033[K") // Clear the line
	}
}
