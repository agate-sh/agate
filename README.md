# Agate

Run Claude Code, Codex, Gemini, and other agents against the same prompt to compare outputs, gather usage metrics, and parallelize work.

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

Run a set of agents against the same prompt:

```bash
agate --agents claude,codex,gemiini "Create or update AGENTS.md"
```

Launch the full Agate TUI:

```bash
agate
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
