package skv

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// LogLevelDebug includes all messages (very verbose)
	LogLevelDebug LogLevel = iota
	// LogLevelInfo includes info, warn, and error messages
	LogLevelInfo
	// LogLevelWarn includes warn and error messages
	LogLevelWarn
	// LogLevelError includes only error messages
	LogLevelError
	// LogLevelNone disables all logging
	LogLevelNone
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelNone:
		return "none"
	default:
		return "unknown"
	}
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// Logger is the interface for structured logging in SKV
type Logger interface {
	// Debug logs a debug-level message with optional fields
	Debug(msg string, fields ...Field)
	// Info logs an info-level message with optional fields
	Info(msg string, fields ...Field)
	// Warn logs a warning-level message with optional fields
	Warn(msg string, fields ...Field)
	// Error logs an error-level message with optional fields
	Error(msg string, fields ...Field)
	// SetLevel changes the minimum log level
	SetLevel(level LogLevel)
}

// nullLogger is a logger that does nothing (default)
type nullLogger struct{}

func (l *nullLogger) Debug(msg string, fields ...Field) {}
func (l *nullLogger) Info(msg string, fields ...Field)  {}
func (l *nullLogger) Warn(msg string, fields ...Field)  {}
func (l *nullLogger) Error(msg string, fields ...Field) {}
func (l *nullLogger) SetLevel(level LogLevel)           {}

// NullLogger returns a logger that discards all messages
func NullLogger() Logger {
	return &nullLogger{}
}

// jsonLogger writes structured logs in JSON format
type jsonLogger struct {
	writer   io.Writer
	level    LogLevel
	mu       sync.Mutex
	encoder  *json.Encoder
	hostname string
}

// NewJSONLogger creates a new JSON logger
// writer: where to write logs (os.Stderr, file, etc.)
// level: minimum log level to output
func NewJSONLogger(writer io.Writer, level LogLevel) Logger {
	hostname, _ := os.Hostname()
	return &jsonLogger{
		writer:   writer,
		level:    level,
		encoder:  json.NewEncoder(writer),
		hostname: hostname,
	}
}

func (l *jsonLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *jsonLogger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := make(map[string]interface{})
	entry["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level.String()
	entry["msg"] = msg

	if l.hostname != "" {
		entry["hostname"] = l.hostname
	}

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	// Ignore encoding errors (best effort)
	_ = l.encoder.Encode(entry)
}

func (l *jsonLogger) Debug(msg string, fields ...Field) {
	l.log(LogLevelDebug, msg, fields...)
}

func (l *jsonLogger) Info(msg string, fields ...Field) {
	l.log(LogLevelInfo, msg, fields...)
}

func (l *jsonLogger) Warn(msg string, fields ...Field) {
	l.log(LogLevelWarn, msg, fields...)
}

func (l *jsonLogger) Error(msg string, fields ...Field) {
	l.log(LogLevelError, msg, fields...)
}

// textLogger writes human-readable logs
type textLogger struct {
	writer   io.Writer
	level    LogLevel
	mu       sync.Mutex
	hostname string
}

// NewTextLogger creates a new text logger
// writer: where to write logs (os.Stdout, file, etc.)
// level: minimum log level to output
func NewTextLogger(writer io.Writer, level LogLevel) Logger {
	hostname, _ := os.Hostname()
	return &textLogger{
		writer:   writer,
		level:    level,
		hostname: hostname,
	}
}

func (l *textLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *textLogger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	levelStr := level.String()

	// Build field string
	fieldStr := ""
	for i, f := range fields {
		if i > 0 {
			fieldStr += " "
		}
		fieldStr += fmt.Sprintf("%s=%v", f.Key, f.Value)
	}

	if fieldStr != "" {
		fmt.Fprintf(l.writer, "%s [%s] %s | %s\n", timestamp, levelStr, msg, fieldStr)
	} else {
		fmt.Fprintf(l.writer, "%s [%s] %s\n", timestamp, levelStr, msg)
	}
}

func (l *textLogger) Debug(msg string, fields ...Field) {
	l.log(LogLevelDebug, msg, fields...)
}

func (l *textLogger) Info(msg string, fields ...Field) {
	l.log(LogLevelInfo, msg, fields...)
}

func (l *textLogger) Warn(msg string, fields ...Field) {
	l.log(LogLevelWarn, msg, fields...)
}

func (l *textLogger) Error(msg string, fields ...Field) {
	l.log(LogLevelError, msg, fields...)
}
