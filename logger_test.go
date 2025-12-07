package skv

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestNullLogger(t *testing.T) {
	logger := NullLogger()

	// Should not panic or do anything
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.SetLevel(LogLevelDebug)
}

func TestJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(&buf, LogLevelDebug)

	logger.Info("test message", Field{Key: "key1", Value: "value1"}, Field{Key: "count", Value: 42})

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output missing message: %s", output)
	}

	// Parse JSON to verify structure
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["level"] != "info" {
		t.Errorf("expected level=info, got: %v", entry["level"])
	}
	if entry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got: %v", entry["msg"])
	}
	if entry["key1"] != "value1" {
		t.Errorf("expected key1='value1', got: %v", entry["key1"])
	}
	if entry["count"].(float64) != 42 {
		t.Errorf("expected count=42, got: %v", entry["count"])
	}
}

func TestJSONLoggerLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		logFunc  func(Logger)
		expected bool
	}{
		{LogLevelDebug, func(l Logger) { l.Debug("test") }, true},
		{LogLevelDebug, func(l Logger) { l.Info("test") }, true},
		{LogLevelDebug, func(l Logger) { l.Warn("test") }, true},
		{LogLevelDebug, func(l Logger) { l.Error("test") }, true},
		{LogLevelInfo, func(l Logger) { l.Debug("test") }, false},
		{LogLevelInfo, func(l Logger) { l.Info("test") }, true},
		{LogLevelWarn, func(l Logger) { l.Info("test") }, false},
		{LogLevelWarn, func(l Logger) { l.Warn("test") }, true},
		{LogLevelError, func(l Logger) { l.Warn("test") }, false},
		{LogLevelError, func(l Logger) { l.Error("test") }, true},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		logger := NewJSONLogger(&buf, tt.level)
		tt.logFunc(logger)

		hasOutput := buf.Len() > 0
		if hasOutput != tt.expected {
			t.Errorf("level %v: expected output=%v, got output=%v", tt.level, tt.expected, hasOutput)
		}
	}
}

func TestTextLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewTextLogger(&buf, LogLevelDebug)

	logger.Info("test message", Field{Key: "key1", Value: "value1"}, Field{Key: "count", Value: 42})

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output missing message: %s", output)
	}
	if !strings.Contains(output, "[info]") {
		t.Errorf("log output missing level: %s", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("log output missing field: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("log output missing field: %s", output)
	}
}

func TestTextLoggerLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		logFunc  func(Logger)
		expected bool
	}{
		{LogLevelDebug, func(l Logger) { l.Debug("test") }, true},
		{LogLevelDebug, func(l Logger) { l.Info("test") }, true},
		{LogLevelInfo, func(l Logger) { l.Debug("test") }, false},
		{LogLevelInfo, func(l Logger) { l.Info("test") }, true},
		{LogLevelWarn, func(l Logger) { l.Info("test") }, false},
		{LogLevelWarn, func(l Logger) { l.Warn("test") }, true},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		logger := NewTextLogger(&buf, tt.level)
		tt.logFunc(logger)

		hasOutput := buf.Len() > 0
		if hasOutput != tt.expected {
			t.Errorf("level %v: expected output=%v, got output=%v", tt.level, tt.expected, hasOutput)
		}
	}
}

func TestLoggerWithSKV(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(&buf, LogLevelDebug)

	dbPath := "test_logger.skv"
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + ".wal")
	}()

	db, err := OpenWithOptions(dbPath, &Options{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Perform operations that should log
	if err := db.Put([]byte("test_key"), []byte("test_value")); err != nil {
		t.Fatalf("failed to put: %v", err)
	}

	// Check that logs were written
	output := buf.String()
	if !strings.Contains(output, "Put successful") {
		t.Errorf("expected Put log, got: %s", output)
	}

	// Clear buffer
	buf.Reset()

	// Get operation
	if _, err := db.Get([]byte("test_key")); err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "Get successful") {
		t.Errorf("expected Get log, got: %s", output)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(&buf, LogLevelError)

	// Should not log at Info level
	logger.Info("test")
	if buf.Len() > 0 {
		t.Errorf("expected no output at Info when level is Error")
	}

	// Change level
	logger.SetLevel(LogLevelDebug)

	// Should now log at Info level
	logger.Info("test")
	if buf.Len() == 0 {
		t.Errorf("expected output at Info after changing level to Debug")
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "debug"},
		{LogLevelInfo, "info"},
		{LogLevelWarn, "warn"},
		{LogLevelError, "error"},
		{LogLevelNone, "none"},
		{LogLevel(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}
