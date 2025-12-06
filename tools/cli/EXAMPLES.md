# SKV CLI - Quick Examples

## Installation

```bash
# Install globally
go install github.com/jncss/skv/tools/cli@latest

# Or build from source
cd tools/cli
go build -o skv .
```

## Basic Usage

```bash
# Create/open database and add data
skv put users.skv user:1 "Alice"
skv put users.skv user:2 "Bob"
skv put users.skv user:3 "Charlie"

# Read data
skv get users.skv user:1                    # Output: Alice

# Update
skv update users.skv user:1 "Alice Smith"

# Check existence
skv exists users.skv user:1                 # Output: true
skv exists users.skv user:99                # Output: false

# Count keys
skv count users.skv                         # Output: 3

# List all keys
skv keys users.skv
# Output:
# user:1
# user:2
# user:3

# Show all key-value pairs
skv foreach users.skv
# Output:
# user:1=Alice Smith
# user:2=Bob
# user:3=Charlie

# Delete
skv delete users.skv user:2
skv count users.skv                         # Output: 2
```

## File Operations

```bash
# Store files
skv putfile assets.skv logo logo.png
skv putfile assets.skv config config.json
skv putfile assets.skv readme README.md

# Retrieve files
skv getfile assets.skv logo retrieved_logo.png

# Update with file
skv updatefile assets.skv config new_config.json

# List stored files
skv keys assets.skv
```

## Streaming (for Large Files)

```bash
# Store large files efficiently (no full memory load)
skv putstream media.skv intro intro.mp4
skv putstream media.skv backup backup.tar.gz

# Retrieve large files
skv getstream media.skv intro output.mp4

# Update large files
skv updatestream media.skv intro new_intro.mp4
```

## Batch Operations

```bash
# Store multiple keys at once
skv putbatch config.skv \
  database.host "localhost" \
  database.port "5432" \
  database.name "mydb" \
  app.debug "true"

# Retrieve multiple keys
skv getbatch config.skv database.host database.port database.name
# Output:
# database.host=localhost
# database.port=5432
# database.name=mydb
```

## Database Management

```bash
# Check database health and stats
skv verify mydb.skv
# Output:
# Database Statistics:
# ====================
# Total Records:    150
# Active Records:   120
# Deleted Records:  30
# ...
# Wasted Percent:   15.75%
# Efficiency:       80.25%

# Compact database (remove deleted records)
skv compact mydb.skv
# Output:
# ✓ Database compacted
# Size before: 524288 bytes (0.50 MB)
# Size after:  380000 bytes (0.36 MB)
# Saved:       144288 bytes (0.14 MB, 27.5%)
```

## Backup & Restore

```bash
# Create JSON backup
skv backup production.skv backup_20241206.json

# Restore from backup
skv restore production.skv backup_20241206.json

# Backup is human-readable JSON
cat backup_20241206.json
# [
#   {
#     "key": "username",
#     "value": "alice",
#     "is_binary": false
#   },
#   ...
# ]
```

## Real-World Examples

### User Session Storage

```bash
# Store user sessions
skv put sessions.skv "sess:abc123" '{"user_id":42,"expires":"2024-12-31"}'
skv put sessions.skv "sess:def456" '{"user_id":99,"expires":"2024-12-30"}'

# Retrieve session
skv get sessions.skv "sess:abc123"

# Clean up expired session
skv delete sessions.skv "sess:def456"
```

### Configuration Management

```bash
# Store app configs
skv putfile configs.skv prod:nginx nginx.conf
skv putfile configs.skv prod:app app.yaml
skv putfile configs.skv dev:app app-dev.yaml

# Deploy configuration
skv getfile configs.skv prod:nginx /etc/nginx/nginx.conf

# Backup all configs
skv backup configs.skv configs_backup.json
```

### Cache System

```bash
# Cache API responses
skv put cache.skv "api:/users/123" '{"id":123,"name":"Alice"}'
skv put cache.skv "api:/posts/456" '{"id":456,"title":"Hello"}'

# Read from cache
skv get cache.skv "api:/users/123"

# Check if cached
skv exists cache.skv "api:/users/999"

# Clear cache
skv clear cache.skv
```

### File Archive

```bash
# Archive files with streaming (memory-efficient)
skv putstream archive.skv "video:2024-12-06" recording.mp4
skv putstream archive.skv "backup:2024-12-06" backup.tar.gz

# List archived files
skv keys archive.skv

# Extract specific file
skv getstream archive.skv "backup:2024-12-06" restored_backup.tar.gz

# Check archive size
skv verify archive.skv

# Optimize archive
skv compact archive.skv
```

## Tips

1. **Use streaming for large files (> 1MB)**: `putstream`/`getstream` instead of `putfile`/`getfile`
2. **Batch operations are faster**: Use `putbatch`/`getbatch` for multiple keys
3. **Monitor wasted space**: Run `verify` periodically, compact when > 30%
4. **Backup important data**: Use `backup` before major operations
5. **Key naming conventions**: Use prefixes for organization (e.g., `user:`, `sess:`, `cache:`)

## Getting Help

```bash
# Show all commands
skv

# Show detailed help
skv help
```

## More Information

- [Full CLI Documentation](README.md)
- [Main SKV Documentation](../../README.md)
- [Examples](../../examples/)
