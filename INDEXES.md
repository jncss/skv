# Secondary Indexes

SKV supports secondary indexes for fast lookups by alternative keys, beyond the primary key.

## Overview

Secondary indexes allow you to efficiently query records by fields other than the primary key. For example, you can look up users by email instead of just by user ID.

## Features

- **Custom key extraction**: Define your own function to extract secondary keys from values
- **Automatic maintenance**: Indexes are updated automatically on Put/Update/Delete
- **O(1) lookups**: Hash-based index for constant-time access
- **In-memory**: Fast access with minimal overhead
- **Persistence**: Save/load indexes to JSON files
- **Thread-safe**: Safe for concurrent use

## Basic Usage

### Creating an Index

```go
db, _ := skv.Open("users.skv")

// Create index by email
err := db.CreateIndex("by_email", func(data []byte) []byte {
    var user struct {
        Email string `json:"email"`
    }
    json.Unmarshal(data, &user)
    return []byte(user.Email)  // Extract secondary key
})
```

### Looking Up by Index

```go
// Get first matching user by email
data, err := db.GetByIndex("by_email", []byte("user@example.com"))

// Or get all matching records (if secondary key is not unique)
values, err := db.GetAllByIndex("by_email", []byte("shared@example.com"))
for _, data := range values {
    var user User
    json.Unmarshal(data, &user)
    fmt.Printf("Found user: %s\n", user.Name)
}

// Using string convenience methods
data, err := db.GetByIndexString("by_email", "user@example.com")
```

### Checking if Key Exists

```go
if db.HasIndex("by_email", []byte("user@example.com")) {
    fmt.Println("Email exists in index")
}

// String version
if db.HasIndexString("by_email", "user@example.com") {
    fmt.Println("Email exists")
}
```

## Index Management

### Listing Indexes

```go
indexes := db.ListIndexes()
for _, name := range indexes {
    size := db.IndexSize(name)
    fmt.Printf("%s: %d entries\n", name, size)
}
```

### Dropping an Index

```go
err := db.DropIndex("by_email")
```

### Rebuilding an Index

Useful if an index becomes out of sync or needs to be refreshed:

```go
err := db.RebuildIndex("by_email")
```

## Persistence

### Saving an Index

```go
err := db.SaveIndex("by_email", "email_index.json")
```

The index is saved as a JSON file with the mapping of secondary keys to primary keys.

### Loading an Index

```go
// Define the same key extraction function
keyFunc := func(data []byte) []byte {
    var user struct { Email string `json:"email"` }
    json.Unmarshal(data, &user)
    return []byte(user.Email)
}

// Load index
err := db.LoadIndex("by_email", "email_index.json", keyFunc)
```

**Note**: You must provide the same key extraction function when loading.

## Automatic Index Updates

Indexes are automatically maintained:

### On Put

When you insert a new record, the secondary key is automatically added to all indexes:

```go
user := User{Email: "new@example.com", Name: "New User"}
data, _ := json.Marshal(user)
db.Put([]byte("user123"), data)
// "new@example.com" → "user123" added to by_email index
```

### On Update

When you update a record, the old secondary key is removed and the new one is added:

```go
updated := User{Email: "updated@example.com", Name: "New User"}
data, _ := json.Marshal(updated)
db.Update([]byte("user123"), data)
// "new@example.com" removed from index
// "updated@example.com" → "user123" added to index
```

### On Delete

When you delete a record, its secondary key is removed from all indexes:

```go
db.Delete([]byte("user123"))
// "updated@example.com" removed from index
```

## Advanced Usage

### Multiple Indexes

You can create multiple indexes on the same database:

```go
// Index by email
db.CreateIndex("by_email", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(user.Email)
})

// Index by username
db.CreateIndex("by_username", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(user.Username)
})

// Index by age (as string)
db.CreateIndex("by_age", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(strconv.Itoa(user.Age))
})
```

### Conditional Indexing

You can choose which records to index by returning `nil`:

