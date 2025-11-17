package tmux

import (
	"fmt"
	"os/exec"
	"strings"

	"agate/internal/debug"
	"agate/pkg/agents"
	"agate/pkg/git"
)

// PaneLayout represents the layout strategy for multiple agents in a single tmux session
type PaneLayout struct {
	AgentCount     int
	Commands       []string // tmux commands to create the layout
	AgentToPaneMap []int    // Maps agent index to tmux pane index (1-based)
}

// getLayoutName returns the tmux layout name to apply for equalizing pane sizes
func getLayoutName(agentCount int) string {
	switch agentCount {
	case 1:
		return "" // No layout needed for single pane
	case 2, 3:
		return "even-horizontal" // Equal horizontal splits
	case 4, 5, 6:
		return "tiled" // Tiled layout for grids
	default:
		return "tiled"
	}
}

// CalculatePaneLayout returns the tmux commands needed to create a pane layout for N agents
// Layouts follow the specification:
// 1: single pane (no splits)
// 2: 1x2 (horizontal split)
// 3: 1x3 (two horizontal splits)
// 4: 2x2 (horizontal split, then vertical on each)
// 5: 1x2 top, 1x3 bottom
// 6: 2x3 (full grid)
func CalculatePaneLayout(count int) *PaneLayout {
	if count < 1 {
		count = 1
	}
	if count > 6 {
		count = 6
	}

	layout := &PaneLayout{
		AgentCount: count,
		Commands:   make([]string, 0),
	}

	switch count {
	case 1:
		// No splits needed for single pane
		// Commands will be empty
		layout.AgentToPaneMap = []int{1}
	case 2:
		// 1x2: Split horizontally once
		layout.Commands = []string{
			"split-window -h",
		}
		layout.AgentToPaneMap = []int{1, 2} // Agent 0→Pane 1 (left), Agent 1→Pane 2 (right)
	case 3:
		// 1x3: Split horizontally twice
		layout.Commands = []string{
			"split-window -h",
			"split-window -h",
		}
		layout.AgentToPaneMap = []int{1, 2, 3} // Left to right
	case 4:
		// 2x2: Split horizontally, then split each pane vertically
		// Final panes: 1=top-left, 3=top-right, 2=bottom-left, 4=bottom-right
		layout.Commands = []string{
			"split-window -h",          // Create right column (panes: 1, 2)
			"select-pane -t 1",         // Select left pane
			"split-window -v",          // Split left vertically (panes: 1, 2, 3)
			"select-pane -t 3",         // Select right pane (was 2, now 3)
			"split-window -v",          // Split right vertically (panes: 1, 2, 3, 4)
		}
		// Visual layout:
		// [Agent 0][Agent 1]
		// [Agent 2][Agent 3]
		layout.AgentToPaneMap = []int{1, 3, 2, 4}
	case 5:
		// Top row: 1x2, Bottom row: 1x3
		layout.Commands = []string{
			"split-window -h",          // Create top-right (panes: 1, 2)
			"select-pane -t 1",         // Select top-left
			"split-window -v",          // Create bottom-left (panes: 1, 2, 3)
			"split-window -h",          // Create bottom-middle (panes: 1, 2, 3, 4)
			"split-window -h",          // Create bottom-right (panes: 1, 2, 3, 4, 5)
		}
		// Visual layout:
		// [Agent 0][Agent 1]
		// [Agent 2][Agent 3][Agent 4]
		// Actual pane positions: 1=top-left, 5=top-right, 2=bottom-left, 3=bottom-mid, 4=bottom-right
		layout.AgentToPaneMap = []int{1, 5, 2, 3, 4}
	case 6:
		// 2x3: Three columns, each split vertically
		layout.Commands = []string{
			"split-window -h",          // Create middle column (panes: 1, 2)
			"split-window -h",          // Create right column (panes: 1, 2, 3)
			"select-pane -t 1",         // Select left
			"split-window -v",          // Split left vertically (panes: 1, 2, 3, 4)
			"select-pane -t 3",         // Select middle (was 2, now 3)
			"split-window -v",          // Split middle vertically (panes: 1, 2, 3, 4, 5)
			"select-pane -t 5",         // Select right (was 3, now 5)
			"split-window -v",          // Split right vertically (panes: 1, 2, 3, 4, 5, 6)
		}
		// Visual layout:
		// [Agent 0][Agent 1][Agent 2]
		// [Agent 3][Agent 4][Agent 5]
		// Actual pane positions: top row is 1,3,5; bottom row is 2,4,6
		layout.AgentToPaneMap = []int{1, 3, 5, 2, 4, 6}
	}

	return layout
}

