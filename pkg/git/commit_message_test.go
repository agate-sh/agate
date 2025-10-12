package git

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockAgent implements HeadlessCommandRunner for testing
type mockAgent struct {
	cmdArgs []string
}

func (m *mockAgent) HeadlessCommand(prompt string) []string {
	return m.cmdArgs
}

// mockCommandExecutor implements CommandExecutor for testing
type mockCommandExecutor struct {
	commands []executedCommand
	responses map[string]commandResponse
}

type executedCommand struct {
	name       string
	args       []string
	workingDir string
}

type commandResponse struct {
	output string
	err    error
}

func newMockCommandExecutor() *mockCommandExecutor {
	return &mockCommandExecutor{
		commands:  make([]executedCommand, 0),
		responses: make(map[string]commandResponse),
	}
}

func (m *mockCommandExecutor) setResponse(cmdKey string, output string, err error) {
	m.responses[cmdKey] = commandResponse{output: output, err: err}
}

func (m *mockCommandExecutor) ExecuteCommand(ctx context.Context, name string, args []string, workingDir string) (string, error) {
	// Record the command execution
	m.commands = append(m.commands, executedCommand{
		name:       name,
		args:       args,
		workingDir: workingDir,
	})

	// Build a key to lookup the response
	cmdKey := name + " " + strings.Join(args, " ")

	if resp, ok := m.responses[cmdKey]; ok {
		return resp.output, resp.err
	}

	// Default response if not found
	return "", nil
}

func TestGenerateCommitMessage_Success(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	executor.setResponse("claude -p test prompt", "feat: add new feature", nil)

	message, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if message != "feat: add new feature" {
		t.Errorf("expected 'feat: add new feature', got: %q", message)
	}

	// Verify the right commands were executed
	if len(executor.commands) != 2 {
		t.Fatalf("expected 2 commands executed, got %d", len(executor.commands))
	}

	// Check git diff was called
	if executor.commands[0].name != "git" {
		t.Errorf("expected first command to be 'git', got %q", executor.commands[0].name)
	}
	if len(executor.commands[0].args) != 2 || executor.commands[0].args[0] != "diff" || executor.commands[0].args[1] != "--cached" {
		t.Errorf("expected 'diff --cached' args, got %v", executor.commands[0].args)
	}

	// Check agent command was called
	if executor.commands[1].name != "claude" {
		t.Errorf("expected second command to be 'claude', got %q", executor.commands[1].name)
	}
}

func TestGenerateCommitMessage_NoHeadlessSupport(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: nil, // Agent doesn't support headless mode
	}

	executor := newMockCommandExecutor()

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error for agent without headless support")
	}

	if !strings.Contains(err.Error(), "does not support headless mode") {
		t.Errorf("expected 'does not support headless mode' error, got: %v", err)
	}

	// No commands should have been executed
	if len(executor.commands) != 0 {
		t.Errorf("expected 0 commands executed, got %d", len(executor.commands))
	}
}

func TestGenerateCommitMessage_NoStagedChanges(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "", nil) // Empty diff

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error for no staged changes")
	}

	if !strings.Contains(err.Error(), "no staged changes") {
		t.Errorf("expected 'no staged changes' error, got: %v", err)
	}

	// Only git diff should have been called
	if len(executor.commands) != 1 {
		t.Errorf("expected 1 command executed, got %d", len(executor.commands))
	}
}

func TestGenerateCommitMessage_GitDiffFails(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "", errors.New("git command failed"))

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error when git diff fails")
	}

	if !strings.Contains(err.Error(), "failed to get git diff") {
		t.Errorf("expected 'failed to get git diff' error, got: %v", err)
	}
}

func TestGenerateCommitMessage_AgentCommandFails(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	executor.setResponse("claude -p test prompt", "", errors.New("agent timeout"))

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error when agent command fails")
	}

	if !strings.Contains(err.Error(), "agent command failed") {
		t.Errorf("expected 'agent command failed' error, got: %v", err)
	}
}

func TestGenerateCommitMessage_EmptyAgentResponse(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	executor.setResponse("claude -p test prompt", "   \n\t  ", nil) // Whitespace only

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error for empty agent response")
	}

	if !strings.Contains(err.Error(), "empty commit message") {
		t.Errorf("expected 'empty commit message' error, got: %v", err)
	}
}

func TestGenerateCommitMessage_TrimsWhitespace(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	executor.setResponse("claude -p test prompt", "\n\n  fix: bug  \n\t", nil)

	message, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if message != "fix: bug" {
		t.Errorf("expected 'fix: bug', got: %q", message)
	}
}

func TestGenerateCommitMessage_WorkingDirPassedThrough(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	executor.setResponse("claude -p test prompt", "feat: new feature", nil)

	workingDir := "/custom/working/dir"
	_, err := GenerateCommitMessage(agent, workingDir, executor)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify working directory was passed to both commands
	for i, cmd := range executor.commands {
		if cmd.workingDir != workingDir {
			t.Errorf("command %d: expected workingDir %q, got %q", i, workingDir, cmd.workingDir)
		}
	}
}

func TestGenerateCommitMessage_TruncatesTo50Chars(t *testing.T) {
	agent := &mockAgent{
		cmdArgs: []string{"claude", "-p", "test prompt"},
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff --cached", "diff --git a/file.go b/file.go\n+added line", nil)
	// Response is longer than 50 characters
	executor.setResponse("claude -p test prompt", "feat: add a very long commit message that exceeds the fifty character limit", nil)

	message, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(message) != 50 {
		t.Errorf("expected message length 50, got %d", len(message))
	}

	expected := "feat: add a very long commit message that exceeds "
	if message != expected {
		t.Errorf("expected %q, got %q", expected, message)
	}
}
