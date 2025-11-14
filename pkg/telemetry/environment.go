package telemetry

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// GetEnvironmentProperties collects user environment properties.
func GetEnvironmentProperties(version string) map[string]interface{} {
	props := map[string]interface{}{
		"os":           runtime.GOOS,
		"architecture": runtime.GOARCH,
		"version":      version,
	}

	// OS version
	if osVersion := getOSVersion(); osVersion != "" {
		props["os_version"] = osVersion
	}

	// Terminal program
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		props["terminal_program"] = termProgram
	}

	// Terminal type
	if term := os.Getenv("TERM"); term != "" {
		props["terminal_type"] = term
	}

	// Shell type
	if shell := getShellType(); shell != "" {
		props["shell"] = shell
	}

	return props
}

// getOSVersion returns the operating system version.
func getOSVersion() string {
	switch runtime.GOOS {
	case "darwin":
		// Get macOS version
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}

	case "linux":
		// Try to get Linux distribution info
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					version := strings.TrimPrefix(line, "PRETTY_NAME=")
					version = strings.Trim(version, "\"")
					return version
				}
			}
		}

		// Fallback to kernel version
		if out, err := exec.Command("uname", "-r").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}

	case "windows":
		// Get Windows version
		if out, err := exec.Command("cmd", "/c", "ver").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	return ""
}

// getShellType returns the user's shell type.
func getShellType() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ""
	}

	// Extract shell name from path
	parts := strings.Split(shell, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return shell
}
