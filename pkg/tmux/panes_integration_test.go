package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestTmuxPaneIndexing tests actual tmux pane behavior to understand indexing
// This is an integration test that requires tmux to be installed
func TestTmuxPaneIndexing(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping integration test")
	}

	sessionName := "agate_test_panes"

	// Cleanup any existing session
	exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	// Create new session
	cmd := exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create tmux session: %v", err)
	}
	defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

	// Helper to list panes
	listPanes := func() ([]string, error) {
		cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}")
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		return lines, nil
	}

	// Helper to get pane positions
	getPanePositions := func() (string, error) {
		cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top},#{pane_left}")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}

	t.Run("Initial state", func(t *testing.T) {
		panes, err := listPanes()
		if err != nil {
			t.Fatalf("Failed to list panes: %v", err)
		}
		t.Logf("Initial panes: %v", panes)
		if len(panes) != 1 {
			t.Errorf("Expected 1 pane, got %d", len(panes))
		}
	})

	t.Run("After horizontal split", func(t *testing.T) {
		cmd := exec.Command("tmux", "-L", "agate", "split-window", "-h", "-t", sessionName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to split horizontally: %v", err)
		}

		panes, err := listPanes()
		if err != nil {
			t.Fatalf("Failed to list panes: %v", err)
		}
		positions, _ := getPanePositions()
		t.Logf("After horizontal split - panes: %v", panes)
		t.Logf("Positions:\n%s", positions)

		if len(panes) != 2 {
			t.Errorf("Expected 2 panes, got %d", len(panes))
		}
	})

	t.Run("After first vertical split on left pane", func(t *testing.T) {
		// Try selecting pane 0 first
		cmd := exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.0", sessionName))
		err := cmd.Run()
		t.Logf("Selecting pane 0: error=%v", err)

		// Try selecting pane 1
		cmd = exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.1", sessionName))
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to select pane 1: %v", err)
		}

		cmd = exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to split vertically: %v", err)
		}

		panes, err := listPanes()
		if err != nil {
			t.Fatalf("Failed to list panes: %v", err)
		}
		positions, _ := getPanePositions()
		t.Logf("After first vertical split - panes: %v", panes)
		t.Logf("Positions:\n%s", positions)

		if len(panes) != 3 {
			t.Errorf("Expected 3 panes, got %d", len(panes))
		}
	})

	t.Run("After second vertical split on right pane for 2x2 grid", func(t *testing.T) {
		// We need to figure out which pane is the top-right
		// After the previous split, we should have:
		// - Left column split vertically
		// - Right column still single pane
		positions, _ := getPanePositions()
		t.Logf("Current positions before 2nd split:\n%s", positions)

		// The right pane should be the one that wasn't split
		// Let's try pane 2
		cmd := exec.Command("tmux", "-L", "agate", "select-pane", "-t", fmt.Sprintf("%s.2", sessionName))
		if err := cmd.Run(); err != nil {
			t.Logf("Failed to select pane 2: %v", err)
		}

		cmd = exec.Command("tmux", "-L", "agate", "split-window", "-v", "-t", sessionName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to split vertically: %v", err)
		}

		panes, err := listPanes()
		if err != nil {
			t.Fatalf("Failed to list panes: %v", err)
		}
		positions, _ = getPanePositions()
		t.Logf("After second vertical split - panes: %v", panes)
		t.Logf("Final positions:\n%s", positions)

		if len(panes) != 4 {
			t.Errorf("Expected 4 panes, got %d", len(panes))
		}
	})
}

// TestTmuxPaneIndexingAllLayouts tests all 1-6 agent layouts
func TestTmuxPaneIndexingAllLayouts(t *testing.T) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping integration test")
	}

	tests := []struct {
		name       string
		agentCount int
	}{
		{"1 agent", 1},
		{"2 agents", 2},
		{"3 agents", 3},
		{"4 agents", 4},
		{"5 agents", 5},
		{"6 agents", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName := fmt.Sprintf("agate_test_%d", tt.agentCount)

			// Cleanup
			exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			// Create session
			cmd := exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName)
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to create tmux session: %v", err)
			}
			defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			// Get layout commands
			layout := CalculatePaneLayout(tt.agentCount)

			// Execute each command
			for i, cmdStr := range layout.Commands {
				args := strings.Fields(cmdStr)
				args = append([]string{"-L", "agate"}, args...)
				args = append(args, "-t", sessionName)

				cmd := exec.Command("tmux", args...)
				if err := cmd.Run(); err != nil {
					t.Errorf("Command %d (%s) failed: %v", i, cmdStr, err)
					continue
				}
			}

			// Verify final pane count
			cmd = exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}")
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("Failed to list panes: %v", err)
			}

			panes := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(panes) != tt.agentCount {
				t.Errorf("Expected %d panes, got %d", tt.agentCount, len(panes))
				t.Logf("Panes: %v", panes)
			}

			// Log positions for debugging
			cmd = exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top},#{pane_left}")
			out, _ = cmd.Output()
			t.Logf("Final layout for %d agents:\n%s", tt.agentCount, strings.TrimSpace(string(out)))
		})
	}
}
