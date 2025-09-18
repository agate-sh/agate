package tmux

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// PtyFactory is an interface for creating PTYs, allowing for mocking in tests
type PtyFactory interface {
	Start(cmd *exec.Cmd) (*os.File, error)
}

// RealPtyFactory uses the creack/pty library to create actual PTYs
type RealPtyFactory struct{}

// NewPtyFactory creates a new PtyFactory for production use
func NewPtyFactory() PtyFactory {
	return &RealPtyFactory{}
}

// Start creates a PTY and starts the given command
func (f *RealPtyFactory) Start(cmd *exec.Cmd) (*os.File, error) {
	return pty.Start(cmd)
}

// MockPtyFactory is a mock implementation for testing
type MockPtyFactory struct {
	Commands []*exec.Cmd
	Files    []*os.File
	Err      error
}

// NewMockPtyFactory creates a new mock PTY factory for testing
func NewMockPtyFactory() *MockPtyFactory {
	return &MockPtyFactory{
		Commands: make([]*exec.Cmd, 0),
		Files:    make([]*os.File, 0),
	}
}

// Start simulates creating a PTY for testing
func (f *MockPtyFactory) Start(cmd *exec.Cmd) (*os.File, error) {
	if f.Err != nil {
		return nil, f.Err
	}

	// Create a temporary file to simulate a PTY
	tmpFile, err := os.CreateTemp("", "mock-pty-")
	if err != nil {
		return nil, err
	}

	f.Commands = append(f.Commands, cmd)
	f.Files = append(f.Files, tmpFile)

	return tmpFile, nil
}

// Close cleans up the mock PTY factory
func (f *MockPtyFactory) Close() {
	for _, file := range f.Files {
		if file != nil {
			file.Close()
			os.Remove(file.Name())
		}
	}
}