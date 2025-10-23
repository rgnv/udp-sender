package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_OutputFormat(t *testing.T) {
	// Create a logger that writes to a buffer for testing
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Test Info log
	logger.Info("Test info message", map[string]any{
		"key1": "value1",
		"key2": 42,
	})

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Log output should end with newline")
	}

	// Parse the JSON as a map since fields are now at top level
	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify core fields
	if logMap["level"] != "info" {
		t.Errorf("Expected level 'info', got '%s'", logMap["level"])
	}
	if logMap["message"] != "Test info message" {
		t.Errorf("Expected message 'Test info message', got '%s'", logMap["message"])
	}
	if logMap["time"] == nil || logMap["time"] == "" {
		t.Error("Time field should not be empty")
	}
	// Verify additional fields are at top level
	if logMap["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", logMap["key1"])
	}
	if logMap["key2"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected key2=42, got '%v'", logMap["key2"])
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*Logger, string, map[string]any)
		expected LogLevel
	}{
		{
			name: "Debug",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Debug(msg, fields)
			},
			expected: LogLevelDebug,
		},
		{
			name: "Info",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			expected: LogLevelInfo,
		},
		{
			name: "Warn",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Warn(msg, fields)
			},
			expected: LogLevelWarn,
		},
		{
			name: "Error",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Error(msg, fields)
			},
			expected: LogLevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				output:   &buf,
				minLevel: LogLevelDebug,
			}

			tt.logFunc(logger, "test message", nil)

			var logMap map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			if logMap["level"] != string(tt.expected) {
				t.Errorf("Expected level '%s', got '%s'", tt.expected, logMap["level"])
			}
		})
	}
}

func TestLogger_WithoutFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	logger.Info("Simple message")

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if logMap["message"] != "Simple message" {
		t.Errorf("Expected message 'Simple message', got '%s'", logMap["message"])
	}

	// Should only have the core fields (time, level, message)
	expectedFieldCount := 3
	if len(logMap) != expectedFieldCount {
		t.Errorf("Expected %d fields (time, level, message), got %d: %v", expectedFieldCount, len(logMap), logMap)
	}
}

func TestLogger_MultipleLogEntries(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Write multiple log entries
	logger.Info("First message")
	logger.Warn("Second message")
	logger.Error("Third message")

	// Split by newlines and verify each is valid JSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var logMap map[string]any
		if err := json.Unmarshal([]byte(line), &logMap); err != nil {
			t.Errorf("Failed to parse line %d as JSON: %v", i+1, err)
		}
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  LogLevel
		logLevel  LogLevel
		shouldLog bool
	}{
		{"Debug logs at debug level", LogLevelDebug, LogLevelDebug, true},
		{"Info logs at debug level", LogLevelDebug, LogLevelInfo, true},
		{"Debug blocked at info level", LogLevelInfo, LogLevelDebug, false},
		{"Info logs at info level", LogLevelInfo, LogLevelInfo, true},
		{"Error logs at info level", LogLevelInfo, LogLevelError, true},
		{"Debug blocked at error level", LogLevelError, LogLevelDebug, false},
		{"Info blocked at error level", LogLevelError, LogLevelInfo, false},
		{"Error logs at error level", LogLevelError, LogLevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				output:   &buf,
				minLevel: tt.minLevel,
			}

			// Log based on the test level
			switch tt.logLevel {
			case LogLevelDebug:
				logger.Debug("Test message")
			case LogLevelInfo:
				logger.Info("Test message")
			case LogLevelWarn:
				logger.Warn("Test message")
			case LogLevelError:
				logger.Error("Test message")
			}

			logged := buf.Len() > 0
			if logged != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v, but logged=%v", tt.shouldLog, logged)
			}
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelInfo,
	}

	// Debug should be filtered
	logger.Debug("Debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should be filtered at Info level")
	}

	// Change to debug level
	logger.SetLevel(LogLevelDebug)
	logger.Debug("Debug message after level change")
	if buf.Len() == 0 {
		t.Error("Debug message should be logged after setting level to Debug")
	}
}

func TestLogger_FieldConflicts(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Test that conflicting fields are prefixed with underscore
	logger.Info("Test message", map[string]any{
		"time":    "custom_time",
		"level":   "custom_level",
		"message": "custom_message",
		"safe":    "safe_value",
	})

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Core fields should have their proper values
	if logMap["message"] != "Test message" {
		t.Errorf("Expected message='Test message', got '%v'", logMap["message"])
	}
	if logMap["level"] != "info" {
		t.Errorf("Expected level='info', got '%v'", logMap["level"])
	}

	// Conflicting fields should be prefixed with underscore
	if logMap["_time"] != "custom_time" {
		t.Errorf("Expected _time='custom_time', got '%v'", logMap["_time"])
	}
	if logMap["_level"] != "custom_level" {
		t.Errorf("Expected _level='custom_level', got '%v'", logMap["_level"])
	}
	if logMap["_message"] != "custom_message" {
		t.Errorf("Expected _message='custom_message', got '%v'", logMap["_message"])
	}

	// Non-conflicting fields should be added as-is
	if logMap["safe"] != "safe_value" {
		t.Errorf("Expected safe='safe_value', got '%v'", logMap["safe"])
	}
}

func TestLogger_FieldOrder(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Log with additional fields
	logger.Info("Test message", map[string]any{
		"zebra": "last",
		"alpha": "first",
		"extra": 123,
	})

	jsonStr := strings.TrimSpace(buf.String())

	// Find positions of the three core fields
	timePos := strings.Index(jsonStr, `"time":`)
	levelPos := strings.Index(jsonStr, `"level":`)
	messagePos := strings.Index(jsonStr, `"message":`)

	// Verify they exist
	if timePos == -1 || levelPos == -1 || messagePos == -1 {
		t.Fatalf("Missing core fields in JSON output: %s", jsonStr)
	}

	// Verify they appear in the correct order: time < level < message
	if timePos >= levelPos || levelPos >= messagePos {
		t.Errorf("Fields not in correct order. Positions: time=%d, level=%d, message=%d. JSON: %s",
			timePos, levelPos, messagePos, jsonStr)
	}

	// Verify time is the first field (position 1, after opening brace)
	if timePos != 1 {
		t.Errorf("time should be the first field, but found at position %d. JSON: %s", timePos, jsonStr)
	}
}
