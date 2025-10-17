// +build integration

package git

import (
	"agate/pkg/app"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateCommitMessage_RealCLI_Claude tests commit message generation with real Claude CLI
func TestGenerateCommitMessage_RealCLI_Claude(t *testing.T) {
	// Skip if Claude CLI is not installed
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("Claude CLI not installed")
	}

	// Skip if API key is not set
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	testGenerateCommitMessageWithRealCLI(t, app.ClaudeAgent)
}

// TestGenerateCommitMessage_RealCLI_Gemini tests commit message generation with real Gemini CLI
func TestGenerateCommitMessage_RealCLI_Gemini(t *testing.T) {
	// Skip if Gemini CLI is not installed
	if _, err := exec.LookPath("gemini"); err != nil {
		t.Skip("Gemini CLI not installed")
	}

	// Skip if API key is not set
	if os.Getenv("GOOGLE_API_KEY") == "" {
		t.Skip("GOOGLE_API_KEY not set")
	}

	testGenerateCommitMessageWithRealCLI(t, app.GeminiAgent)
}

// TestGenerateCommitMessage_RealCLI_Codex tests commit message generation with real Codex CLI
func TestGenerateCommitMessage_RealCLI_Codex(t *testing.T) {
	// Skip if Codex CLI is not installed
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("Codex CLI not installed")
	}

	// Skip if API key is not set
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	testGenerateCommitMessageWithRealCLI(t, app.CodexAgent)
}

// testGenerateCommitMessageWithRealCLI is a helper that tests commit message generation
// with a real agent CLI by creating a temporary git repo with changes
func testGenerateCommitMessageWithRealCLI(t *testing.T, agent app.AgentConfig) {
	// Create temporary directory for test repo
	tempDir, err := os.MkdirTemp("", "agate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git name: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "example.go")
	content := `package example

func Hello() string {
	return "Hello, World!"
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Stage the file
	cmd = exec.Command("git", "add", "example.go")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	// Generate commit message using real CLI
	executor := &DefaultCommandExecutor{}
	message, err := GenerateCommitMessage(&agent, tempDir, executor, "")

	if err != nil {
		t.Fatalf("failed to generate commit message: %v", err)
	}

	// Verify message is not empty
	if strings.TrimSpace(message) == "" {
		t.Error("generated commit message is empty")
	}

	// Verify message is 50 characters or less
	if len(message) > 50 {
		t.Errorf("generated commit message is too long (%d chars): %q", len(message), message)
	}

	t.Logf("Generated commit message with %s: %q", agent.Name, message)
}
