package logging

import (
	"log/slog"
)

// Level represents the severity level of a log message
type Level int

const (
	// DEBUG level for detailed debugging information
	DEBUG Level = iota
	// INFO level for general operational messages
	INFO
	// WARN level for warning messages (non-critical issues)
	WARN
	// ERROR level for error messages (recoverable failures)
	ERROR
	// FATAL level for critical errors (application termination)
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// ToSlogLevel converts the Level to slog.Level
func (l Level) ToSlogLevel() slog.Level {
	switch l {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	case FATAL:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
