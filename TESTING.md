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
```

## Test Statistics

- **Total tests**: 102
- **Test coverage**: 81.8%
- **Total lines**: 4,238
- **Average test file size**: ~605 lines

## Test Categories

### Functional Tests (76 tests)
- Basic operations
- Advanced features
- Data integrity
- Lifecycle management

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

Reorganized into 7 logically grouped files for better maintainability:
- Core tests remain in `skv_test.go`
- Advanced features consolidated into `advanced_test.go`
- Integrity checks merged into `integrity_test.go`
- Lifecycle operations combined into `lifecycle_test.go`
- Specialized tests kept separate: `stress_test.go`, `concurrent_test.go`, `errors_test.go`
