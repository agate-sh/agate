package main

import (
	"fmt"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/app"
	"agate/pkg/git"
	"agate/pkg/session"
	"agate/pkg/state"
)

// newSessionFromCLI creates a new session from CLI arguments and attaches directly to tmux
func newSessionFromCLI(agentsFlag string, prompt string) error {
	if err := checkTmuxInstalled(); err != nil {
		return err
	}

	// Initialize debug logger
	debugLogger := debug.InitDebugLogger()
	defer debugLogger.Close()
	debug.DebugLog("CLI: newSessionFromCLI started with agents=%s, prompt=%s", agentsFlag, prompt)

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
	agentConfigs := make([]app.AgentConfig, 0, len(agentNames))
	for _, name := range agentNames {
		agentConfigs = append(agentConfigs, app.GetAgentConfig(name))
	}

	// Start spinner
	stopSpinner := startSpinner("Starting agents...")
	defer stopSpinner()

	// Generate branch name from prompt
	branchName, err := session.GenerateBranchNameFromPrompt(prompt, agentConfigs[0])
	if err != nil {
		debug.DebugLog("Branch name generation failed: %v, using random name", err)
		branchName = session.GenerateRandomBranchName()
	}

	// Create session
	sess, err := sessionManager.CreateSession(prompt, branchName, agentNames)
	if err != nil {
		stopSpinner()
		return fmt.Errorf("failed to create session: %v", err)
	}

	// Wait for session to be ready (poll until hasPrompt is true)
	if sess.SharedTmux != nil {
		debug.DebugLog("CLI: Waiting for session to be ready...")
		for {
			content, err := sess.SharedTmux.CapturePaneContent()
			if err != nil {
				debug.DebugLog("CLI: Error capturing pane content: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			debug.DebugLog("CLI: Captured %d bytes of content", len(content))
			// Log last 200 chars to see what we're checking
			preview := content
			if len(preview) > 200 {
				preview = "..." + preview[len(preview)-200:]
			}
			debug.DebugLog("CLI: Content preview: %q", preview)

			// Call HasUpdated to check hasPrompt (we don't care about updated flag here)
			_, hasPrompt := sess.SharedTmux.HasUpdated()
			debug.DebugLog("CLI: hasPrompt=%v", hasPrompt)

			if hasPrompt {
				debug.DebugLog("CLI: Session is ready! Breaking out of wait loop")
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Stop spinner and clear line
	stopSpinner()

	// Attach to tmux session
	if sess.SharedTmux != nil {
		detachCh, err := sess.SharedTmux.Attach()
		if err != nil {
			return fmt.Errorf("failed to attach to tmux session: %v", err)
		}
		<-detachCh
	}

	return nil
}

// startSpinner starts a terminal spinner and returns a function to stop it
func startSpinner(message string) func() {
	// Agate purple color
	purple := "\033[38;2;157;135;174m"
	reset := "\033[0m"

	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	done := make(chan bool)

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
