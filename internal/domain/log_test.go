package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevelPriority(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected int
	}{
		{LogLevelDebug, 0},
		{LogLevelInfo, 1},
		{LogLevelDefault, 2},
		{LogLevelError, 3},
		{LogLevelFault, 4},
		{LogLevel("unknown"), 2}, // Should default to Default priority
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.Priority())
		})
	}
}

func TestLogLevelPriorityOrdering(t *testing.T) {
	// Verify that priorities are in correct order
	assert.Less(t, LogLevelDebug.Priority(), LogLevelInfo.Priority())
	assert.Less(t, LogLevelInfo.Priority(), LogLevelDefault.Priority())
	assert.Less(t, LogLevelDefault.Priority(), LogLevelError.Priority())
	assert.Less(t, LogLevelError.Priority(), LogLevelFault.Priority())
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"Debug", LogLevelDebug},
		{"info", LogLevelInfo},
		{"Info", LogLevelInfo},
		{"default", LogLevelDefault},
		{"Default", LogLevelDefault},
		{"error", LogLevelError},
		{"Error", LogLevelError},
		{"fault", LogLevelFault},
		{"Fault", LogLevelFault},
		{"unknown", LogLevelDefault},
		{"", LogLevelDefault},
		{"WARNING", LogLevelDefault}, // Unsupported level
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Level:     LogLevelError,
		Process:   "TestApp",
		PID:       12345,
		Subsystem: "com.test.app",
		Category:  "network",
		Message:   "Connection failed",
	}

	require.NotNil(t, entry)
	assert.Equal(t, LogLevelError, entry.Level)
	assert.Equal(t, "TestApp", entry.Process)
	assert.Equal(t, 12345, entry.PID)
	assert.Equal(t, "com.test.app", entry.Subsystem)
	assert.Equal(t, "network", entry.Category)
	assert.Equal(t, "Connection failed", entry.Message)
}

func TestLogLevelConstants(t *testing.T) {
	// Ensure constants are defined correctly
	assert.Equal(t, LogLevel("Debug"), LogLevelDebug)
	assert.Equal(t, LogLevel("Info"), LogLevelInfo)
	assert.Equal(t, LogLevel("Default"), LogLevelDefault)
	assert.Equal(t, LogLevel("Error"), LogLevelError)
	assert.Equal(t, LogLevel("Fault"), LogLevelFault)
}
