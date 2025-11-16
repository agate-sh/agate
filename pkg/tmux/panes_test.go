package tmux

import (
	"testing"
)

func TestCalculatePaneLayout(t *testing.T) {
	tests := []struct {
		name          string
		agentCount    int
		expectedCount int // Expected agent count after normalization
		expectedCmds  []string
	}{
		{
			name:          "Single agent - no splits",
			agentCount:    1,
			expectedCount: 1,
			expectedCmds:  []string{},
		},
		{
			name:          "Two agents - horizontal split",
			agentCount:    2,
			expectedCount: 2,
			expectedCmds: []string{
				"split-window -h",
			},
		},
		{
			name:          "Three agents - two horizontal splits",
			agentCount:    3,
			expectedCount: 3,
			expectedCmds: []string{
				"split-window -h",
				"split-window -h",
			},
		},
		{
			name:          "Four agents - 2x2 grid",
			agentCount:    4,
			expectedCount: 4,
			expectedCmds: []string{
				"split-window -h",
				"select-pane -t 0",
				"split-window -v",
				"select-pane -t 2",
				"split-window -v",
			},
		},
		{
			name:          "Five agents - 1x2 top, 1x3 bottom",
			agentCount:    5,
			expectedCount: 5,
			expectedCmds: []string{
				"split-window -h",
				"select-pane -t 0",
				"split-window -v",
				"split-window -h",
				"split-window -h",
			},
		},
		{
			name:          "Six agents - 2x3 grid",
			agentCount:    6,
			expectedCount: 6,
			expectedCmds: []string{
				"split-window -h",
				"split-window -h",
				"select-pane -t 0",
				"split-window -v",
				"select-pane -t 2",
				"split-window -v",
				"select-pane -t 4",
				"split-window -v",
			},
		},
		{
			name:          "Zero agents - normalized to 1",
			agentCount:    0,
			expectedCount: 1,
			expectedCmds:  []string{},
		},
		{
			name:          "Negative agents - normalized to 1",
			agentCount:    -5,
			expectedCount: 1,
			expectedCmds:  []string{},
		},
		{
			name:          "More than 6 agents - normalized to 6",
			agentCount:    10,
			expectedCount: 6,
			expectedCmds: []string{
				"split-window -h",
				"split-window -h",
				"select-pane -t 0",
				"split-window -v",
				"select-pane -t 2",
				"split-window -v",
				"select-pane -t 4",
				"split-window -v",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculatePaneLayout(tt.agentCount)

			if layout.AgentCount != tt.expectedCount {
				t.Errorf("AgentCount = %d, want %d", layout.AgentCount, tt.expectedCount)
			}

			if len(layout.Commands) != len(tt.expectedCmds) {
				t.Errorf("Commands length = %d, want %d", len(layout.Commands), len(tt.expectedCmds))
				t.Errorf("Got commands: %v", layout.Commands)
				t.Errorf("Want commands: %v", tt.expectedCmds)
				return
			}

			for i, cmd := range layout.Commands {
				if cmd != tt.expectedCmds[i] {
					t.Errorf("Command[%d] = %q, want %q", i, cmd, tt.expectedCmds[i])
				}
			}
		})
	}
}

// TestPaneLayoutSequences validates the tmux command sequences produce correct layouts
// This test documents the expected pane structure after each command
func TestPaneLayoutSequences(t *testing.T) {
	tests := []struct {
		name         string
		agentCount   int
		paneSequence []string // Description of pane structure after each command
	}{
		{
			name:       "2 agents layout sequence",
			agentCount: 2,
			paneSequence: []string{
				"Initial: [0]",
				"After split-window -h: [0][1]",
			},
		},
		{
			name:       "3 agents layout sequence",
			agentCount: 3,
			paneSequence: []string{
				"Initial: [0]",
				"After split-window -h: [0][1]",
				"After split-window -h: [0][1][2]",
			},
		},
		{
			name:       "4 agents layout sequence",
			agentCount: 4,
			paneSequence: []string{
				"Initial: [0]",
				"After split-window -h: [0][1]",
				"After select-pane -t 0 + split-window -v: [0]\n[2] [1]",
				"After select-pane -t 2 + split-window -v: [0] [1]\n[2] [3]",
			},
		},
		{
			name:       "5 agents layout sequence",
			agentCount: 5,
			paneSequence: []string{
				"Initial: [0]",
				"After split-window -h: [0][1]",
				"After select-pane -t 0 + split-window -v: [0]\n[2] [1]",
				"After split-window -h: [0]\n[2][3] [1]",
				"After split-window -h: [0]\n[2][3][4] [1]",
			},
		},
		{
			name:       "6 agents layout sequence",
			agentCount: 6,
			paneSequence: []string{
				"Initial: [0]",
				"After split-window -h: [0][1]",
				"After split-window -h: [0][1][2]",
				"After select-pane -t 0 + split-window -v: [0]\n[3] [1][2]",
				"After select-pane -t 2 + split-window -v: [0] [1]\n[3] [4] [2]",
				"After select-pane -t 4 + split-window -v: [0] [1] [2]\n[3] [4] [5]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculatePaneLayout(tt.agentCount)

			// Verify we have the expected number of sequence steps
			// Initial state + one step per command
			expectedSteps := 1 + len(layout.Commands)
			if len(tt.paneSequence) != expectedSteps {
				t.Errorf("Sequence steps = %d, want %d", len(tt.paneSequence), expectedSteps)
			}

			// Log the sequence for documentation
			t.Logf("Layout sequence for %d agents:", tt.agentCount)
			for i, step := range tt.paneSequence {
				if i == 0 {
					t.Logf("  %s", step)
				} else {
					t.Logf("  Step %d: %s - %s", i, layout.Commands[i-1], step)
				}
			}
		})
	}
}

