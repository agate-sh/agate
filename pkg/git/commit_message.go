package git

import (
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

	return stdout.String(), nil
}

// HeadlessCommandRunner defines an interface for getting headless commands
type HeadlessCommandRunner interface {
	HeadlessCommand(prompt string) []string
}

// GenerateCommitMessage generates a commit message using the specified agent
// Returns the generated message and any error encountered
func GenerateCommitMessage(agent HeadlessCommandRunner, workingDir string, executor CommandExecutor) (string, error) {
	// Check if agent supports headless mode
	prompt := "Generate a concise commit message based on the current git diff. The message MUST be 50 characters or less."
	cmdArgs := agent.HeadlessCommand(prompt)
	if cmdArgs == nil {
		return "", fmt.Errorf("agent does not support headless mode")
	}

	// Get the git diff
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	diff, err := executor.ExecuteCommand(ctx, "git", []string{"diff", "--cached"}, workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	// Check if there are any staged changes
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("no staged changes to commit")
	}

	// Execute the agent's headless command
	// Create a new context for the agent command
	agentCtx, agentCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer agentCancel()

	output, err := executor.ExecuteCommand(agentCtx, cmdArgs[0], cmdArgs[1:], workingDir)
	if err != nil {
		return "", fmt.Errorf("agent command failed: %w", err)
	}

	// Parse and clean the output
	message := strings.TrimSpace(output)
	if message == "" {
		return "", fmt.Errorf("agent returned empty commit message")
	}

	// Limit to 50 characters
	if len(message) > 50 {
		message = message[:50]
	}

	return message, nil
}
