# SKV Test Organization

This document describes the organization of test files in the SKV project.

## Test Files

### `skv_test.go` (1,267 lines)
**Core functionality tests**
- Basic CRUD operations (Put, Get, Update, Delete)
- File operations and database lifecycle
- Key management
- Data persistence
- Record type handling (1-byte, 2-byte, 4-byte, 8-byte)
- Compaction functionality
- File verification

### `advanced_test.go` (739 lines)
**Advanced features and convenience functions**
- Exists/Has checks
- Count operations
- Clear functionality
- GetOrDefault operations
- ForEach iteration (binary and string variants)
- Batch operations (PutBatch, GetBatch)
- String convenience functions (PutString, GetString, UpdateString, etc.)

### `integrity_test.go` (614 lines)
**Data integrity and storage management**
- Extended statistics (Verify with detailed stats)
- File header validation
- Version checking
- Free space tracking and reuse
- Storage efficiency metrics
- Wasted space calculations
- Compaction effectiveness

### `lifecycle_test.go` (470 lines)
**Database lifecycle management**
- Close operations (normal and with compaction)
- Backup functionality (JSON export)
- Restore functionality (JSON import)
- Encoding strategies (text vs base64)
- Partial restore behavior

### `stress_test.go` (441 lines)
**Performance and reliability under load**
- 10,000+ record operations
- Concurrent access (10 goroutines)
- Large value handling (up to 1MB)
- Database reopen/recovery cycles
- Performance benchmarks

### `concurrent_test.go` (326 lines)
**Thread safety and concurrent access**
- Concurrent reads
- Concurrent writes
- Mixed read/write operations
- Concurrent compaction
- Cache consistency under concurrent access

### `errors_test.go` (381 lines)
**Error handling and edge cases**
- File permission errors
- Corrupted file handling
- Invalid input validation
- Resource cleanup on errors
- Edge cases for all major operations
- Type calculation coverage

### `context_test.go` (400 lines)
**Context-aware operations**
- Context cancellation during Put, Get, Update, Delete operations
- Timeout handling (PutCtx, GetCtx, UpdateCtx, DeleteCtx)
- Compaction cancellation (CompactCtx)
- Context propagation and error handling
- Integration with long-running operations
- Proper cleanup on cancellation

### `readrecord_optimization_test.go` (215 lines)
**Memory optimization for readRecord**
- Streaming CRC verification without loading data into memory
- Tests for small, medium, large, and very large values (up to 5MB)
- Verification that deleted records skip CRC efficiently
- Corruption detection with streaming CRC
- Performance validation for cache loading and delete operations

### `fuzz_test.go` (335 lines)
**Fuzzing tests for robustness**
- FuzzPutGet: Random key/value combinations
- FuzzUpdate: Random update sequences
- FuzzDelete: Random deletion patterns
- FuzzMultipleOperations: Random operation sequences
- FuzzReopenPersistence: Random data persistence validation
- FuzzCompaction: Random compaction scenarios
- FuzzBinaryKeys: Binary keys with special characters (including 0x00, 0xFF, 0x80)

### `index_test.go` (560 lines)
**Secondary indexes**
- Index creation and management
- Lookups by secondary keys
- Automatic index updates on Put/Update/Delete
- Index persistence (save/load to JSON)
- Index rebuilding
- Binary data indexing

### `cursor_test.go` (682 lines)
**Cursors for ordered iteration**
- Primary key cursors (all records, prefix, range)
- Secondary index cursors
- Forward and reverse iteration
- Cursor navigation (Next, Seek, Reset)
- Utility methods (ForEach, Collect, Keys, Count)
- Position checks (IsFirst, IsLast, IsValid)
- Edge cases (empty database, closed cursors)

### `logger_test.go` (275 lines)
**Structured logging**
- NullLogger (zero overhead, no output)
- TextLogger (human-readable format)
- JSONLogger (structured JSON output)
- Log levels (Debug, Info, Warn, Error)
- Field-based logging with key-value pairs
- Logger integration with all operations
- Custom logger implementations

### `stream_rollback_test.go` (316 lines)
**Stream operation rollback protection**
- PutStream rollback on write errors
- UpdateStream rollback preserving original value
- Database integrity after failed stream operations
- File size verification after rollback
- Checkpoint-based error recovery
- errorReader test helper for simulating failures

