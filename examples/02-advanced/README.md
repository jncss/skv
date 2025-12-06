# 02 - Advanced Operations

Advanced features for efficient data management.

## Examples

### `batch_operations/`
Efficiently handle multiple keys at once:
- **PutBatchString**: Insert multiple key-value pairs in a single operation
- **PutBatch**: Batch insert with byte data
- **GetBatchString**: Retrieve multiple values at once
- **GetBatch**: Batch retrieval with byte data
- **Performance Benefits**: Reduced overhead and better efficiency

**Run:**
```bash
cd examples/02-advanced/batch_operations
go run batch_operations.go
```

### `file_operations/`
Store and retrieve files directly:
- **PutFile**: Store a file from disk into the database
- **GetFile**: Retrieve a value and write it to a file
- **UpdateFile**: Update an existing key with file contents
- **Binary Support**: Works with text and binary files
- **Use Cases**: Configuration files, templates, assets, documents

**Run:**
```bash
cd examples/02-advanced/file_operations
go run file_operations.go
```

### `iteration/`
Process all key-value pairs in the database:
- **ForEachString**: Iterate over all entries with string callback
- **ForEach**: Iterate with byte data
- **Filtering**: Process only matching keys during iteration
- **Early Termination**: Stop iteration by returning an error
- **Collecting Results**: Build lists or maps during iteration
- **KeysString/Keys**: Get all keys as a slice

**Run:**
```bash
cd examples/02-advanced/iteration
go run iteration.go
```

### `maintenance/`
Keep your database optimized:
- **Verify**: Get detailed statistics about database health
- **Stats**: Total/active/deleted records, duplicate keys, wasted space
- **Compact**: Remove deleted records and old versions to reduce file size
- **Conditional Compaction**: Compact only when wasted space exceeds a threshold
- **CloseWithCompact**: Automatically compact before closing

**Run:**
```bash
cd examples/02-advanced/maintenance
go run maintenance.go
```

## Key Concepts

### Batch Operations
```go
// Insert multiple at once (more efficient)
users := map[string]string{
    "user:1": "Alice",
    "user:2": "Bob",
    "user:3": "Charlie",
}
db.PutBatchString(users)

// Retrieve multiple at once
keys := []string{"user:1", "user:2"}
results, _ := db.GetBatchString(keys)
```

### File Operations
```go
// Store a file in the database
db.PutFile("config:app", "config.ini")

// Retrieve file from database
db.GetFile("config:app", "retrieved_config.ini")

// Update with new file contents
db.UpdateFile("config:app", "updated_config.ini")

// Stream writing (memory-efficient for large files)
file, _ := os.Open("large_video.mp4")
info, _ := file.Stat()
db.PutStreamString("video", file, info.Size())
file.Close()

// Stream reading (no full memory load)
output, _ := os.Create("output.dat")
defer output.Close()
bytesWritten, _ := db.GetStreamString("video", output)
```

### Iteration
```go
// Process all key-value pairs
db.ForEachString(func(key string, value string) error {
    fmt.Printf("%s: %s\n", key, value)
    return nil
})

// Early termination
db.ForEachString(func(key string, value string) error {
    if someCondition {
        return fmt.Errorf("stop") // Stops iteration
    }
    return nil
})
```

### Maintenance
```go
// Check database health
stats, _ := db.Verify()
fmt.Printf("Wasted space: %d bytes (%.2f%%)\n", 
    stats.WastedSpace,
    float64(stats.WastedSpace)/float64(stats.TotalSize)*100)

// Compact if needed
if stats.DeletedRecords > 100 {
    db.Compact()
}

// Or compact on close
defer db.CloseWithCompact()
```

### Statistics Structure
```go
type Stats struct {
    TotalRecords    int    // All records (active + deleted)
    ActiveRecords   int    // Currently active records
    DeletedRecords  int    // Marked as deleted
    DuplicateKeys   int    // Old versions of updated keys
    TotalSize       int64  // Total file size
    ActiveDataSize  int64  // Size of active data only
    WastedSpace     int64  // Space from deleted/duplicate records
}
```

## Performance Tips

1. **Use batch operations** when inserting/retrieving multiple keys
2. **Compact periodically** to maintain optimal file size (e.g., 30% wasted space threshold)
3. **Use ForEach** instead of Keys + Get loop for better performance
4. **Monitor statistics** with Verify to understand database health
5. **CloseWithCompact** for long-running applications that accumulate many updates

## Next Steps

- **03-concurrent**: Thread-safe concurrent operations
- **04-usecases**: Real-world application examples
- **05-backup**: Backup and restore functionality
