# Cursors in SKV

SKV provides a powerful cursor system for ordered traversal of records, supporting both primary keys and secondary indexes. Cursors enable efficient iteration, range queries, and batch processing with minimal memory overhead.

## Table of Contents

- [Overview](#overview)
- [Basic Usage](#basic-usage)
- [Primary Key Cursors](#primary-key-cursors)
- [Secondary Index Cursors](#secondary-index-cursors)
- [Cursor Navigation](#cursor-navigation)
- [Iteration Patterns](#iteration-patterns)
- [Range Queries](#range-queries)
- [Performance](#performance)
- [Best Practices](#best-practices)
- [Limitations](#limitations)

## Overview

A cursor is a stateful iterator that allows you to traverse records in a defined order. Cursors in SKV:

- **Ordered Traversal**: Always iterate in lexicographic (sorted) order
- **Range Support**: Query records within a specific key range
- **Bidirectional**: Support both forward and reverse iteration
- **Prefix Matching**: Filter records by key prefix
- **Thread-Safe Initialization**: Snapshot keys safely during creation
- **Memory Efficient**: Load values on demand during iteration

## Basic Usage

```go
// Open database
db, _ := skv.Open("data.skv")
defer db.Close()

// Create a cursor for all records
cursor := db.AllCursor(false) // forward iteration
defer cursor.Close()

// Iterate through all records
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process key/value
}
```

## Primary Key Cursors

Primary key cursors iterate over records ordered by their primary keys.

### All Records

```go
// Forward iteration (a → z)
cursor := db.AllCursor(false)

// Reverse iteration (z → a)
cursor := db.AllCursor(true)
```

### Prefix-Based

Find all keys starting with a specific prefix:

```go
cursor := db.PrefixCursor([]byte("user:"), false)
// Iterates: user:1, user:2, user:3, etc.
```

### Range Queries

Iterate over keys within a range (inclusive on both ends):

```go
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("book:100"),
    To:   []byte("book:199"),
})
// Iterates: book:100, book:101, ..., book:199
```

### Custom Options

```go
cursor := db.NewCursor(&skv.CursorOptions{
    From:    []byte("start"),
    To:      []byte("end"),
    Reverse: true, // Iterate backwards
})
```

## Secondary Index Cursors

Index cursors traverse records ordered by indexed field values.

### Prerequisites

First, create a secondary index:

```go
db.CreateIndex("by_email", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(user.Email)
})
```

### All Index Entries

```go
cursor, _ := db.AllIndexCursor("by_email", false)
// Iterates through all records, ordered by email
```

**Note**: Index cursors return records ordered by the indexed field value, then by primary key for stable ordering when multiple records share the same index value.

### Prefix on Indexed Field

```go
cursor, _ := db.PrefixIndexCursor("by_email", []byte("admin@"), false)
// All emails starting with "admin@"
```

### Range on Indexed Field

```go
cursor, _ := db.NewIndexCursor("by_email", &skv.CursorOptions{
    From: []byte("alice@"),
    To:   []byte("bob@"),
})
// All emails between alice@ and bob@ (inclusive)
```

### Multiple Records with Same Index Value

Index cursors fully support multiple records sharing the same secondary key value:

```go
// Multiple products in the same category
db.Put([]byte("prod1"), []byte(`{"category":"electronics",...}`))
db.Put([]byte("prod2"), []byte(`{"category":"electronics",...}`))
db.Put([]byte("prod3"), []byte(`{"category":"electronics",...}`))

db.CreateIndex("by_category", extractCategory)

// Cursor will iterate through all 3 products
cursor, _ := db.PrefixIndexCursor("by_category", []byte("electronics"), false)
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process each electronics product
    // key is the primary key (prod1, prod2, prod3)
}
```

## Cursor Navigation

### Next()

Advance to the next record and return key/value:

```go
key, value, err := cursor.Next()
if err == io.EOF {
    // No more records
}
```

### Seek()

Jump to a specific position:

```go
cursor.Seek([]byte("target-key"))
key, value, _ := cursor.Next() // Returns first key >= "target-key"
```

For reverse cursors, Seek positions at the first key <= the target.

### Reset()

Go back to the beginning:

```go
cursor.Reset()
// Next iteration starts from the first record again
```

### Key() and Value()

Access current record without advancing:

```go
cursor.Next() // Advance first

key, _ := cursor.Key()     // Get current key
value, _ := cursor.Value() // Get current value
```

### Position Checks

```go
if cursor.IsFirst() {
    // At the first record
}

if cursor.IsLast() {
    // At the last record
}

if cursor.IsValid() {
    // Cursor has a valid current position
}
```

### Count()

Get total number of records in cursor:

```go
count := cursor.Count()
```

## Iteration Patterns

### Manual Iteration

```go
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    // Process key/value
}
```

### ForEach

Iterate with a callback function:

```go
err := cursor.ForEach(func(key, value []byte) bool {
    // Process key/value
    return true // continue iteration
    // return false to stop early
})
```

### Collect All

Retrieve all key-value pairs into slices:

```go
keys, values, err := cursor.Collect()
// keys[i] corresponds to values[i]
```

### Keys Only

Get just the keys:

```go
keys := cursor.Keys()
```

## String Convenience Functions

For easier string handling, SKV provides string-based convenience functions for all cursor operations. These avoid the need for explicit byte slice conversions.

### Creating Cursors with Strings

```go
// Primary key cursors with string range
cursor := db.NewCursorString("start", "end", false)

// Index cursors with string range
cursor, _ := db.NewIndexCursorString("by_email", "alice@", "bob@", false)

// Prefix cursors with string
cursor := db.PrefixCursorString("user:", false)
cursor, _ := db.PrefixIndexCursorString("by_email", "admin@", false)

// All cursors (string variants for consistency)
cursor := db.AllCursorString(false)
cursor, _ := db.AllIndexCursorString("by_category", true)
```

### Reading Data as Strings

```go
// Get current key/value as strings
key := cursor.KeyString()
value, err := cursor.ValueString()

// Iterate with strings
for {
    key, value, err := cursor.NextString()
    if err == io.EOF {
        break
    }
    // key and value are strings
    fmt.Printf("%s = %s\n", key, value)
}
```

### Navigation with Strings

```go
// Seek to a string key
cursor.SeekString("target-key")

// Check prefix with string
if cursor.HasPrefixString("user:") {
    // Current key starts with "user:"
}
```

### Collecting as Strings

```go
// Get all keys as strings
keys := cursor.KeysString()

// Collect all key-value pairs as strings
keys, values, err := cursor.CollectString()
for i := range keys {
    fmt.Printf("%s = %s\n", keys[i], values[i])
}
```

### Complete Example

```go
// Create cursor with string range
cursor := db.NewCursorString("user:100", "user:199", false)
defer cursor.Close()

// Iterate with strings
err := cursor.ForEach(func(key, value []byte) bool {
    // Convert once for all operations
    keyStr := string(key)
    valueStr := string(value)
    
    fmt.Printf("%s = %s\n", keyStr, valueStr)
    return true
})

// Or use string iterators directly
cursor.Reset()
for {
    key, value, err := cursor.NextString()
    if err == io.EOF {
        break
    }
    fmt.Printf("%s = %s\n", key, value)
}
```

## Range Queries

### Inclusive Ranges

Both `From` and `To` are inclusive:

```go
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("a"),
    To:   []byte("c"),
})
// Returns: a, b, c (if they exist)
```

### Open-Ended Ranges

```go
// From key onwards
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("m"),
    // To: nil (default)
})

// Up to key
cursor := db.NewCursor(&skv.CursorOptions{
    // From: nil (default)
    To: []byte("m"),
})
```

### Reverse Ranges

```go
cursor := db.NewCursor(&skv.CursorOptions{
    From:    []byte("a"),
    To:      []byte("z"),
    Reverse: true,
})
// Iterates: z, y, x, ..., a
```

## Performance

### Time Complexity

- **Cursor Creation**: O(n log n) where n is total records (for sorting)
- **Next()**: O(1) amortized, O(log n) for value lookup
- **Seek()**: O(n) linear search through sorted keys
- **Count()**: O(1)

### Memory Usage

- **Cursor Object**: O(n) for storing sorted key strings
- **Values**: Loaded on demand (O(1) per Next() call)
- **Range Filtering**: Applied during cursor creation

### Optimization Tips

1. **Use Prefix Cursors** instead of filtering manually
2. **Range Queries** filter early, reducing memory
3. **Close Cursors** when done to release resources
4. **Avoid Count()** if not needed; it's precomputed but unnecessary work

## Best Practices

### 1. Always Close Cursors

```go
cursor := db.AllCursor(false)
defer cursor.Close() // Clean up resources
```

### 2. Handle Errors

```go
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break // Normal termination
    }
    if err != nil {
        return err // Handle other errors
    }
    // Process
}
```

### 3. Use Appropriate Iteration Method

- **Manual loop**: When you need fine control
- **ForEach**: For simple processing
- **Collect**: When you need all data in memory

### 4. Avoid Long-Lived Cursors

Cursors snapshot keys at creation. For long-running operations:

```go
// Create cursor
cursor := db.NewCursor(opts)
// Process quickly
cursor.Close()

// Don't keep cursor open for extended periods
```

### 5. Index Cursors for Non-Primary Ordering

```go
// Don't iterate all records and filter
// ❌ BAD
cursor := db.AllCursor(false)
for {
    key, value, _ := cursor.Next()
    var user User
    json.Unmarshal(value, &user)
    if user.Email == target {
        // Found it
    }
}

// Use index cursor
// ✅ GOOD
cursor, _ := db.PrefixIndexCursor("by_email", []byte(target), false)
key, value, _ := cursor.Next()
```

## Limitations

### 1. Snapshot Consistency

Cursors capture a snapshot of keys at creation time:

```go
cursor := db.AllCursor(false)

// New records added here won't appear in cursor
db.Put([]byte("new-key"), data)

// Cursor won't see "new-key"
```

### 2. Memory Overhead

Cursors store all keys in memory. For very large databases:

```go
// May use significant memory
cursor := db.AllCursor(false) // Loads all keys

// Better: Use ranges
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("batch:1000"),
    To:   []byte("batch:1999"),
})
```

### 3. No Concurrent Modification Protection

Changes during iteration aren't reflected:

```go
cursor := db.AllCursor(false)

for {
    key, _, err := cursor.Next()
    if err == io.EOF {
        break
    }
    
    // ⚠️ Deleting while iterating
    db.Delete(key) // This works, but cursor still has the key
}
```

## Examples

See the [examples/09-cursors](../examples/09-cursors/) directory for complete working examples demonstrating:

- All cursor types (primary, index, prefix, range)
- Forward and reverse iteration
- Seek, Reset, and navigation methods
- ForEach and Collect patterns
- Practical use cases (aggregation, batch processing)

## See Also

- [Secondary Indexes](./INDEXES.md) - Index creation and management
- [README](./README.md) - General SKV documentation
- [Testing](./TESTING.md) - Test coverage and examples
