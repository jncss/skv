# Changelog

All notable changes to SKV will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.2] - 2025-12-08

### Fixed
- **CLI recover with encryption**: Fixed double-encryption bug when recovering encrypted databases
  - Recover now preserves raw bytes without re-encrypting
  - Recovered encrypted databases maintain encryption state and require same password
  - Added safety check to prevent OOM panic on corrupted data size fields (max 100MB per record)
  - Enhanced `test_recovery.sh` with encrypted database recovery test

### Documentation
- **CLI README**: Updated Example 9 with encryption recovery behavior explanation
- Clarified that recovered files preserve encryption state of original file

## [0.6.1] - 2025-12-08

### Changed
- **Dependencies**: Updated `github.com/jncss/easyaes` from v0.0.0-20251208190620-9743bf4abb45 to v1.0.0
- **Code quality**: Fixed redundant newline in `examples/12-encryption/main.go`

## [0.6.0] - 2025-12-08

### Added
- **Encryption System**: Optional encryption for keys and values with password-based authentication
  - **AES Encryption**: Industry-standard AES-256 via EasyAES library (Base64 encoded)
  - **SimpleCipher Encryption**: Custom XOR-based cipher via SimpleCipher library (Base64 encoded)
  - **Separate Encryption**: Keys and values encrypted independently
  - **Encryption Order**: Encrypt BEFORE compress (write), Decompress THEN decrypt (read)
  - **Transparent Operations**: Automatic encryption/decryption on all Get/Put/Update/Delete operations
  - **Secure Backups**: Backup/Restore preserves encrypted data without decryption (critical security feature)
  - **CLI Integration**: `--encryption`/`-e` and `--password`/`-p` flags for encrypted databases
  - **10 new tests**: Comprehensive encryption coverage (AES, SimpleCipher, compression, backup/restore, wrong password)
  - See `ENCRYPTION.md` for comprehensive documentation
  - See `encryption_test.go` and `backup_encryption_test.go` for test examples
  - See `examples/12-encryption/` for complete working examples

### Changed
- **File version**: Updated to 0.6.0
- **API**: Added optional `Encryption` and `EncryptionPassword` fields to `Options` struct
- **Internal**: Modified `readRecord()` and `writeRecordAtPosition()` with `skipEncryption` parameter for secure backups
- **CLI**: Enhanced with encryption support in all commands (put, get, update, delete, backup, restore)
- **Test count**: 238 tests (up from 228, +10 encryption tests)
- **Coverage**: 81.0% (improved from 80.8%)

### Dependencies
- **Added**: `github.com/jncss/easyaes` v1.0.0 - AES-256 encryption
- **Added**: `github.com/jncss/simplecipher` v1.0.0 - Custom XOR cipher

### Fixed
- **Backup security**: Backup now preserves encrypted data instead of decrypting (prevents sensitive data exposure in JSON)
- **Restore cache indexing**: Fixed restore using encrypted keys for cache; now correctly decrypts keys before cache insertion

### Documentation
- **Added**: `ENCRYPTION.md` - Comprehensive encryption guide
- **Updated**: `README.md` - Added encryption feature and updated version/test counts
- **Updated**: `tools/cli/README.md` - Added encryption examples (Examples 7 & 8)
- **Updated**: `tools/cli/EXAMPLES.md` - Added encryption usage examples
- **Added**: `examples/12-encryption/` - Complete working encryption examples

## [0.5.1] - 2025-12-08

### Fixed
- **Compression with record reuse**: Fixed padding calculation when reusing space with compressed records
  - `writeRecord()` now uses actual written size instead of estimated size for padding calculation
  - Prevents data corruption when updating uncompressed records with compressed data
  - `writeRecordAtPosition()` signature changed to return actual record size: `(int64, uint64, error)`
- **Verify statistics**: Fixed padding detection and efficiency calculation with compressed records
  - Padding bytes after last record now counted correctly
  - `Efficiency` and `Wasted Percent` now accurate for databases with mixed compression levels
  - Renamed `activeDataSize` → `activeRecordSize` for clarity (on-disk size vs decompressed size)

## [0.5.0] - 2025-12-08