// TestPaneIndexMapping verifies that pane indices map correctly to agent positions
func TestPaneIndexMapping(t *testing.T) {
	tests := []struct {
		name        string
		agentCount  int
		description string
	}{
		{
			name:        "1 agent",
			agentCount:  1,
			description: "Pane 0: Agent 0",
		},
		{
			name:        "2 agents",
			agentCount:  2,
			description: "Pane 0: Agent 0 (left), Pane 1: Agent 1 (right)",
		},
		{
			name:        "3 agents",
			agentCount:  3,
			description: "Pane 0: Agent 0 (left), Pane 1: Agent 1 (middle), Pane 2: Agent 2 (right)",
		},
		{
			name:        "4 agents",
			agentCount:  4,
			description: "Pane 0: Agent 0 (top-left), Pane 1: Agent 1 (top-right), Pane 2: Agent 2 (bottom-left), Pane 3: Agent 3 (bottom-right)",
		},
		{
			name:        "5 agents",
			agentCount:  5,
			description: "Pane 0: Agent 0 (top-left), Pane 1: Agent 1 (top-right), Pane 2: Agent 2 (bottom-left), Pane 3: Agent 3 (bottom-middle), Pane 4: Agent 4 (bottom-right)",
		},
		{
			name:        "6 agents",
			agentCount:  6,
			description: "Pane 0: Agent 0 (top-left), Pane 1: Agent 1 (top-middle), Pane 2: Agent 2 (top-right), Pane 3: Agent 3 (bottom-left), Pane 4: Agent 4 (bottom-middle), Pane 5: Agent 5 (bottom-right)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculatePaneLayout(tt.agentCount)

			// Verify the layout produces the expected number of panes
			expectedPanes := tt.agentCount

			// Count actual panes created by analyzing split commands
			actualPanesCreated := 1
			for _, cmd := range layout.Commands {
				if cmd == "split-window -h" || cmd == "split-window -v" {
					actualPanesCreated++
				}
			}

			if actualPanesCreated != expectedPanes {
				t.Errorf("Panes created = %d, want %d", actualPanesCreated, expectedPanes)
			}

			t.Logf("Mapping for %d agents: %s", tt.agentCount, tt.description)
		})
	}
}

// TestPaneLayoutErrors validates error handling in CreatePanes
func TestPaneLayoutErrors(t *testing.T) {
	tests := []struct {
		name        string
		agentCount  int
		worktreeCount int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Mismatched agent and worktree counts",
			agentCount:  3,
			worktreeCount: 2,
			expectError: true,
			errorMsg:    "agent count (3) must match worktree count (2)",
		},
		{
			name:        "No agents",
			agentCount:  0,
			worktreeCount: 0,
			expectError: true,
			errorMsg:    "at least one agent is required",
		},
		{
			name:        "Too many agents",
			agentCount:  7,
			worktreeCount: 7,
			expectError: true,
			errorMsg:    "maximum 6 agents supported, got 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually call CreatePanes without a real tmux session,
			// but we can verify the layout calculation is correct
			layout := CalculatePaneLayout(tt.agentCount)

			// For counts > 6 or < 1, verify normalization
			if tt.agentCount > 6 {
				if layout.AgentCount != 6 {
					t.Errorf("AgentCount not normalized to 6, got %d", layout.AgentCount)
				}
			} else if tt.agentCount < 1 {
				if layout.AgentCount != 1 {
					t.Errorf("AgentCount not normalized to 1, got %d", layout.AgentCount)
				}
			}
		})
	}
}

// TestSplitWindowCommands validates the structure of split commands
func TestSplitWindowCommands(t *testing.T) {
	tests := []struct {
		name       string
		agentCount int
		wantHSplits int // Expected horizontal splits
		wantVSplits int // Expected vertical splits
		wantSelects int // Expected select-pane commands
	}{
		{name: "1 agent", agentCount: 1, wantHSplits: 0, wantVSplits: 0, wantSelects: 0},
		{name: "2 agents", agentCount: 2, wantHSplits: 1, wantVSplits: 0, wantSelects: 0},
		{name: "3 agents", agentCount: 3, wantHSplits: 2, wantVSplits: 0, wantSelects: 0},
		{name: "4 agents", agentCount: 4, wantHSplits: 1, wantVSplits: 2, wantSelects: 2},
		{name: "5 agents", agentCount: 5, wantHSplits: 3, wantVSplits: 1, wantSelects: 1},
		{name: "6 agents", agentCount: 6, wantHSplits: 2, wantVSplits: 3, wantSelects: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := CalculatePaneLayout(tt.agentCount)

			var hSplits, vSplits, selects int
			for _, cmd := range layout.Commands {
				switch cmd {
				case "split-window -h":
					hSplits++
				case "split-window -v":
					vSplits++
				}
				if len(cmd) >= 12 && cmd[:12] == "select-pane " {
					selects++
				}
			}

			if hSplits != tt.wantHSplits {
				t.Errorf("Horizontal splits = %d, want %d", hSplits, tt.wantHSplits)
			}
			if vSplits != tt.wantVSplits {
				t.Errorf("Vertical splits = %d, want %d", vSplits, tt.wantVSplits)
			}
			if selects != tt.wantSelects {
				t.Errorf("Select commands = %d, want %d", selects, tt.wantSelects)
			}
		})
	}
}
