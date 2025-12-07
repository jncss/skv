# Structured Logging in SKV

SKV provides comprehensive structured logging capabilities to help you monitor, debug, and analyze database operations. Logging is **optional** and has **zero performance overhead** when disabled (the default).

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Log Levels](#log-levels)
- [Logger Implementations](#logger-implementations)
- [Logged Operations](#logged-operations)
- [Log Fields](#log-fields)
- [Examples](#examples)
- [Performance](#performance)
- [Best Practices](#best-practices)

## Overview

Structured logging provides machine-readable logs with key-value pairs, making it easy to:

- **Debug** database operations and identify issues
- **Monitor** performance with operation durations
- **Audit** data access patterns
- **Analyze** cache hit rates and compression effectiveness
- **Track** WAL recovery and compaction operations

### Key Features

- **Zero overhead by default** - NullLogger discards all messages
- **Multiple formats** - JSON for machines, Text for humans
- **Structured fields** - Key-value pairs for easy parsing
- **Flexible log levels** - Debug, Info, Warn, Error
- **Thread-safe** - Safe for concurrent operations
- **Rich context** - Operation details, timings, errors

## Quick Start

```go
package main

import (
    "os"
    "github.com/jncss/skv"
)

func main() {
    // Create a JSON logger writing to stderr
    logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelInfo)
    
    // Open database with logging
    db, err := skv.OpenWithOptions("data.skv", &skv.Options{
        Logger: logger,
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Operations are now logged
    db.Put([]byte("user:1"), []byte("Alice"))
    // Logs: {"time":"2024-12-07T20:00:00Z","level":"debug","msg":"Put successful","key":"user:1","data_size":5,...}
}
```

## Log Levels

SKV supports five log levels, from most to least verbose:

| Level | Value | Description | Use Case |
|-------|-------|-------------|----------|
| `LogLevelDebug` | 0 | Very detailed information | Development, debugging |
| `LogLevelInfo` | 1 | General informational messages | Production monitoring |
| `LogLevelWarn` | 2 | Warning messages | Potential issues |
| `LogLevelError` | 3 | Error messages only | Critical issues |
| `LogLevelNone` | 4 | No logging | Disable all logs |

When you set a log level, only messages at that level or higher are logged:

```go
// Only Error messages
logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelError)

// All messages (Debug, Info, Warn, Error)
logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelDebug)

// Change level dynamically
logger.SetLevel(skv.LogLevelInfo)
```

## Logger Implementations

### NullLogger (Default)

The null logger discards all log messages with **zero overhead**. This is the default when no logger is specified.

```go
logger := skv.NullLogger()
// or just don't specify a logger
db, _ := skv.Open("data.skv") // Uses NullLogger by default
```

### JSONLogger

Outputs structured logs in JSON format, ideal for log aggregation systems (Elasticsearch, Splunk, CloudWatch, etc.).

```go
logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelInfo)
```

**Example output:**
```json
{"time":"2024-12-07T20:00:00.123Z","level":"info","msg":"Put successful","key":"user:1","data_size":256,"compression":"lz4","position":1024,"duration_ms":2}
```

### TextLogger

Outputs human-readable logs, ideal for development and console output.

```go
logger := skv.NewTextLogger(os.Stdout, skv.LogLevelDebug)
```

**Example output:**
```
2024-12-07T20:00:00.123Z [info] Put successful | key=user:1 data_size=256 compression=lz4 position=1024 duration_ms=2
```

### Custom Logger

Implement the `Logger` interface to create custom loggers:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    SetLevel(level LogLevel)
}

type Field struct {
    Key   string
    Value interface{}
}
```

## Logged Operations

### Basic Operations

**Put** (LogLevelDebug):
- Message: "Put successful"
- Fields: `key`, `data_size`, `compression`, `position`, `duration_ms`
- Errors: "Put failed" (LogLevelError) with `error`

**Get** (LogLevelDebug):
- Message: "Get successful"
- Fields: `key`, `data_size`, `cache_hit`, `duration_ms`
- Not found: "Get failed: key not found" (LogLevelDebug)
- Errors: "Get failed: seek error" or "Get failed: read error" (LogLevelError)

**Update** (LogLevelDebug):
- Message: "Update successful"
- Fields: `key`, `data_size`, `position`, `duration_ms`
- Errors: "Update failed during delete" or "Update failed during write" (LogLevelError)

**Delete** (LogLevelDebug):
- Message: "Delete successful"
- Fields: `key`, `duration_ms`
- Errors: "Delete failed" (LogLevelError) with `error`

### Maintenance Operations

**Verify** (LogLevelInfo):
- Message: "Verify completed"
- Fields: `file_size`, `total_records`, `active_records`, `deleted_records`, `wasted_percent`, `efficiency`, `duration_ms`

**Compact** (LogLevelInfo):
- Message: "Compact completed"
- Fields: `before_bytes`, `after_bytes`, `saved_bytes`, `saved_percent`, `active_records`, `duration_ms`
- Errors: "Compact failed during header write" (LogLevelError)

### WAL Operations

**Recovery** (LogLevelInfo):
- Message: "WAL recovery completed"
- Fields: `recovered_entries`, `corrupted_entries`
- Warnings: "WAL recovery stopped at corrupted entry" (LogLevelWarn)

**Commit** (LogLevelDebug):
- Message: "WAL commit logged"

**Truncate** (LogLevelDebug):
- Message: "WAL truncated"
- Errors: "WAL truncate failed" (LogLevelError)

## Log Fields

Common fields across operations:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `time` | string | ISO 8601 timestamp | "2024-12-07T20:00:00.123Z" |
| `level` | string | Log level | "info", "debug", "warn", "error" |
| `msg` | string | Human-readable message | "Put successful" |
| `key` | string | Record key | "user:1" |
| `data_size` | int | Data size in bytes | 256 |
| `compression` | string | Compression type | "none", "snappy", "lz4" |
| `position` | int | File position | 1024 |
| `duration_ms` | int | Operation duration | 2 |
| `cache_hit` | bool | Cache hit status | true |
| `error` | string | Error message | "key not found" |
| `file_size` | int | Total file size | 10485760 |
| `total_records` | int | Total record count | 1000 |
| `active_records` | int | Active record count | 850 |
| `deleted_records` | int | Deleted record count | 150 |
| `wasted_percent` | string | Wasted space % | "15.23" |
| `efficiency` | string | File efficiency % | "84.77" |
| `before_bytes` | int | Size before compact | 10485760 |
| `after_bytes` | int | Size after compact | 8912896 |
| `saved_bytes` | int | Bytes saved | 1572864 |
| `saved_percent` | string | Percentage saved | "15.00" |
| `recovered_entries` | int | WAL entries recovered | 5 |
| `corrupted_entries` | int | Corrupted WAL entries | 0 |

## Examples

### Development Logging (Console)

```go
logger := skv.NewTextLogger(os.Stdout, skv.LogLevelDebug)
db, _ := skv.OpenWithOptions("data.skv", &skv.Options{
    Logger: logger,
})
defer db.Close()

db.Put([]byte("key"), []byte("value"))
// Output: 2024-12-07T20:00:00.123Z [debug] Put successful | key=key data_size=5 compression=none position=6 duration_ms=1
```

### Production Logging (JSON to File)

```go
logFile, _ := os.OpenFile("db.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
defer logFile.Close()

logger := skv.NewJSONLogger(logFile, skv.LogLevelInfo)
db, _ := skv.OpenWithOptions("data.skv", &skv.Options{
    Logger:      logger,
    Compression: skv.CompressionLZ4,
})
defer db.Close()
```

### Monitoring Performance

```go
logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelInfo)
db, _ := skv.OpenWithOptions("data.skv", &skv.Options{Logger: logger})
defer db.Close()

// Perform maintenance
stats, _ := db.Verify()
// Logs: {"level":"info","msg":"Verify completed","file_size":1048576,"total_records":100,...}

db.Compact()
// Logs: {"level":"info","msg":"Compact completed","before_bytes":1048576,"after_bytes":892160,"saved_percent":"14.91",...}
```

### Debugging Issues

```go
logger := skv.NewTextLogger(os.Stderr, skv.LogLevelDebug)
db, _ := skv.OpenWithOptions("data.skv", &skv.Options{Logger: logger})
defer db.Close()

_, err := db.Get([]byte("missing_key"))
// Output: 2024-12-07T20:00:00.123Z [debug] Get failed: key not found | key=missing_key duration_ms=0
```

### Selective Logging

```go
// Log only errors in production
logger := skv.NewJSONLogger(os.Stderr, skv.LogLevelError)

// Dynamically increase verbosity for debugging
if debugMode {
    logger.SetLevel(skv.LogLevelDebug)
}
```

## Performance

Logging performance characteristics:

| Logger Type | Overhead When Enabled | Overhead When Disabled |
|-------------|----------------------|------------------------|
| NullLogger | **0%** (no-op) | **0%** (no-op) |
| JSONLogger (Error level) | < 1% | ~0% (level check) |
| JSONLogger (Debug level) | 2-5% | ~0% (level check) |
| TextLogger (Error level) | < 1% | ~0% (level check) |
| TextLogger (Debug level) | 2-5% | ~0% (level check) |

**Tips for minimal overhead:**
- Use `NullLogger()` by default (zero overhead)
- Use higher log levels (`Info`, `Error`) in production
- Use `Debug` level only when troubleshooting
- Write logs to fast destinations (local SSD, memory-mapped files)
- Avoid logging to slow network destinations synchronously

## Best Practices

### 1. Use Appropriate Log Levels

```go
// Production: Log maintenance and errors
logger := skv.NewJSONLogger(logFile, skv.LogLevelInfo)

// Development: Log everything for debugging
logger := skv.NewTextLogger(os.Stdout, skv.LogLevelDebug)

// Critical systems: Errors only
logger := skv.NewJSONLogger(alertFile, skv.LogLevelError)
```

### 2. Aggregate JSON Logs

JSON logs work well with log aggregation systems:

```bash
# Parse and filter JSON logs
cat db.log | jq 'select(.duration_ms > 100)'

# Extract specific fields
cat db.log | jq '{time, msg, key, duration_ms}'

# Group by operation
cat db.log | jq -r .msg | sort | uniq -c
```

### 3. Monitor Key Metrics

Important metrics to watch:

- **operation duration** - Identify slow operations
- **cache_hit rate** - Monitor cache effectiveness
- **wasted_percent** - Know when to compact
- **saved_percent** - Track compaction benefits
- **recovered_entries** - Detect crashes
- **corrupted_entries** - Identify corruption

### 4. Log Rotation

Implement log rotation to prevent disk fill:

```go
// Use a log rotation library like lumberjack
import "gopkg.in/natefinch/lumberjack.v2"

logWriter := &lumberjack.Logger{
    Filename:   "db.log",
    MaxSize:    100, // megabytes
    MaxBackups: 3,
    MaxAge:     28, // days
}

logger := skv.NewJSONLogger(logWriter, skv.LogLevelInfo)
```

### 5. Structured Analysis

```bash
# Find slow operations (> 10ms)
cat db.log | jq 'select(.duration_ms > 10)'

# Count operations by type
cat db.log | jq -r .msg | sort | uniq -c

# Average operation duration
cat db.log | jq -s 'map(.duration_ms) | add / length'

# Monitor compaction effectiveness
cat db.log | jq 'select(.msg == "Compact completed") | {time, saved_percent, duration_ms}'
```

## Integration Examples

### With Zerolog

```go
import "github.com/rs/zerolog"

type ZerologAdapter struct {
    logger zerolog.Logger
}

func (z *ZerologAdapter) Debug(msg string, fields ...skv.Field) {
    event := z.logger.Debug()
    for _, f := range fields {
        event = event.Interface(f.Key, f.Value)
    }
    event.Msg(msg)
}

// Implement Info, Warn, Error, SetLevel...
```

### With Slog (Go 1.21+)

```go
import "log/slog"

type SlogAdapter struct {
    logger *slog.Logger
}

func (s *SlogAdapter) Info(msg string, fields ...skv.Field) {
    args := make([]any, 0, len(fields)*2)
    for _, f := range fields {
        args = append(args, f.Key, f.Value)
    }
    s.logger.Info(msg, args...)
}

// Implement Debug, Warn, Error, SetLevel...
```

## See Also

- [README.md](README.md) - Main documentation
- [COMPRESSION.md](COMPRESSION.md) - Compression guide
- [examples/](examples/) - Usage examples
