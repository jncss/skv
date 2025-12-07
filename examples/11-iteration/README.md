# Iteration Methods Example

This example demonstrates the different ways to iterate over records in SKV, comparing their performance characteristics and use cases.

## Methods Compared

### 1. ForEach - Fast, Unordered Iteration

```go
db.ForEachString(func(key, value string) error {
    fmt.Printf("%s = %s\n", key, value)
    return nil
})
```

**Characteristics:**
- ✅ **Memory efficient**: O(1) for values (reads on-demand from disk)
- ❌ **No order guarantee**: Iterates over Go map (undefined order)
- ✅ **Fast**: Direct iteration over cache
- **Best for**: Processing all records when order doesn't matter

### 2. NewCursor - Ordered Iteration

```go
cursor := db.NewCursor(nil)
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err != nil {
        break // End of iteration
    }
    fmt.Printf("%s = %s\n", string(key), string(value))
}
```

**Characteristics:**
- ✅ **Ordered**: Keys are sorted alphabetically
- ❌ **Memory cost**: O(n) for keys (must load all keys to sort)
- ✅ **Values on-demand**: Still reads values from disk as needed
- **Best for**: When you need sorted output

### 3. Range Cursor - Filtered Iteration

```go
cursor := db.NewCursor(&skv.CursorOptions{
    From: []byte("user:"),
    To:   []byte("user:\xff"),
})
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err != nil {
        break
    }
    // Process only keys in range
}
```

**Characteristics:**
- ✅ **Filtered**: Only keys in range [From, To]
- ✅ **Ordered**: Sorted within range
- ❌ **Memory**: Still O(n) for all keys (loads all, then filters)
- **Best for**: When you need a subset of keys in a specific range

### 4. Prefix Cursor - Prefix Matching

```go
cursor := db.PrefixCursor([]byte("product:"), false)
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err != nil {
        break
    }
    // Process only keys with prefix
}
```

**Characteristics:**
- ✅ **Efficient filtering**: Only loads keys matching prefix
- ✅ **Ordered**: Sorted within matching keys
- ❌ **Memory**: O(n) for matching keys
- **Best for**: Keys with common prefixes (user:*, product:*, etc.)

### 5. Reverse Cursor - Descending Order

```go
cursor := db.NewCursor(&skv.CursorOptions{
    Reverse: true,
})
defer cursor.Close()

for {
    key, value, err := cursor.Next()
    if err != nil {
        break
    }
    // Keys in reverse order (Z to A)
}
```

**Characteristics:**
- ✅ **Reverse order**: Descending sort
- ❌ **Memory**: O(n) for keys
- **Best for**: Newest-first iteration (if keys include timestamps)

## Performance Comparison

| Method | Memory (keys) | Memory (values) | Order | Speed | Use Case |
|--------|---------------|-----------------|-------|-------|----------|
| ForEach | O(1) | O(1) | ❌ No | ⚡ Fastest | Process all, any order |
| NewCursor | O(n) | O(1) | ✅ Yes | 🐢 Slower | Sorted output |
| Range | O(n) | O(1) | ✅ Yes | 🐢 Slower | Subset [from..to] |
| Prefix | O(m)* | O(1) | ✅ Yes | 🐢 Slower | Keys with prefix |
| Reverse | O(n) | O(1) | ✅ Yes | 🐢 Slower | Descending order |

\* m = number of keys matching prefix

## Memory Considerations

**Why cursors use O(n) memory for keys:**
- Cursors collect all (or matching) keys into a slice
- Keys are sorted with `sort.Strings()`
- This requires loading all keys into memory
- Values are still read on-demand from disk

**For databases with millions of keys:**
- Cursors may consume significant memory (strings in Go are ~16 bytes + key length)
- Example: 1 million keys of 20 bytes each ≈ 36 MB RAM
- ForEach remains O(1) regardless of database size

**Optimization tips:**
- Use ForEach when order doesn't matter
- Use PrefixCursor to limit key set
- Use Range cursors to process in batches
- Consider chunking large datasets

## Running the Example

```bash
cd examples/11-iteration
go run main.go
```

## Expected Output

```
=== Iteration Methods Comparison ===

1. ForEach (fast, unordered iteration)
   - Memory: O(1) for values (read on-demand)
   - Order: NOT guaranteed (iterates over map)
   - Use case: Process all records, don't care about order

   [1] product:001 = Laptop
   [2] user:alice = Alice Johnson
   [3] user:bob = Bob Smith
   ... (unordered)

2. NewCursor (ordered iteration, sorted by key)
   - Memory: O(n) for keys (must sort all keys)
   - Order: Guaranteed (sorted)
   - Use case: Need sorted output

   [1] product:001 = Laptop
   [2] product:002 = Mouse
   [3] product:003 = Keyboard
   [4] user:alice = Alice Johnson
   [5] user:bob = Bob Smith
   [6] user:charlie = Charlie Brown

3. NewCursor with Range (ordered, filtered)
   - Memory: O(n) for keys, but can limit range
   - Use case: Iterate only a subset of keys

   [1] user:alice = Alice Johnson
   [2] user:bob = Bob Smith
   [3] user:charlie = Charlie Brown

4. PrefixCursor (ordered, prefix match)
   - Memory: O(n) for keys matching prefix
   - Use case: All keys with specific prefix

   [1] product:001 = Laptop
   [2] product:002 = Mouse
   [3] product:003 = Keyboard

5. Reverse Cursor (descending order)
   - Order: Reverse sorted
   - Use case: Iterate from end to start

   [1] user:charlie = Charlie Brown
   [2] user:bob = Bob Smith
   [3] user:alice = Alice Johnson

=== Performance Guidelines ===
ForEach:       Best for: Processing all records, don't care about order
               Memory: O(1) for values

NewCursor:     Best for: Need sorted output of all records
               Memory: O(n) for keys

RangeCursor:   Best for: Iterate subset (from..to)
               Memory: O(n) for keys in range

PrefixCursor:  Best for: All keys starting with prefix
               Memory: O(n) for matching keys
```

## See Also

- [Cursor Documentation](../../CURSORS.md)
- [ForEach Tests](../../foreach_test.go)
- [Cursor Tests](../../cursor_test.go)