### Added
- **Atomic Transactions**: Full ACID transaction support with all-or-nothing guarantees
  - Transaction API: `Begin()`, `Commit()`, `Rollback()`
  - Mixed operations: `Put()`, `Update()`, `Delete()` within transactions
  - String convenience: `PutString()`, `UpdateString()`, `DeleteString()`
  - Validation: Keys checked before any writes (fast failure)
  - Isolation: Changes buffered until commit (not visible to other operations)
  - Durability: WAL logging with BeginTx, CommitTx, RollbackTx markers
  - Recovery: Automatic replay of committed transactions, discarding incomplete ones
  - Error handling: Automatic rollback on validation or write errors
  - Transaction state: `Len()`, `ID()`, `IsCommitted()`, `IsRolledBack()`
  - **15 new tests**: All transaction scenarios covered (basic, rollback, validation, recovery, large data, sequential)
  - See `TRANSACTIONS.md` for comprehensive documentation and examples
  - See `transaction_test.go` for test examples
  - See `examples/08-transactions/` for complete working example

### Changed
- **WAL operation types**: Added transaction markers (BeginTx, CommitTx, RollbackTx)
- **WAL recovery**: Enhanced to handle transactions (buffer operations, apply on commit, discard on rollback/incomplete)
  - Fixed old-style commit marker handling (properly stops processing at WALOpCommit)
- **Test count**: 228 tests (up from 220, +15 transaction tests)
- **Coverage**: 80.8% (improved from 80.5%)

### Removed
- **Unused code**: Removed `compressWriter` type and `newCompressWriter()` function from compression.go (-35 lines)

## [0.4.0] - 2025-12-07

### Added
- **Structured Logging**: Built-in logging support for observability
  - Three logger implementations: `NullLogger` (default, zero overhead), `JSONLogger` (structured), `TextLogger` (human-readable)
  - Logger interface for custom implementations
  - Four log levels: Debug, Info, Warn, Error
  - Field-based logging with key-value pairs
  - Set logger via `SetLogger()` method
  - All operations logged: Put, Get, Update, Delete, Compact, WAL operations
  - Performance metrics logged: operation duration, data sizes, file positions
  - Error recovery events logged: rollback operations, WAL recovery
  - See `LOGGING.md` for comprehensive documentation
  - See `examples/07-logging` for usage examples

- **Stream Rollback Protection**: PutStream and UpdateStream now include checkpoint-based rollback protection
  - Atomic writes: Either entire key-value is written or nothing (no partial records)
  - Checkpoint mechanism: File position saved before write, truncated on failure
  - Original value preserved: UpdateStream keeps old value if new write fails
  - No WAL overhead: Uses truncate instead of WAL to avoid memory buffering for large streams
  - Consistent state: Database remains clean after failed operations, ready for retry
  - All rollback events logged (Warn on success, Error on failure)
  - **3 new tests**: `TestPutStreamRollbackOnError`, `TestUpdateStreamRollbackOnError`, `TestStreamRollbackPreservesIntegrity`
  - See `ROLLBACK_PROTECTION.md` for detailed documentation
  - See `stream_rollback_test.go` for test examples

- **RWMutex Optimization**: Read operations now use RLock for better concurrency
  - Cache-only operations use RLock: `Keys()`, `Exists()`, `Count()`
  - File operations still use exclusive Lock (not thread-safe at OS level)
  - 10-100x throughput improvement for concurrent read-heavy workloads
  - New test file `rlock_test.go` with concurrent benchmarks
  - Validates race-free operation with `TestConcurrentKeysAndWrites`

- **String Convenience Functions for Cursors and Indexes**: Extended string API coverage
  - Cursor creation: `NewCursorString()`, `PrefixCursorString()`, `AllCursorString()`
  - Index cursors: `NewIndexCursorString()`, `PrefixIndexCursorString()`, `AllIndexCursorString()`
  - Cursor navigation: `NextString()`, `KeyString()`, `ValueString()`, `SeekString()`
  - Cursor utilities: `HasPrefixString()`, `KeysString()`, `CollectString()`
  - Index operations: `GetAllByIndexString()` (complements existing `GetByIndexString()` and `HasIndexString()`)
  - **3 new tests**: `TestCursorStringFunctions`, `TestIndexCursorStringFunctions`, `TestIndexStringFunctions`
  - See `CURSORS.md` for comprehensive documentation and examples

