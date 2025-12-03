# Advanced Examples

Master advanced features for optimized and efficient database operations.

## Examples

### `batch_operations.go`
Efficient multi-key operations:
- `PutBatch()` - Insert multiple keys atomically
- `GetBatch()` - Retrieve multiple keys at once
- Atomic all-or-nothing behavior
- Performance benefits over individual operations

**Use when:** Inserting or retrieving many keys at once

**Run:**
```bash
go run batch_operations.go
```

### `iteration.go`
Process all key-value pairs:
- `ForEach()` / `ForEachString()` - Iterate over all data
- Calculate aggregates (sum, count, average)
- Filter and search data
- Early termination on specific conditions
- Handle binary data

**Use when:** Need to process all records, generate reports, or migrate data

**Run:**
```bash
go run iteration.go
```

### `maintenance.go`
Database optimization and monitoring:
- `Verify()` - Get database statistics
- `Compact()` - Reclaim space from deleted/updated records
- `CloseWithCompact()` - Compact on shutdown
- File size monitoring
- Understanding storage growth

**Use when:** Running production databases that accumulate updates/deletes

**Run:**
```bash
go run maintenance.go
```

## Performance Tips

1. **Batch operations** are faster than individual operations for multiple keys
2. **Compact regularly** if you have many updates/deletes (e.g., >30% deleted)
3. **Use ForEach** instead of Keys() + Get() for processing all data
4. **Monitor with Verify()** to understand database health

## Related Examples

- **01-basics/** - Foundation concepts
- **03-concurrent/** - Thread-safe operations
- **04-usecases/** - Real-world patterns
