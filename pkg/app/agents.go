// Package app provides the core application logic for agate
package app

import "strings"

// AgentConfig defines the configuration for different AI agents
type AgentConfig struct {
	Name           string // Display name
	BorderColor    string // Hex color value for pane borders
	ExecutableName string // What to match against in subprocess names
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

// CN agent configuration with the specific color
var CNAgent = AgentConfig{
	Name:           "cn",
	BorderColor:    "#82a2be",
	ExecutableName: "cn",
	CompanyName:    "Continue",
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

// Default configuration for unknown agents
var DefaultAgent = AgentConfig{
	Name:           "default",
	BorderColor:    "#86", // Default cyan color
	ExecutableName: "default",
	CompanyName:    "Default",
}

// GetAgentConfig returns the appropriate agent configuration based on the subprocess name
func GetAgentConfig(subprocess string) AgentConfig {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(subprocess)

	// List of all available agents
	agents := []AgentConfig{
		ClaudeAgent,
		AmpAgent,
		GeminiAgent,
		CodexAgent,
		CNAgent,
		OpenCodeAgent,
		CursorAgent,
		GithubCopilotAgent,
	}

	// Check if the subprocess contains any known agent executable names
	for _, agent := range agents {
		if strings.Contains(lower, strings.ToLower(agent.ExecutableName)) {
			return agent
		}
	}

	// Return default if no match found
	return DefaultAgent
}