```go
// Only index active users
db.CreateIndex("active_users", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    if user.Active {
        return []byte(user.Email)
    }
    return nil  // Don't index inactive users
})
```

### Binary Data Indexing

Indexes work with binary data too:

```go
// Index images by file hash
db.CreateIndex("by_hash", func(data []byte) []byte {
    // First 32 bytes are the SHA256 hash
    if len(data) >= 32 {
        return data[:32]
    }
    return nil
})
```

## Performance Considerations

### Memory Usage

- Indexes are stored in memory
- Each index entry uses: `len(secondary_key) + len(primary_key) + overhead`
- For 1 million users with 30-byte emails and 10-byte IDs: ~40 MB per index

### Lookup Performance

- **Index lookup**: O(1) - hash map lookup
- **Record retrieval**: O(1) - cache lookup by primary key
- **Total**: O(1) for complete operation

### Index Creation

- **Initial creation**: O(n) where n = number of records
- **Must read all records** to build index
- For large databases, this can take time

### Maintenance Overhead

- **Put**: O(1) per index - adds one entry to each index
- **Update**: O(1) per index - removes old entry, adds new entry
- **Delete**: O(k) where k = number of indexes - must scan each index

## Limitations

### Current Limitations

1. **In-memory only**: Indexes are not persisted automatically
   - Use `SaveIndex`/`LoadIndex` for persistence
   - Indexes must be rebuilt on database open

2. **No range queries**: Indexes only support exact key lookups
   - Cannot query for "all users with age > 30"
   - For range queries on indexed fields, use index cursors with range options
   - For complex queries, use `ForEach` or iterate manually

3. **No partial matches**: Cannot search for partial keys
   - "alice@" won't match "alice@example.com"
   - For text search, consider external full-text search

### Duplicate Secondary Keys

**Supported!** Multiple records can have the same secondary key value:

```go
// Multiple users with same category
db.Put([]byte("user1"), []byte(`{"category":"admin"}`))
db.Put([]byte("user2"), []byte(`{"category":"admin"}`))
db.Put([]byte("user3"), []byte(`{"category":"admin"}`))

db.CreateIndex("by_category", extractCategory)

// Get first match
data, _ := db.GetByIndex("by_category", []byte("admin"))

// Get all matches
allAdmins, _ := db.GetAllByIndex("by_category", []byte("admin"))
// Returns all 3 users

// Or use index cursor for ordered iteration
cursor, _ := db.PrefixIndexCursor("by_category", []byte("admin"), false)
for {
    key, value, err := cursor.Next()
    if err == io.EOF {
        break
    }
    // Process each admin user
}
```

### Workarounds
```go
// Index age as "age_030" so you can iterate by prefix
return []byte(fmt.Sprintf("age_%03d", user.Age))
```

## Error Handling

```go
// Index already exists
err := db.CreateIndex("existing", keyFunc)
if err != nil {
    // Handle error: index already exists
}

// Index not found
data, err := db.GetByIndex("nonexistent", []byte("key"))
if err != nil {
    // Handle error: index not found
}

// Key not found in index
data, err := db.GetByIndex("by_email", []byte("unknown@example.com"))
if err == skv.ErrKeyNotFound {
    // Key doesn't exist
}
```

## Best Practices

1. **Create indexes early**: Before inserting data, or rebuild after creation
2. **Use meaningful names**: "by_email", "by_username", not "index1", "index2"
3. **Index selectively**: Only create indexes you actually use
4. **Save indexes for persistence**: Use `SaveIndex` if you want indexes to survive restarts
5. **Rebuild on corruption**: Use `RebuildIndex` if index gets out of sync
6. **Consider memory**: Each index uses memory proportional to record count

## Examples

See `examples/08-indexes` for a complete working example demonstrating:
- Creating indexes
- Looking up by secondary keys
- Automatic index updates
- Saving and loading indexes
- Managing multiple indexes

## See Also

- [API Reference](README.md#api-reference)
- [Examples](examples/)
- [Testing Documentation](TESTING.md)
