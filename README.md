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
- **All standard terminal keys**: Supported in the right pane (arrows, backspace, etc.)

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