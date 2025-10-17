//go:build windows

package tmux

// monitorWindowSize is a no-op on Windows
func (t *TmuxSession) monitorWindowSize() {
	// Windows doesn't have SIGWINCH, so we don't monitor window size changes
}