### `rlock_test.go` (202 lines)
**Concurrent read optimization**
- Benchmarks for concurrent reads vs sequential
- RLock performance for Keys(), Exists(), Count()
- Concurrent safety validation
- Race condition prevention tests
- Mixed read/write concurrent operations

### `string_convenience_test.go` (350 lines)
**String convenience functions for cursors and indexes**
- Cursor creation with strings: NewCursorString, PrefixCursorString, AllCursorString
- Index cursor creation: NewIndexCursorString, PrefixIndexCursorString, AllIndexCursorString
- String navigation: NextString, KeyString, ValueString, SeekString
- String utilities: HasPrefixString, KeysString, CollectString
- Index operations: GetByIndexString, GetAllByIndexString, HasIndexString
- Reverse iteration validation
- Category-based index testing

### `transaction_test.go` (690+ lines)
**Atomic transactions with ACID guarantees**
- Basic transaction operations (Put, Update, Delete)
- Transaction commit and rollback
- Validation rules (Put requires non-existing key, Update/Delete require existing key)
- Mixed operations within single transaction
- Empty transaction handling
- Double commit/rollback prevention
- Commit after rollback prevention
- Transaction recovery after crash (committed, incomplete, rolled back)
- Large data in transactions (1MB records)
- Sequential transactions (100+ transactions)
- Transaction state queries (Len, ID, IsCommitted, IsRolledBack)

## Running Tests

```bash
# Run all tests
go test

# Run tests with coverage
go test -cover

# Run specific test file
go test -run TestPut

# Run tests verbosely
go test -v

# Run only stress tests
go test -run TestStress

# Run tests with race detector
go test -race

# Run fuzzing tests (indefinitely until failure)
go test -fuzz=FuzzPutGet

# Run fuzzing for specific duration
go test -fuzz=FuzzPutGet -fuzztime=30s
go test -fuzz=FuzzBinaryKeys -fuzztime=1m
go test -fuzz=FuzzCompaction -fuzztime=10s

# List all fuzz tests
go test -list=Fuzz
```

## Test Statistics

- **Total tests**: 228 (221 regular + 7 fuzz functions)
- **Test coverage**: 80.8%
- **Total lines**: ~9,500+
- **Test files**: 17

## Test Categories

### Functional Tests (153 tests)
- Basic operations
- Advanced features
- Data integrity
- Lifecycle management
- Context-aware operations
- Memory optimization
- Secondary indexes
- Cursors for ordered iteration
- Structured logging
- Stream rollback protection
- Concurrent read optimization
- String convenience functions for cursors and indexes
- Atomic transactions (ACID guarantees)

### Fuzzing Tests (7 functions)
- Random input generation
- Edge case discovery
- Binary data handling
- Persistence validation
- Compaction robustness

### Stress Tests (15 tests)
- Large datasets (10,000+ records)
- Concurrent operations
- Large values (up to 1MB)
- Recovery scenarios

### Error/Coverage Tests (11 tests)
- Error conditions
- Edge cases
- Code path coverage

## Reorganization History

Originally, tests were split across 11 files:
- `skv_test.go`, `extended_test.go`, `string_test.go`, `concurrent_test.go`
- `stress_test.go`, `freespace_test.go`, `header_test.go`, `close_test.go`
- `backup_test.go`, `verify_stats_test.go`, `coverage_test.go`

Reorganized into 17 logically grouped files for better maintainability:
- Core tests remain in `skv_test.go`
- Advanced features consolidated into `advanced_test.go`
- Integrity checks merged into `integrity_test.go`
- Lifecycle operations combined into `lifecycle_test.go`
- Specialized tests kept separate: `stress_test.go`, `concurrent_test.go`, `errors_test.go`
- Context support added in `context_test.go`
- Memory optimization tests in `readrecord_optimization_test.go`
- Fuzzing tests in `fuzz_test.go`
- Secondary indexes in `index_test.go`
- Cursors in `cursor_test.go`
- Structured logging in `logger_test.go` (new)
- Stream rollback protection in `stream_rollback_test.go` (new)
- Concurrent read optimization in `rlock_test.go` (new)
- String convenience functions in `string_convenience_test.go` (new)
- Atomic transactions in `transaction_test.go` (new)
