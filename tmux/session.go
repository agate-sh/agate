package tmux

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// TmuxSession represents a managed tmux session
type TmuxSession struct {
	// Session identification
	name          string
	sanitizedName string
	program       string

	// PTY management
	ptyFactory PtyFactory
	ptmx       *os.File

	// Status monitoring
	monitor *StatusMonitor

	// Attachment state
	attachCh chan struct{}
	ctx      context.Context
	cancel   func()
	wg       *sync.WaitGroup

	// Terminal dimensions
	width  int
	height int
}

// NewTmuxSession creates a new tmux session manager
func NewTmuxSession(name, program string) *TmuxSession {
	sanitizedName := sanitizeName(name)
	return &TmuxSession{
		name:          name,
		sanitizedName: sanitizedName,
		program:       program,
		ptyFactory:    NewPtyFactory(),
		monitor:       newStatusMonitor(program),
	}
}

// SetPtyFactory sets a custom PTY factory (useful for testing)
func (t *TmuxSession) SetPtyFactory(factory PtyFactory) {
	t.ptyFactory = factory
}

// sanitizeName creates a valid tmux session name
func sanitizeName(name string) string {
	// Replace spaces and special characters with underscores
	sanitized := strings.ReplaceAll(name, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")

	// Add prefix to ensure uniqueness
	timestamp := time.Now().Unix()
	return fmt.Sprintf("agate_%s_%d", sanitized, timestamp)
}

// Start creates and starts a new tmux session
func (t *TmuxSession) Start(workDir string) error {
	// Check if session already exists
	exists, err := t.sessionExists()
	if err != nil {
		return fmt.Errorf("error checking session existence: %w", err)
	}

	if !exists {
		// Create new tmux session using PTY like Claude Squad
		cmd := exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName, "-c", workDir, t.program)

		ptmx, err := t.ptyFactory.Start(cmd)
		if err != nil {
			// Cleanup any partially created session if any exists.
			if exists, _ := t.sessionExists(); exists {
				cleanupCmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
				if cleanupErr := cleanupCmd.Run(); cleanupErr != nil {
					err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
				}
			}
			return fmt.Errorf("error starting tmux session: %w", err)
		}

		// Poll for session existence with exponential backoff like Claude Squad
		timeout := time.After(2 * time.Second)
		sleepDuration := 5 * time.Millisecond
		for {
			if exists, _ := t.sessionExists(); exists {
				break
			}
			select {
			case <-timeout:
				if cleanupErr := t.Kill(); cleanupErr != nil {
					err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
				}
				return fmt.Errorf("timed out waiting for tmux session %s: %v", t.sanitizedName, err)
			default:
				time.Sleep(sleepDuration)
				// Exponential backoff up to 50ms max
				if sleepDuration < 50*time.Millisecond {
					sleepDuration *= 2
				}
			}
		}
		ptmx.Close()

		// Set history limit to enable scrollback (default is 2000, we'll use 10000 for more history)
		historyCmd := exec.Command("tmux", "set-option", "-t", t.sanitizedName, "history-limit", "10000")
		if err := historyCmd.Run(); err != nil {
			// Log warning but don't fail
		}

		// Enable mouse scrolling for the session
		mouseCmd := exec.Command("tmux", "set-option", "-t", t.sanitizedName, "mouse", "on")
		if err := mouseCmd.Run(); err != nil {
			// Log warning but don't fail
		}
	}

	// Attach to the session in detached mode
	return t.Restore()
}

// Restore sets up monitoring for an existing tmux session without attaching
func (t *TmuxSession) Restore() error {
	// Close existing PTY if any
	if t.ptmx != nil {
		t.ptmx.Close()
		t.ptmx = nil
	}

	// Create a PTY connected to tmux attach-session (like Claude Squad)
	cmd := exec.Command("tmux", "attach-session", "-t", t.sanitizedName)
	ptmx, err := t.ptyFactory.Start(cmd)
	if err != nil {
		return fmt.Errorf("error opening PTY for session %s: %w", t.sanitizedName, err)
	}
	t.ptmx = ptmx

	// Initialize status monitor if needed
	if t.monitor == nil {
		t.monitor = newStatusMonitor(t.program)
	}

	return nil
}

