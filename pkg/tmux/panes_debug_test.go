package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestDebug4AgentLayout traces through the 4-agent layout step by step
func TestDebug4AgentLayout(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	sessionName := "agate_debug_4"
	exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	// Create session
	cmd := exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	getLayout := func() string {
		cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top},#{pane_left} size=#{pane_width}x#{pane_height}")
		out, _ := cmd.Output()
		return strings.TrimSpace(string(out))
	}

	t.Log("Initial state:")
	t.Log(getLayout())

	// Step 1: split-window -h
	exec.Command("tmux", "-L", "agate", "split-window", "-h", "-t", sessionName).Run()
	t.Log("\nAfter split-window -h:")
	t.Log(getLayout())

	// Step 2: select-pane -t 1
	exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.1", sessionName)).Run()
	t.Log("\nAfter select-pane -t 1 (selecting left pane):")
	t.Log(getLayout())

	// Step 3: split-window -v
	exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName).Run()
	t.Log("\nAfter split-window -v (splitting left pane vertically):")
	t.Log(getLayout())

	// Step 4: select-pane -t 3
	exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.3", sessionName)).Run()
	t.Log("\nAfter select-pane -t 3 (should be right pane):")
	t.Log(getLayout())

	// Step 5: split-window -v
	exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName).Run()
	t.Log("\nAfter split-window -v (splitting right pane vertically):")
	t.Log(getLayout())

	t.Log("\n=== Expected 2x2 grid ===")
	t.Log("Top row: panes 1,2")
	t.Log("Bottom row: panes 3,4")
}
