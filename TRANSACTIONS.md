# SKV Atomic Transactions

SKV provides full support for atomic transactions with ACID guarantees. All operations within a transaction are applied atomically - either all succeed or none are applied.

## Table of Contents

- [Overview](#overview)
- [Basic Usage](#basic-usage)
- [API Reference](#api-reference)
- [Transaction Semantics](#transaction-semantics)
- [Error Handling](#error-handling)
- [Recovery and Durability](#recovery-and-durability)
- [Performance Considerations](#performance-considerations)
- [Examples](#examples)

## Overview

### What are Atomic Transactions?

An atomic transaction is a sequence of operations that are treated as a single unit of work:

- **All-or-Nothing**: Either all operations succeed and are committed, or all fail and are rolled back
- **Isolation**: Operations within a transaction are not visible to other operations until committed
- **Durability**: Committed transactions are persisted to the Write-Ahead Log (WAL) and survive crashes
- **Consistency**: Transactions maintain database integrity with validation before commit

### Key Features

✅ **ACID Guarantees**
- Atomicity: All-or-nothing execution
- Consistency: Validation ensures database integrity
- Isolation: Changes are buffered until commit
- Durability: WAL ensures crash recovery

✅ **Flexible Operations**
- `Put`: Insert new keys (must not exist)
- `Update`: Modify existing keys (must exist)
- `Delete`: Remove existing keys (must exist)

✅ **Crash Recovery**
- Incomplete transactions are automatically discarded
- Committed transactions are replayed from WAL
- Rolled back transactions are ignored

✅ **Validation**
- Keys are validated before any writes
- Fast failure if constraints are violated
- Original state restored on error

## Basic Usage

### Simple Transaction

```go
package main

import (
    "fmt"
    "github.com/jncss/skv"
)

func main() {
    db, _ := skv.Open("data.skv")
    defer db.Close()

    // Begin transaction
    tx := db.Begin()

    // Add operations
    tx.PutString("user:1", `{"name": "Alice", "age": 30}`)
    tx.PutString("user:2", `{"name": "Bob", "age": 25}`)
    tx.PutString("user:3", `{"name": "Charlie", "age": 35}`)

    // Commit atomically
    if err := tx.Commit(); err != nil {
        fmt.Printf("Transaction failed: %v\n", err)
        return
    }

    fmt.Println("All users created atomically!")
}
```

### Transaction with Rollback

```go
tx := db.Begin()

tx.PutString("account:1", "100.00")
tx.PutString("account:2", "200.00")

// Decide not to commit
if err := tx.Rollback(); err != nil {
    fmt.Printf("Rollback failed: %v\n", err)
}

// No changes were applied
```

### Mixed Operations

```go
// Initial data
db.PutString("balance:alice", "1000.00")
db.PutString("balance:bob", "500.00")
db.PutString("temp:data", "temporary")

// Transaction with Put, Update, Delete
tx := db.Begin()

tx.UpdateString("balance:alice", "900.00")  // Deduct
tx.UpdateString("balance:bob", "600.00")    // Add
tx.PutString("transfer:1", "alice->bob:100") // Log
tx.DeleteString("temp:data")                 // Cleanup

if err := tx.Commit(); err != nil {
    fmt.Printf("Transfer failed: %v\n", err)
    return
}

fmt.Println("Transfer completed atomically!")
```

## API Reference

### Creating Transactions

#### `Begin() *Transaction`

Creates a new transaction. The transaction must be committed with `Commit()` or rolled back with `Rollback()`.

```go
tx := db.Begin()
```

### Adding Operations

#### `Put(key, data []byte) error`

Adds a Put operation to the transaction. The key must NOT exist when the transaction is committed.

```go
err := tx.Put([]byte("key"), []byte("value"))
```

#### `PutString(key, value string) error`

Convenience method for Put using strings.

```go
err := tx.PutString("key", "value")
```

#### `Update(key, data []byte) error`

Adds an Update operation to the transaction. The key MUST exist when the transaction is committed.

```go
err := tx.Update([]byte("key"), []byte("new_value"))
```

#### `UpdateString(key, value string) error`

Convenience method for Update using strings.

```go
err := tx.UpdateString("key", "new_value")
```

#### `Delete(key []byte) error`

Adds a Delete operation to the transaction. The key MUST exist when the transaction is committed.

```go
err := tx.Delete([]byte("key"))
```

#### `DeleteString(key string) error`

Convenience method for Delete using strings.

```go
err := tx.DeleteString("key")
```

### Finalizing Transactions

#### `Commit() error`

Commits the transaction atomically. All operations are validated first, then applied together. Returns an error if any validation fails or if there's a write error.

```go
if err := tx.Commit(); err != nil {
    // Transaction failed, no changes were applied
}
```

#### `Rollback() error`

Rolls back the transaction. All operations are discarded and no changes are applied.

```go
if err := tx.Rollback(); err != nil {
    // Rollback failed (unlikely)
}
```

### Query Methods

#### `Len() int`

Returns the number of operations in the transaction.

```go
count := tx.Len()
fmt.Printf("Transaction has %d operations\n", count)
```

#### `ID() uint64`

Returns the unique identifier for this transaction.

```go
id := tx.ID()
fmt.Printf("Transaction ID: %d\n", id)
```

#### `IsCommitted() bool`

Returns true if the transaction has been committed.

```go
if tx.IsCommitted() {
    fmt.Println("Transaction committed")
}
```

#### `IsRolledBack() bool`

Returns true if the transaction has been rolled back.

```go
if tx.IsRolledBack() {
    fmt.Println("Transaction rolled back")
}
```

## Transaction Semantics

### Validation Rules

Transactions validate all operations before applying any changes:

| Operation | Validation Rule | Error if Failed |
|-----------|----------------|-----------------|
| `Put` | Key must NOT exist | `ErrKeyExists` |
| `Update` | Key MUST exist | `ErrKeyNotFound` |
| `Delete` | Key MUST exist | `ErrKeyNotFound` |

### Isolation

Operations within a transaction are isolated:

```go
db.PutString("counter", "0")

tx := db.Begin()
tx.UpdateString("counter", "1")
tx.UpdateString("counter", "2") // Updates buffered value
tx.Commit()

// Now counter = "2"
```

External operations cannot see uncommitted changes:

```go
tx := db.Begin()
tx.PutString("new_key", "value")

// This will fail - key not visible yet
value, err := db.GetString("new_key") // ErrKeyNotFound

tx.Commit()

// Now it's visible
value, err = db.GetString("new_key") // "value"
```

### Atomicity Guarantees

**All operations succeed or all fail:**

```go
tx := db.Begin()
tx.PutString("key1", "value1")
tx.PutString("key2", "value2")
tx.PutString("existing_key", "value") // This key already exists!

err := tx.Commit()
// Error: key "existing_key" already exists
// Neither key1 nor key2 were created
```

### State Transitions

A transaction has three states:

```
┌─────────┐
│  Begin  │
└────┬────┘
     │
     ├─────────────┬─────────────┐
     ▼             ▼             ▼
┌─────────┐   ┌─────────┐   ┌─────────┐
│ Active  │   │Committed│   │RolledBack│
└─────────┘   └─────────┘   └─────────┘
```

Once committed or rolled back, a transaction cannot be reused:

```go
tx := db.Begin()
tx.PutString("key", "value")
tx.Commit()

// This will fail
err := tx.Commit() // Error: transaction already committed

// This will also fail
err = tx.Rollback() // Error: transaction already committed
```

## Error Handling

### Common Errors

#### ErrKeyExists

Occurs when a `Put` operation tries to create a key that already exists:

```go
db.PutString("user:1", "data")

tx := db.Begin()
tx.PutString("user:1", "new_data") // Will fail at commit
err := tx.Commit()
// Error: operation 0: key "user:1" already exists
```

#### ErrKeyNotFound

Occurs when `Update` or `Delete` operations reference non-existing keys:

```go
tx := db.Begin()
tx.UpdateString("nonexistent", "value") // Will fail at commit
err := tx.Commit()
// Error: operation 0: key "nonexistent" not found
```

#### Transaction State Errors

```go
tx := db.Begin()
tx.Commit()

// Cannot commit twice
err := tx.Commit()
// Error: transaction already committed

// Cannot rollback after commit
err = tx.Rollback()
// Error: transaction already committed
```

### Error Recovery

If a transaction fails, the original state is preserved:

```go
// Original state
db.PutString("balance", "1000")

tx := db.Begin()
tx.UpdateString("balance", "900")
tx.PutString("existing_key", "value") // Oops, already exists!

err := tx.Commit()
if err != nil {
    // Transaction failed, balance is still "1000"
    val, _ := db.GetString("balance")
    fmt.Println(val) // Output: 1000
}
```

## Recovery and Durability

### Write-Ahead Logging

All transactions are logged to the WAL before being applied:

```
┌──────────────┐
│ Begin TX 42  │
├──────────────┤
│ Put key1     │
│ Put key2     │
│ Update key3  │
├──────────────┤
│ Commit TX 42 │
└──────────────┘
```

### Crash Recovery

On database reopening, the WAL is replayed:

**Committed transactions** are fully applied:
```
BEGIN TX 42
  PUT key1=value1
  PUT key2=value2
COMMIT TX 42
→ Both operations applied
```

**Incomplete transactions** are discarded:
```
BEGIN TX 43
  PUT key3=value3
  PUT key4=value4
(no commit marker)
→ Both operations discarded
```

**Rolled back transactions** are ignored:
```
BEGIN TX 44
  PUT key5=value5
ROLLBACK TX 44
→ Operations discarded
```

### Recovery Example

```go
// Session 1: Crash before commit
db1, _ := skv.Open("data.skv")
tx := db1.Begin()
tx.PutString("key1", "value1")
tx.PutString("key2", "value2")
// ... crash here (no commit)
db1.Close()

// Session 2: Recovery
db2, _ := skv.Open("data.skv")
defer db2.Close()

// Incomplete transaction was discarded during recovery
_, err1 := db2.GetString("key1") // ErrKeyNotFound
_, err2 := db2.GetString("key2") // ErrKeyNotFound
```

```go
// Session 1: Commit before crash
db1, _ := skv.Open("data.skv")
tx := db1.Begin()
tx.PutString("key1", "value1")
tx.PutString("key2", "value2")
tx.Commit()
// ... crash here
db1.Close()

// Session 2: Recovery
db2, _ := skv.Open("data.skv")
defer db2.Close()

// Committed transaction was replayed during recovery
val1, _ := db2.GetString("key1") // "value1"
val2, _ := db2.GetString("key2") // "value2"
```

## Performance Considerations

### Transaction Size

Transactions buffer all operations in memory until commit:

```go
// Small transaction - efficient
tx := db.Begin()
for i := 0; i < 100; i++ {
    tx.PutString(fmt.Sprintf("key%d", i), "value")
}
tx.Commit()

// Very large transaction - uses more memory
tx = db.Begin()
for i := 0; i < 1000000; i++ {
    tx.PutString(fmt.Sprintf("key%d", i), "value")
}
tx.Commit() // Large memory usage during commit
```

**Recommendation**: For bulk operations, consider batching into multiple transactions of moderate size (e.g., 1,000-10,000 operations per transaction).

### Validation Overhead

All operations are validated before any writes:

```go
tx := db.Begin()
for i := 0; i < 1000; i++ {
    tx.PutString(fmt.Sprintf("key%d", i), "value")
}
// Commit validates all 1000 keys before writing
err := tx.Commit()
```

### WAL Overhead

Each transaction generates WAL entries:

- `BeginTx` marker
- All operations (Put, Update, Delete)
- `CommitTx` marker

The WAL is synced to disk on commit for durability.

### Concurrency

Transactions acquire a lock for the entire commit duration:

```go
// Transaction 1
tx1 := db.Begin()
tx1.PutString("key1", "value1")
tx1.Commit() // Locks database during commit

// Transaction 2 (waits for tx1)
tx2 := db.Begin()
tx2.PutString("key2", "value2")
tx2.Commit() // Waits if tx1 is committing
```

**Best Practice**: Keep transactions short to minimize lock contention.

## Examples

### Bank Transfer

```go
// Atomic money transfer between accounts
func transfer(db *skv.SKV, from, to string, amount float64) error {
    tx := db.Begin()
    
    // Deduct from source
    balanceFrom, _ := db.GetString(from)
    newFrom := parseBalance(balanceFrom) - amount
    tx.UpdateString(from, fmt.Sprintf("%.2f", newFrom))
    
    // Add to destination
    balanceTo, _ := db.GetString(to)
    newTo := parseBalance(balanceTo) + amount
    tx.UpdateString(to, fmt.Sprintf("%.2f", newTo))
    
    // Log transaction
    tx.PutString(
        fmt.Sprintf("transfer:%d", time.Now().Unix()),
        fmt.Sprintf("%s->%s:%.2f", from, to, amount),
    )
    
    // Commit atomically
    return tx.Commit()
}
```

### Bulk User Creation

```go
// Create multiple users atomically
func createUsers(db *skv.SKV, users []User) error {
    tx := db.Begin()
    
    for _, user := range users {
        key := fmt.Sprintf("user:%s", user.ID)
        data, _ := json.Marshal(user)
        
        if err := tx.Put([]byte(key), data); err != nil {
            return err
        }
    }
    
    // All users created or none
    return tx.Commit()
}
```

### Conditional Update

```go
// Update only if condition is met
func updateIfMatch(db *skv.SKV, key, expected, newValue string) error {
    // Check current value
    current, err := db.GetString(key)
    if err != nil {
        return err
    }
    
    if current != expected {
        return fmt.Errorf("value mismatch")
    }
    
    // Update atomically
    tx := db.Begin()
    tx.UpdateString(key, newValue)
    return tx.Commit()
}
```

### Batch Delete with Logging

```go
// Delete multiple keys and log the operation
func batchDelete(db *skv.SKV, keys []string) error {
    tx := db.Begin()
    
    // Delete all keys
    for _, key := range keys {
        if err := tx.DeleteString(key); err != nil {
            return err
        }
    }
    
    // Log the operation
    logEntry := fmt.Sprintf("deleted %d keys at %s", 
        len(keys), time.Now().Format(time.RFC3339))
    tx.PutString(
        fmt.Sprintf("log:delete:%d", time.Now().Unix()),
        logEntry,
    )
    
    // All deletes + log entry or nothing
    return tx.Commit()
}
```

### Safe Initialization

```go
// Initialize database schema atomically
func initDatabase(db *skv.SKV) error {
    tx := db.Begin()
    
    // Create schema version
    tx.PutString("schema:version", "1.0")
    
    // Create default settings
    tx.PutString("settings:max_users", "1000")
    tx.PutString("settings:timeout", "30")
    
    // Create admin user
    admin := `{"id":"admin","role":"superuser"}`
    tx.PutString("user:admin", admin)
    
    // All schema setup or nothing
    return tx.Commit()
}
```

## Best Practices

### ✅ DO

- Use transactions for multi-step operations that must be atomic
- Keep transactions short to minimize lock contention
- Validate business logic before starting a transaction
- Use appropriate operation types (Put/Update/Delete)
- Handle commit errors gracefully

### ❌ DON'T

- Don't create extremely large transactions (millions of operations)
- Don't hold transactions open for long periods
- Don't ignore commit errors
- Don't reuse committed or rolled back transactions
- Don't perform expensive computations inside transaction commits

### Example: Good vs Bad

**❌ Bad - Long-running transaction**
```go
tx := db.Begin()
for i := 0; i < 1000; i++ {
    // Expensive computation inside transaction
    data := expensiveComputation(i)
    tx.PutString(fmt.Sprintf("key%d", i), data)
}
tx.Commit()
```

**✅ Good - Prepare data first**
```go
// Prepare all data first
items := make(map[string]string)
for i := 0; i < 1000; i++ {
    data := expensiveComputation(i)
    items[fmt.Sprintf("key%d", i)] = data
}

// Then commit quickly
tx := db.Begin()
for key, value := range items {
    tx.PutString(key, value)
}
tx.Commit()
```

## Summary

SKV's atomic transactions provide:

- **ACID guarantees** for multi-operation workflows
- **Simple API** with Begin/Commit/Rollback
- **Automatic recovery** from crashes via WAL
- **Validation** to ensure database integrity
- **Isolation** until commit

Use transactions when you need to ensure that multiple related operations either all succeed or all fail together, maintaining database consistency even in the face of errors or crashes.
