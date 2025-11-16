package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestTracePaneIndices traces how pane indices change during layout creation
func TestTracePaneIndices(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	sessionName := "agate_trace"
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

	listPanes("Step 0: Initial")

	// Step 1
	exec.Command("tmux", "-L", "agate", "split-window", "-h", "-t", sessionName).Run()
	listPanes("Step 1: split-window -h")

	// Step 2
	exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.1", sessionName)).Run()
	listPanes("Step 2: select-pane -t 1")

	// Step 3
	exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName).Run()
	listPanes("Step 3: split-window -v (split pane 1)")

	// Step 4
	exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.3", sessionName)).Run()
	listPanes("Step 4: select-pane -t 3")

	// Step 5
	exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName).Run()
	listPanes("Step 5: split-window -v (split pane 3)")
}
