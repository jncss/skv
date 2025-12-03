# Basic Examples

Learn the fundamentals of SKV with these introductory examples.

## Examples

### `basic_usage.go`
Complete introduction covering:
- Opening and closing databases
- String operations (PutString, GetString, UpdateString, DeleteString)
- Key existence checking (HasString)
- Counting keys (Count)
- Listing all keys (KeysString)
- Default values (GetOrDefaultString)

**Run:**
```bash
go run basic_usage.go
```

### `put_vs_update.go`
Understanding insert vs update semantics:
- `Put()` - Creates new keys (fails if exists)
- `Update()` - Modifies existing keys (fails if not found)
- `ErrKeyExists` and `ErrKeyNotFound` errors

**Run:**
```bash
go run put_vs_update.go
```

## Next Steps

After mastering these basics, explore:
- **02-advanced/** - Batch operations, iteration, and maintenance
- **03-concurrent/** - Thread-safe concurrent access
- **04-usecases/** - Real-world application patterns