// sessionExists checks if a tmux session exists
func (t *TmuxSession) sessionExists() (bool, error) {
	cmd := exec.Command("tmux", "has-session", "-t", t.sanitizedName)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means session doesn't exist
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// Attach attaches to the tmux session for interactive use
func (t *TmuxSession) Attach() (chan struct{}, error) {
	// Use the existing PTY that's already connected (like Claude Squad)
	if t.ptmx == nil {
		return nil, fmt.Errorf("no PTY available for session %s", t.sanitizedName)
	}

	t.attachCh = make(chan struct{})

	t.wg = &sync.WaitGroup{}
	t.wg.Add(1)
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// The first goroutine should terminate when the ptmx is closed. We use the
	// waitgroup to wait for it to finish.
	// The 2nd one returns when you press escape to Detach. It doesn't need to be
	// in the waitgroup because is the goroutine doing the Detaching; it waits for
	// all the other ones.
	go func() {
		defer t.wg.Done()
		_, _ = io.Copy(os.Stdout, t.ptmx)
		// When io.Copy returns, it means the connection was closed
		// This could be due to normal detach or Ctrl-D
		// Check if the context is done to determine if it was a normal detach
		if t.ctx != nil {
			select {
			case <-t.ctx.Done():
				// Normal detach, do nothing
			default:
				// If context is not done, it was likely an abnormal termination (Ctrl-D)
				// Print warning message
				fmt.Fprintf(os.Stderr, "\n\033[31mError: Session terminated without detaching. Use Ctrl-Q to properly detach from tmux sessions.\033[0m\n")
			}
		}
	}()

	go func() {
		// Close the channel after 50ms
		timeoutCh := make(chan struct{})
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(timeoutCh)
		}()

		// Read input from stdin and check for Ctrl+q
		buf := make([]byte, 32)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				continue
			}

			// Nuke the first bytes of stdin, up to 64, to prevent tmux from reading it.
			// When we attach, there tends to be terminal control sequences like ?[?62c0;95;0c or
			// ]10;rgb:f8f8f8. The control sequences depend on the terminal (warp vs iterm). We should use regex ideally
			// but this works well for now. Log this for debugging.
			//
			// There seems to always be control characters, but I think it's possible for there not to be. The heuristic
			// here can be: if there's characters within 50ms, then assume they are control characters and nuke them.
			select {
			case <-timeoutCh:
			default:
				// For now, skip logging since we don't have a logger setup
				continue
			}

			// Check for Ctrl+q (ASCII 17)
			if nr == 1 && buf[0] == 17 {
				// Detach from the session
				t.Detach()
				return
			}

			// Forward other input to tmux
			_, _ = t.ptmx.Write(buf[:nr])
		}
	}()

	t.monitorWindowSize()
	return t.attachCh, nil
}

// Detach disconnects from the current tmux session.
func (t *TmuxSession) Detach() {
	// Store references to avoid race condition with cleanup
	cancel := t.cancel
	wg := t.wg
	attachCh := t.attachCh

	// Cancel goroutines created by Attach first
	if cancel != nil {
		cancel()
	}

	// Close the attached pty session.
	if t.ptmx != nil {
		err := t.ptmx.Close()
		if err != nil {
			// This is a fatal error. We can't detach if we can't close the PTY. It's better to just panic and have the
			// user re-invoke the program than to ruin their terminal pane.
			msg := fmt.Sprintf("error closing attach pty session: %v", err)
			panic(msg)
		}
	}

	// Wait for goroutines to finish before cleanup
	if wg != nil {
		wg.Wait()
	}

	// Clean up state
	if attachCh != nil {
		close(attachCh)
	}
	t.attachCh = nil
	t.cancel = nil
	t.ctx = nil
	t.wg = nil

	// Attach goroutines should die on EOF due to the ptmx closing. Call
	// t.Restore to set a new t.ptmx.
	if err := t.Restore(); err != nil {
		// This is a fatal error. Our invariant that a started TmuxSession always has a valid ptmx is violated.
		msg := fmt.Sprintf("error restoring pty session: %v", err)
		panic(msg)
	}
}

