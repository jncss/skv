# SKV Examples

This directory contains practical examples demonstrating various features and use cases of the SKV library.

## Directory Structure

```
examples/
├── 01-basics/          # Basic operations and getting started
├── 02-advanced/        # Advanced features and optimization
├── 03-concurrent/      # Concurrency and thread-safety
├── 04-usecases/        # Real-world use cases
├── 05-backup/          # Backup and restore operations
└── 06-fileformat/      # File format and header information
```

## Running Examples

```bash
cd examples/01-basics
go run basic_usage.go

cd ../02-advanced
go run batch_operations.go

# ... etc
```

## Examples by Category

### 📚 01-basics/ - Getting Started

#### `basic_usage.go`
Introduction to basic SKV operations:
- Opening/closing database
- Storing and retrieving data
- Updating values
- Checking key existence
- Counting keys
- Listing all keys
- Deleting keys
- Using default values

**Perfect for**: First-time users, quick start guide

#### `put_vs_update.go`
Demonstrates the difference between `Put()` and `Update()`:
- `Put()` creates new keys only
- `Update()` modifies existing keys only
- Error handling for duplicate keys
- Error handling for missing keys

**Perfect for**: Understanding the insert vs update semantics

### 🚀 02-advanced/ - Advanced Features

#### `batch_operations.go`
Shows how to work with multiple keys efficiently:
- Batch insert with `PutBatch()`
- Batch retrieve with `GetBatch()`
- Atomic behavior (all-or-nothing)
- Error handling in batch operations

**Perfect for**: Performance optimization, bulk operations

#### `iteration.go`
Demonstrates iteration over all keys:
- Using `ForEach()` and `ForEachString()`
- Processing all key-value pairs
- Calculating aggregates
- Filtering data
- Early termination
- Working with binary data

**Perfect for**: Data processing, reporting, migrations

#### `maintenance.go`
Database maintenance and optimization:
- Checking database statistics with `Verify()`
- Understanding file growth with updates/deletes
- Compacting database to reclaim space
- `CloseWithCompact()` for automatic cleanup
- Monitoring file size

**Perfect for**: Production deployments, long-running applications

### ⚡ 03-concurrent/ - Concurrency

#### `concurrent.go`
Thread-safety and concurrent access:
- Concurrent writes from multiple goroutines
- Concurrent reads
- Mixed read/write operations
- Thread-safe updates
- Concurrent iteration

**Perfect for**: Multi-threaded applications, web servers

### 💡 04-usecases/ - Real-World Applications

#### `usecases.go`
Real-world use cases:
- Configuration storage
- Session management
- Structured data (JSON)
- Cache implementation
- Feature flags
- Counters and metrics
- Namespaced keys

**Perfect for**: Understanding practical applications

### 💾 05-backup/ - Backup and Restore

#### `demo.go`
Backup and restore operations:
- Creating JSON backups
- Smart encoding (text vs base64)
- Restoring from backup
- Partial restoration (preserves keys not in backup)
- Disaster recovery workflows
- Human-readable backup format

**Perfect for**: Data migration, disaster recovery, database inspection

### 🔧 06-fileformat/ - File Format Details

#### `demo.go`
File format and header information:
- SKV file header structure
- Version information (Major.Minor.Patch)
- Backward compatibility with old format
- File size breakdown

**Perfect for**: Understanding the file format internals

## Key Concepts Demonstrated

### Thread Safety
All examples can be run concurrently without external locking. The library handles synchronization internally.

### Performance
- O(1) lookups using in-memory cache
- Batch operations for efficiency
- Compact operations to optimize storage

### Data Integrity
- Atomic batch operations
- File locking prevents corruption
- Verify operation checks integrity

## Common Patterns

### Configuration Storage
```go
db.PutString("config.timeout", "30")
timeout := db.GetOrDefaultString("config.timeout", "10")
```

### Session Management
```go
db.PutString("session:"+sessionID, sessionData)
if db.HasString("session:"+sessionID) {
    // Session exists
}
```

### Caching
```go
if db.Exists(cacheKey) {
    return db.Get(cacheKey) // Cache hit
}
// Cache miss - compute and store
db.Put(cacheKey, computedValue)
```

### Feature Flags
```go
enabled := db.GetOrDefaultString("feature:new_ui", "false") == "true"
```

## Tips

1. **Use string functions** for text data: `PutString()`, `GetString()`, etc.
2. **Use batch operations** when working with multiple keys
3. **Compact regularly** if you do many updates/deletes
4. **Use namespaces** in keys for organization: `user:123`, `config:app`
5. **Check `Count()`** before iterating if you need to know size first

## Clean Up

Example databases are created in the examples directory. To clean up:

```bash
rm *.skv
```
