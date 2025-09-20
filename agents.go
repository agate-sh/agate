package main

import "strings"

// AgentConfig defines the configuration for different AI agents
type AgentConfig struct {
	Name        string
	BorderColor string // Hex color value for pane borders
}

// Claude agent configuration with the specific color
var ClaudeAgent = AgentConfig{
	Name:        "claude",
	BorderColor: "#da7756",
}

// Amp agent configuration with the specific color
var AmpAgent = AgentConfig{
	Name:        "amp",
	BorderColor: "#04c160",
}

// Default configuration for unknown agents
var DefaultAgent = AgentConfig{
	Name:        "default",
	BorderColor: "#86", // Default cyan color
}

// GetAgentConfig returns the appropriate agent configuration based on the subprocess name
func GetAgentConfig(subprocess string) AgentConfig {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(subprocess)

	// Check if the subprocess contains known agent names
	if strings.Contains(lower, "claude") {
		return ClaudeAgent
	}
	if strings.Contains(lower, "amp") {
		return AmpAgent
	}

	// Add more agent mappings here as needed
	// Example for future additions:
	// if strings.Contains(lower, "codex") {
	//     return CodexAgent
	// }
	// if strings.Contains(lower, "gpt") {
	//     return GPTAgent
	// }

	// Return default if no match found
	return DefaultAgent
}