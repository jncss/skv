# SKV - Simple Key Value

A Go library for storing key/value data in a sequential binary file format.

## Features

- **Sequential file format** - All writes are append-only for simplicity and reliability
- **Binary encoding** - Efficient storage with variable-length data size fields
- **In-memory cache** - Automatic caching of all keys for O(1) read performance
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
go get github.com/yourusername/skv
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/skv"
)

func main() {
    // Open or create a database
    db, err := skv.Open("mydata.skv")
    if err != nil {
        log.Fatal(err)
    }
    // Option 1: Normal close (fast)
    defer db.Close()
    // Option 2: Close with compact to optimize file size
    // defer func() { 
    //     if err := db.CloseWithCompact(); err != nil {
    //         log.Fatal(err)
    //     }
    // }()

    // Store a key-value pair
    if err := db.Put([]byte("username"), []byte("alice")); err != nil {
        log.Fatal(err)
    }

    // Update an existing key
    if err := db.Update([]byte("username"), []byte("alice_smith")); err != nil {
        log.Fatal(err)
    }

    // Retrieve a value
    value, err := db.Get([]byte("username"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("username = %s\n", value)

    // Delete a key
    if err := db.Delete([]byte("username")); err != nil {
        log.Fatal(err)
    }

    // List all keys
    keys, err := db.Keys()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Active keys: %v\n", keys)

    // Verify database integrity and get statistics
    stats, err := db.Verify()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total: %d, Active: %d, Deleted: %d\n", 
        stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)

    // Manual compact (alternative to CloseWithCompact)
    // if err := db.Compact(); err != nil {
    //     log.Fatal(err)
    // }
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

## Testing

Run the test suite:

```bash
go test -v
```

All 23 tests cover:
- File opening and creation
- Put operations (including duplicate key prevention)
- Update operations
- Get operations
- Different data types (1-byte, 2-byte, 4-byte size fields)
- Delete operations
- Update scenarios
- Verify functionality
- Compact operations
- Keys listing
- Cache performance (1000+ key operations)
- Cache rebuild after compaction

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
