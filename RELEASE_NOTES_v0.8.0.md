# Release Notes - v0.8.0

**Release Date:** January 14, 2026

## Overview

SKV v0.8.0 introduces significant improvements to space management and file efficiency. This release features automatic coalescing of deleted records, file truncation for deleted records at end of file, and automatic WAL file cleanup on clean shutdown.

## ✨ New Features

### Free Space Coalescing
When deleting records, SKV now automatically merges adjacent deleted records into a single free space block. This allows larger records to reuse space that would otherwise be fragmented.

```go
// Before v0.8.0: 3 separate small free spaces
// After v0.8.0: 1 large coalesced free space
db.Delete([]byte("key1"))  // Creates free space
db.Delete([]byte("key2"))  // Coalesces with key1's space
db.Delete([]byte("key3"))  // Coalesces into single block
```

### Automatic File Truncation
When deleted records are at the end of the file, SKV now truncates the file instead of keeping wasted space. This automatically reduces file size without requiring explicit compaction.

```go
db.Put([]byte("a"), data)
db.Put([]byte("b"), data)
db.Put([]byte("c"), data)  // Last record

db.Delete([]byte("c"))     // File is truncated, no wasted space!
```

### WAL Auto-Cleanup
The Write-Ahead Log (WAL) file is now automatically removed on clean shutdown if it's empty. This eliminates leftover `.wal` files after normal database closure.

```go
db, _ := Open("mydb.skv")
// ... operations ...
db.Close()  // WAL file removed if empty
```

## 🔧 Improvements

### Enhanced Delete Operations
- Adjacent deleted records are automatically merged
- End-of-file deletions trigger file truncation
- Reduced fragmentation improves space reuse

### Better Space Efficiency
- Coalesced free spaces allow larger record reuse
- File size is automatically optimized on delete
- Less need for manual compaction in many scenarios

### Cleaner Shutdown
- WAL files are removed when empty
- No leftover files after clean closure
- Simpler file management for users

## 📦 Installation

```bash
go get github.com/jncss/skv@v0.8.0
```

Or update your `go.mod`:

```go
require github.com/jncss/skv v0.8.0
```

## 🔄 Upgrade Notes

This release has no breaking API changes but introduces behavioral changes that improve efficiency:

- **Backward Compatibility:** Full compatibility with v0.7.x
- **Database Format:** Compatible, existing databases work with v0.8.0
- **Behavioral Change:** Delete operations may now truncate files
- **Behavioral Change:** WAL files are removed on clean shutdown

### Important Notes

1. **File sizes may shrink**: Deleting records at the end of the file now truncates rather than marking as deleted.

2. **Fewer wasted space entries**: With coalescing, you'll see fewer (but larger) free space blocks.

3. **No .wal files after clean shutdown**: This is expected behavior, not file corruption.

## 📝 Technical Details

### New Internal Functions
- `findAdjacentFreeSpaces()`: Finds free spaces adjacent to a given position
- `coalesceFreeSpaces()`: Merges adjacent free spaces into single blocks
- `WAL.Close()`: Now removes WAL file if empty (only header)

### File Format
- Version header: 0.8.0 (0x00 0x08 0x00)
- No changes to record structure
- Compatible with all v0.7.x databases

### Performance Impact
- Delete operations: Slight overhead for coalescing check
- Space reuse: Improved due to larger free blocks
- File I/O: Reduced due to smaller files after end-deletions

## 🧪 Testing

New tests added:
- `TestFreeSpaceCoalescing`: Verifies adjacent space merging
- `TestDeleteAtEndTruncatesFile`: Verifies file truncation behavior
- WAL cleanup verification in existing tests

## ✅ Verification

After installing, verify the version:

```go
// Check version constants
fmt.Printf("SKV v%d.%d.%d\n", skv.VersionMajor, skv.VersionMinor, skv.VersionPatch)
// Output: SKV v0.8.0
```

## 🔗 Links

- **Repository**: https://github.com/jncss/skv
- **Documentation**: See README.md for complete feature list
- **Examples**: See `examples/` directory for usage examples

---

**Full Changelog**: https://github.com/jncss/skv/compare/v0.7.0...v0.8.0
