package git

import (
	"agate/internal/debug"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandExecutor is an interface for executing commands
// This allows us to mock command execution in tests
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, name string, args []string, workingDir string) (string, error)
}

// DefaultCommandExecutor implements CommandExecutor using os/exec
type DefaultCommandExecutor struct{}

// ExecuteCommand executes a command and returns its output
func (e *DefaultCommandExecutor) ExecuteCommand(ctx context.Context, name string, args []string, workingDir string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("command failed: %w (stderr: %s)", err, stderr.String())
	}

	output := stdout.String()
	return output, nil
}

// HeadlessCommandRunner defines an interface for getting headless commands
type HeadlessCommandRunner interface {
	HeadlessCommand(prompt string) []string
}

// GenerateCommitMessage generates a commit message using the specified agent
// Returns the generated message and any error encountered
func GenerateCommitMessage(agent HeadlessCommandRunner, workingDir string, executor CommandExecutor) (string, error) {
	// Check if agent supports headless mode first (before doing any work)
	if agent.HeadlessCommand("") == nil {
		return "", fmt.Errorf("agent does not support commit message generation")
	}

	// Get the git diff summary (--stat for overview + limited diff)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use --stat to get file summary + limit diff context
	diff, err := executor.ExecuteCommand(ctx, "git", []string{"diff", "--cached", "--stat", "--compact-summary"}, workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	// Check if there are any staged changes
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("no staged changes to commit")
	}

	// Build prompt with diff embedded
	prompt := fmt.Sprintf("Generate a concise commit message for this diff. Use conventional commit syntax (type: description). Output ONLY the commit message, no preamble, no explanation, no extra text:\n\n%s", diff)

	// Get headless command with prompt
	cmdArgs := agent.HeadlessCommand(prompt)

	// Execute the agent's headless command (no timeout)
	agentCtx := context.Background()

	output, err := executor.ExecuteCommand(agentCtx, cmdArgs[0], cmdArgs[1:], workingDir)
	if err != nil {
		return "", fmt.Errorf("agent command failed: %w", err)
	}

	// Log the full response from the agent
	debug.DebugLog("Agent commit message response: %q", output)

	// Parse and clean the output
	message := strings.TrimSpace(output)
	if message == "" {
		return "", fmt.Errorf("agent returned empty commit message")
	}

	return message, nil
}
