package debug

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger manages application logging to a file
type Logger struct {
	logger  *log.Logger
	logFile *os.File
	mu      sync.Mutex
	level   LogLevel
}

// NewLogger creates a new logger that writes to agate.log in the current working directory
func NewLogger() *Logger {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		// Fall back to executable directory if we can't get working directory
		execPath, err := os.Executable()
		if err != nil {
			workDir = "."
		} else {
			workDir = filepath.Dir(execPath)
		}
	}

	// Create log file path
	logPath := filepath.Join(workDir, "agate.log")

	// Create or open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fall back to stderr if file creation fails
		logFile = os.Stderr
	}

	// Create logger with timestamp
	logger := log.New(logFile, "", log.LstdFlags)

	appLogger := &Logger{
		logger:  logger,
		logFile: logFile,
		level:   LevelInfo,
	}

	// Log session start
	appLogger.Info("=== Application session started ===")

	return appLogger
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		l.log(LevelDebug, format, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		l.log(LevelInfo, format, args...)
	}
}

// Warn logs a warning message (also writes to console)
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		l.log(LevelWarn, format, args...)
		// Warnings also go to console
		fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
	}
}

// Error logs an error message (also writes to console)
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		l.log(LevelError, format, args...)
		// Errors always go to console
		fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
	}
}

// log writes a log message to the file
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logger == nil {
		return
	}

	levelPrefix := l.levelPrefix(level)
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("%s %s", levelPrefix, message)
}

// levelPrefix returns the prefix for a log level
func (l *Logger) levelPrefix(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "[DEBUG]"
	case LevelInfo:
		return "[INFO]"
	case LevelWarn:
		return "[WARN]"
	case LevelError:
		return "[ERROR]"
	default:
		return "[UNKNOWN]"
	}
}

// Close closes the log file
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Info("=== Application session ended ===")

	if l.logFile != nil && l.logFile != os.Stderr {
		if err := l.logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close log file: %v\n", err)
		}
	}
}

// Global logger instance
var globalLogger *Logger

// InitLogger initializes the global logger
func InitLogger() *Logger {
	globalLogger = NewLogger()
	return globalLogger
}

// GetLogger returns the global logger (creates it if it doesn't exist)
func GetLogger() *Logger {
	if globalLogger == nil {
		return InitLogger()
	}
	return globalLogger
}

// Log logs a message at INFO level using the global logger
func Log(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

// Error logs an error message using the global logger
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// DebugLog is an alias for Log for backwards compatibility
func DebugLog(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}
