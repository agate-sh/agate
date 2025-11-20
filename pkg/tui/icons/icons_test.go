package icons

import (
	"testing"
)

func TestGetGitStatusIcon_Modified(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"unstaged modification", " M"},
		{"staged modification", "M "},
		{"both staged and unstaged", "MM"},
		{"added and modified", "AM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitModified.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_Added(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"staged addition", "A "},
		{"added and deleted", "AD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitAdded.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_Deleted(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"unstaged deletion", " D"},
		{"staged deletion", "D "},
		{"both deleted", "DD"},
		{"deleted and modified", "DM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitDeleted.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_Untracked(t *testing.T) {
	icon := GetGitStatusIcon("??")
	expected := GitUntracked.Get()
	if icon != expected {
		t.Errorf("GetGitStatusIcon(%q) = %q, want %q", "??", icon, expected)
	}
}

func TestGetGitStatusIcon_Renamed(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"renamed", "R "},
		{"renamed and modified", "RM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitRenamed.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_TypeChanged(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"staged type change", "T "},
		{"unstaged type change", " T"},
		{"both type changed", "TT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitTypeChanged.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_Conflicted(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"both modified conflict", "UU"},
		{"both added conflict", "AA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetGitStatusIcon(tt.status)
			expected := GitConflicted.Get()
			if icon != expected {
				t.Errorf("GetGitStatusIcon(%q) = %q, want %q", tt.status, icon, expected)
			}
		})
	}
}

func TestGetGitStatusIcon_DefaultFallback(t *testing.T) {
	// Unknown status should return modified icon as fallback
	icon := GetGitStatusIcon("XX")
	expected := GitModified.Get()
	if icon != expected {
		t.Errorf("GetGitStatusIcon(%q) = %q, want %q (default fallback)", "XX", icon, expected)
	}
}
