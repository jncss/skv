# Cursors Example

This example demonstrates SKV's cursor system for ordered iteration and range queries.

## Features Demonstrated

1. **All Cursor**: Iterate through all records in sorted order
2. **Prefix Cursor**: Find all keys with a specific prefix
3. **Range Cursor**: Query records within a key range (from-to, inclusive)
4. **Index Cursor**: Iterate through secondary index entries
5. **Reverse Iteration**: Traverse records in descending order
6. **ForEach Method**: Process all records with a callback function
7. **Collect Method**: Retrieve all keys/values at once
8. **Seek Operation**: Jump to a specific position in the cursor

## Running the Example

```bash
cd examples/09-cursors
go run main.go
```

## Key Concepts

### Primary Key Cursors

Cursors on primary keys iterate through records in lexicographic (sorted) order:

```go
// All records
cursor := db.AllCursor(false) // forward
cursor := db.AllCursor(true)  // reverse

// With prefix
cursor := db.PrefixCursor([]byte("BOOK-"), false)

// With range
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("start"),
    To:   []byte("end"),
})
```

### Secondary Index Cursors

Cursors on indexes iterate through records ordered by the indexed field:

```go
// All records in index
cursor, _ := db.AllIndexCursor("by_category", false)

// With prefix on indexed field
cursor, _ := db.PrefixIndexCursor("by_category", []byte("electronics"), false)

// With range on indexed field
cursor, _ := db.NewIndexCursor("by_category", &skv.CursorOptions{
    From: []byte("books"),
    To:   []byte("furniture"),
})
```

### Iteration Patterns

```go
// Pattern 1: Manual iteration
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process key/value
}

// Pattern 2: ForEach
cursor.ForEach(func(key, value []byte) bool {
    // Process key/value
    return true // continue, false to stop
})

// Pattern 3: Collect all
keys, values, _ := cursor.Collect()
```

### Navigation Methods

```go
cursor.Seek([]byte("specific-key"))  // Jump to position
cursor.Reset()                       // Go back to start
cursor.IsFirst()                     // Check if at first record
cursor.IsLast()                      // Check if at last record
```

## Use Cases

1. **Listing Records**: Display all records in sorted order
2. **Pagination**: Use range queries with cursors for efficient pagination
3. **Prefix Search**: Find all keys starting with a pattern
4. **Batch Processing**: Process records in order with ForEach
5. **Aggregation**: Calculate totals, counts using cursor iteration
6. **Index Scanning**: Efficiently query by secondary keys

## Performance Notes

- Cursors capture a snapshot of keys at creation time
- Thread-safe initialization (acquires RLock)
- Sorted traversal is efficient (O(n log n) for initial sort)
- Range filtering happens during cursor creation
- No additional I/O during iteration (values loaded on demand)
