# Real-World Use Cases

Practical examples showing how to use SKV in real applications.

## Example

### `usecases.go`
Complete use case demonstrations:

1. **Configuration Storage**
   - Application settings
   - Environment-specific configs
   - Feature toggles

2. **Session Management**
   - User session storage
   - Session expiration tracking
   - Active session counting

3. **Structured Data (JSON)**
   - Store complex objects
   - Serialize/deserialize automatically
   - Type-safe storage

4. **Cache Implementation**
   - Simple key-value cache
   - Cache hit/miss detection
   - Lazy computation

5. **Feature Flags**
   - Enable/disable features dynamically
   - A/B testing support
   - Gradual rollouts

6. **Counters and Metrics**
   - Page view counters
   - API call tracking
   - Simple analytics

7. **Namespaced Keys**
   - Organize data by namespace
   - User-specific settings
   - Application-level configs

**Run:**
```bash
go run usecases.go
```

## Common Patterns

### Configuration
```go
db.PutString("config.timeout", "30")
timeout := db.GetOrDefaultString("config.timeout", "10")
```

### Sessions
```go
sessionKey := "session:" + sessionID
db.PutString(sessionKey, sessionData)
if db.HasString(sessionKey) {
    // Session is active
}
```

### Caching
```go
if db.Exists(cacheKey) {
    return db.Get(cacheKey) // Fast cache hit
}
// Cache miss - compute and store
result := expensiveComputation()
db.Put(cacheKey, result)
```

### Feature Flags
```go
enabled := db.GetOrDefaultString("feature:new_ui", "false") == "true"
if enabled {
    // Show new UI
}
```

### Namespaces
```go
// User settings
db.PutString("user:123:theme", "dark")
db.PutString("user:123:lang", "en")

// App config
db.PutString("app:version", "1.0.0")
db.PutString("app:name", "MyApp")
```

## Best Practices

1. **Use namespaces** to organize keys: `user:id:property`
2. **Store JSON** for complex data structures
3. **Use GetOrDefault** for optional settings
4. **Compact periodically** if you update configs frequently
5. **Use batch operations** when initializing multiple settings

## When to Use SKV

✅ **Good for:**
- Application configuration
- Session storage (small to medium scale)
- Feature flags
- Simple caching
- Settings and preferences
- Counters and simple metrics

❌ **Not ideal for:**
- High-volume transactional workloads
- Complex queries (no SQL)
- Relationships between entities
- Real-time analytics at scale

## Related Examples

- **01-basics/** - Core operations
- **02-advanced/** - Batch operations for initializing configs
- **03-concurrent/** - Thread-safe access for web applications
