// Package app provides the core application logic for agate
package app

import (
	"os/exec"
	"strings"
)

// AgentConfig defines the configuration for different AI agents
type AgentConfig struct {
	Name           string // Display name
	BorderColor    string // Hex color value for pane borders
	ExecutableName string // What to match against in agent names
	CompanyName    string // Company name to display in UI
}

// Claude agent configuration with the specific color
var ClaudeAgent = AgentConfig{
	Name:           "claude",
	BorderColor:    "#da7756",
	ExecutableName: "claude",
	CompanyName:    "Claude Code",
}

// Amp agent configuration with the specific color
var AmpAgent = AgentConfig{
	Name:           "amp",
	BorderColor:    "#b6bf69",
	ExecutableName: "amp",
	CompanyName:    "Amp",
}

// Gemini agent configuration with the specific color
var GeminiAgent = AgentConfig{
	Name:           "gemini",
	BorderColor:    "#cda9fc",
	ExecutableName: "gemini",
	CompanyName:    "Gemini",
}

// Codex agent configuration with the specific color
var CodexAgent = AgentConfig{
	Name:           "codex",
	BorderColor:    "#6c908e",
	ExecutableName: "codex",
	CompanyName:    "Codex",
}

// OpenCode agent configuration with the specific color
var OpenCodeAgent = AgentConfig{
	Name:           "opencode",
	BorderColor:    "#ffba88",
	ExecutableName: "opencode",
	CompanyName:    "opencode",
}

// Cursor agent configuration with the specific color
var CursorAgent = AgentConfig{
	Name:           "cursor",
	BorderColor:    "#ffffff",
	ExecutableName: "cursor-agent",
	CompanyName:    "Cursor",
}

// GithubCopilot agent configuration with the specific color
var GithubCopilotAgent = AgentConfig{
	Name:           "copilot",
	BorderColor:    "#81a1be",
	ExecutableName: "copilot",
	CompanyName:    "GitHub Copilot",
}

var ContinueAgent = AgentConfig{
	Name:           "cn",
	BorderColor:    "#3782a6",
	ExecutableName: "cn",
	CompanyName:    "Continue",
}

var ClineAgent = AgentConfig{
	Name:           "cline",
	BorderColor:    "#f3cb76",
	ExecutableName: "cline",
	CompanyName:    "Cline",
}

// Default configuration for unknown agents
var DefaultAgent = AgentConfig{
	Name:           "default",
	BorderColor:    "#86", // Default cyan color
	ExecutableName: "default",
	CompanyName:    "Default",
}

// GetAgentConfig returns the appropriate agent configuration based on the agent name
func GetAgentConfig(agentName string) AgentConfig {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(agentName)

	// List of all available agents
	agents := []AgentConfig{
		ClaudeAgent,
		AmpAgent,
		GeminiAgent,
		CodexAgent,
		ContinueAgent,
		ClineAgent,
		OpenCodeAgent,
		CursorAgent,
		GithubCopilotAgent,
	}

	// Check if the agent name contains any known agent executable names
	for _, agent := range agents {
		if strings.Contains(lower, strings.ToLower(agent.ExecutableName)) {
			return agent
		}
	}

	// Return default if no match found
	return DefaultAgent
}

// GetAllAgents returns a list of all configured agents
func GetAllAgents() []AgentConfig {
	return []AgentConfig{
		ClaudeAgent,
		AmpAgent,
		GeminiAgent,
		CodexAgent,
		ContinueAgent,
		ClineAgent,
		OpenCodeAgent,
		CursorAgent,
		GithubCopilotAgent,
	}
}

// IsValidAgent checks if the given agent name is valid
func IsValidAgent(name string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(name)

	agents := GetAllAgents()
	for _, agent := range agents {
		if strings.ToLower(agent.ExecutableName) == lower {
			return true
		}
	}

	return false
}

// IsInstalled checks if the agent's binary is installed
func (a AgentConfig) IsInstalled() bool {
	_, err := exec.LookPath(a.ExecutableName)
	return err == nil
}

// HeadlessCommand returns the command and arguments to run the agent in headless mode
// with the given prompt. Returns nil if the agent does not support headless mode.
func (a AgentConfig) HeadlessCommand(prompt string) []string {
	switch a.ExecutableName {
	case "claude":
		return []string{"claude", "-p", prompt}
	case "opencode":
		return []string{"opencode", "-p", prompt}
	case "cursor-agent":
		return []string{"cursor-agent", "-p", prompt}
	case "gemini":
		return []string{"gemini", "-p", prompt}
	case "codex":
		return []string{"codex", "exec", prompt}
	case "copilot":
		return []string{"copilot", "-p", prompt}
	case "cn":
		return []string{"cn", "-p", prompt}
	case "cline":
		return []string{"cline", prompt}
	case "amp":
		return []string{"amp", "-x", prompt}
	default:
		return nil
	}
}