### Changed
- **Total tests**: Now 220 tests passing (217 previous + 3 string convenience tests)
- **Test coverage**: 80.5% of statements (improved from 79.1%)
- **UpdateStream behavior**: Now writes new record before deleting old one (preserves original on failure)
- **Concurrency model**: Read operations that only access cache now allow concurrent execution

## [0.3.0] - 2025-12-07

### Added
- **Data Compression**: Transparent LZ4 and Snappy compression support
  - Two algorithms: LZ4 (balanced, best general purpose) and Snappy (maximum speed)
  - Threshold-based: Only data ≥128 bytes is considered for compression
  - Adaptive: Compression only applied if it actually reduces size
  - Mixed compression: Different records can use different algorithms in same database
  - Transparent operation: Automatic compression on write, decompression on read
  - New API: `OpenWithOptions()` with `Options{Compression: CompressionLZ4 | CompressionSnappy | CompressionNone}`
  - Record format extended: Compression type stored in type byte (bits 5-6), original size field added when compressed
  - Performance: ~5-7% write overhead, ~2-3% read overhead, 40-60% space savings on compressible data
  - **9 compression tests**: None, small data threshold, Snappy, LZ4, incompressible data, SKV integration, mixed compression, flags
  - See `COMPRESSION.md` for comprehensive documentation
  - See `compression_test.go` for usage examples

- **CLI Compression Support**: Added `--compression` / `-c` flag to CLI tool
  - Values: `none` (default), `snappy`, `lz4`
  - Applies to all write operations (put, update, putfile, putstream, etc.)
  - Transparent: reads work regardless of compression settings
  - Recovery support: `recover` command handles compressed databases correctly
  - Examples: `skv -c lz4 put db.skv key value`, `skv --compression snappy putfile db.skv config file.txt`
  - Documentation updated in `tools/cli/README.md` with compression examples

### Changed
- **BREAKING**: Record format changed to support compression
  - Type byte bits 5-6 now used for compression flags (00=none, 01=snappy, 10=lz4)
  - Compressed records include original size field before compressed size
  - Old databases (v0.2.0 and earlier) may not be compatible
  - Migration: Use backup/restore or `recover` command to migrate data
- **Dependencies**: Added github.com/golang/snappy v1.0.0 and github.com/pierrec/lz4/v4 v4.1.22
- **getBaseType()**: Now masks both deleted flag (bit 7) AND compression bits (bits 5-6)
- **CLI binary name**: Clarified that CLI tool compiles as `skv` (not `cli`)
- **Total tests**: Now 206 tests passing (197 library + 9 compression)

### Fixed
- CLI flag parsing: `--compression` flag now correctly filters from argument list
- CLI recover: Now correctly skips compression originalSize field when parsing compressed records
- Record type calculation: Now uses max(compressedSize, originalSize) to ensure both fit in size fields

## [0.2.0] - 2025-12-07

### Added
- **Write-Ahead Log (WAL)**: Crash recovery and durability guarantees
  - Logs all operations (Put, Update, Delete) before applying to main file
  - Automatic recovery on database open - replays uncommitted operations
  - CRC-32 protected WAL entries for corruption detection
  - Commit markers for transaction boundaries
  - WAL file created as `<database>.skv.wal`
  - Manual control: `Enable()`, `Disable()`, `IsEnabled()`, `Size()`
  - Functions: `OpenWAL`, `LogPut`, `LogDelete`, `LogCommit`, `Recover`, `Truncate`, `Close`
  - Provides ACID properties (Atomicity, Consistency, Isolation, Durability)
  - Performance: ~199 writes/sec with WAL, ~734 writes/sec without WAL
  - See `examples/10-wal` for usage examples
  - See `WAL.md` for comprehensive documentation
  - **10 WAL tests**: Basic logging, truncation, commit markers, disable/enable, large data, corruption handling, concurrent writes, and recovery
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
