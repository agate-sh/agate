package app

import (
	"sync"
)

// agentState holds the global agent configuration state
type agentState struct {
	mu     sync.RWMutex
	config AgentConfig
}

// globalAgentState is the singleton instance
var globalAgentState = &agentState{
	config: DefaultAgent, // Start with default agent
}

// SetCurrentAgent sets the current agent configuration globally
func SetCurrentAgent(config AgentConfig) {
	globalAgentState.mu.Lock()
	defer globalAgentState.mu.Unlock()
	globalAgentState.config = config
}

// GetCurrentAgent returns the current agent configuration
func GetCurrentAgent() AgentConfig {
	globalAgentState.mu.RLock()
	defer globalAgentState.mu.RUnlock()
	return globalAgentState.config
}

// GetCurrentAgentColor returns just the border color of the current agent
func GetCurrentAgentColor() string {
	globalAgentState.mu.RLock()
	defer globalAgentState.mu.RUnlock()
	return globalAgentState.config.BorderColor
}

// GetCurrentAgentName returns just the company name of the current agent
func GetCurrentAgentName() string {
	globalAgentState.mu.RLock()
	defer globalAgentState.mu.RUnlock()
	return globalAgentState.config.CompanyName
}
