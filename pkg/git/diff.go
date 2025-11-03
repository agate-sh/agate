package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetFileDiffRendered returns the diff for a specific file rendered with delta
func GetFileDiffRendered(repoPath, filePath string, width int) (string, error) {
	// Get unified diff for the file
	cmd := exec.Command("git", "diff", "HEAD", "--", filePath)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Try getting diff for staged changes if HEAD diff fails
		cmd = exec.Command("git", "diff", "--cached", "--", filePath)
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff: %w", err)
		}
	}

	// If output is empty, file might be untracked - show the whole file as additions
	if len(output) == 0 {
		cmd = exec.Command("git", "show", fmt.Sprintf(":%s", filePath))
		cmd.Dir = repoPath
		content, err := cmd.Output()
		if err != nil {
			// File is untracked, read it directly
			cmd = exec.Command("cat", filePath)
			cmd.Dir = repoPath
			content, err = cmd.Output()
			if err != nil {
				return "", fmt.Errorf("failed to read untracked file: %w", err)
			}
		}

		// Create a synthetic diff showing the entire file as new
		lines := strings.Split(string(content), "\n")
		var diffText strings.Builder
		diffText.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
		diffText.WriteString("new file mode 100644\n")
		diffText.WriteString("--- /dev/null\n")
		diffText.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
		diffText.WriteString("@@ -0,0 +1," + fmt.Sprintf("%d", len(lines)) + " @@\n")
		for _, line := range lines {
			diffText.WriteString("+" + line + "\n")
		}
		output = []byte(diffText.String())
	}

	// Pipe diff through delta for rendering
	// Use user's delta config but override decorations to fit in our layout
	deltaCmd := exec.Command("delta",
		"--width", fmt.Sprintf("%d", width),
		"--paging", "never",
		"--file-decoration-style", "none",
		"--hunk-header-decoration-style", "none",
	)
	deltaCmd.Stdin = bytes.NewReader(output)

	rendered, err := deltaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to render diff with delta: %w", err)
	}

	return string(rendered), nil
}
