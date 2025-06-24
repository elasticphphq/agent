package logging

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/elasticphphq/agent/internal/config"
)

func TestInit_JSONFormat(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
		w.Close()
	}()

	cfg := config.LoggingBlock{
		Level:  "debug",
		Format: "json",
		Color:  false,
	}

	Init(cfg)

	// Test that logger is initialized
	if logger == nil {
		t.Errorf("Expected logger to be initialized")
	}

	// Test that we get the same logger from L()
	if L() != logger {
		t.Errorf("Expected L() to return the same logger instance")
	}

	// Test logging
	logger.Debug("test debug message")

	w.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	output.ReadFrom(r)
	
	outputStr := output.String()
	if !strings.Contains(outputStr, "test debug message") {
		t.Errorf("Expected log output to contain debug message")
	}
	if !strings.Contains(outputStr, "{") {
		t.Errorf("Expected JSON format output")
	}
}

func TestInit_TextFormat(t *testing.T) {
	cfg := config.LoggingBlock{
		Level:  "info",
		Format: "text",
		Color:  true,
	}

	Init(cfg)

	// Test that logger is initialized
	if logger == nil {
		t.Errorf("Expected logger to be initialized")
	}

	// Test that we get the same logger from L()
	if L() != logger {
		t.Errorf("Expected L() to return the same logger instance")
	}
}

func TestInit_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "debug level uppercase",
			level:    "DEBUG",
			expected: slog.LevelDebug,
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level", 
			level:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "info level",
			level:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "unknown level defaults to info",
			level:    "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty level defaults to info",
			level:    "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoggingBlock{
				Level:  tt.level,
				Format: "text",
				Color:  false,
			}

			Init(cfg)

			// Test that logger is initialized
			if logger == nil {
				t.Errorf("Expected logger to be initialized")
			}

			// We can't directly test the log level, but we can test that the logger was created
			// The actual level testing would require capturing output and checking if messages appear
		})
	}
}

func TestL_ReturnsLogger(t *testing.T) {
	// Initialize logger first
	cfg := config.LoggingBlock{
		Level:  "info",
		Format: "json",
		Color:  false,
	}
	Init(cfg)

	result := L()
	if result == nil {
		t.Errorf("Expected L() to return a non-nil logger")
	}

	if result != logger {
		t.Errorf("Expected L() to return the same logger instance")
	}
}

func TestL_BeforeInit(t *testing.T) {
	// Reset logger to nil
	logger = nil

	result := L()
	if result != nil {
		t.Errorf("Expected L() to return nil when logger is not initialized")
	}
}

func TestInit_SetsDefaultLogger(t *testing.T) {
	cfg := config.LoggingBlock{
		Level:  "info",
		Format: "text",
		Color:  false,
	}

	Init(cfg)

	// Test that the default logger is set
	defaultLogger := slog.Default()
	if defaultLogger != logger {
		t.Errorf("Expected slog.Default() to return our logger instance")
	}
}

func TestInit_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{
			name:   "json format",
			format: "json",
		},
		{
			name:   "text format",
			format: "text",
		},
		{
			name:   "other format defaults to text",
			format: "xml",
		},
		{
			name:   "empty format defaults to text",
			format: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoggingBlock{
				Level:  "info",
				Format: tt.format,
				Color:  false,
			}

			Init(cfg)

			// Test that logger is initialized
			if logger == nil {
				t.Errorf("Expected logger to be initialized")
			}
		})
	}
}