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
- Designed for single-process use only

**Run:**
```bash
go run concurrent.go
```

## Thread Safety Guarantees

SKV provides **automatic thread safety** for goroutine-level concurrency:

### Goroutine-level (within process)
- `sync.RWMutex` protects all operations
- Multiple goroutines can safely access the same SKV instance
- No data races (verified with `go test -race`)

**Note:** This library is designed for single-process use. Multiple processes accessing the same database file simultaneously is not supported and may result in data corruption.

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
