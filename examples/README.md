# SKV Examples

Comprehensive examples demonstrating all features of SKV with up-to-date code and best practices.

## 📚 Examples Overview

### [01-basics](01-basics/) - Fundamental Operations
Learn the core CRUD operations and error handling.

**Examples:**
- `basic_usage/` - Put, Get, Update, Delete, Has, Count, GetOrDefault
- `put_vs_update/` - Understanding the difference between Put and Update

**Key Topics:** Error handling, string vs byte operations, key existence checks

---

### [02-advanced](02-advanced/) - Advanced Features
Efficient data management with batch operations and maintenance.

**Examples:**
- `batch_operations/` - PutBatch, GetBatch for multiple keys at once
- `iteration/` - ForEach to process all key-value pairs
- `maintenance/` - Verify, Compact, CloseWithCompact

**Key Topics:** Performance optimization, database health, space management

---

### [03-concurrent](03-concurrent/) - Thread-Safe Operations
Safe concurrent access from multiple goroutines.

**Examples:**
- `concurrent/` - Concurrent reads, writes, updates, and mixed operations

**Key Topics:** Thread safety, mutex locks, concurrent patterns, race detection

---

### [04-usecases](04-usecases/) - Real-World Applications
Practical examples showing common use cases.

**Examples:**
- Session storage
- Application configuration
- Simple cache
- Job queue / task storage
- Feature flags
- User preferences

**Key Topics:** Practical patterns, production-ready code, best practices

---

### [05-backup](05-backup/) - Data Protection
Backup and restore functionality for data safety.

**Examples:**
- `backup_restore/` - Creating backups, restoring data, optimization

**Key Topics:** JSON backups, disaster recovery, timestamped backups, compaction

---

### [06-fileformat](06-fileformat/) - Binary Format
Understanding the SKV file format internals.

**Examples:**
- `demo/` - File format inspection, record structure, type bytes

**Key Topics:** Binary format, header structure, record types, space reuse

---

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/jncss/skv
cd skv/examples

# Run basic usage example
cd 01-basics/basic_usage
go run basic_usage.go

# Try other examples
cd ../../02-advanced/batch_operations
go run batch_operations.go
```

## 📖 Learning Path

**For beginners:**
1. **01-basics/basic_usage** - Start here to learn fundamental operations
2. **01-basics/put_vs_update** - Understand Put vs Update
3. **04-usecases** - See real-world applications
4. **02-advanced/batch_operations** - Learn efficient batch operations

**For advanced users:**
1. **02-advanced/iteration** - Process all data with ForEach
2. **02-advanced/maintenance** - Keep database optimized
3. **03-concurrent** - Safe multi-threaded access
4. **05-backup** - Protect your data
5. **06-fileformat** - Understand internals

## 🎯 Common Patterns

### Basic CRUD
```go
db, _ := skv.Open("mydata")
defer db.Close()

// Create
db.PutString("key", "value")

// Read
value, _ := db.GetString("key")

// Update
db.UpdateString("key", "new_value")

// Delete
db.DeleteString("key")
```

### Batch Operations
```go
// Insert multiple keys at once
users := map[string]string{
    "user:1": "Alice",
    "user:2": "Bob",
}
db.PutBatchString(users)
```

### Iteration
```go
// Process all key-value pairs
db.ForEachString(func(key, value string) error {
    fmt.Printf("%s: %s\n", key, value)
    return nil
})
```

### Maintenance
```go
// Check database health
stats, _ := db.Verify()
if stats.WastedSpace > threshold {
    db.Compact()
}
```

### Backup
```go
// Create backup
db.Backup("backup.json")

// Restore from backup
db.Restore("backup.json")
```

## 🛠️ Requirements

- **Go**: 1.24.0 or higher
- **Dependencies**: Only `github.com/jncss/skv` (no external deps)
- **OS**: Linux, macOS, Windows, BSD

## 📝 Running Examples

Each example is self-contained and can be run independently:

```bash
cd examples/<category>/<example>
go run *.go
```

## 🧪 Testing with Examples

Some examples create data files in a `data/` subdirectory. These are temporary and can be safely deleted.

```bash
# Clean up all example data
find examples -type d -name "data" -exec rm -rf {} +
```

## 💡 Tips

- **Read the README** in each directory for detailed explanations
- **Run examples in order** to build understanding progressively
- **Modify and experiment** - all examples are meant to be educational
- **Check error handling** - examples show proper error patterns
- **Use race detector** with concurrent examples: `go run -race concurrent.go`

## 🔗 Additional Resources

- **Main README**: [../README.md](../README.md) - Project overview and API reference
- **Testing Guide**: [../TESTING.md](../TESTING.md) - Comprehensive test suite
- **GitHub**: [github.com/jncss/skv](https://github.com/jncss/skv)

## 📊 Performance Reference

From project benchmarks:
- **Writes**: ~750 inserts/sec
- **Reads**: ~270,000 reads/sec (cached)
- **Updates**: ~365 updates/sec
- **Concurrent**: ~1,900 ops/sec (10 goroutines)
- **Compaction**: ~37% average size reduction

## ❓ Need Help?

1. Check the example code and README in each directory
2. Read the main project README for API reference
3. Look at test files (`*_test.go`) for more examples
4. Open an issue on GitHub for questions or bugs
