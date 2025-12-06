# 03 - Concurrent Operations

Demonstrates thread-safe concurrent access to SKV databases.

## Example

### `concurrent/`
Shows how multiple goroutines can safely access the same database:
- **Concurrent Writes**: Multiple goroutines writing different keys
- **Concurrent Reads**: Multiple goroutines reading simultaneously
- **Mixed Operations**: Reads, writes, updates, and deletes from multiple goroutines
- **Batch Operations**: Concurrent batch inserts
- **Performance Metrics**: Operations per second with concurrent access

**Run:**
```bash
cd examples/03-concurrent/concurrent
go run concurrent.go
```

**Run with race detector:**
```bash
cd examples/03-concurrent/concurrent
go run -race concurrent.go
```

## Key Concepts

### Thread Safety
SKV is thread-safe for concurrent access within a single process:
- All operations are protected by `sync.RWMutex`
- Multiple readers can access simultaneously (read locks)
- Writers get exclusive access (write locks)
- No external synchronization needed

### Safe Concurrent Patterns
```go
var wg sync.WaitGroup

// Multiple writers
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        db.PutString(fmt.Sprintf("key:%d", id), "value")
    }(i)
}

wg.Wait()
```

### Mixed Operations
```go
// Different goroutines can perform different operations
go func() {
    db.PutString("key1", "value1")
}()

go func() {
    db.GetString("key2")
}()

go func() {
    db.UpdateString("key3", "value3")
}()

go func() {
    db.DeleteString("key4")
}()
```

### Batch Operations
```go
// Concurrent batch inserts are safe
var wg sync.WaitGroup
for i := 0; i < 5; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        batch := make(map[string]string)
        for j := 0; j < 100; j++ {
            batch[fmt.Sprintf("batch:%d:item:%d", id, j)] = "value"
        }
        
        db.PutBatchString(batch)
    }(i)
}
wg.Wait()
```

## Important Notes

### Single Process Only
- SKV is thread-safe for **concurrent goroutines within the same process**
- **Not designed for multi-process access** (multiple programs accessing the same file)
- For multi-process scenarios, implement external locking or use a client-server architecture

### File Locking
- No file-level locking is implemented
- Opening the same database from multiple processes simultaneously may lead to corruption
- Use at the application level with proper coordination if multi-process access is needed

### Performance Considerations
- Read-heavy workloads benefit from concurrent access (shared read locks)
- Write-heavy workloads may experience contention (exclusive write locks)
- Batch operations reduce lock contention compared to individual operations

## Testing for Race Conditions

Always test concurrent code with the race detector:
```bash
go run -race concurrent.go
```

Or run tests:
```bash
cd /workspaces/skv
go test -race -run Concurrent
```

## Next Steps

- **04-usecases**: Real-world application examples
- **05-backup**: Backup and restore functionality
