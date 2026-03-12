package logging

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// Test with default level
	logger := NewLogger("")
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}

	// Test with custom level
	logger = NewLogger("DEBUG")
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Level
		hasError bool
	}{
		{
			name:     "debug",
			input:    "debug",
			expected: DEBUG,
			hasError: false,
		},
		{
			name:     "info",
			input:    "info",
			expected: INFO,
			hasError: false,
		},
		{
			name:     "warn",
			input:    "warn",
			expected: WARN,
			hasError: false,
		},
		{
			name:     "error",
			input:    "error",
			expected: ERROR,
			hasError: false,
		},
		{
			name:     "fatal",
			input:    "fatal",
			expected: FATAL,
			hasError: false,
		},
		{
			name:     "invalid",
			input:    "invalid",
			expected: INFO,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.input)
			if logger.level != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, logger.level)
			}
		})
	}
}

func TestDebug(t *testing.T) {
	logger := NewLogger("DEBUG")

	logger.Debug("test debug message")
	// No assertion needed, just verify it doesn't panic
}

func TestInfo(t *testing.T) {
	logger := NewLogger("INFO")

	logger.Info("test info message")
	// No assertion needed, just verify it doesn't panic
}

func TestWarn(t *testing.T) {
	logger := NewLogger("WARN")

	logger.Warn("test warn message")
	// No assertion needed, just verify it doesn't panic
}

func TestError(t *testing.T) {
	logger := NewLogger("ERROR")

	logger.Error("test error message")
	// No assertion needed, just verify it doesn't panic
}

func TestFatal(t *testing.T) {
	// Set environment variable to prevent os.Exit
	os.Setenv("GO_TESTING", "true")
	defer os.Unsetenv("GO_TESTING")

	logger := NewLogger("FATAL")

	// Fatal should not panic in tests, just log
	logger.Fatal("test fatal message")
	// No assertion needed, just verify it doesn't panic
}

func TestToSlogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected slog.Level
	}{
		{
			name:     "debug",
			level:    DEBUG,
			expected: slog.LevelDebug,
		},
		{
			name:     "info",
			level:    INFO,
			expected: slog.LevelInfo,
		},
		{
			name:     "warn",
			level:    WARN,
			expected: slog.LevelWarn,
		},
		{
			name:     "error",
			level:    ERROR,
			expected: slog.LevelError,
		},
		{
			name:     "fatal",
			level:    FATAL,
			expected: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.ToSlogLevel()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
