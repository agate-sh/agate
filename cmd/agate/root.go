package main

import (
	"fmt"
	"os"

	"agate/internal/version"
	"agate/pkg/agents"
	"agate/pkg/common"
	"agate/pkg/state"

	"github.com/spf13/cobra"
)

// ResolveAgentNames returns agent names based on CLI flags and state.
// Priority: CLI flag > saved state > ClaudeAgent
// Guaranteed to return at least one non-empty agent name.
func ResolveAgentNames(agentsFlag string, stateManager *state.Manager) []string {
	// Priority 1: Explicit CLI flag
	if agentsFlag != "" {
		if parsed := common.ParseCommaSeparated(agentsFlag); len(parsed) > 0 {
			return parsed
		}
	}

	// Priority 2: Saved state (last used agents)
	if stateManager != nil {
		if saved := stateManager.GetSelectedAgents(); len(saved) > 0 {
			return saved
		}
	}

	// Priority 3: Default to Claude
	return []string{agents.ClaudeAgent.ExecutableName}
}

// SetupRootCommand creates and configures the root cobra command
func SetupRootCommand() *cobra.Command {
	var showVersion bool
	var agentsFlag string
	var tmuxAttachFlag bool

	rootCmd := &cobra.Command{
		Use:   "agate",
		Short: "A tmux-based terminal UI for AI agents",
		Long: `Agate provides a split-pane terminal interface for interacting with AI agents.

Supports any agent name (claude, codex, gemini, etc.) and automatically configures
colors and settings based on the agent type.

Agate provides two interaction modes:
  Preview Mode (default): Read-only view with fast, lag-free rendering
  Attached Mode (a): Full tmux experience with complete control

Press 'a' when focused on the right pane to attach to tmux.
Press Ctrl+Q when attached to detach back to preview.
Press ? for help once running.

Examples:
  agate                                           # Launch with last selected agents (defaults to claude)
  agate -a claude                                 # Launch with Claude
  agate --agents claude,codex                     # Launch with Claude and Codex
  agate "add a new feature"                       # Create session with prompt in TUI
  agate -t "add a new feature"                    # Create session with prompt and attach directly to tmux`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(version.Short())
				return nil
			}

			// If -t flag is used without prompt, error
			if tmuxAttachFlag && len(args) == 0 {
				return fmt.Errorf("-t/--tmux flag requires a prompt argument")
			}

			// If prompt provided with -t flag, direct attach (old behavior)
			if len(args) > 0 && tmuxAttachFlag {
				prompt := args[0]
				return newSessionFromCLI(agentsFlag, prompt)
			}

			// If prompt provided WITHOUT -t flag, create session and show in TUI
			if len(args) > 0 {
				prompt := args[0]
				return runAgentWithPrompt(agentsFlag, prompt)
			}

			// No prompt, no -t flag: normal TUI
			return runAgent(agentsFlag)
		},
	}

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	rootCmd.Flags().StringVarP(&agentsFlag, "agents", "a", "", "Comma-separated list of agents (e.g., claude,codex)")
	rootCmd.Flags().BoolVarP(&tmuxAttachFlag, "tmux", "t", false, "Directly attach to tmux session (requires prompt)")

	return rootCmd
}

// ExecuteCLI sets up and executes the CLI
func ExecuteCLI() {
	rootCmd := SetupRootCommand()
	metricsCmd := SetupMetricsCommand()
	rootCmd.AddCommand(metricsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
