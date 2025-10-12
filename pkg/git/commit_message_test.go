package git

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockAgent implements HeadlessCommandRunner for testing
type mockAgent struct {
	executableName string
}

func (m *mockAgent) HeadlessCommand(prompt string) []string {
	if m.executableName == "" {
		return nil // Agent doesn't support headless mode
	}
	return []string{m.executableName, "-p", prompt}
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
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	diffOutput := " file.go | 5 +++--\n 1 file changed, 3 insertions(+), 2 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	// The agent command will be built with the full prompt
	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "feat: add new feature", nil)

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

	// Check git diff was called with HEAD (not --cached)
	if executor.commands[0].name != "git" {
		t.Errorf("expected first command to be 'git', got %q", executor.commands[0].name)
	}
	expectedArgs := []string{"diff", "HEAD", "--stat", "--compact-summary"}
	if !sliceEqual(executor.commands[0].args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, executor.commands[0].args)
	}

	// Check agent command was called
	if executor.commands[1].name != "claude" {
		t.Errorf("expected second command to be 'claude', got %q", executor.commands[1].name)
	}
}

func TestGenerateCommitMessage_NoHeadlessSupport(t *testing.T) {
	agent := &mockAgent{
		executableName: "", // Agent doesn't support headless mode
	}

	executor := newMockCommandExecutor()

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error for agent without headless support")
	}

	if !strings.Contains(err.Error(), "does not support commit message generation") {
		t.Errorf("expected 'does not support commit message generation' error, got: %v", err)
	}

	// No commands should have been executed
	if len(executor.commands) != 0 {
		t.Errorf("expected 0 commands executed, got %d", len(executor.commands))
	}
}

func TestGenerateCommitMessage_NoChanges(t *testing.T) {
	agent := &mockAgent{
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff HEAD --stat --compact-summary", "", nil) // Empty diff

	_, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err == nil {
		t.Fatal("expected error for no changes")
	}

	if !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("expected 'no changes to commit' error, got: %v", err)
	}

	// Only git diff should have been called
	if len(executor.commands) != 1 {
		t.Errorf("expected 1 command executed, got %d", len(executor.commands))
	}
}

func TestGenerateCommitMessage_UnstagedChanges(t *testing.T) {
	// This test verifies that unstaged changes work correctly
	agent := &mockAgent{
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	// Simulate unstaged changes (git diff HEAD shows them)
	diffOutput := " CLAUDE.md                              | 12 ++++++++++++\n pkg/gui/overlays/commit.go             |  6 ++++--\n pkg/gui/overlays/session_delete_conf.go|  8 +++-----\n 3 files changed, 19 insertions(+), 7 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "docs: add debugging section to CLAUDE.md", nil)

	message, err := GenerateCommitMessage(agent, "/test/dir", executor)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if message != "docs: add debugging section to CLAUDE.md" {
		t.Errorf("expected commit message, got: %q", message)
	}

	// Verify git diff HEAD was called (not git diff --cached)
	if len(executor.commands) != 2 {
		t.Fatalf("expected 2 commands executed, got %d", len(executor.commands))
	}
	expectedArgs := []string{"diff", "HEAD", "--stat", "--compact-summary"}
	if !sliceEqual(executor.commands[0].args, expectedArgs) {
		t.Errorf("expected git args %v, got %v", expectedArgs, executor.commands[0].args)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGenerateCommitMessage_GitDiffFails(t *testing.T) {
	agent := &mockAgent{
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	executor.setResponse("git diff HEAD --stat --compact-summary", "", errors.New("git command failed"))

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
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	diffOutput := " file.go | 5 +++--\n 1 file changed, 3 insertions(+), 2 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "", errors.New("agent timeout"))

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
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	diffOutput := " file.go | 5 +++--\n 1 file changed, 3 insertions(+), 2 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "   \n\t  ", nil) // Whitespace only

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
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	diffOutput := " file.go | 5 +++--\n 1 file changed, 3 insertions(+), 2 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "\n\n  fix: bug  \n\t", nil)

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
		executableName: "claude",
	}

	executor := newMockCommandExecutor()
	diffOutput := " file.go | 5 +++--\n 1 file changed, 3 insertions(+), 2 deletions(-)"
	executor.setResponse("git diff HEAD --stat --compact-summary", diffOutput, nil)

	prompt := "Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n" + diffOutput
	executor.setResponse("claude -p "+prompt, "feat: new feature", nil)

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
