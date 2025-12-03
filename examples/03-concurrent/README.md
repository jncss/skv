# Concurrent Access Examples

Learn how SKV handles concurrent operations safely from multiple goroutines.

## Example

### `concurrent.go`
Comprehensive concurrency demonstrations:
- Concurrent writes from multiple goroutines
- Concurrent reads (multiple readers simultaneously)
- Mixed read/write operations
- Thread-safe updates without external locking
- Concurrent iteration

**Key Points:**
- All SKV operations are thread-safe
- No external locking required
- Uses internal `sync.RWMutex` for goroutine-level safety
- Uses file locking for process-level coordination

**Run:**
```bash
go run concurrent.go
```

## Thread Safety Guarantees

SKV provides **automatic thread safety** at two levels:

### 1. Goroutine-level (within process)
- `sync.RWMutex` protects all operations
- Multiple goroutines can safely access the same SKV instance
- No data races (verified with `go test -race`)

### 2. Process-level (between processes)
- File locking coordinates access between different processes
- Shared locks for reads (concurrent readers allowed)
- Exclusive locks for writes (serialized)

## When to Use

Perfect for:
- Web servers handling multiple requests
- Background workers processing jobs
- Multi-threaded applications
- Shared caches accessed by multiple goroutines

## Performance Notes

- **Reads are concurrent** - Multiple goroutines can read simultaneously
- **Writes are serialized** - Only one write at a time (prevents corruption)
- **Cache lookups are O(1)** - Fast even under high concurrency

## Related Examples

- **01-basics/** - Learn core operations first
- **02-advanced/** - Batch operations for better concurrent performance
- **04-usecases/** - Real-world concurrent patterns (caching, sessions)
