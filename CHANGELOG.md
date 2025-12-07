# Changelog

All notable changes to SKV will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **CRC Integrity Checking**: Every record now includes a CRC checksum for corruption detection
  - CRC-16-CCITT (2 bytes) for Type 0x01 records (data ≤ 255 bytes)
  - CRC-32-IEEE (4 bytes) for Types 0x02, 0x04, 0x08 (larger data)
  - Automatic verification on all read operations for active records
  - Deleted records skip CRC verification (type byte is modified during deletion)
- **Atomic Compaction**: Compact operation now uses temporary files for safe atomic updates
  - Prevents corruption if process is interrupted during compaction
  - Original file is only replaced after new file is completely written and verified
  - See `COMPACT_SAFETY.md` for detailed documentation
- **Context Support**: Context-aware operations for timeouts and cancellation
  - New functions: `PutCtx`, `GetCtx`, `UpdateCtx`, `DeleteCtx`, `CompactCtx`
  - Enable graceful shutdown and timeout handling in production environments
  - Periodic context checking during long-running compaction operations
  - Backward compatible: original functions call Ctx versions with `context.Background()`
- **CLI Hexdump Mode**: Added `--hex` / `-x` flag to display keys and values in hexadecimal format
  - Classic hexdump format with offset, hex bytes, and ASCII representation
  - Available in: `get`, `keys`, `foreach`, `getbatch` commands
  - Useful for inspecting binary data and debugging
- **CLI Recovery Command**: New `recover` command to salvage valid records from corrupted databases
  - Byte-by-byte scanning for potential valid records
  - CRC verification ensures only valid data is recovered
  - Automatically skips corrupted sections and deleted records
  - Usage: `skv recover corrupted.skv repaired.skv`
  - See `tools/cli/RECOVERY.md` for detailed documentation
- **Example: CRC Verification** (`examples/07-integrity/crc_verification`)
  - Demonstrates automatic CRC checking on active records
  - Shows verification of database integrity
- **Fuzzing Tests**: Comprehensive fuzz testing for robustness
  - 7 fuzz test functions covering Put/Get, Update, Delete, and binary keys
  - Automatic discovery of edge cases with random inputs
  - Tests for persistence, compaction, and multi-operation sequences
  - Binary key handling including special characters (0x00, 0xFF, 0x80)
- **Secondary Indexes**: Fast lookups by alternative keys
  - Create indexes with custom key extraction functions
  - **Supports duplicate secondary keys**: Multiple records can have the same indexed value
  - Automatic index maintenance on Put/Update/Delete operations
  - In-memory indexes for O(1) lookup performance
  - Save/Load indexes to JSON for persistence
  - Rebuild indexes when needed
  - Functions: `CreateIndex`, `GetByIndex`, `GetAllByIndex`, `HasIndex`, `SaveIndex`, `LoadIndex`, `RebuildIndex`, `DropIndex`
  - See `examples/08-indexes` for usage examples
- **Cursors**: Ordered iteration and range queries
  - Primary key cursors for sorted traversal
  - Secondary index cursors for indexed field ordering
  - **Full support for duplicate secondary keys**: Index cursors iterate through all records with the same indexed value
  - Range queries with inclusive from/to boundaries
  - Prefix-based cursors for pattern matching
  - Bidirectional iteration (forward and reverse)
  - Navigation methods: `Next`, `Seek`, `Reset`, `Key`, `Value`
  - Utility methods: `Count`, `ForEach`, `Collect`, `Keys`
  - Position checks: `IsFirst`, `IsLast`, `IsValid`
  - Functions: `AllCursor`, `NewCursor`, `PrefixCursor`, `AllIndexCursor`, `NewIndexCursor`, `PrefixIndexCursor`
  - See `examples/09-cursors` for usage examples
  - See `CURSORS.md` for comprehensive documentation

### Changed
- **BREAKING**: Record format changed to include CRC at end of each record
  - Old databases (without CRC) are **not compatible** with this version
  - You must recreate databases or migrate data using backup/restore
  - Record sizes increased by 2-4 bytes per record (for CRC)
- **Memory Optimization**: Streaming CRC calculation for write operations
  - `writeRecordAtPosition` now uses incremental CRC with 64KB chunks
  - `compactInternal` processes records one-at-a-time instead of loading all in memory
  - `readRecord(false)` now uses streaming CRC verification without loading data into memory
  - **O(1) constant memory** usage regardless of database size
  - Essential for databases with very large values (>1GB)
  - Improves performance of cache loading and delete operations on large records

### Fixed
- CRC verification is now skipped for deleted records (only type byte changes on delete)
- `readRecord()` now always reads data for CRC verification (no seeking back)

## [0.1.0] - Previous Version

### Features
- Sequential binary file format
- In-memory cache for O(1) lookups
- Thread-safe operations with mutex locks
- Free space reuse for deleted records
- Backup and restore with JSON
- Streaming operations for large values
- Cross-platform support (Linux, macOS, BSD, Windows)
- Command-line tool with 22 commands (before recover was added)
- 142 comprehensive tests