// CreatePanes creates panes in a tmux session and starts agents in each pane
// sessionName: the tmux session name (already created)
// agents: ordered list of agent configs
// worktrees: ordered list of worktree paths (matches agents order)
// sessionID: the session ID (used for metrics pane)
func CreatePanes(sessionName string, agents []agents.AgentConfig, worktrees []*git.WorktreeInfo, sessionID string) error {
	if len(agents) != len(worktrees) {
		return fmt.Errorf("agent count (%d) must match worktree count (%d)", len(agents), len(worktrees))
	}

	if len(agents) == 0 {
		return fmt.Errorf("at least one agent is required")
	}

	if len(agents) > 6 {
		return fmt.Errorf("maximum 6 agents supported, got %d", len(agents))
	}

	debug.DebugLog("CreatePanes: Creating %d panes for session %s", len(agents), sessionName)

	// Get the layout commands
	layout := CalculatePaneLayout(len(agents))

	// Execute each layout command
	for i, cmd := range layout.Commands {
		// Split the command into arguments
		cmdParts := strings.Fields(cmd)
		args := []string{"-L", "agate"}
		args = append(args, cmdParts...)

		// For select-pane commands, the -t argument needs to be "sessionName.paneIndex"
		// For other commands, the -t argument is just "sessionName"
		if len(cmdParts) > 0 && cmdParts[0] == "select-pane" {
			// Find the -t argument and prepend session name
			for j, arg := range args {
				if arg == "-t" && j+1 < len(args) {
					args[j+1] = fmt.Sprintf("%s.%s", sessionName, args[j+1])
					break
				}
			}
		} else {
			// For split-window commands, append -t sessionName
			args = append(args, "-t", sessionName)
		}

		tmuxCmd := exec.Command(tmuxPath, args...)
		if err := tmuxCmd.Run(); err != nil {
			return fmt.Errorf("failed to execute layout command %d (%s): %w", i, cmd, err)
		}
		debug.DebugLog("CreatePanes: Executed layout command: %s", cmd)
	}

	// Apply layout to equalize agent pane sizes FIRST
	layoutName := getLayoutName(len(agents))
	if layoutName != "" {
		selectLayoutCmd := Command("select-layout", "-t", sessionName, layoutName)
		if err := selectLayoutCmd.Run(); err != nil {
			debug.DebugLog("Warning: failed to apply layout %s: %v", layoutName, err)
			// Don't fail - layout is cosmetic
		} else {
			debug.DebugLog("CreatePanes: Applied layout: %s", layoutName)
		}
	}

	// Now start each agent in its corresponding pane
	for i, agent := range agents {
		paneIndex := layout.AgentToPaneMap[i]
		if err := startAgentInPane(sessionName, paneIndex, agent, worktrees[i]); err != nil {
			return fmt.Errorf("failed to start agent %s in pane %d: %w", agent.Name, paneIndex, err)
		}
	}

	debug.DebugLog("CreatePanes: Successfully created %d agent panes", len(agents))
	return nil
}

// CreateMetricsPane creates the metrics pane at the bottom of the session
// This should be called AFTER the window has been resized to its final size
// to avoid proportional scaling of the pane height
func CreateMetricsPane(sessionName string, sessionID string) error {
	// Create metrics pane at bottom using 10 absolute lines
	// This pane is only visible when attached to tmux, not in Bubble Tea preview
	// Use -f flag to create a full-width split (not nested within current pane)
	splitCmd := Command("split-window", "-v", "-f", "-l", "10", "-t", sessionName)
	if err := splitCmd.Run(); err != nil {
		debug.DebugLog("Warning: failed to create metrics pane: %v", err)
		return fmt.Errorf("failed to create metrics pane: %w", err)
	}

	debug.DebugLog("CreateMetricsPane: Created metrics pane at bottom (10 lines)")

	// Start metrics command in the bottom pane
	metricsCmd := fmt.Sprintf("agate metrics --session-id %s", sessionID)
	sendKeysCmd := Command("send-keys", "-t", sessionName, metricsCmd, "C-m")
	if err := sendKeysCmd.Run(); err != nil {
		debug.DebugLog("Warning: failed to start metrics command: %v", err)
		// Don't fail - metrics is optional
	} else {
		debug.DebugLog("CreateMetricsPane: Started metrics command in bottom pane")
	}

	return nil
}

// startAgentInPane starts an agent CLI in a specific pane
func startAgentInPane(sessionName string, paneIndex int, agent agents.AgentConfig, worktree *git.WorktreeInfo) error {
	debug.DebugLog("startAgentInPane: Starting %s in pane %d at %s", agent.Name, paneIndex, worktree.Path)

	// Select the target pane
	selectCmd := Command("select-pane", "-t", fmt.Sprintf("%s.%d", sessionName, paneIndex))
	if err := selectCmd.Run(); err != nil {
		return fmt.Errorf("failed to select pane %d: %w", paneIndex, err)
	}

	// Change directory to the worktree
	cdCmd := Command("send-keys", "-t", fmt.Sprintf("%s.%d", sessionName, paneIndex),
		fmt.Sprintf("cd %s", worktree.Path), "C-m")
	if err := cdCmd.Run(); err != nil {
		return fmt.Errorf("failed to cd to worktree: %w", err)
	}

	// Start the agent CLI
	agentCmd := agent.ExecutableName
	startCmd := Command("send-keys", "-t", fmt.Sprintf("%s.%d", sessionName, paneIndex),
		agentCmd, "C-m")
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	debug.DebugLog("startAgentInPane: Successfully started %s in pane %d", agent.Name, paneIndex)
	return nil
}

// SelectPane focuses a specific pane in the session
func SelectPane(sessionName string, paneIndex int) error {
	cmd := Command("select-pane", "-t", fmt.Sprintf("%s.%d", sessionName, paneIndex))
	return cmd.Run()
}
