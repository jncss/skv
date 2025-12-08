# SKV - Solid Key Value

A Go library for storing key/value data in a sequential binary file format.

[![Production Ready](https://img.shields.io/badge/production-ready-green.svg)](https://github.com/jncss/skv)
[![Test Coverage](https://img.shields.io/badge/coverage-80.8%25-green.svg)](https://github.com/jncss/skv)
[![Tests Passing](https://img.shields.io/badge/tests-228%20passing-brightgreen.svg)](https://github.com/jncss/skv)
[![Go Version](https://img.shields.io/badge/go-1.24.0+-blue.svg)](https://golang.org/dl/)

**Performance Metrics:**
- ⚡ **199 writes/sec** - With WAL enabled (crash-safe, default)
- 🔒 **734 writes/sec** - Without WAL (3.7x faster, no durability guarantee)
- 🚀 **732,000 reads/sec** - In-memory cached lookups (4.9 μs/op)
- 🔄 **154 updates/sec** - With WAL and automatic space reuse
- 🔄 **374 updates/sec** - Without WAL (2.4x faster)
- 🗑️ **109 deletes/sec** - With WAL enabled
- 🗑️ **746 deletes/sec** - Without WAL (6.8x faster)
- 📊 **WAL overhead** - ~73% slower writes, 0% impact on reads
- 🧵 **1,900 ops/sec** - Concurrent operations (10 goroutines, race-free)
- 📦 **37% reduction** - Average compaction savings
- ✅ **228 tests** - All passing (comprehensive coverage including streaming, safety, context, indexes, cursors, WAL, rollback protection, string convenience functions, and atomic transactions)
- 💾 **O(1) memory** - Streaming operations with constant memory usage regardless of file size

## Features

- **Atomic Transactions** - ACID guarantees with Begin/Commit/Rollback for multi-operation atomicity (all-or-nothing)
- **Write-Ahead Log (WAL)** - Crash recovery with automatic operation replay for guaranteed durability and transaction recovery
- **Structured Logging** - Built-in logging support with JSONLogger, TextLogger, and custom loggers for observability
- **Compression** - Transparent LZ4/Snappy compression for reduced storage (optional, configurable per database)
- **Sequential file format** - All writes are append-only for simplicity and reliability
- **Binary encoding** - Efficient storage with variable-length data size fields
- **CRC integrity checking** - Every record includes CRC-16 or CRC-32 checksum for corruption detection
- **In-memory cache** - Automatic caching of all keys for O(1) read performance
- **Free space reuse** - Automatically reuses space from deleted records, reducing file bloat
- **Thread-safe** - All operations are protected with mutex locks for safe concurrent access within a single process
- **Production-ready** - Stress tested with 10,000+ records and concurrent operations
- **Backup/Restore** - JSON-based backups with smart encoding (text/base64) for portability
- **Streaming operations** - Memory-efficient PutStream/GetStream for large values with incremental CRC calculation and rollback protection
- **Context support** - Context-aware operations (PutCtx, GetCtx, UpdateCtx, DeleteCtx, CompactCtx) for timeouts and cancellation
- **Secondary indexes** - Fast lookups by alternative keys with automatic index maintenance
- **Cursors** - Ordered iteration with range queries, prefix matching, and bidirectional traversal for both primary and secondary keys
- **Cross-platform** - Works on Linux, macOS, BSD, and Windows
- **String convenience functions** - Direct string operations without byte conversion
- **Batch operations** - Efficiently insert or retrieve multiple keys at once
- **Iterator support** - ForEach for processing all key-value pairs
- **File operations** - Direct file storage/retrieval with PutFile/GetFile/UpdateFile
- **Command-line tool** - Full-featured CLI with 23 commands for database management
- **Data recovery** - CLI recover command to salvage valid records from corrupted databases
- **Soft deletes** - Deleted records are marked with a flag (bit 7) preserving original type
- **Last-write-wins** - When a key is updated, the new value is appended; Get returns the last active occurrence
- **Compact operation** - Remove deleted records and duplicate keys to reduce file size
- **Atomic compaction** - Safe compaction using temporary files to prevent corruption
- **Type safety** - Automatic selection of data size field (1, 2, 4, or 8 bytes) based on value length

## File Format (.skv)

### File Header

Every SKV file starts with a 6-byte header:

| Field | Size | Description |
|-------|------|-------------|
| Magic | 3 bytes | Always "SKV" (0x53 0x4B 0x56) to identify the file format |
| Version | 3 bytes | Version number: Major.Minor.Patch (e.g., 0.5.0) |

**Current version:** 0.5.0

### Record Format

After the header, records are stored sequentially with the following binary structure:

| Field | Size | Description |
|-------|------|-------------|
| Type | 1 byte | 0x01=1-byte size, 0x02=2-byte size, 0x04=4-byte size, 0x08=8-byte size<br>Bit 7 set (0x80) indicates deleted record |
| Key Size | 1 byte | Length of the key (max 255 bytes) |
| Key | [key_size] bytes | Key data |
| Data Size | 1/2/4/8 bytes | Length of the data (according to Type field) |
| Data | [data_size] bytes | Value data |
| CRC | 2 or 4 bytes | CRC-16 for Type 0x01, CRC-32 for other types<br>Covers entire record: type + key_size + key + data_size + data |

**Data Integrity**: Each record includes a CRC checksum to detect corruption:
- **CRC-16-CCITT** (2 bytes): Used for Type 0x01 records (data ≤ 255 bytes)
  - Polynomial: 0x1021, Initial value: 0xFFFF
- **CRC-32-IEEE** (4 bytes): Used for Types 0x02, 0x04, 0x08 (larger data)
  - Standard IEEE polynomial (same as PNG, Ethernet, ZIP)

The CRC is calculated over the entire record (type + key_size + key + data_size + data) and stored 
in little-endian format. On read, if the CRC doesn't match, an error is returned with details about 
the mismatch to help identify corruption.

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

### Command Line Tool

SKV includes a full-featured CLI tool for interacting with SKV databases from the command line.

**Install the CLI:**

```bash
# Install globally
go install github.com/jncss/skv/tools/cli@latest

# Or build from source
cd tools/cli
go build -o skv .
```

**Quick CLI Examples:**

```bash
# Basic operations
skv put mydb.skv username "alice"
skv get mydb.skv username
skv update mydb.skv username "bob"
skv delete mydb.skv username

# Hexdump mode for binary data inspection
skv --hex get mydb.skv username     # View as hexdump
skv -x keys mydb.skv                # List keys in hex format

# File operations
skv putfile mydb.skv config config.ini
skv getfile mydb.skv config retrieved.ini

# Streaming (memory-efficient for large files)
skv putstream mydb.skv video intro.mp4
skv getstream mydb.skv video output.mp4

# Batch operations
skv putbatch mydb.skv key1 val1 key2 val2 key3 val3
skv getbatch mydb.skv key1 key2 key3

# Database management
skv verify mydb.skv       # Check stats and health
skv compact mydb.skv      # Optimize file size
skv backup mydb.skv backup.json
skv restore mydb.skv backup.json

# List and iterate
skv keys mydb.skv         # List all keys
skv count mydb.skv        # Count active keys
skv foreach mydb.skv      # Show all key=value pairs
```

**Available CLI Commands (23 total):**
- **Basic**: `put`, `get`, `update`, `delete`, `exists`, `count`, `keys`, `clear`, `foreach`
- **Files**: `putfile`, `getfile`, `updatefile`
- **Streaming**: `putstream`, `getstream`, `updatestream` (memory-efficient for large files)
- **Batch**: `putbatch`, `getbatch`
- **Maintenance**: `backup`, `restore`, `verify`, `compact`
- **Help**: `help`

See [tools/cli/README.md](tools/cli/README.md) for complete CLI documentation with examples and use cases.

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

    // Atomic transactions
    tx := db.Begin()
    tx.PutString("account:alice", "1000.00")
    tx.PutString("account:bob", "500.00")
    if err := tx.Commit(); err != nil {
        log.Fatal(err) // All operations or none
    }

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

### Context-Aware Operations

All primary operations support context for timeout and cancellation control. This is essential for production applications that need fine-grained control over long-running operations.

#### `PutCtx(ctx context.Context, key []byte, data []byte) error`
Stores a new key-value pair with context support. Returns `ctx.Err()` if the context is cancelled or times out.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := db.PutCtx(ctx, []byte("key"), []byte("value"))
if err == context.DeadlineExceeded {
    fmt.Println("Operation timed out")
} else if err == context.Canceled {
    fmt.Println("Operation was cancelled")
}
```

#### `GetCtx(ctx context.Context, key []byte) ([]byte, error)`
Retrieves a value with context support.

**Example:**
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel from another goroutine if needed
go func() {
    time.Sleep(100 * time.Millisecond)
    cancel()
}()

value, err := db.GetCtx(ctx, []byte("key"))
if err == context.Canceled {
    fmt.Println("Read was cancelled")
}
```

#### `UpdateCtx(ctx context.Context, key []byte, data []byte) error`
Updates an existing key with context support.

**Example:**
```go
ctx := context.Background()
err := db.UpdateCtx(ctx, []byte("key"), []byte("new_value"))
```

#### `DeleteCtx(ctx context.Context, key []byte) error`
Deletes a key with context support.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

err := db.DeleteCtx(ctx, []byte("key"))
```

#### `CompactCtx(ctx context.Context) error`
Compacts the database with context support. This is particularly useful for large databases where compaction might take significant time and you want the ability to cancel the operation.

The context is checked periodically during compaction, allowing for responsive cancellation even when processing many records.

**Example:**
```go
// Compact with a timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := db.CompactCtx(ctx)
if err == context.DeadlineExceeded {
    fmt.Println("Compaction took too long, cancelled")
    // Original database file remains intact
}
```

**Context checking points:**
- Before acquiring the lock
- After acquiring the lock
- Periodically during record processing (in CompactCtx)

**Use cases:**
- **HTTP servers**: Cancel operations when client disconnects
- **Batch jobs**: Respect shutdown signals and timeouts
- **Microservices**: Propagate cancellation across service boundaries
- **Resource management**: Prevent runaway operations

**Note:** The original functions (`Put`, `Get`, `Update`, `Delete`, `Compact`) remain available and internally call the context versions with `context.Background()` for backward compatibility.

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
Creates a new file containing only the last active occurrence of each key, then atomically replaces the original file. This removes all deleted records and old versions of updated keys. The in-memory cache is automatically rebuilt after compaction.

**Safety:** Uses atomic file operations with a temporary file to ensure data integrity:
1. Writes compacted data to a temporary file
2. Syncs the temporary file to disk
3. Closes the original file
4. Atomically renames the temporary file over the original (OS-level atomic operation)
5. Reopens the database file

This approach ensures that if compaction fails at any point (power loss, disk full, etc.), the original database file remains intact and uncorrupted.

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

**Basic Operations:**
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

**Cursor Operations:**
- `NewCursorString(from, to string, reverse bool) *Cursor`
- `PrefixCursorString(prefix string, reverse bool) *Cursor`
- `AllCursorString(reverse bool) *Cursor`
- `NewIndexCursorString(indexName, from, to string, reverse bool) (*Cursor, error)`
- `PrefixIndexCursorString(indexName, prefix string, reverse bool) (*Cursor, error)`
- `AllIndexCursorString(indexName string, reverse bool) (*Cursor, error)`
- `cursor.NextString() (string, string, error)`
- `cursor.KeyString() string`
- `cursor.ValueString() (string, error)`
- `cursor.SeekString(key string) error`
- `cursor.HasPrefixString(prefix string) bool`
- `cursor.KeysString() []string`
- `cursor.CollectString() ([]string, []string, error)`

**Index Operations:**
- `GetByIndexString(indexName, secondaryKey string) ([]byte, error)`
- `GetAllByIndexString(indexName, secondaryKey string) ([][]byte, error)`
- `HasIndexString(indexName, secondaryKey string) bool`

**Example:**
```go
db.PutString("username", "alice")
name, _ := db.GetString("username")
db.UpdateString("username", "alice_smith")

// Cursor with strings
cursor := db.PrefixCursorString("user:", false)
for {
    key, value, err := cursor.NextString()
    if err == io.EOF {
        break
    }
    fmt.Printf("%s = %s\n", key, value)
}
```

### File Operations

Store and retrieve files directly from the database:

- `PutFile(key string, filePath string) error` - Store a file from disk
- `GetFile(key string, filePath string) error` - Retrieve to a file on disk
- `UpdateFile(key string, filePath string) error` - Update with file contents
- `PutStream(key []byte, reader io.Reader, size int64) error` - Store value from a reader (memory-efficient, with rollback protection)
- `PutStreamString(key string, reader io.Reader, size int64) error` - Store using string key
- `UpdateStream(key []byte, reader io.Reader, size int64) error` - Update value from a reader (with rollback protection)
- `UpdateStreamString(key string, reader io.Reader, size int64) error` - Update using string key
- `GetStream(key []byte, writer io.Writer) (int64, error)` - Stream value to a writer (memory-efficient)
- `GetStreamString(key string, writer io.Writer) (int64, error)` - Stream value using string key

**Memory Efficiency:**

All streaming operations (PutStream, GetStream, UpdateStream) and internal operations (cache loading, deletion, compaction) use incremental CRC calculation with constant memory buffers (64KB), making them safe for files of any size:

- **Small files** (≤255 bytes): Negligible difference between regular Put/Get and streaming
- **Large files** (>1MB): Significant memory savings - uses only 64KB regardless of file size
- **Very large files** (>1GB): Essential to use streaming - regular Put/Get would require loading entire file into memory

**Rollback Protection:**

PutStream and UpdateStream include checkpoint-based rollback protection to ensure database integrity:

- **Atomic writes**: Either the entire key-value is written or nothing (no partial records)
- **Checkpoint mechanism**: File position saved before write, truncated on failure
- **Original value preserved**: UpdateStream keeps the old value if the new write fails
- **No WAL overhead**: Unlike regular Put/Update, streams use truncate instead of WAL to avoid memory buffering
- **Consistent state**: Database remains in clean state after failed operations, ready for retry
- **Logged events**: All rollback operations are logged (Warn on successful rollback, Error on rollback failure) for observability

The library automatically handles CRC verification incrementally during all operations:
- **Writing**: `PutStream`, `UpdateStream`, and `writeRecordAtPosition` calculate CRC while streaming data
- **Reading with data**: `GetStream` verifies CRC incrementally while streaming to output
- **Reading metadata only**: Cache loading and delete operations use streaming CRC verification without loading data into memory
- **Compaction**: Processes records one-at-a-time with streaming CRC, never loading all data in memory

This ensures **O(1) constant memory usage** for databases of any size, ensuring data integrity without memory overhead.

**Example:**
```go
// Store a configuration file
db.PutFile("config:app", "config.ini")

// Retrieve it later
db.GetFile("config:app", "retrieved_config.ini")

// Update with new contents
db.UpdateFile("config:app", "new_config.ini")

// Stream large values efficiently (writing)
file, _ := os.Open("large_video.mp4")
defer file.Close()
info, _ := file.Stat()
db.PutStreamString("video:intro", file, info.Size())

// Stream large values efficiently (reading)
output, _ := os.Create("output.mp4")
defer output.Close()
n, _ := db.GetStreamString("video:intro", output)
fmt.Printf("Streamed %d bytes\n", n)

// Round-trip streaming (no memory load)
reader := getDataReader() // some io.Reader
db.PutStream([]byte("backup:data"), reader, dataSize)
var buf bytes.Buffer
db.GetStream([]byte("backup:data"), &buf)
```

**Use cases:**
- Configuration file storage
- Template management
- Asset storage (images, CSS, JS)
- Document archiving
- Binary data storage
- Large file streaming (videos, backups, logs)

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

## Cursors

SKV provides a powerful cursor system for ordered iteration and range queries.

### Basic Cursor Usage

```go
// Iterate all records in sorted order
cursor := db.AllCursor(false) // false = forward, true = reverse
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process key/value
}
```

### Range Queries

```go
// Query records from "user:100" to "user:199" (inclusive)
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("user:100"),
    To:   []byte("user:199"),
})
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process key/value in range
}
```

### Prefix Cursors

```go
// Find all keys starting with "config:"
cursor := db.PrefixCursor([]byte("config:"), false)
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process matching keys
}
```

### Index Cursors

```go
// First, create a secondary index
db.CreateIndex("by_email", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(user.Email)
})

// Iterate records ordered by email
cursor, _ := db.AllIndexCursor("by_email", false)
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Records are ordered by indexed field
}

// Or use prefix on indexed field
cursor, _ = db.PrefixIndexCursor("by_email", []byte("admin@"), false)
```

### Cursor Methods

```go
cursor := db.AllCursor(false)

// Navigation
cursor.Seek([]byte("start-here"))  // Jump to position
cursor.Reset()                     // Go back to beginning

// Utility methods
count := cursor.Count()            // Total records in cursor
keys := cursor.Keys()              // Get all keys

// Iteration patterns
cursor.ForEach(func(key, value []byte) bool {
    // Process each record
    return true // continue, false to stop
})

keys, values, _ := cursor.Collect() // Collect all into slices

// Position checks
if cursor.IsFirst() { /* ... */ }
if cursor.IsLast()  { /* ... */ }
if cursor.IsValid() { /* ... */ }
```

**For complete cursor documentation, see [CURSORS.md](CURSORS.md)**

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
  - Insert: ~750 records/sec (with WAL disabled for stress test performance)
  - Read: ~732,000 reads/sec (cached)
  - Update: ~365 updates/sec (with WAL disabled)
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

- **Writes with WAL** are crash-safe (~199 inserts/sec, ~154 updates/sec, ~109 deletes/sec)
- **Writes without WAL** are 2.4-6.8x faster (~734 inserts/sec, ~374 updates/sec, ~746 deletes/sec) but lose durability guarantee
- **Reads** are extremely fast thanks to in-memory cache (~732,000 reads/sec, 4.9 μs/op)
- **WAL overhead** is ~73% on writes, 0% on reads (3 fsyncs per operation dominate write time)
- **Deletes** are O(1) for key lookups (cache) + O(1) for marking deleted
- **Keys listing** is O(1) using the cache
- **Concurrent operations**: ~1,700-1,900 ops/sec with 10 goroutines
- **Memory usage:** Only key strings and file positions are cached (approximately 8 bytes overhead per key)

### Benchmark Results (Go benchmarks, 100-byte values)

```
Operation                Throughput      Time/op      Speedup   Notes
─────────────────────────────────────────────────────────────────────────────
Put (with WAL)              199 ops/sec     5.0 ms              Crash-safe, default
Put (without WAL)           734 ops/sec     1.4 ms    3.7x      No durability guarantee
Get (cached)            732,000 ops/sec     4.9 μs              From memory cache
Update (with WAL)           154 ops/sec     6.5 ms              Crash-safe
Update (without WAL)        374 ops/sec     2.7 ms    2.4x      No durability guarantee
Delete (with WAL)           109 ops/sec     9.2 ms              Crash-safe
Delete (without WAL)        746 ops/sec     1.3 ms    6.8x      No durability guarantee
Concurrent (10 threads)   1,900 ops/sec                         Race detector clean
Compaction                 10 seconds                            10K records, 37% reduction
```

**WAL Impact:** Write-Ahead Log provides crash recovery and durability at the cost of ~73% write throughput due to fsync overhead (3 fsyncs per operation). Reads are completely unaffected. For bulk imports, temporarily disable WAL with `db.wal.Disable()` to achieve 3-7x faster writes.

This library is best suited for:
- Applications requiring data durability and crash recovery (WAL enabled by default)
- Small to large datasets where all keys can fit in memory (tested with 10,000+ keys)
- Read-heavy workloads (thanks to O(1) cache lookups with 728K+ reads/sec)
- Write-moderate workloads where durability is valued over raw speed
- Concurrent applications (thread-safe, tested with 10 concurrent goroutines)
- Scenarios where simplicity and reliability are important
- Applications that can periodically compact the database during low-traffic periods
- Use cases with large data values (values tested up to 1MB, not cached in memory)

**Cache benefits:** The in-memory cache dramatically improves read performance compared to sequential file scanning. For databases with thousands or millions of keys, Get/Delete/Keys operations are instant. The cache stores only positions, not data values, making it memory-efficient even for databases with very large values.

**Free space reuse:** When records are deleted or updated, the library automatically tracks and reuses free space, reducing file bloat. Tested with thousands of delete/update cycles, showing effective space management and ~37% file size reduction after compaction.

## Compression

SKV supports transparent data compression to reduce storage space:

```go
// Open with LZ4 compression
db, err := skv.OpenWithOptions("mydata.skv", &skv.Options{
    Compression: skv.CompressionLZ4,
})

// All operations transparently compress/decompress
db.Put([]byte("key"), largeData) // Automatically compressed
data, _ := db.Get([]byte("key"))  // Automatically decompressed
```

**Supported algorithms:**
- **LZ4** - Best general purpose (very fast, good compression)
- **Snappy** - Maximum speed with acceptable compression
- **None** - No compression (default)

**Key features:**
- Threshold-based: Only data ≥128 bytes is considered
- Adaptive: Compression only used if it reduces size
- Mixed support: Different records can use different compression
- Transparent: Automatic compression/decompression

See [COMPRESSION.md](COMPRESSION.md) for detailed documentation including:
- Performance benchmarks
- When to use each algorithm
- Compression format details
- Best practices

## Additional Documentation

- **[WAL.md](WAL.md)** - Write-Ahead Log internals and crash recovery
- **[LOGGING.md](LOGGING.md)** - Logging support
- **[COMPRESSION.md](COMPRESSION.md)** - Compression algorithms and performance
- **[CURSORS.md](CURSORS.md)** - Cursor iteration and range queries
- **[TESTING.md](TESTING.md)** - Test coverage and methodology
- **[examples/](examples/)** - Comprehensive usage examples
- **[tools/cli/README.md](tools/cli/README.md)** - Command-line interface guide

## License

MIT License
