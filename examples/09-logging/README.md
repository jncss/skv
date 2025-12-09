# Logging Example

This example demonstrates SKV's structured logging capabilities.

## What This Example Shows

1. **Default behavior** - No logging (NullLogger)
2. **Text logger** - Human-readable logs for development
3. **JSON logger** - Machine-readable logs for production
4. **Log levels** - Controlling verbosity (Debug, Info, Warn, Error)
5. **File logging** - Writing logs to a file
6. **Dynamic levels** - Changing log level at runtime

## Running the Example

```bash
cd examples/07-logging
go run main.go
```

## Key Concepts

### Logger Types

- **NullLogger**: Default, zero overhead, no output
- **TextLogger**: Human-readable, good for console/development
- **JSONLogger**: Structured JSON, good for log aggregation

### Log Levels

From most to least verbose:
- `LogLevelDebug` - Everything (Put, Get, Update, Delete)
- `LogLevelInfo` - Important events (Verify, Compact, WAL recovery)
- `LogLevelWarn` - Warnings (corrupted WAL entries)
- `LogLevelError` - Errors only

### Logged Information

Each log entry includes:
- **Timestamp** - When the operation occurred
- **Level** - Log severity
- **Message** - What happened
- **Fields** - Contextual data (key, size, duration, etc.)

## Example Output

### Text Logger (Debug Level)
```
2024-12-07T20:00:00.123Z [debug] Put successful | key=user:1 data_size=5 compression=none position=6 duration_ms=1
2024-12-07T20:00:00.124Z [debug] Get successful | key=user:1 data_size=5 cache_hit=true duration_ms=0
```

### JSON Logger (Info Level)
```json
{"time":"2024-12-07T20:00:00.123Z","level":"info","msg":"Verify completed","file_size":1024,"total_records":2,"active_records":2,"deleted_records":0,"wasted_percent":"0.00","efficiency":"100.00","duration_ms":1}
```

## Use Cases

1. **Development**: Use TextLogger with Debug level to see everything
2. **Production**: Use JSONLogger with Info level to monitor key operations
3. **Debugging**: Temporarily increase to Debug level to troubleshoot issues
4. **Compliance**: Log all operations to file for audit trail

## See Also

- [LOGGING.md](../../LOGGING.md) - Complete logging documentation
- [README.md](../../README.md) - Main documentation
