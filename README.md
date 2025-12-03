# SKV - Simple Key Value

A Go library for storing key/value data in a sequential binary file format.

## Features

- **Sequential file format** - All writes are append-only for simplicity and reliability
- **Binary encoding** - Efficient storage with variable-length data size fields
- **In-memory cache** - Automatic caching of all keys for O(1) read performance
- **Thread-safe** - All operations are protected with mutex locks for safe concurrent access
- **File locking** - OS-level per-operation locks: shared locks for reads (allow concurrent readers), exclusive locks for writes (serialize writes)
- **Cross-platform** - Works on Linux, macOS, BSD, and Windows
- **String convenience functions** - Direct string operations without byte conversion
- **Batch operations** - Efficiently insert or retrieve multiple keys at once
- **Iterator support** - ForEach for processing all key-value pairs
- **Soft deletes** - Deleted records are marked with a flag (bit 7) preserving original type
- **Last-write-wins** - When a key is updated, the new value is appended; Get returns the last active occurrence
- **Compact operation** - Remove deleted records and duplicate keys to reduce file size
- **Type safety** - Automatic selection of data size field (1, 2, 4, or 8 bytes) based on value length

## File Format (.skv)

Each record is stored sequentially with the following binary structure:

| Field | Size | Description |
|-------|------|-------------|
| Type | 1 byte | 0x01=1-byte size, 0x02=2-byte size, 0x04=4-byte size, 0x08=8-byte size<br>Bit 7 set (0x80) indicates deleted record |
| Key Size | 1 byte | Length of the key (max 255 bytes) |
| Key | [key_size] bytes | Key data |
| Data Size | 1/2/4/8 bytes | Length of the data (according to Type field) |
| Data | [data_size] bytes | Value data |

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
Verifies the integrity of the database file and returns statistics.

**Stats structure:**
```go
type Stats struct {
    TotalRecords   int  // Total records in file
    ActiveRecords  int  // Non-deleted records
    DeletedRecords int  // Deleted records
}
```

**Example:**
```go
stats, err := db.Verify()
fmt.Printf("Database has %d active records\n", stats.ActiveRecords)
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

The library provides two levels of concurrency protection:

### Goroutine-level (within a single process)
All public methods are thread-safe and can be safely called from multiple goroutines concurrently:

- **Mutex protection**: Read and write operations are protected with `sync.RWMutex`
- **Safe concurrent access**: Multiple goroutines can safely perform operations on the same SKV instance
- **No external locking needed**: The library handles all synchronization internally

**Concurrency characteristics:**
- `Keys()` uses read lock (RLock) - allows concurrent reads of the cache
- `Get()`, `Put()`, `Update()`, `Delete()`, `Compact()`, `Verify()` use exclusive lock - serialized for safety
- File operations (seek/read/write) are protected to prevent race conditions

### Process-level (multiple processes)
File-level locks coordinate access between different processes:

- **Shared locks (LOCK_SH)**: Multiple processes can read concurrently (`Get()`, `Verify()`)
- **Exclusive locks (LOCK_EX)**: Only one process can write at a time (`Put()`, `Update()`, `Delete()`, `Compact()`)
- **Automatic coordination**: Locks acquired per-operation, released automatically

**Testing:** All operations have been tested with Go's race detector (`go test -race`) to ensure thread safety.

## File Locking

The library uses OS-level file locking to coordinate access between multiple processes:

- **Per-operation locking**: Locks are acquired at the start of each operation and released when complete
- **Shared locks (LOCK_SH)**: Used for read operations (`Get()`, `Verify()`) - multiple processes can read simultaneously
- **Exclusive locks (LOCK_EX)**: Used for write operations (`Put()`, `Update()`, `Delete()`, `Compact()`) - only one process can write at a time
- **Process coordination**: Prevents data corruption while allowing concurrent reads from multiple processes
- **Automatic management**: All locking is handled internally - no manual lock management needed

**Behavior:**
- Multiple processes can open the same database file simultaneously
- Read operations can proceed concurrently from different processes
- Write operations are serialized - if one process is writing, others wait
- Locks are held only during the operation, not across the entire database lifetime
- If a process crashes during an operation, the OS automatically releases the lock

**Platform support:** File locking uses `github.com/gofrs/flock` which provides cross-platform support for Unix-like systems (Linux, macOS, BSD) and Windows.

## Testing

Run the test suite:

```bash
go test -v

# With race detector
go test -v -race
```

All 57 tests cover:
- **Basic operations**: File opening, Put, Update, Get, Delete
- **String functions**: All string convenience methods
- **Extended operations**: Exists/Has, Count, Clear, GetOrDefault
- **Batch operations**: PutBatch, GetBatch (both bytes and strings)
- **Iterator**: ForEach and ForEachString
- **Data types**: Different size fields (1-byte, 2-byte, 4-byte)
- **Cache**: Performance tests, rebuild after compaction
- **Concurrency**: Reads, writes, mixed operations, concurrent compact
- **File locking**: Per-operation locks, concurrent access from multiple processes
- **Integrity**: Verify functionality, compact operations
- **Edge cases**: Duplicate key prevention, missing keys, delete and re-add
- **Concurrent read/write** (mixed operations)
- **Concurrent compact** (compact while reading/writing)
- **Concurrent Keys()** calls
- **File locking** (per-operation locks, concurrent access from multiple processes)
- **Lock release** (automatic on operation completion)
- **Concurrent process access** (multiple processes reading simultaneously)
- **Crash simulation** (lock release on process termination)

## Performance Considerations

- **Sequential writes** are very fast (append-only)
- **Reads** are O(1) thanks to the in-memory cache
- **Updates** create duplicate keys until `Compact()` is called
- **Deletes** are O(1) for key lookups (cache) + O(1) for marking deleted
- **Keys listing** is O(1) using the cache
- **Memory usage:** Only key strings and file positions are cached (approximately 8 bytes overhead per key)

This library is best suited for:
- Small to large datasets where all keys can fit in memory
- Read-heavy workloads (thanks to O(1) cache lookups)
- Write-heavy workloads (append-only is very fast)
- Scenarios where simplicity and reliability are important
- Applications that can periodically compact the database during low-traffic periods
- Use cases with large data values (since values are not cached, only their positions)

**Cache benefits:** The in-memory cache dramatically improves read performance compared to sequential file scanning. For databases with thousands or millions of keys, Get/Delete/Keys operations are instant. The cache stores only positions, not data values, making it memory-efficient even for databases with very large values.

## License

MIT License
