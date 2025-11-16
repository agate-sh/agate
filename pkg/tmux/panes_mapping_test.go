package tmux

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// paneInfo represents a tmux pane's position
type paneInfo struct {
	index int
	row   int
	col   int
}

// TestPaneIndexMapping tests the actual pane index to position mapping for all layouts
func TestPaneIndexToPositionMapping(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	tests := []struct {
		count int
		// Expected agent index to pane index mapping
		// agentToPaneMap[agentIndex] = tmuxPaneIndex
		expectedMapping map[int]int
	}{
		{
			count: 1,
			expectedMapping: map[int]int{
				0: 1, // Agent 0 → Pane 1
			},
		},
		{
			count: 2,
			expectedMapping: map[int]int{
				0: 1, // Agent 0 → Pane 1 (left)
				1: 2, // Agent 1 → Pane 2 (right)
			},
		},
		{
			count: 3,
			expectedMapping: map[int]int{
				0: 1, // Agent 0 → Pane 1 (left)
				1: 2, // Agent 1 → Pane 2 (middle)
				2: 3, // Agent 2 → Pane 3 (right)
			},
		},
		// We'll discover the correct mappings for 4, 5, 6
		{count: 4},
		{count: 5},
		{count: 6},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d agents", tt.count), func(t *testing.T) {
			sessionName := fmt.Sprintf("agate_map_%d", tt.count)
			exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			// Create session and layout
			exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName).Run()
			defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			layout := CalculatePaneLayout(tt.count)
			for _, cmdStr := range layout.Commands {
				args := strings.Fields(cmdStr)
				args = append([]string{"-L", "agate"}, args...)
				args = append(args, "-t", sessionName)
				exec.Command("tmux", args...).Run()
			}

			// Get all panes with their positions
			cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top}:#{pane_left}")
			out, _ := cmd.Output()
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")

			var panes []paneInfo
			for _, line := range lines {
				var p paneInfo
				fmt.Sscanf(line, "%d:%d:%d", &p.index, &p.row, &p.col)
				panes = append(panes, p)
			}

			// Sort panes by position (row, then col) to get visual order
			sort.Slice(panes, func(i, j int) bool {
				if panes[i].row != panes[j].row {
					return panes[i].row < panes[j].row
				}
				return panes[i].col < panes[j].col
			})

			t.Logf("Visual layout (agent index → pane index):")
			for agentIdx, p := range panes {
				t.Logf("  Agent %d → Pane %d (row %d, col %d)", agentIdx, p.index, p.row, p.col)
			}

			// Verify against expected mapping if provided
			if tt.expectedMapping != nil {
				for agentIdx, expectedPane := range tt.expectedMapping {
					actualPane := panes[agentIdx].index
					if actualPane != expectedPane {
						t.Errorf("Agent %d: expected pane %d, got %d", agentIdx, expectedPane, actualPane)
					}
				}
			}
		})
	}
}
