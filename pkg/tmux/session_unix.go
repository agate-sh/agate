//go:build !windows

package tmux

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

// monitorWindowSize monitors and handles window resize events while attached.
func (t *TmuxSession) monitorWindowSize() {
	winchChan := make(chan os.Signal, 1)
	signal.Notify(winchChan, syscall.SIGWINCH)
	// Send initial SIGWINCH to trigger the first resize
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)

	doUpdate := func() {
		// Use the current terminal height and width.
		cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			// If we can't get terminal size, skip this update
			return
		}
		_ = t.updateWindowSize(cols, rows) // Log error but continue
	}
	// Do one at the end of the function to set the initial size.
	defer doUpdate()

	// Debounce resize events
	t.wg.Add(2)
	debouncedWinch := make(chan os.Signal, 1)
	go func() {
		defer t.wg.Done()
		var resizeTimer *time.Timer
		for {
			if t.ctx == nil {
				return
			}
			select {
			case <-t.ctx.Done():
				return
			case <-winchChan:
				if resizeTimer != nil {
					resizeTimer.Stop()
				}
				resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
					if t.ctx != nil {
						select {
						case debouncedWinch <- syscall.SIGWINCH:
						case <-t.ctx.Done():
						}
					}
				})
			}
		}
	}()
	go func() {
		defer t.wg.Done()
		defer signal.Stop(winchChan)
		// Handle resize events
		for {
			if t.ctx == nil {
				return
			}
			select {
			case <-t.ctx.Done():
				return
			case <-debouncedWinch:
				doUpdate()
			}
		}
	}()
}
