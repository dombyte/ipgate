package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is the main logger struct that wraps slog.Logger
type Logger struct {
	logger *slog.Logger
	level  Level
}

// NewLogger creates a new Logger instance with the specified log level
// If levelStr is empty or invalid, it defaults to INFO level
func NewLogger(levelStr string) *Logger {
	level := parseLevel(levelStr)
	opts := &slog.HandlerOptions{
		Level: level.ToSlogLevel(),
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	return &Logger{
		logger: logger,
		level:  level,
	}
}

// parseLevel converts a string to a Level
// Valid values are: DEBUG, INFO, WARN, ERROR, FATAL (case-insensitive)
// Returns INFO level for invalid or empty values
func parseLevel(levelStr string) Level {
	if levelStr == "" {
		return INFO
	}
	levelStr = strings.ToUpper(strings.TrimSpace(levelStr))
	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// Debug logs a debug message if the current level allows it
func (l *Logger) Debug(msg string, args ...any) {
	if l.level <= DEBUG {
		l.logger.Debug(msg, args...)
	}
}

// Info logs an info message if the current level allows it
func (l *Logger) Info(msg string, args ...any) {
	if l.level <= INFO {
		l.logger.Info(msg, args...)
	}
}

// Warn logs a warning message if the current level allows it
func (l *Logger) Warn(msg string, args ...any) {
	if l.level <= WARN {
		l.logger.Warn(msg, args...)
	}
}

// Error logs an error message if the current level allows it
func (l *Logger) Error(msg string, args ...any) {
	if l.level <= ERROR {
		l.logger.Error(msg, args...)
	}
}

// Fatal logs a fatal message and exits the application
// Note: In tests, os.Exit is mocked or prevented from actually exiting
func (l *Logger) Fatal(msg string, args ...any) {
	if l.level <= FATAL {
		l.logger.Error(msg, args...)
		// Only exit in production, not in tests
		if testing := os.Getenv("GO_TESTING"); testing != "true" {
			os.Exit(1)
		}
	}
}
