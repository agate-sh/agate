# Agate Technical Overview

## Purpose and Intent

Agate is a specialized terminal multiplexer designed to optimize the experience of working with CLI-based AI agents. The project addresses a specific gap in the ecosystem: while tools like tmux and screen excel at general terminal multiplexing, and applications like lazygit demonstrate the power of information-dense TUIs, there hasn't been a purpose-built solution for managing AI agent interactions that combines both approaches effectively.

The core insight driving Agate is that AI agents benefit from a hybrid interface that provides both structured UI elements for status and control, while maintaining full terminal emulation for the agent's actual output and interaction.

## Architectural Challenges

The architecture of Agate addresses three fundamental challenges in modern terminal application development:

### 1. Custom UI Rendering (Bubble Tea + Lipgloss)
Managing forms, status displays, and visual feedback requires precise control over terminal rendering. This includes handling layouts, colors, borders, and responsive design that adapts to terminal resizing. We use **Bubble Tea** as our TUI framework for component management and event handling, paired with **Lipgloss** for styling and layout composition.

### 2. Full Terminal Emulation (VT10x)
AI agents output complex ANSI escape sequences for colors, cursor positioning, and screen manipulation. Proper terminal emulation must handle these sequences correctly to preserve the agent's intended presentation, especially for agents like Claude Code that use sophisticated terminal features. We use **VT10x** to maintain a complete terminal state buffer and interpret ANSI/VT100 sequences.

### 3. Process Multiplexing (creack/pty)
Managing multiple terminal sessions requires careful orchestration of pseudo-terminals (PTYs), process lifecycle management, and input/output routing between the UI and the underlying processes. We use **creack/pty** to create and manage PTY pairs, handle terminal sizing, and facilitate bidirectional communication with subprocesses.

## Technical Architecture

### Data Flow
1. **User Input** → Bubble Tea captures keyboard events
2. **Event Router** → Routes to UI or terminal based on focus
3. **PTY Write** → Sends to subprocess via PTY master
4. **Terminal Emulator** → VT10x processes ANSI from subprocess
5. **Render** → Terminal state converted to styled text
6. **Update** → Bubble Tea refreshes display

### Design Patterns
- **60/40 Split**: Left pane (status/control), right pane (terminal)
- **Fallback Mode**: Degrades to raw output if terminal emulation fails
- **Tab Focus**: Quick switching between panes

## Key Implementation Details

### Concurrency
- Main thread: Bubble Tea UI loop
- Goroutine: PTY I/O handling
- Communication: Channel-based message passing

### Performance
- 1KB PTY read buffer
- 50ms output polling
- 100KB max output buffer
- Cell-based terminal state (VT10x maintains 2D grid)

## Language Context

Go was chosen for its concurrency primitives and mature TUI ecosystem. Similar patterns exist across languages:
- **Rust**: `crossterm`/`tui-rs` + `portable-pty`
- **JavaScript**: `blessed`/`ink` + `node-pty`
- **Python**: `rich`/`textual` + `pexpect`

## Future Considerations

Architecture supports future extensions:
- Multiple agent sessions
- Session recording/playback
- Custom pane plugins
- Remote agent support