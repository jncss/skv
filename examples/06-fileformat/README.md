# 06 - File Format

Understand the binary file format used by SKV.

## Example

### `demo/`
Detailed inspection of the SKV binary file format:
- **File Header**: Magic bytes and version
- **Record Structure**: Type, key size, key, data size, data
- **Type Byte**: Size field selection and deleted flag
- **Deleted Records**: How deletions are marked (bit 7)
- **Padding**: Gap filling with 0x80 bytes
- **Binary Visualization**: Hex dump of actual file contents

**Run:**
```bash
cd examples/06-fileformat/demo
go run demo.go
```

## File Format Specification

### Header (6 bytes)
```
Offset  Size  Description
------  ----  -----------
0       3     Magic bytes: "SKV" (0x53 0x4B 0x56)
3       1     Version major
4       1     Version minor
5       1     Version patch
```

**Current version:** 0.1.0

### Record Structure
```
┌─────────────────────────────────────┐
│ Type (1 byte)                       │
├─────────────────────────────────────┤
│ Key Size (1 byte)                   │
├─────────────────────────────────────┤
│ Key (variable, max 255 bytes)       │
├─────────────────────────────────────┤
│ Data Size (1/2/4/8 bytes)           │
├─────────────────────────────────────┤
│ Data (variable)                     │
└─────────────────────────────────────┘
```

### Type Byte (1 byte)
```
Bit:  7 6 5 4 3 2 1 0
      │ └─────┴─────┘
      │       │
      │       └─ Size field type (0x01, 0x02, 0x04, 0x08)
      └─ Deleted flag (1 = deleted, 0 = active)
```

**Type Values:**
- `0x01`: Data size in 1 byte (max 255 bytes)
- `0x02`: Data size in 2 bytes (max 64 KB)
- `0x04`: Data size in 4 bytes (max 4 GB)
- `0x08`: Data size in 8 bytes (max 18 exabytes)

**Deleted Flag:**
- `0x81`: Deleted, was type 0x01
- `0x82`: Deleted, was type 0x02
- `0x84`: Deleted, was type 0x04
- `0x88`: Deleted, was type 0x08

### Example Records

**Small Record (type 0x01):**
```
Hex: 01 05 68 65 6C 6C 6F 05 77 6F 72 6C 64
     │  │  └─────┬─────┘  │  └─────┬─────┘
     │  │       │         │        └─ Data: "world" (5 bytes)
     │  │       │         └─ Data size: 5 (1 byte)
     │  │       └─ Key: "hello" (5 bytes)
     │  └─ Key size: 5
     └─ Type: 0x01 (1-byte size field)
```

**Deleted Record (type 0x81):**
```
Hex: 81 04 74 65 6D 70 03 6F 6C 64
     │  │  └────┬────┘  │  └──┬──┘
     │  │       │        │     └─ Data: "old" (3 bytes)
     │  │       │        └─ Data size: 3 (1 byte)
     │  │       └─ Key: "temp" (4 bytes)
     │  └─ Key size: 4
     └─ Type: 0x81 (deleted, was 0x01)
```

**Padding Byte:**
```
Hex: 80
     └─ Padding byte (fills gap too small for a record)
```

## Record Types by Size

### Type 0x01 (1-byte size)
- **Data size field**: 1 byte
- **Maximum data size**: 255 bytes
- **Total overhead**: 3 bytes (type + key_size + data_size)
- **Use case**: Small values (< 256 bytes)

**Example:** Configuration flags, user IDs, short strings

### Type 0x02 (2-byte size)
- **Data size field**: 2 bytes
- **Maximum data size**: 65,535 bytes (64 KB)
- **Total overhead**: 4 bytes
- **Use case**: Medium values (256 bytes - 64 KB)

**Example:** JSON documents, small files, formatted text

### Type 0x04 (4-byte size)
- **Data size field**: 4 bytes
- **Maximum data size**: 4,294,967,295 bytes (4 GB)
- **Total overhead**: 6 bytes
- **Use case**: Large values (64 KB - 4 GB)

**Example:** Binary files, images, large documents

### Type 0x08 (8-byte size)
- **Data size field**: 8 bytes
- **Maximum data size**: 18,446,744,073,709,551,615 bytes (18 EB)
- **Total overhead**: 10 bytes
- **Use case**: Very large values (> 4 GB)

**Example:** Video files, large archives (theoretical, not practical)

## Space Management

### Deleted Records
When a record is deleted:
1. Type byte bit 7 is set (e.g., 0x01 → 0x81)
2. Record remains in file (soft delete)
3. Position and size added to free space list
4. Can be reused by new records of similar size

### Free Space Reuse
When writing a new record:
1. Calculate required size
2. Search free space list for best fit
3. If found, write at that position
4. If leftover space exists, fill with padding (0x80)
5. Otherwise, append to end of file

### Padding Bytes
- Padding byte: `0x80`
- Used to fill gaps too small for a record
- Automatically skipped during file scanning
- Minimum record size is 4 bytes (type + key_size + key(1) + data_size)

### Compaction
`Compact()` creates a new file with:
- Only the latest version of each key
- No deleted records
- No padding bytes
- Optimal file size

## Design Principles

### Sequential Write
- All writes are appends (except free space reuse)
- Simple and reliable
- Good for HDDs (sequential performance)

### Variable-Length Encoding
- Automatic size field selection
- Small values use 1-byte size
- Large values use appropriate size field
- Efficient storage

### Soft Deletes
- Deleted records marked with flag
- Can be reused immediately
- No need to rewrite entire file
- Compact when needed for optimization

### Last-Write-Wins
- Updates append new version
- Old version marked as deleted
- Get() returns last active occurrence
- Simple conflict resolution

### In-Memory Cache
- All keys cached with file positions
- O(1) read performance
- Built on Open() by scanning file
- Updated on writes

## File Format Tools

### Inspect Raw File
```bash
# View hex dump
hexdump -C data/mydb.skv | head -n 20

# View with od
od -A x -t x1z -v data/mydb.skv
```

### File Statistics
```go
stats, _ := db.Verify()
fmt.Printf("Total records: %d\n", stats.TotalRecords)
fmt.Printf("Active records: %d\n", stats.ActiveRecords)
fmt.Printf("Deleted records: %d\n", stats.DeletedRecords)
fmt.Printf("File size: %d\n", stats.TotalSize)
fmt.Printf("Wasted space: %d\n", stats.WastedSpace)
```

## Compatibility

### Version Compatibility
- File format version stored in header
- Current implementation: 0.1.0
- Future versions may add features
- Version check on Open() ensures compatibility

### Portability
- Binary format (little-endian for multi-byte integers)
- Platform-independent
- Works on Linux, macOS, Windows, BSD
- Can be copied between systems

### Endianness
- Multi-byte integers use **little-endian** encoding
- Consistent across all platforms
- Compatible with most modern CPUs

## Next Steps

- **01-basics**: Learn basic operations
- **02-advanced**: Understand maintenance and compaction
- **05-backup**: Export to portable JSON format
