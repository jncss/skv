# Write-Ahead Log (WAL)

SKV includes a built-in Write-Ahead Log (WAL) system that provides **durability guarantees** by ensuring that all operations are safely logged before being applied to the main data file. This protects against data loss in case of system crashes or unexpected shutdowns.

## Table of Contents

1. [Overview](#overview)
2. [How It Works](#how-it-works)
3. [File Format](#file-format)
4. [Usage](#usage)
5. [Recovery Process](#recovery-process)
6. [Performance Considerations](#performance-considerations)
7. [Advanced Topics](#advanced-topics)

## Overview

The Write-Ahead Log is a crash-recovery mechanism that ensures **ACID** properties for database operations:

- **Atomicity**: Operations either complete fully or not at all
- **Consistency**: The database remains in a valid state
- **Isolation**: Operations don't interfere with each other (via mutex)
- **Durability**: Committed operations survive crashes

### Key Benefits

✅ **Automatic crash recovery**: Uncommitted operations are replayed on startup  
✅ **Zero data loss**: All committed operations are guaranteed to persist  
✅ **Transparent**: Works automatically without code changes  
✅ **Minimal overhead**: Sequential writes with small entries  
✅ **Simple**: Single WAL file per database  

## How It Works

### Operation Flow

Every write operation (`Put`, `Update`, `Delete`) follows this sequence:

```
1. Log operation to WAL file (.skv.wal)
   ├─> Write entry with operation details
   └─> Sync to disk (fsync)

2. Apply operation to main file (.skv)
   ├─> Write/modify record
   └─> Update cache and indexes

3. Log commit marker
   ├─> Write commit entry
   └─> Sync to disk (fsync)

4. Truncate WAL
   ├─> Write new header
   └─> Sync to disk (fsync)
   └─> Clear WAL (operation fully committed)
```

### Crash Scenarios

| When crash occurs | What happens | Result |
|-------------------|--------------|--------|
| **Before step 1** | Nothing logged | No operation applied |
| **After step 1, before step 2** | WAL has entry, main file unchanged | Operation replayed on recovery |
| **After step 2, before step 4** | Both WAL and main file updated | WAL truncated on recovery (idempotent) |
| **After step 4** | WAL empty, main file updated | Normal state |

## File Format

### WAL File Structure

```
┌─────────────────────────────────────────┐
│ Header (6 bytes)                        │
│  ├─ Magic: "WAL" (3 bytes)              │
│  └─ Version: major.minor.patch (3 bytes)│
├─────────────────────────────────────────┤
│ Entry 1                                 │
│  ├─ OpType (1 byte)                     │
│  ├─ KeySize (2 bytes, little-endian)    │
│  ├─ Key (variable)                      │
│  ├─ DataSize (4 bytes, little-endian)   │
│  ├─ Data (variable)                     │
│  └─ CRC32 (4 bytes)                     │
├─────────────────────────────────────────┤
│ Entry 2                                 │
│  ...                                    │
├─────────────────────────────────────────┤
│ Commit Marker (optional)                │
│  └─ OpType = 0x03                       │
└─────────────────────────────────────────┘
```

### Operation Types

| OpType | Value | Description | Key | Data |
|--------|-------|-------------|-----|------|
| `WALOpPut` | `0x01` | Put/Update operation | Required | Required |
| `WALOpDelete` | `0x02` | Delete operation | Required | Empty |
| `WALOpCommit` | `0x03` | Commit marker | Empty | Empty |

### Entry Format Details

```go
type WALEntry struct {
    OpType byte   // 0x01=Put, 0x02=Delete, 0x03=Commit
    Key    []byte // Key (max 65535 bytes)
    Data   []byte // Value (max 4GB, empty for Delete)
}
```

Each entry is protected by a CRC-32 checksum to detect corruption.

## Usage

### Automatic (Default)

WAL is **enabled automatically** when you open a database:

```go
db, err := skv.Open("mydata.skv")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// All operations automatically use WAL
db.Put([]byte("key"), []byte("value"))
db.Update([]byte("key"), []byte("new value"))
db.Delete([]byte("key"))
```

The WAL file is created as `mydata.skv.wal`.

### Manual Control (Advanced)

You can access the WAL directly if needed:

```go
// Disable WAL temporarily (e.g., for bulk imports)
db.wal.Disable()

// ... bulk operations ...

// Re-enable WAL
db.wal.Enable()

// Check if WAL is enabled
if db.wal.IsEnabled() {
    fmt.Println("WAL is active")
}

// Get WAL file size
size, _ := db.wal.Size()
fmt.Printf("WAL size: %d bytes\n", size)
```

**⚠️ Warning**: Disabling WAL removes crash-recovery protection. Only disable for bulk operations where you can re-run the entire batch if needed.

## Recovery Process

### On Database Open

When you call `skv.Open()`:

1. **Check WAL existence**: Look for `<database>.skv.wal`
2. **Check WAL size**: If only header (6 bytes), skip recovery
3. **Read entries**: Parse all WAL entries
4. **Replay operations**:
   - Apply Put operations
   - Apply Delete operations
   - Stop at Commit marker (or EOF)
5. **Truncate WAL**: Clear WAL after successful recovery
6. **Build cache**: Rebuild in-memory cache from main file

### Recovery Example

```go
// Database crashed after logging but before committing
// WAL contains: [Put("user:1", data), Put("user:2", data)]

db, err := skv.Open("crashed.skv")
// During Open():
// 1. WAL is detected
// 2. Operations are replayed
// 3. user:1 and user:2 are written to main file
// 4. WAL is cleared
// 5. Database is ready for use

// You can now access the recovered data
val, _ := db.Get([]byte("user:1")) // Returns data
```

### Partial Entry Handling

If the WAL file is corrupted (e.g., incomplete entry due to crash during write):

- **Valid entries before corruption**: Replayed successfully
- **Corrupted entry**: Recovery stops (CRC mismatch detected)
- **Entries after corruption**: Ignored

This provides **best-effort recovery** up to the last valid operation.

## Performance Considerations

### Overhead

- **Write amplification**: Each operation writes to both WAL and main file (~2x writes)
- **Sync operations**: Each operation performs multiple `fsync()` calls (WAL write, WAL commit, WAL truncate, plus main file)
- **Space**: WAL file size varies (typically small, truncated after each operation)

### Optimization Strategies

#### 1. Batch Operations (Future)

For multiple operations, batching can reduce overhead:

```go
// Future feature (not yet implemented)
db.BeginBatch()
db.Put([]byte("key1"), []byte("value1"))
db.Put([]byte("key2"), []byte("value2"))
db.Put([]byte("key3"), []byte("value3"))
db.CommitBatch() // Single WAL commit for all operations
```

#### 2. Disable for Bulk Imports

For large imports where re-running is acceptable:

```go
db.wal.Disable()

for _, record := range millionsOfRecords {
    db.Put(record.Key, record.Value)
}

db.wal.Enable()
```

#### 3. Use Update Instead of Put+Delete

`Update()` uses a single WAL transaction instead of two:

```go
// Less efficient (2 WAL transactions)
db.Delete([]byte("key"))
db.Put([]byte("key"), []byte("new value"))

// More efficient (1 WAL transaction)
db.Update([]byte("key"), []byte("new value"))
```

### Typical Performance

On modern SSDs:
- **With WAL**: ~500-750 ops/sec (sequential writes)
- **Without WAL**: ~1000-1500 ops/sec (no fsync)

WAL overhead is **minimal** for most applications and provides crucial durability guarantees.

## Advanced Topics

### WAL File Location

The WAL file is always `<database>.skv.wal`:

```
myapp.skv       <- Main database file
myapp.skv.wal   <- WAL file
```

### Concurrent Access

- **Thread-safe**: WAL has internal mutex for concurrent writes
- **No locking issues**: SKV already uses mutex for all operations
- **Atomic operations**: Each `Put`/`Update`/`Delete` is atomic

### WAL vs. Main File Consistency

The WAL acts as a **source of truth** during recovery:

| Scenario | Main File | WAL | Recovery Action |
|----------|-----------|-----|-----------------|
| Normal shutdown | Updated | Empty | No recovery needed |
| Crash before commit | Old data | Has entry | Replay entry |
| Crash after commit | Updated | Empty | No action needed |
| Crash during commit | Updated | Has entry + commit | Truncate WAL (idempotent) |

### Error Handling

```go
db, err := skv.Open("mydata.skv")
if err != nil {
    // Could be:
    // - WAL recovery failed
    // - Corrupted WAL file
    // - Disk I/O error
    log.Fatalf("Failed to open database: %v", err)
}
```

If WAL recovery fails, the main database file remains unchanged.

### Maintenance

#### Checking WAL Size

```go
size, err := db.wal.Size()
if err != nil {
    log.Printf("Error getting WAL size: %v", err)
}

// WAL should typically be just the header (6 bytes) when idle
if size > 6 {
    log.Printf("Warning: WAL has uncommitted data (%d bytes)", size-6)
}
```

#### Manual Cleanup

WAL files are automatically managed. However, if you have orphaned WAL files:

```bash
# Find WAL files without corresponding .skv files
find . -name "*.wal" -type f | while read wal; do
    skv="${wal%.wal}"
    [ ! -f "$skv" ] && echo "Orphaned: $wal"
done
```

### Limitations

1. **No checkpointing**: WAL is truncated after every operation (not batched)
2. **No compression**: WAL entries are not compressed
3. **No rotation**: Single WAL file per database (no log rotation)
4. **No remote replication**: WAL is local-only

These limitations may be addressed in future versions.

## See Also

- [Examples](../examples/10-wal/README.md) - WAL usage examples
- [Testing](../wal_test.go) - WAL test suite
- [File Format](FILEFORMAT.md) - SKV file format details

## References

- [Write-Ahead Logging (Wikipedia)](https://en.wikipedia.org/wiki/Write-ahead_logging)
- [ACID Properties](https://en.wikipedia.org/wiki/ACID)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
