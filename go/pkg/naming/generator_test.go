package naming

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name     string
		input    string
		wantContains []string
	}{
		{
			name:  "basic sanitization",
			input: "agate_main_claude",
			wantContains: []string{"agate_", "agate_main_claude"},
		},
		{
			name:  "special characters",
			input: "repo/name with spaces",
			wantContains: []string{"agate_", "repo_name_with_spaces"},
		},
		{
			name:  "empty string",
			input: "",
			wantContains: []string{"agate_", "default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.Sanitize(tt.input)

			// Check that result contains expected substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Sanitize(%q) = %q, want to contain %q", tt.input, got, want)
				}
			}

			// Check that result has the expected pattern: agate_*_<8-char-hex>
			if !strings.HasPrefix(got, "agate_") {
				t.Errorf("Sanitize(%q) = %q, want to start with 'agate_'", tt.input, got)
			}

			parts := strings.Split(got, "_")
			if len(parts) < 3 {
				t.Errorf("Sanitize(%q) = %q, want at least 3 parts separated by '_'", tt.input, got)
			}

			// Last part should be 8-char hex hash
			lastPart := parts[len(parts)-1]
			if len(lastPart) != 8 {
				t.Errorf("Sanitize(%q) = %q, want hash suffix of length 8, got %d", tt.input, got, len(lastPart))
			}
		})
	}
}

func TestIsAlreadySanitized(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "sanitized name",
			input: "agate_foo_bar_a1b2c3d4",
			want:  true,
		},
		{
			name:  "raw name",
			input: "foo_bar",
			want:  false,
		},
		{
			name:  "name with agate prefix but no hash",
			input: "agate_foo_bar",
			want:  false,
		},
		{
			name:  "name with hash but no agate prefix",
			input: "foo_bar_a1b2c3d4",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.IsAlreadySanitized(tt.input)
			if got != tt.want {
				t.Errorf("IsAlreadySanitized(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeName_Idempotency(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "raw name",
			input: "foo_bar_baz",
		},
		{
			name:  "already sanitized",
			input: "agate_foo_bar_a1b2c3d4",
		},
		{
			name:  "complex name",
			input: "my-repo/feature-branch_agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First normalization
			once := gen.NormalizeName(tt.input)

			// Second normalization should be identical (idempotent)
			twice := gen.NormalizeName(once)

			if once != twice {
				t.Errorf("NormalizeName is not idempotent: NormalizeName(%q) = %q, but NormalizeName(%q) = %q",
					tt.input, once, once, twice)
			}

			// Verify the result is sanitized
			if !gen.IsAlreadySanitized(twice) {
				t.Errorf("NormalizeName(%q) = %q is not sanitized", tt.input, twice)
			}
		})
	}
}

func TestSanitize_NotIdempotent(t *testing.T) {
	gen := NewGenerator()

	// Verify that Sanitize is NOT idempotent (by design)
	input := "foo_bar"
	once := gen.Sanitize(input)
	twice := gen.Sanitize(once)

	if once == twice {
		t.Errorf("Sanitize should NOT be idempotent, but Sanitize(Sanitize(%q)) returned same result", input)
	}

	// The second sanitization should add another "agate_" prefix
	if !strings.HasPrefix(twice, "agate_agate_") {
		t.Errorf("Sanitize(Sanitize(%q)) = %q, want to start with 'agate_agate_'", input, twice)
	}
}

func TestGenerateFromWorktree(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name      string
		repoName  string
		branch    string
		agentName string
		wantContains []string
	}{
		{
			name:      "standard worktree",
			repoName:  "agate",
			branch:    "main",
			agentName: "claude",
			wantContains: []string{"agate_", "agate", "main", "claude"},
		},
		{
			name:      "feature branch",
			repoName:  "myproject",
			branch:    "feature/new-ui",
			agentName: "gpt",
			wantContains: []string{"agate_", "myproject", "feature", "new", "ui", "gpt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.GenerateFromWorktree(tt.repoName, tt.branch, tt.agentName)

			// Check that result contains expected substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateFromWorktree(%q, %q, %q) = %q, want to contain %q",
						tt.repoName, tt.branch, tt.agentName, got, want)
				}
			}

			// Verify the result is sanitized
			if !gen.IsAlreadySanitized(got) {
				t.Errorf("GenerateFromWorktree(%q, %q, %q) = %q is not sanitized",
					tt.repoName, tt.branch, tt.agentName, got)
			}
		})
	}
}

func TestGenerateShellSessionName(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name              string
		agentSessionName  string
		wantContains      []string
	}{
		{
			name:             "standard agent session",
			agentSessionName: "agate_myrepo_main_claude_a1b2c3d4",
			wantContains:     []string{"agate_", "shell", "myrepo", "main", "claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.GenerateShellSessionName(tt.agentSessionName)

			// Check that result contains expected substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateShellSessionName(%q) = %q, want to contain %q",
						tt.agentSessionName, got, want)
				}
			}

			// Verify the result is sanitized
			if !gen.IsAlreadySanitized(got) {
				t.Errorf("GenerateShellSessionName(%q) = %q is not sanitized",
					tt.agentSessionName, got)
			}

			// Verify it's different from the agent session name
			if got == tt.agentSessionName {
				t.Errorf("GenerateShellSessionName(%q) should generate a different name",
					tt.agentSessionName)
			}
		})
	}
}