// monitorWindowSize is implemented in platform-specific files (session_unix.go, etc.)

// updateWindowSize updates the PTY size
func (t *TmuxSession) updateWindowSize(cols, rows int) error {
	t.width = cols
	t.height = rows

	if t.ptmx != nil {
		return pty.Setsize(t.ptmx, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
			X:    0,
			Y:    0,
		})
	} else {
		// In detached mode, resize the tmux session directly
		cmd := exec.Command("tmux", "resize-window", "-t", t.sanitizedName, "-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows))
		return cmd.Run()
	}
}

// SetDetachedSize sets the size for detached mode
func (t *TmuxSession) SetDetachedSize(width, height int) error {
	return t.updateWindowSize(width, height)
}

// CapturePaneContent captures the current content of the tmux pane
func (t *TmuxSession) CapturePaneContent() (string, error) {
	// -e preserves escape sequences (ANSI colors)
	// -J joins wrapped lines
	// -p prints to stdout
	cmd := exec.Command("tmux", "capture-pane", "-p", "-e", "-J", "-t", t.sanitizedName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error capturing pane content: %w", err)
	}
	return string(output), nil
}

// CapturePaneContentWithOptions captures specific lines from the tmux pane
func (t *TmuxSession) CapturePaneContentWithOptions(startLine, endLine int) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-p", "-e", "-J", "-t", t.sanitizedName,
		"-S", fmt.Sprintf("%d", startLine), "-E", fmt.Sprintf("%d", endLine))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error capturing pane content: %w", err)
	}
	return string(output), nil
}

// HasUpdated checks if the tmux pane content has changed
func (t *TmuxSession) HasUpdated() (updated bool, hasPrompt bool) {
	content, err := t.CapturePaneContent()
	if err != nil {
		return false, false
	}

	return t.monitor.HasUpdated(content)
}

// SendKeys sends keystrokes to the tmux session
func (t *TmuxSession) SendKeys(keys string) error {
	// Use tmux send-keys command for detached sessions
	cmd := exec.Command("tmux", "send-keys", "-t", t.sanitizedName, keys)
	return cmd.Run()
}

// TapEnter sends an Enter key to the tmux session
func (t *TmuxSession) TapEnter() error {
	return t.SendKeys("\r")
}

// SendScrollUp sends scroll up command to tmux session
func (t *TmuxSession) SendScrollUp() error {
	// Use tmux copy-mode with scroll up
	cmd := exec.Command("tmux", "copy-mode", "-t", t.sanitizedName)
	if err := cmd.Run(); err != nil {
		return err
	}
	// Send multiple up arrows for smoother scrolling (3 lines up)
	cmd = exec.Command("tmux", "send-keys", "-t", t.sanitizedName, "Up", "Up", "Up")
	return cmd.Run()
}

// SendScrollDown sends scroll down command to tmux session
func (t *TmuxSession) SendScrollDown() error {
	// Try to scroll down - if at bottom, this will exit copy mode automatically
	cmd := exec.Command("tmux", "send-keys", "-t", t.sanitizedName, "Down", "Down", "Down")
	return cmd.Run()
}

// Kill terminates the tmux session
func (t *TmuxSession) Kill() error {
	// Cancel any active operations
	if t.cancel != nil {
		t.cancel()
	}

	// Close PTY
	if t.ptmx != nil {
		t.ptmx.Close()
		t.ptmx = nil
	}

	// Kill the tmux session
	cmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
	return cmd.Run()
}

// GetPTY returns the current PTY file descriptor
func (t *TmuxSession) GetPTY() *os.File {
	return t.ptmx
}

// GetSessionName returns the sanitized session name
func (t *TmuxSession) GetSessionName() string {
	return t.sanitizedName
}