package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestFourAgentLayoutFromCode tests the exact commands that CalculatePaneLayout generates for 4 agents
func TestFourAgentLayoutFromCode(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	sessionName := "agate_4_from_code"
	exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	// Create session
	cmd := exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	listPanes := func(label string) {
		cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top},#{pane_left}")
		out, _ := cmd.Output()
		t.Logf("%s:\n%s", label, strings.TrimSpace(string(out)))
	}

	// Get the layout from CalculatePaneLayout
	layout := CalculatePaneLayout(4)
	t.Logf("Commands from CalculatePaneLayout(4): %v", layout.Commands)
	t.Logf("AgentToPaneMap: %v", layout.AgentToPaneMap)

	listPanes("Initial")

	// Execute each command (using same logic as CreatePanes)
	for i, cmdStr := range layout.Commands {
		cmdParts := strings.Fields(cmdStr)
		args := []string{"-L", "agate"}
		args = append(args, cmdParts...)

		// For select-pane commands, the -t argument needs to be "sessionName.paneIndex"
		if len(cmdParts) > 0 && cmdParts[0] == "select-pane" {
			for j, arg := range args {
				if arg == "-t" && j+1 < len(args) {
					args[j+1] = fmt.Sprintf("%s.%s", sessionName, args[j+1])
					break
				}
			}
		} else {
			args = append(args, "-t", sessionName)
		}

		t.Logf("Executing: tmux %v", args)
		if err := exec.Command("tmux", args...).Run(); err != nil {
			t.Fatalf("Command %d failed: %v", i, err)
		}

		listPanes(fmt.Sprintf("After command %d: %s", i, cmdStr))
	}
}
