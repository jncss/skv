# SKV - Simple Key Value

A Go library for storing key/value data in a sequential binary file format.

[![Production Ready](https://img.shields.io/badge/production-ready-green.svg)](https://github.com/jncss/skv)
[![Test Coverage](https://img.shields.io/badge/coverage-81.8%25-brightgreen.svg)](https://github.com/jncss/skv)
[![Tests Passing](https://img.shields.io/badge/tests-102%20passing-brightgreen.svg)](https://github.com/jncss/skv)
[![Go Version](https://img.shields.io/badge/go-1.24.0+-blue.svg)](https://golang.org/dl/)

**Performance Metrics:**
- ⚡ **750 inserts/sec** - Sequential writes (10,000 records tested)
- 🚀 **270,000 reads/sec** - In-memory cached lookups
- 🔄 **365 updates/sec** - With automatic space reuse
- 🧵 **1,900 ops/sec** - Concurrent operations (10 goroutines, race-free)
- 📦 **37% reduction** - Average compaction savings
- ✅ **102 tests** - All passing (76 functional + 26 stress/coverage tests)

## Features

- **Sequential file format** - All writes are append-only for simplicity and reliability
- **Binary encoding** - Efficient storage with variable-length data size fields
- **In-memory cache** - Automatic caching of all keys for O(1) read performance
- **Free space reuse** - Automatically reuses space from deleted records, reducing file bloat
- **Thread-safe** - All operations are protected with mutex locks for safe concurrent access within a single process
- **Production-ready** - Stress tested with 10,000+ records and concurrent operations
- **Backup/Restore** - JSON-based backups with smart encoding (text/base64) for portability
- **Cross-platform** - Works on Linux, macOS, BSD, and Windows
- **String convenience functions** - Direct string operations without byte conversion
- **Batch operations** - Efficiently insert or retrieve multiple keys at once
- **Iterator support** - ForEach for processing all key-value pairs
- **Soft deletes** - Deleted records are marked with a flag (bit 7) preserving original type
- **Last-write-wins** - When a key is updated, the new value is appended; Get returns the last active occurrence
- **Compact operation** - Remove deleted records and duplicate keys to reduce file size
- **Type safety** - Automatic selection of data size field (1, 2, 4, or 8 bytes) based on value length

## File Format (.skv)

### File Header

Every SKV file starts with a 6-byte header:

| Field | Size | Description |
|-------|------|-------------|
| Magic | 3 bytes | Always "SKV" (0x53 0x4B 0x56) to identify the file format |
| Version | 3 bytes | Version number: Major.Minor.Patch (e.g., 0.1.0) |

**Current version:** 0.1.0

### Record Format

After the header, records are stored sequentially with the following binary structure:

| Field | Size | Description |
|-------|------|-------------|
| Type | 1 byte | 0x01=1-byte size, 0x02=2-byte size, 0x04=4-byte size, 0x08=8-byte size<br>Bit 7 set (0x80) indicates deleted record |
| Key Size | 1 byte | Length of the key (max 255 bytes) |
| Key | [key_size] bytes | Key data |
| Data Size | 1/2/4/8 bytes | Length of the data (according to Type field) |
| Data | [data_size] bytes | Value data |

**Note on free space reuse**: When records are deleted or updated, the library tracks free space locations. 
New records will automatically reuse these spaces if they fit, improving storage efficiency. 
Padding bytes (0x80) may be added to fill small gaps that cannot hold a complete record.

### Type Field Details

- `0x01`: Data size stored in 1 byte (max 255 bytes)
- `0x02`: Data size stored in 2 bytes (max 65,535 bytes / 64 KB)
- `0x04`: Data size stored in 4 bytes (max 4,294,967,295 bytes / 4 GB)
- `0x08`: Data size stored in 8 bytes (max 18 exabytes)
- `0x81`, `0x82`, `0x84`, `0x88`: Same as above but with deleted flag (bit 7) set

## Installation

```bash
go get github.com/jncss/skv
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/jncss/skv"
)

func main() {
    // Open or create a database
    db, err := skv.Open("mydata.skv")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Using string functions for convenience
    db.PutString("username", "alice")
    db.PutString("email", "alice@example.com")
    
    // Update existing key
    db.UpdateString("username", "alice_smith")

    // Get with default value
    theme := db.GetOrDefaultString("theme", "dark")
    fmt.Printf("Theme: %s\n", theme)

    // Check if key exists
    if db.HasString("username") {
        name, _ := db.GetString("username")
        fmt.Printf("Username: %s\n", name)
    }

    // Batch operations
    users := map[string]string{
        "user1": "alice",
        "user2": "bob",
        "user3": "charlie",
    }
    db.PutBatchString(users)

    // Iterate over all keys
    db.ForEachString(func(key string, value string) error {
        fmt.Printf("%s: %s\n", key, value)
        return nil
    })

    // Get statistics
    fmt.Printf("Total keys: %d\n", db.Count())

    // Verify and compact if needed
    stats, _ := db.Verify()
    if stats.DeletedRecords > 100 {
        db.Compact()
    }
}
```

**More examples:** See the [examples/](examples/) directory for detailed examples covering all features including backup/restore, concurrent operations, and real-world use cases.

## API Reference

### `Open(name string) (*SKV, error)`
Opens or creates a .skv file. Automatically adds `.skv` extension if not present.

**Example:**
```go
db, err := skv.Open("mydata")  // Creates/opens mydata.skv
```

### `Close() error`
Closes the database file without compaction.

**Example:**
```go
defer db.Close()
```

### `CloseWithCompact() error`
Compacts the database (removes deleted records and old versions) before closing. This is useful to optimize file size when closing the database, especially for long-running applications that accumulate many updates and deletes.

**Example:**
```go
// Optimize file size on close
if err := db.CloseWithCompact(); err != nil {
    log.Fatal(err)
}
```

**Note:** Use `Close()` for faster shutdown, or `CloseWithCompact()` to optimize file size at the cost of additional processing time during shutdown.

### `Put(key []byte, data []byte) error`
Stores a new key-value pair. Returns `ErrKeyExists` if the key already exists. To modify an existing key, use `Update()` instead.

**Constraints:**
- Key must not be empty
- Key must be ≤ 255 bytes
- Data can be any size (up to 8 bytes size field limit)
- Key must not already exist in the database

**Example:**
```go
err := db.Put([]byte("name"), []byte("John Doe"))
if err == skv.ErrKeyExists {
    fmt.Println("Key already exists, use Update() instead")
}
```

### `Update(key []byte, data []byte) error`
Updates the value of an existing key. The old value is marked as deleted and the new value is appended to the end of the file. Returns `ErrKeyNotFound` if the key doesn't exist.

**Constraints:**
- Key must not be empty
- Key must exist in the database

**Example:**
```go
err := db.Update([]byte("name"), []byte("Jane Doe"))
if err == skv.ErrKeyNotFound {
    fmt.Println("Key not found, use Put() to create it")
}
```

### `Get(key []byte) ([]byte, error)`
Retrieves the value for a given key. Returns `ErrKeyNotFound` if the key doesn't exist or has been deleted.

**Performance:** O(1) lookup using in-memory cache.

**Example:**
```go
value, err := db.Get([]byte("name"))
if err == skv.ErrKeyNotFound {
    fmt.Println("Key not found")
}
```

### `Delete(key []byte) error`
Marks a key as deleted by setting bit 7 of the type field on the last occurrence. Returns `ErrKeyNotFound` if the key doesn't exist. The key is also removed from the in-memory cache.

**Performance:** O(1) cache lookup to locate the key.

**Example:**
```go
err := db.Delete([]byte("name"))
```

### `Keys() ([][]byte, error)`
Returns a list of all active keys in the database. Deleted keys and old versions of updated keys are excluded.

**Performance:** O(1) - returns keys directly from the in-memory cache.

**Example:**
```go
keys, err := db.Keys()
for _, key := range keys {
    fmt.Printf("Key: %s\n", key)
}
```

### `Verify() (*Stats, error)`
Verifies the integrity of the database file and returns detailed statistics about storage usage and efficiency.

**Stats structure:**
```go
type Stats struct {
    TotalRecords    int     // Total records in file
    ActiveRecords   int     // Non-deleted records
    DeletedRecords  int     // Deleted records
    FileSize        int64   // Total file size in bytes
    HeaderSize      int64   // Size of file header (6 bytes)
    DataSize        int64   // Size of all records (active + deleted)
    WastedSpace     int64   // Space occupied by deleted records
    PaddingBytes    int64   // Space occupied by padding bytes
    WastedPercent   float64 // Percentage of wasted space
    Efficiency      float64 // Percentage of space used by active records
    AverageKeySize  float64 // Average key size in bytes
    AverageDataSize float64 // Average data value size in bytes
}
```

**Example:**
```go
stats, err := db.Verify()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Database Statistics:\n")
fmt.Printf("  Total records: %d (%d active, %d deleted)\n", 
    stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)
fmt.Printf("  File size: %d bytes\n", stats.FileSize)
fmt.Printf("  Wasted space: %d bytes (%.2f%%)\n", 
    stats.WastedSpace + stats.PaddingBytes, stats.WastedPercent)
fmt.Printf("  Efficiency: %.2f%%\n", stats.Efficiency)
fmt.Printf("  Average key size: %.1f bytes\n", stats.AverageKeySize)
fmt.Printf("  Average value size: %.1f bytes\n", stats.AverageDataSize)

// Consider compacting if wasted space is high
if stats.WastedPercent > 30.0 {
    fmt.Println("  Recommendation: Run Compact() to reclaim space")
}
```

### `Compact() error`
Creates a new file containing only the last active occurrence of each key, then replaces the original file. This removes all deleted records and old versions of updated keys. The in-memory cache is automatically rebuilt after compaction.

**Example:**
```go
// Before: 100 total records (60 active, 40 deleted)
err := db.Compact()
// After: 60 total records (60 active, 0 deleted)
```

### `Exists(key []byte) bool`
Checks if a key exists in the database without retrieving its value.

**Performance:** O(1) - uses in-memory cache.

**Example:**
```go
if db.Exists([]byte("username")) {
    fmt.Println("User exists")
}
```

### `Has(key []byte) bool`
Alias for `Exists()` with a more idiomatic name.

### `Count() int`
Returns the number of active keys in the database.

**Performance:** O(1) - returns cache size.

**Example:**
```go
count := db.Count()
fmt.Printf("Database has %d keys\n", count)
```

### `Clear() error`
Removes all keys from the database by truncating the file and clearing the cache.

**Example:**
```go
if err := db.Clear(); err != nil {
    log.Fatal(err)
}
```

### `GetOrDefault(key []byte, defaultValue []byte) []byte`
Retrieves a value, returning a default if the key doesn't exist. Never returns an error.

**Example:**
```go
value := db.GetOrDefault([]byte("theme"), []byte("dark"))
fmt.Printf("Theme: %s\n", value)
```

### `ForEach(fn func(key []byte, value []byte) error) error`
Iterates over all active keys and values in the database. If the callback function returns an error, iteration stops.

**Example:**
```go
err := db.ForEach(func(key []byte, value []byte) error {
    fmt.Printf("%s: %s\n", key, value)
    return nil
})
```

### `PutBatch(items map[string][]byte) error`
Stores multiple key-value pairs in a single operation. If any key already exists, the entire operation fails atomically.

**Example:**
```go
items := map[string][]byte{
    "user1": []byte("alice"),
    "user2": []byte("bob"),
    "user3": []byte("charlie"),
}
if err := db.PutBatch(items); err != nil {
    log.Fatal(err)
}
```

### `GetBatch(keys [][]byte) (map[string][]byte, error)`
Retrieves multiple keys at once. Missing keys are excluded from the result.

**Example:**
```go
keys := [][]byte{[]byte("user1"), []byte("user2")}
results, err := db.GetBatch(keys)
for key, value := range results {
    fmt.Printf("%s: %s\n", key, value)
}
```

### String Convenience Functions

For easier string handling, the library provides string versions of all operations:

- `PutString(key string, value string) error`
- `UpdateString(key string, value string) error`
- `GetString(key string) (string, error)`
- `DeleteString(key string) error`
- `KeysString() ([]string, error)`
- `ExistsString(key string) bool` / `HasString(key string) bool`
- `GetOrDefaultString(key string, defaultValue string) string`
- `ForEachString(fn func(key string, value string) error) error`
- `PutBatchString(items map[string]string) error`
- `GetBatchString(keys []string) (map[string]string, error)`

**Example:**
```go
db.PutString("username", "alice")
name, _ := db.GetString("username")
db.UpdateString("username", "alice_smith")
```

### Backup and Restore

The library provides JSON-based backup and restore functionality for data portability and disaster recovery.

#### `Backup(filename string) error`
Creates a JSON backup of all key-value pairs in the database. The backup format automatically chooses the most appropriate encoding for each value:

- **String format**: Values ≤ 256 bytes that are valid UTF-8 text
- **Base64 format**: Values > 256 bytes OR binary data

**Backup JSON Structure:**
```json
[
  {
    "key": "username",
    "value": "alice",
    "is_binary": false
  },
  {
    "key": "avatar",
    "value_b64": "iVBORw0KGgoAAAANS...",
    "is_binary": true
  }
]
```

**Example:**
```go
// Create a backup
if err := db.Backup("backup.json"); err != nil {
    log.Fatal(err)
}
```

#### `Restore(filename string) error`
Loads key-value pairs from a JSON backup file. The restore operation:

- **Overwrites** existing keys with values from the backup
- **Preserves** keys not present in the backup
- **Does not clear** the database before restoring

**Example:**
```go
// Restore from backup
if err := db.Restore("backup.json"); err != nil {
    log.Fatal(err)
}
```

**Use Cases:**
- **Migration**: Transfer data between different SKV databases
- **Disaster recovery**: Restore database from a known good state
- **Inspection**: Human-readable format for debugging
- **Versioning**: JSON format is diff-friendly for version control
- **Partial updates**: Restore only specific keys from backup

**Example Workflow:**
```go
// 1. Create backup before risky operation
db.Backup("before_migration.json")

// 2. Perform migration
// ... risky operations ...

// 3. If something goes wrong, restore
if err != nil {
    db.Restore("before_migration.json")
}
```

## Error Handling

The library defines the following errors:

- `ErrKeyNotFound`: Returned when a key is not found in the database
- `ErrKeyExists`: Returned when trying to insert a key that already exists

## Behavior Details

### Inserts vs Updates
- **`Put()`** only creates new keys. If the key already exists, it returns `ErrKeyExists`.

- **`Update()`** only modifies existing keys. If the key doesn't exist, it returns `ErrKeyNotFound`.
- This design prevents accidental overwrites and makes the intent explicit.

### Updates
When you update a key with `Update()`, the old value is marked as deleted and the new value is appended to the end of the file. The `Get` operation scans the file and returns the last active occurrence.

To reclaim space from old versions, call `Compact()`.

### Deletes
When you delete a key with `Delete`, the record is **not** removed from the file. Instead, bit 7 of the type field is set to mark it as deleted. The original type information (bits 0-6) is preserved.

To permanently remove deleted records, call `Compact()`.

### In-Memory Cache
The library maintains an in-memory cache of all active keys for optimal read performance:

- **Cache building:** Automatically built when opening the database (skips reading data values for efficiency)
- **Cache updates:** Automatically maintained on all write operations (Put, Update, Delete)
- **Cache rebuild:** Automatically rebuilt after `Compact()` operations (skips reading data values for efficiency)
- **Memory usage:** Each cached key stores only its file position (8 bytes per key), not the data value

**Benefits:**
- `Get()` operations are O(1) instead of O(n)
- `Delete()` operations are O(1) for key lookups
- `Keys()` operations are O(1) instead of O(n)
- Low memory overhead: only key strings and positions are cached, not the actual data values

**Trade-off:** All active keys are kept in memory. Memory usage is approximately: `(average_key_size + 8) * number_of_keys`. For example, with 1 million keys of average 20 bytes each, the cache would use approximately 28 MB of RAM.

## Thread Safety

The library provides thread-safe access for concurrent operations within a single process:

### Goroutine-level (within a single process)
All public methods are thread-safe and can be safely called from multiple goroutines concurrently:

- **Mutex protection**: Read and write operations are protected with `sync.RWMutex`
- **Safe concurrent access**: Multiple goroutines can safely perform operations on the same SKV instance
- **No external locking needed**: The library handles all synchronization internally

**Concurrency characteristics:**
- `Keys()` uses read lock (RLock) - allows concurrent reads of the cache
- `Get()`, `Put()`, `Update()`, `Delete()`, `Compact()`, `Verify()` use exclusive lock - serialized for safety
- File operations (seek/read/write) are protected to prevent race conditions

**Testing:** All operations have been tested with Go's race detector (`go test -race`) to ensure thread safety.

**Note:** This library is designed for single-process use. Multiple processes accessing the same database file simultaneously is not supported and may result in data corruption.

## Testing

Run the test suite:

```bash
# Run all tests
go test -v

# With race detector
go test -v -race

# Run stress tests only
go test -v -run TestStress

# Run specific stress test
go test -v -run TestStress10000Records -timeout 10m
```

### Test Coverage

The library includes comprehensive tests covering:

**Basic operations:**
- File opening, Put, Update, Get, Delete
- String functions: All string convenience methods
- Extended operations: Exists/Has, Count, Clear, GetOrDefault
- Batch operations: PutBatch, GetBatch (both bytes and strings)
- Iterator: ForEach and ForEachString
- Data types: Different size fields (1-byte, 2-byte, 4-byte, 8-byte)
- Cache: Performance tests, rebuild after compaction

**Concurrency tests:**
- Concurrent reads from multiple goroutines
- Concurrent writes from multiple goroutines
- Mixed concurrent operations (read/write/update/delete)
- Concurrent compaction
- All verified with race detector

**Stress tests:**
- **TestStress10000Records**: Intensive test with 10,000 records
  - Insert: ~750 records/sec
  - Read: ~270,000 reads/sec (cached)
  - Update: ~365 updates/sec
  - Mixed operations: ~1,000 ops/sec
  - Compaction: ~37% file size reduction
  
- **TestStressConcurrent**: 10 goroutines × 1,000 operations each
  - Throughput: ~1,700-1,900 ops/sec
  - Zero race conditions detected
  
- **TestStressLargeValues**: Values from 1KB to 1MB
  - 1,000 records processed successfully
  - Verified integrity of large data
  
- **TestStressReopenAndRecover**: Database persistence
  - 5 cycles of open/close/reopen
  - 5,000 records persisted correctly
  - Cache rebuilt successfully each time

**Total test count:** 102 tests
- 76 functional tests (basic operations, advanced features, integrity, lifecycle)
- 15 stress tests (large datasets, concurrent operations, large values)
- 11 error/coverage tests (error handling, edge cases)

**Test coverage:** 81.8% of statements

**Production readiness verified:**
- ✅ Stable performance with thousands of records
- ✅ Thread-safe concurrent operations (race detector clean)
- ✅ Data integrity maintained across complex operations
- ✅ Successful recovery after close/reopen cycles
- ✅ Effective compaction (30-40% size reduction)
- ✅ Free space reuse working correctly
- ✅ Support for large values (tested up to 1MB+)

## Performance Considerations

- **Sequential writes** are very fast (append-only, ~750 inserts/sec tested with 10K records)
- **Reads** are extremely fast thanks to in-memory cache (~270,000 reads/sec)
- **Updates** are efficient (~365 updates/sec) with automatic space reuse
- **Deletes** are O(1) for key lookups (cache) + O(1) for marking deleted
- **Keys listing** is O(1) using the cache
- **Concurrent operations**: ~1,700-1,900 ops/sec with 10 goroutines
- **Memory usage:** Only key strings and file positions are cached (approximately 8 bytes overhead per key)

### Benchmark Results (from stress tests)

```
Operation Type           Throughput      Notes
─────────────────────────────────────────────────────────────
Sequential Insert        750 ops/sec     10,000 records
Cached Reads            270,000 ops/sec  From memory cache
Updates                  365 ops/sec     With space reuse
Mixed Operations       1,000 ops/sec     50% read, 25% update, 25% other
Concurrent (10 threads) 1,900 ops/sec    Race detector clean
Compaction              10 seconds       10K records, 37% reduction
```

This library is best suited for:
- Small to large datasets where all keys can fit in memory (tested with 10,000+ keys)
- Read-heavy workloads (thanks to O(1) cache lookups with 270K+ reads/sec)
- Write-heavy workloads (append-only is very fast, tested at 750 inserts/sec)
- Concurrent applications (thread-safe, tested with 10 concurrent goroutines)
- Scenarios where simplicity and reliability are important
- Applications that can periodically compact the database during low-traffic periods
- Use cases with large data values (values tested up to 1MB, not cached in memory)

**Cache benefits:** The in-memory cache dramatically improves read performance compared to sequential file scanning. For databases with thousands or millions of keys, Get/Delete/Keys operations are instant. The cache stores only positions, not data values, making it memory-efficient even for databases with very large values.

**Free space reuse:** When records are deleted or updated, the library automatically tracks and reuses free space, reducing file bloat. Tested with thousands of delete/update cycles, showing effective space management and ~37% file size reduction after compaction.

## License

MIT License
