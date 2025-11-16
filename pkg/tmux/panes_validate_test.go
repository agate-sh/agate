package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestValidateAgentToPaneMapping validates that AgentToPaneMap produces correct visual layouts
func TestValidateAgentToPaneMapping(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	for count := 1; count <= 6; count++ {
		t.Run(fmt.Sprintf("%d agents", count), func(t *testing.T) {
			sessionName := fmt.Sprintf("agate_validate_%d", count)
			exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			// Create session and layout
			exec.Command("tmux", "-L", "agate", "new-session", "-d", "-s", sessionName).Run()
			defer exec.Command("tmux", "-L", "agate", "kill-session", "-t", sessionName).Run()

			layout := CalculatePaneLayout(count)

			// Execute layout commands (same logic as CreatePanes)
			for _, cmdStr := range layout.Commands {
				cmdParts := strings.Fields(cmdStr)
				args := []string{"-L", "agate"}
				args = append(args, cmdParts...)

				// For select-pane commands, prepend session name to pane index
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

				exec.Command("tmux", args...).Run()
			}

			// Get pane positions
			cmd := exec.Command("tmux", "-L", "agate", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_top}:#{pane_left}")
			out, _ := cmd.Output()
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")

			// Parse pane positions
			type panePos struct {
				row, col int
			}
			panePositions := make(map[int]panePos)
			for _, line := range lines {
				var idx, row, col int
				fmt.Sscanf(line, "%d:%d:%d", &idx, &row, &col)
				panePositions[idx] = panePos{row, col}
			}

			// Verify agent→pane mapping produces correct visual order
			t.Logf("Layout for %d agents:", count)
			t.Logf("AgentToPaneMap: %v", layout.AgentToPaneMap)

			for agentIdx, paneIdx := range layout.AgentToPaneMap {
				pos := panePositions[paneIdx]
				t.Logf("  Agent %d → Pane %d (row=%d, col=%d)", agentIdx, paneIdx, pos.row, pos.col)
			}

			// Verify agents are ordered left-to-right, top-to-bottom
			for i := 0; i < len(layout.AgentToPaneMap)-1; i++ {
				pos1 := panePositions[layout.AgentToPaneMap[i]]
				pos2 := panePositions[layout.AgentToPaneMap[i+1]]

				// Agent i+1 should be either:
				// - Same row, further right (pos2.col > pos1.col)
				// - Lower row (pos2.row > pos1.row)
				if pos2.row < pos1.row {
					t.Errorf("Agent %d (row %d) should not be above Agent %d (row %d)", i+1, pos2.row, i, pos1.row)
				} else if pos2.row == pos1.row && pos2.col <= pos1.col {
					t.Errorf("Agent %d (col %d) should be right of Agent %d (col %d) on same row", i+1, pos2.col, i, pos1.col)
				}
			}
		})
	}
}
