//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// Logger provides structured ND-JSON logging
type Logger struct {
	output io.Writer
}

// LogEntry represents a single log entry in JSON format
// Additional fields are added directly to the JSON at the top level
type LogEntry struct {
	Time    string   `json:"time"`
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
}

// NewLogger creates a new logger that writes to stderr
func NewLogger() *Logger {
	return &Logger{
		output: os.Stderr,
	}
}

// log outputs a log entry as ND-JSON
func (l *Logger) log(level LogLevel, message string, fields map[string]any) {
	// Create a map with all fields at the top level
	logMap := make(map[string]any)

	// Add additional fields first, checking for conflicts with reserved fields
	reservedFields := map[string]bool{
		"time":    true,
		"level":   true,
		"message": true,
	}

	for key, value := range fields {
		// If field conflicts with reserved field, prefix with underscore
		if reservedFields[key] {
			logMap["_"+key] = value
		} else {
			logMap[key] = value
		}
	}

	// Add core fields (these override any conflicts)
	logMap["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	logMap["level"] = level
	logMap["message"] = message

	data, err := json.Marshal(logMap)
	if err != nil {
		// Fallback to simple output if JSON marshaling fails
		fmt.Fprintf(l.output, "{\"time\":\"%s\",\"level\":\"error\",\"message\":\"failed to marshal log entry: %v\"}\n",
			time.Now().UTC().Format(time.RFC3339Nano), err)
		return
	}

	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

// Info logs an info-level message
func (l *Logger) Info(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelInfo, message, f)
}

// Error logs an error-level message
func (l *Logger) Error(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelError, message, f)
}

func main() {
	logger := NewLogger()

	fmt.Println("=== ND-JSON Logger Demonstration ===")
	fmt.Println("The following lines are newline-delimited JSON logs:\n")

	// Simple log without fields
	logger.Info("Application starting")

	// Log with fields
	logger.Info("Stream mode: reading packets from stdin", map[string]any{
		"protocol": "[Magic(3)][Version(1)][SrcIP(4/16)][DestIP(4/16)][SrcPort(2)][DestPort(2)][PayloadLen(2)][Payload(N)]",
	})

	// Log with multiple fields
	logger.Info("Progress update", map[string]any{
		"packets_sent": 100,
		"bytes_sent":   8192,
	})

	// Error log with fields
	logger.Error("Error creating UDP sender", map[string]any{
		"error": "permission denied",
	})

	// Log stream completion
	logger.Info("Stream complete", map[string]any{
		"packets_sent": 1000,
		"bytes_sent":   1048576,
	})
}
