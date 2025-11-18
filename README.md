# Agate - _Run and evaluate agents in parallel_

Run agents like Claude Code, Codex, and Gemini in parallel worktrees, or compare them on the same prompt

`agate --agents claude,codex,gemini "Create or update AGENTS.md"`

![Agate Terminal Multiplexer](assets/screenshot.png)

## Features

- **Run multiple agents simultaneously** - Claude Code, Codex, Gemini, and more
- **Isolated environments** - Each agent runs in its own git worktree with tmux
- **Native interfaces** - Use each agent's native terminal interface
- **Compare outputs** - See how different agents approach the same task
- **Usage metrics** - Track performance and behavior across agents

## Installation

```bash
curl -fsSL https://agate.sh/install | bash -s -- --agents claude,codex
```

Or build from source (requires Go 1.21+):

```bash
git clone https://github.com/agate-sh/agate
cd agate
make build
```

## Usage

```
Examples:
  agate                                                           # Launch TUI
  agate --agents claude,codex,gemini                              # Launch TUI with specific agents
  agate --agents claude,codex,gemini "Create or update AGENTS.md" # Launch with a prompt and auto-attach

Usage:
  agate [flags] [prompt]

Flags:
  -a, --agents string   Comma-separated list of agents (e.g., claude,codex,gemini)
  -h, --help            help for agate
  -v, --version         Show version information
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and code quality guidelines.

## Telemetry

Agate collects anonymous usage data (sessions created, agents used, commits made) to improve the product. **We never collect your code, prompts, or personal information.**

To opt out:

```bash
export AGATE_DISABLE_TELEMETRY=1
```

## Learn More

Visit [agate.sh](https://agate.sh) for more information and updates.

## License

MIT
