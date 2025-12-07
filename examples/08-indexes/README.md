# Secondary Indexes Example

This example demonstrates how to use secondary indexes in SKV for efficient lookups by alternative keys.

## Features Demonstrated

- Creating indexes with custom key extraction functions
- Looking up records by secondary keys (email)
- Checking if secondary keys exist
- Automatic index updates on Put/Update/Delete
- Saving and loading indexes for persistence
- Rebuilding indexes
- Listing all indexes

## Running the Example

```bash
cd examples/08-indexes
go run main.go
```

## Key Concepts

### Creating an Index

```go
db.CreateIndex("by_email", func(data []byte) []byte {
    var user User
    json.Unmarshal(data, &user)
    return []byte(user.Email)  // Extract email as secondary key
})
```

### Looking Up by Index

```go
// Get user by email instead of ID
data, err := db.GetByIndexString("by_email", "alice@example.com")
```

### Automatic Index Maintenance

Indexes are automatically updated when you:
- **Put** a new record → secondary key added to index
- **Update** a record → old secondary key removed, new one added
- **Delete** a record → secondary key removed from index

### Persistence

Save indexes to JSON files for persistence across sessions:

```go
// Save
db.SaveIndex("by_email", "email_index.json")

// Load (in a new session)
db.LoadIndex("by_email", "email_index.json", extractEmailFunc)
```

## Use Cases

Secondary indexes are useful for:

- **User lookup by email** (instead of just ID)
- **Product lookup by SKU** (instead of internal ID)
- **Document lookup by title** (instead of document ID)
- **Session lookup by token** (instead of session ID)
- **Any scenario where you need fast lookups by multiple keys**

## Performance

- Index lookups are O(1) (hash map)
- Index creation is O(n) where n = number of records
- Index maintenance on Put/Update/Delete is O(1)
- Indexes are in-memory for fast access

## Cleanup

The example creates:
- `users.skv` - Database file
- `email_index.json` - Saved index file

Both are cleaned up automatically when you run the example again.
