# 01 - Basic Operations

Learn the fundamental CRUD operations in SKV.

## Examples

### `basic_usage/`
Demonstrates core database operations:
- **Put**: Creating new key-value pairs with `PutString()` and `Put()`
- **Get**: Retrieving values with `GetString()` and `Get()`
- **Update**: Modifying existing values with `UpdateString()` and `Update()`
- **Delete**: Removing keys with `DeleteString()` and `Delete()`
- **Exists**: Checking key existence with `HasString()` and `Has()`
- **Count**: Getting total number of keys with `Count()`
- **GetOrDefault**: Safe retrieval with default values
- **Error Handling**: Proper handling of `ErrKeyExists` and `ErrKeyNotFound`

**Run:**
```bash
cd examples/01-basics/basic_usage
go run basic_usage.go
```

### `put_vs_update/`
Explains the critical difference between Put and Update:
- **Put()**: Only works for NEW keys (returns `ErrKeyExists` if key exists)
- **Update()**: Only works for EXISTING keys (returns `ErrKeyNotFound` if key doesn't exist)
- **Best Practices**: Patterns for handling both operations safely
- **Clear()**: Removing all keys from the database

**Run:**
```bash
cd examples/01-basics/put_vs_update
go run put_vs_update.go
```

## Key Concepts

### Put vs Update
```go
// Put - Create NEW key (fails if exists)
err := db.PutString("key", "value")
if err == skv.ErrKeyExists {
    // Key already exists, use Update instead
}

// Update - Modify EXISTING key (fails if not found)
err := db.UpdateString("key", "new_value")
if err == skv.ErrKeyNotFound {
    // Key doesn't exist, use Put instead
}
```

### Error Handling
```go
// Check key existence first
if db.HasString("key") {
    db.UpdateString("key", "value")
} else {
    db.PutString("key", "value")
}

// Or handle errors explicitly
err := db.PutString("key", "value")
if err == skv.ErrKeyExists {
    db.UpdateString("key", "value")
}
```

### String vs Byte Operations
```go
// String convenience functions (most common)
db.PutString("name", "Alice")
value, _ := db.GetString("name")

// Byte operations (for binary data)
db.Put([]byte("key"), []byte{1, 2, 3})
data, _ := db.Get([]byte("key"))
```

## Next Steps

After mastering the basics, explore:
- **02-advanced**: Batch operations, iteration, and maintenance
- **03-concurrent**: Thread-safe concurrent operations
- **04-usecases**: Real-world application examples
- **05-backup**: Backup and restore functionality
