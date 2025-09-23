package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"agate/config"
)

// DebugLogger manages debug output using Go's standard logging
type DebugLogger struct {
	logger  *log.Logger
	logFile *os.File
}

// NewDebugLogger creates a new debug logger using standard Go logging
func NewDebugLogger() *DebugLogger {
	// Ensure .agate directory exists
	if err := config.EnsureAgateDir(); err != nil {
		// Fall back to stderr if directory creation fails
		fmt.Fprintf(os.Stderr, "Warning: Failed to create .agate directory: %v\n", err)
	}

	// Get .agate directory path
	agateDir, err := config.GetAgateDir()
	if err != nil {
		// Fall back to current directory if we can't get .agate dir
		agateDir = "."
	}

	// Create debug log path in .agate directory
	debugLogPath := filepath.Join(agateDir, "debug.log")

	// Create or open debug log file
	logFile, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fall back to stderr if file creation fails
		logFile = os.Stderr
	}

	// Create logger with timestamp and file info
	logger := log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lshortfile)

	debugLogger := &DebugLogger{
		logger:  logger,
		logFile: logFile,
	}

	// Log session start
	logger.Println("=== Debug session started ===")

	return debugLogger
}

// Log adds a message using Go's standard logger
func (d *DebugLogger) Log(format string, args ...interface{}) {
	// Use standard logger for file output only
	d.logger.Printf(format, args...)
}

// Close closes the debug log file
func (d *DebugLogger) Close() {
	d.logger.Println("=== Debug session ended ===")

	if d.logFile != nil && d.logFile != os.Stderr {
		if err := d.logFile.Close(); err != nil {
			// Log to stderr as last resort since logFile is failing
			fmt.Fprintf(os.Stderr, "Warning: Failed to close debug log file: %v\n", err)
		}
	}
}

// IsEnabled returns true (always available now)
func (d *DebugLogger) IsEnabled() bool {
	return true
}

// Global debug logger instance
var globalDebugLogger *DebugLogger

// DebugLog logs a message to the global debug logger
func DebugLog(format string, args ...interface{}) {
	if globalDebugLogger != nil {
		globalDebugLogger.Log(format, args...)
	}
}

// InitDebugLogger initializes the global debug logger
func InitDebugLogger() *DebugLogger {
	globalDebugLogger = NewDebugLogger()
	return globalDebugLogger
}
