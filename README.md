# Agate

A terminal multiplexer built for managing CLI agents with an information-dense, intuitive interface.

![Agate Terminal Multiplexer](https://agate.sh/screenshot.png)

## Overview

Agate provides a streamlined way to interact with CLI-based AI agents like Claude, Gemini, and others. It splits your terminal into intelligent panes, giving you a focused workspace for agent interaction while maintaining full terminal capabilities.

Think of it as a specialized terminal environment where you can seamlessly manage conversations with AI agents, monitor their output, and switch between different agents with ease.

## Features

- **Split-pane interface**: 60/40 layout with status/control on the left, agent interaction on the right
- **Full terminal emulation**: Complete ANSI color and escape sequence support
- **Agent-optimized**: Built specifically for CLI agents with smart handling of their output
- **Keyboard-driven**: Fast pane switching and control with intuitive keybindings
- **Fallback mode**: Automatic degradation to raw output if terminal emulation encounters issues

## Installation

### From Source

Requires Go 1.21 or higher:

```bash
git clone https://github.com/yourusername/agate
cd agate
make build
```

Or using Go directly:
```bash
go build
```

### Pre-built Binaries

<!-- TODO: Add pre-built binary installation instructions -->

## Usage

Launch Agate with your preferred CLI agent:

```bash
# Claude
agate claude

# Claude (shorthand)
agate cn

# Gemini
agate gemini

# Codex
agate codex
```

### Controls

- **Tab**: Switch focus between panes
- **q**: Quit (when left pane is focused)
- **Ctrl+D**: Open debug overlay (debug builds only)
- **All standard terminal keys**: Supported in the right pane (arrows, backspace, etc.)

### Debug Mode

When built with debug support (`go build -tags debug`), Agate includes additional development features:

- **Debug Panel**: A 10-line panel at the bottom showing real-time debug logs
- **File Logging**: All debug output is also written to `debug.log` in the current directory
- **Debug Overlay**: Press `Ctrl+D` to open a full-screen scrollable debug log viewer
- **Persistent Logs**: Debug information is preserved when switching between preview and tmux modes

Debug mode is intended for development and troubleshooting. The debug panel and file logging have minimal performance impact.

## Development

### Building

```bash
# Standard build
make build

# Clean build artifacts
make clean
```

### Code Quality Tools

Agate uses standard Go tooling for maintaining code quality:

#### Formatting
```bash
# Format code with gofmt and goimports
make fmt
```

#### Linting
```bash
# Run comprehensive linting with golangci-lint
make lint

# Run linting with automatic fixes
make lint-fix

# Run go vet static analysis
make vet

# Run all checks (format + vet + lint)
make check

# Auto-fix all possible issues (format + lint-fix)
make fix
```

#### Development Tools Setup
```bash
# Install required development tools
make install-tools
```

The project uses `golangci-lint` with 11 enabled linters including:
- **errcheck** - Check for unchecked errors
- **govet** - Go vet tool ✨ (supports auto-fix)
- **staticcheck** - Advanced static analysis ✨ (supports auto-fix)
- **gosec** - Security issues
- **revive** - Fast configurable linter ✨ (supports auto-fix)
- **gocyclo** - Cyclomatic complexity
- **misspell** - Misspelled words ✨ (supports auto-fix)
- **whitespace** - Leading/trailing whitespace ✨ (supports auto-fix)
- **nolintlint** - Nolint directive issues ✨ (supports auto-fix)
- And more...

Configuration is stored in `.golangci.yml` and tuned specifically for CLI applications.

**Auto-fix capabilities**: 6 out of 11 enabled linters support automatic fixes, including whitespace formatting, deprecated API updates, and code style improvements.

### Testing
```bash
# Run tests
make test
```

All available commands can be viewed with:
```bash
make help
```

## Website

Visit [agate.sh](https://agate.sh) for more information and updates.

## Requirements

- Go 1.21+ (for building from source)
- A terminal emulator with 256 color support
- One or more CLI agents installed (claude, gemini, etc.)

## Contributing

<!-- TODO: Add contributing guidelines -->

## License

[License information to be added]

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [creack/pty](https://github.com/creack/pty) - PTY management
- [vt10x](https://github.com/hinshun/vt10x) - Terminal emulation