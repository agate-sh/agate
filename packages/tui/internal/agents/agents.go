package agents

// Config mirrors the AgentConfig shape defined in @agate/shared.
type Config struct {
	Name           string
	BorderColor    string
	ExecutableName string
	CompanyName    string
}

var all = []Config{
	{Name: "claude", BorderColor: "#da7756", ExecutableName: "claude", CompanyName: "Claude Code"},
	{Name: "amp", BorderColor: "#b6bf69", ExecutableName: "amp", CompanyName: "Amp"},
	{Name: "gemini", BorderColor: "#cda9fc", ExecutableName: "gemini", CompanyName: "Gemini"},
	{Name: "codex", BorderColor: "#6c908e", ExecutableName: "codex", CompanyName: "Codex"},
	{Name: "opencode", BorderColor: "#ffba88", ExecutableName: "opencode", CompanyName: "opencode"},
	{Name: "cursor", BorderColor: "#ffffff", ExecutableName: "cursor-agent", CompanyName: "Cursor"},
	{Name: "copilot", BorderColor: "#81a1be", ExecutableName: "copilot", CompanyName: "GitHub Copilot"},
	{Name: "cn", BorderColor: "#3782a6", ExecutableName: "cn", CompanyName: "Continue"},
	{Name: "cline", BorderColor: "#f3cb76", ExecutableName: "cline", CompanyName: "Cline"},
}

// All returns the list of supported agents.
func All() []Config {
	out := make([]Config, len(all))
	copy(out, all)
	return out
}
