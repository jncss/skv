# 04 - Real-World Use Cases

Practical examples showing how to use SKV in real applications.

## Example

### `usecases/`
Demonstrates six common real-world scenarios:

1. **User Session Storage**
   - Create and manage user sessions
   - Store session data with user info
   - Update session on activity
   - Validate sessions
   - Delete sessions on logout

2. **Application Configuration**
   - Store app settings and configuration
   - Batch insert configuration values
   - Retrieve with defaults
   - Filter settings by category
   - Runtime configuration management

3. **Simple Cache**
   - Cache API responses or computed data
   - Check cache before fetching (cache hit/miss)
   - Cache invalidation
   - Fast in-memory lookups

4. **Job Queue / Task Storage**
   - Queue jobs for background processing
   - Store job metadata and status
   - Process jobs with iteration
   - Update job status
   - Track completion

5. **Feature Flags**
   - Enable/disable features dynamically
   - Check feature status
   - Toggle features at runtime
   - List enabled features
   - A/B testing support

6. **User Preferences**
   - Store per-user settings
   - Batch insert preferences
   - Retrieve user-specific settings
   - Update individual preferences
   - Namespace by user ID

**Run:**
```bash
cd examples/04-usecases/usecases
go run usecases.go
```

## Use Case Patterns

### Session Storage
```go
// Create session
sessionID := "sess_abc123"
sessionData := `{"user_id": "123", "login_time": "..."}`
db.PutString(sessionID, sessionData)

// Validate session
if db.HasString(sessionID) {
    // Session is valid
}

// Logout
db.DeleteString(sessionID)
```

### Configuration Management
```go
// Load all config
config := map[string]string{
    "app:name":    "MyApp",
    "db:host":     "localhost",
    "cache:enabled": "true",
}
db.PutBatchString(config)

// Get with fallback
host := db.GetOrDefaultString("db:host", "localhost")
```

### Caching Pattern
```go
// Check cache first
if !db.HasString(cacheKey) {
    // Cache miss - fetch data
    data := fetchFromAPI()
    db.PutString(cacheKey, data)
} else {
    // Cache hit
    data, _ := db.GetString(cacheKey)
}

// Invalidate cache
db.DeleteString(cacheKey)
```

### Feature Flags
```go
// Check if feature is enabled
enabled := db.GetOrDefaultString("feature:new_ui", "false")
if enabled == "true" {
    // Show new UI
}

// Toggle feature
db.UpdateString("feature:new_ui", "true")
```

### User Preferences
```go
// Store preferences with user namespace
userID := "user_123"
db.PutString(fmt.Sprintf("%s:theme", userID), "dark")
db.PutString(fmt.Sprintf("%s:language", userID), "en")

// Retrieve preference
theme := db.GetOrDefaultString(
    fmt.Sprintf("%s:theme", userID), 
    "light",
)
```

## When to Use SKV

### Good Fits
- **Session storage** - Fast lookups, temporary data
- **Configuration** - Application settings, runtime config
- **Feature flags** - Toggle features without deployment
- **Simple cache** - Cache computed results or API responses
- **User preferences** - Per-user settings
- **Job queues** - Small to medium task storage
- **Key-value metadata** - Tags, labels, attributes

### Not Ideal For
- **Large-scale distributed systems** - Use Redis, etcd, or Consul
- **Multi-process concurrent access** - No file locking (use client-server DB)
- **Complex queries** - No SQL/query language (use SQLite or PostgreSQL)
- **Very large datasets** - Memory cache may be limiting
- **Transactional requirements** - No ACID transactions

## Performance Characteristics

Based on the project's benchmarks:
- **Writes**: ~750 inserts/sec (sequential)
- **Reads**: ~270,000 reads/sec (cached)
- **Updates**: ~365 updates/sec
- **Concurrent**: ~1,900 ops/sec (10 goroutines)

Best for workloads with:
- Read-heavy patterns (benefits from in-memory cache)
- Small to medium datasets (< 1 million keys)
- Single-process applications
- Local storage requirements

## Next Steps

- **05-backup**: Learn how to backup and restore your data
- **01-basics**: Review fundamental operations
- **02-advanced**: Explore batch operations and maintenance
