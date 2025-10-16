package tmux

import (
	"os"
	"os/exec"
	"strings"
)

// FindTmuxPath locates the tmux binary on the system.
// It first checks PATH, then falls back to common installation locations.
func FindTmuxPath() string {
	// First try to find tmux in PATH
	if path, err := exec.LookPath("tmux"); err == nil {
		return path
	}

	// Fallback to common locations if not in PATH
	paths := []string{
		"/opt/homebrew/bin/tmux", // Apple Silicon Homebrew
		"/usr/local/bin/tmux",     // Intel Homebrew / standard macOS
		"/usr/bin/tmux",           // Linux
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Final fallback
	return "tmux"
}

// IsTmuxInstalled verifies that tmux is actually executable
func IsTmuxInstalled() bool {
	tmuxPath := FindTmuxPath()
	cmd := exec.Command(tmuxPath, "-V")
	return cmd.Run() == nil
}

// GetInstallCommand returns the OS-specific command to install tmux
func GetInstallCommand() string {
	// Detect OS for platform-specific install command
	goos := os.Getenv("GOOS")
	if goos == "" {
		// Fallback to detecting via uname-like approach
		cmd := exec.Command("uname", "-s")
		if output, err := cmd.Output(); err == nil {
			osName := strings.ToLower(strings.TrimSpace(string(output)))
			if strings.Contains(osName, "darwin") {
				goos = "darwin"
			} else if strings.Contains(osName, "linux") {
				goos = "linux"
			}
		}
	}

	switch goos {
	case "darwin":
		return "brew install tmux"
	case "linux":
		// Try to detect Linux package manager
		if _, err := exec.LookPath("apt-get"); err == nil {
			return "sudo apt-get install tmux"
		} else if _, err := exec.LookPath("yum"); err == nil {
			return "sudo yum install tmux"
		} else if _, err := exec.LookPath("pacman"); err == nil {
			return "sudo pacman -S tmux"
		}
		return "install tmux using your package manager"
	default:
		return "install tmux using your package manager"
	}
}
