# Atomic Transactions Example

This example demonstrates SKV's atomic transaction support with ACID guarantees.

## Features Demonstrated

1. **Basic transactions** - Creating multiple keys atomically
2. **Bank transfer** - Atomic money transfer between accounts
3. **Transaction rollback** - Explicit rollback of operations
4. **Validation errors** - Handling constraint violations
5. **Recovery** - Transaction recovery after crash

## Running the Example

```bash
cd examples/08-transactions
go run main.go
```

## What It Does

- Creates users atomically (all or nothing)
- Performs atomic money transfer between accounts
- Demonstrates rollback functionality
- Shows validation error handling
- Simulates crash recovery

## Key Concepts

### Atomicity
All operations in a transaction succeed together or fail together. No partial updates.

### Validation
- `Put` requires key does NOT exist
- `Update` requires key MUST exist
- `Delete` requires key MUST exist

### Isolation
Changes are buffered and not visible until commit.

### Durability
Committed transactions are logged to WAL and survive crashes.

## See Also

- [TRANSACTIONS.md](../../TRANSACTIONS.md) - Complete transaction documentation
- [transaction_test.go](../../transaction_test.go) - Comprehensive test suite
