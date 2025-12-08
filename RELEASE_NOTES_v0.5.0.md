# SKV v0.5.0 Release Notes

**Release Date:** December 8, 2025

## 🎯 Highlights

**Atomic Transactions** - SKV now supports full ACID transactions with all-or-nothing guarantees! This major feature allows you to group multiple operations (Put, Update, Delete) into a single atomic unit that either succeeds completely or fails completely, maintaining database consistency even in the face of errors or crashes.

## ✨ New Features

### Atomic Transactions (ACID)

The most significant addition in v0.5.0 is comprehensive transaction support:

```go
// Create multiple records atomically
tx := db.Begin()
tx.PutString("user:alice", `{"name":"Alice","age":30}`)
tx.PutString("user:bob", `{"name":"Bob","age":25}`)
tx.PutString("user:charlie", `{"name":"Charlie","age":35}`)

if err := tx.Commit(); err != nil {
    // All operations rolled back automatically
    return err
}
// All users created successfully
```

**Key Features:**
- **Atomicity**: All-or-nothing execution - either all operations succeed or all fail
- **Consistency**: Validation before any writes (Put requires non-existing key, Update/Delete require existing key)
- **Isolation**: Changes buffered until commit, not visible to other operations
- **Durability**: WAL logging with BeginTx/CommitTx/RollbackTx markers for crash recovery

**Transaction API:**
- `Begin() *Transaction` - Start a new transaction
- `tx.Put(key, data []byte)` - Add Put operation
- `tx.Update(key, data []byte)` - Add Update operation
- `tx.Delete(key []byte)` - Add Delete operation
- `tx.Commit()` - Apply all operations atomically
- `tx.Rollback()` - Discard all operations
- String variants: `PutString()`, `UpdateString()`, `DeleteString()`

**Transaction State:**
- `tx.Len()` - Number of operations
- `tx.ID()` - Unique transaction ID
- `tx.IsCommitted()` - Check if committed
- `tx.IsRolledBack()` - Check if rolled back

**Error Handling:**
- Automatic rollback on validation errors
- Original state restored on write failures
- Clear, descriptive error messages

**Recovery:**
- Committed transactions replayed from WAL
- Incomplete transactions discarded automatically
- Rolled back transactions ignored

## 🔧 Improvements

### WAL Enhancements

**New Operation Types:**
- `WALOpBeginTx` - Transaction begin marker
- `WALOpCommitTx` - Transaction commit marker
- `WALOpRollbackTx` - Transaction rollback marker

**Enhanced Recovery:**
- Buffers transaction operations until commit
- Applies all operations on commit
- Discards operations on rollback or incomplete transactions
- Fixed old-style commit marker handling (properly stops at WALOpCommit)

### Code Cleanup

Removed unused code from `compression.go`:
- `compressWriter` type (not used anywhere)
- `newCompressWriter()` function (not used anywhere)
- Removed unnecessary `io` import
- Reduced file size by 35 lines

## 📊 Statistics

| Metric | v0.4.0 | v0.5.0 | Change |
|--------|--------|--------|--------|
| Tests | 220 | 228 | +8 (+3.6%) |
| Coverage | 80.5% | 80.8% | +0.3% |
| Test Files | 16 | 17 | +1 |

### New Tests (15)

All transaction scenarios comprehensively tested:

1. **TestTransactionBasic** - Basic transaction operations
2. **TestTransactionRollback** - Explicit rollback
3. **TestTransactionPutExistingKey** - Put validation (key must not exist)
4. **TestTransactionUpdateNonExistingKey** - Update validation (key must exist)
5. **TestTransactionDeleteNonExistingKey** - Delete validation (key must exist)
6. **TestTransactionMixedOperations** - Put + Update + Delete in one transaction
7. **TestTransactionEmpty** - Empty transaction handling
8. **TestTransactionDoubleCommit** - Prevent double commit
9. **TestTransactionDoubleRollback** - Prevent double rollback
10. **TestTransactionCommitAfterRollback** - Prevent commit after rollback
11. **TestTransactionRecovery** - Recovery of committed transactions
12. **TestTransactionRecoveryIncomplete** - Incomplete transactions discarded
13. **TestTransactionRecoveryRolledBack** - Rolled back transactions ignored
14. **TestTransactionLargeData** - Transactions with large data (10x 1MB)
15. **TestTransactionSequential** - Sequential transaction throughput (100 txns)

## 📚 Documentation

### New Documentation

**TRANSACTIONS.md** (900+ lines)
Comprehensive guide covering:
- Transaction overview and ACID guarantees
- Complete API reference
- Transaction semantics and validation rules
- Error handling patterns
- Recovery and durability
- Performance considerations
- Real-world examples (bank transfer, batch operations, etc.)

### Updated Documentation

- **README.md**: Added transactions to features, Quick Start example
- **CHANGELOG.md**: Complete v0.5.0 changelog
- **TESTING.md**: Added transaction test section, updated statistics

### Examples

**examples/08-transactions/** - Complete working example demonstrating:
1. Basic transactions (multiple users)
2. Bank transfer (atomic money transfer)
3. Transaction rollback
4. Validation errors
5. Mixed operations (Put + Update + Delete)
6. Large batch transactions (50 products)
7. Recovery simulation
8. Performance benchmarking (~109 tx/sec)

## 🔄 Migration Guide

### From v0.4.0 to v0.5.0

**No breaking changes!** v0.5.0 is fully backward compatible with v0.4.0.

**New capabilities:**
```go
// Old way (still works)
db.PutString("key1", "value1")
db.PutString("key2", "value2")

// New way (atomic)
tx := db.Begin()
tx.PutString("key1", "value1")
tx.PutString("key2", "value2")
tx.Commit() // Both or neither
```

**WAL Compatibility:**
- Old WAL files (pre-transactions) are fully supported
- New WAL files include transaction markers
- Recovery handles both old and new formats transparently

## 🎓 Use Cases

Transactions are perfect for:

✅ **Multi-record operations** that must succeed together
- Creating related records (user + profile + settings)
- Batch imports where partial success is unacceptable

✅ **Financial operations** requiring consistency
- Account transfers (debit one, credit another)
- Balance updates with audit logging

✅ **State transitions** with multiple steps
- Workflow state changes with history tracking
- Multi-table updates in application state

✅ **Cleanup operations**
- Delete related records atomically
- Swap old and new data sets

## 🚀 Performance

**Transaction Throughput** (sequential):
- ~109 transactions/second with WAL enabled
- Each transaction can contain multiple operations
- Example: 100 transactions with 1 operation each = 9.15ms average per transaction

**Memory Efficiency:**
- Operations buffered in memory during transaction
- Efficient for small to medium transactions (< 10,000 operations)
- For very large batches, consider splitting into multiple transactions

## 📦 Installation

```bash
go get github.com/jncss/skv@v0.5.0
```

Or update your `go.mod`:
```
github.com/jncss/skv v0.5.0
```

## 🔗 Resources

- **Documentation**: See [TRANSACTIONS.md](TRANSACTIONS.md) for complete guide
- **Examples**: See [examples/08-transactions/](examples/08-transactions/)
- **Tests**: See [transaction_test.go](transaction_test.go)
- **Changelog**: See [CHANGELOG.md](CHANGELOG.md)

## 🙏 Acknowledgments

This release represents a significant milestone in SKV's evolution, adding enterprise-grade transaction support while maintaining simplicity and performance.

## 📝 Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete details.

---

**Download:** https://github.com/jncss/skv/releases/tag/v0.5.0
**GitHub:** https://github.com/jncss/skv
