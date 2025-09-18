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
	attachCh    chan struct{}
	ctx         context.Context
	cancel      func()
	wg          *sync.WaitGroup
	detaching   bool      // Guard to prevent duplicate detachment processing
	detachMutex sync.Mutex // Protect detachment state

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
		// Create new tmux session
		cmd := exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName, "-c", workDir, t.program)

		// Run the command directly without PTY for session creation
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error creating tmux session: %w, output: %s", err, string(output))
		}

		// Wait for session to be created
		for i := 0; i < 10; i++ {
			exists, _ = t.sessionExists()
			if exists {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if !exists {
			return fmt.Errorf("tmux session failed to start")
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

	// For detached monitoring, we don't need a PTY - we'll use capture-pane
	// Just verify the session exists
	exists, err := t.sessionExists()
	if err != nil {
		return fmt.Errorf("error checking session: %w", err)
	}
	if !exists {
		return fmt.Errorf("tmux session %s does not exist", t.sanitizedName)
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

	// Create a PTY for attaching to the tmux session
	cmd := exec.Command("tmux", "attach-session", "-t", t.sanitizedName)

	ptmx, err := t.ptyFactory.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("error attaching to tmux session: %w", err)
	}

	// Close any existing PTY
	if t.ptmx != nil {
		t.ptmx.Close()
	}
	t.ptmx = ptmx

	t.attachCh = make(chan struct{})
	t.wg = &sync.WaitGroup{}
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// Create a channel to signal when stdin goroutine detects Ctrl+Q
	detachSignal := make(chan struct{}, 1)

	// Start goroutine to copy tmux output to stdout
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		_, _ = io.Copy(os.Stdout, t.ptmx)

		// Check if it was a normal detach
		select {
		case <-t.ctx.Done():
		default:
			// Abnormal termination
			fmt.Fprintf(os.Stderr, "\n\033[31mError: Session terminated without detaching. Use Ctrl+Q to properly detach from tmux sessions.\033[0m\n")
		}
	}()

	// Start goroutine to forward stdin to tmux
	t.wg.Add(1)
	go func() {
		defer func() {
			t.wg.Done()
		}()

		// Small delay to handle initial terminal control sequences
		timeoutCh := make(chan struct{})
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(timeoutCh)
		}()

		buf := make([]byte, 32)
		for {
			// Check if we should exit before reading
			select {
			case <-t.ctx.Done():
				return
			default:
			}

			// Use a channel to make stdin read interruptible
			type readResult struct {
				n   int
				err error
			}

			readCh := make(chan readResult, 1)
			go func() {
				n, err := os.Stdin.Read(buf)
				readCh <- readResult{n, err}
			}()

			// Wait for either read completion or context cancellation
			var nr int
			var err error
			select {
			case <-t.ctx.Done():
				return
			case result := <-readCh:
				nr = result.n
				err = result.err
			}

			if err != nil {
				if err == io.EOF {
					break
				}
				continue
			}

			// Skip initial control sequences
			select {
			case <-timeoutCh:
			default:
				continue
			}

			// Check for detach key (Ctrl+Q)
			if nr == 1 && buf[0] == 17 { // ASCII 17 is Ctrl+Q
				// Check if already detaching to prevent duplicate processing
				t.detachMutex.Lock()
				if t.detaching {
					t.detachMutex.Unlock()
					continue
				}
				t.detaching = true
				t.detachMutex.Unlock()

				select {
				case detachSignal <- struct{}{}:
				default:
				}
				return
			}

			// Forward to tmux
			_, writeErr := t.ptmx.Write(buf[:nr])
			if writeErr != nil {
			}
		}
	}()

	// Monitor for detach signal
	go func() {
		select {
		case <-detachSignal:
			t.Detach()
		case <-t.ctx.Done():
		}
	}()

	// Monitor window size changes
	t.monitorWindowSize()

	return t.attachCh, nil
}

// Detach detaches from the tmux session
func (t *TmuxSession) Detach() error {

	// Clear state variables immediately like claude-squad does
	defer func() {
		t.detachMutex.Lock()
		t.detaching = false
		t.attachCh = nil
		t.ctx = nil
		t.cancel = nil
		t.wg = nil
		t.detachMutex.Unlock()
	}()

	if t.cancel != nil {
		t.cancel()
	}

	// Close PTY first to unblock io.Copy
	if t.ptmx != nil {
		t.ptmx.Close()
		t.ptmx = nil
	}

	if t.wg != nil {
		t.wg.Wait()
	}

	// Close the attachment channel
	if t.attachCh != nil {
		close(t.attachCh)
	}

	// Re-establish PTY for detached monitoring
	err := t.Restore()
	if err != nil {
	} else {
	}

	return err
}

// monitorWindowSize monitors and updates terminal size
func (t *TmuxSession) monitorWindowSize() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				return
			case <-ticker.C:
				// Get current terminal size
				if size, err := pty.GetsizeFull(os.Stdin); err == nil {
					t.updateWindowSize(int(size.Cols), int(size.Rows))
				}
			}
		}
	}()
}

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
	}
	return nil
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

// GetSessionName returns the sanitized session name
func (t *TmuxSession) GetSessionName() string {
	return t.sanitizedName
}

// GetPTY returns the current PTY file descriptor
func (t *TmuxSession) GetPTY() *os.File {
	return t.ptmx
}