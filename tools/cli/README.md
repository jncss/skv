# SKV CLI - Command Line Interface for SKV Database

A powerful command-line tool to interact with SKV (Simple Key-Value) databases.

## Installation

### Build from source

```bash
cd tools/cli
go build -o skv
```

### Install globally

```bash
cd tools/cli
go install
```

Or from the root of the project:

```bash
go install ./tools/cli
```

This will install the `skv` binary to your `$GOPATH/bin` directory.

## Usage

```
skv <command> [arguments]
```

## Commands

### Basic Operations

#### put - Store a new key-value pair
```bash
skv put mydb.skv username "john_doe"
skv put mydb.skv email "john@example.com"
```

#### get - Retrieve a value
```bash
skv get mydb.skv username
# Output: john_doe
```

#### update - Update an existing key
```bash
skv update mydb.skv username "jane_doe"
```

#### delete - Delete a key
```bash
skv delete mydb.skv username
```

#### exists - Check if a key exists
```bash
skv exists mydb.skv username
# Output: true or false
```

#### count - Count active keys
```bash
skv count mydb.skv
# Output: 5
```

#### keys - List all keys
```bash
skv keys mydb.skv
# Output (one per line):
# username
# email
# config
```

#### clear - Remove all keys
```bash
skv clear mydb.skv
```

⚠️ **Warning**: This operation cannot be undone!

#### foreach - Iterate over all key-value pairs
```bash
skv foreach mydb.skv
# Output (key=value):
# username=john_doe
# email=john@example.com
```

### File Operations

#### putfile - Store file contents as a value
```bash
skv putfile mydb.skv config config.ini
skv putfile mydb.skv logo logo.png
```

#### getfile - Retrieve value to a file
```bash
skv getfile mydb.skv config retrieved_config.ini
```

#### updatefile - Update key with file contents
```bash
skv updatefile mydb.skv config new_config.ini
```

### Streaming Operations (Memory-Efficient for Large Files)

#### putstream - Stream large file to database
```bash
skv putstream mydb.skv video intro.mp4
skv putstream mydb.skv backup large_backup.tar.gz
```

Use this for files that are too large to load into memory.

#### getstream - Stream value to file
```bash
skv getstream mydb.skv video output.mp4
```

#### updatestream - Update via streaming
```bash
skv updatestream mydb.skv video new_intro.mp4
```

### Batch Operations

#### putbatch - Store multiple key-value pairs
```bash
skv putbatch mydb.skv \
  username "john" \
  email "john@example.com" \
  role "admin"
```

All keys must be new (not exist). If any key exists, the entire operation fails.

#### getbatch - Retrieve multiple keys
```bash
skv getbatch mydb.skv username email role
# Output:
# username=john
# email=john@example.com
# role=admin
```

### Backup & Maintenance

#### backup - Create JSON backup
```bash
skv backup mydb.skv backup.json
```

Creates a human-readable JSON backup of all key-value pairs.

#### restore - Restore from JSON backup
```bash
skv restore mydb.skv backup.json
```

Restores data from a JSON backup. Overwrites existing keys with the same name.

#### verify - Check database integrity and statistics
```bash
skv verify mydb.skv
```

Displays detailed statistics:
- Total, active, and deleted records
- File size and space usage
- Wasted space percentage
- Efficiency metrics
- Average key and data sizes

Example output:
```
Database Statistics:
====================
Total Records:    150
Active Records:   120
Deleted Records:  30

File Size:        524288 bytes (0.50 MB)
Header Size:      6 bytes
Data Size:        450000 bytes (0.43 MB)
Wasted Space:     70000 bytes (0.07 MB)
Padding Bytes:    4282 bytes

Wasted Percent:   15.75%
Efficiency:       80.25%

Avg Key Size:     12.50 bytes
Avg Data Size:    256.30 bytes

✓ Database health: Good
```

#### compact - Remove deleted records and optimize file size
```bash
skv compact mydb.skv
```

Removes all deleted records and reclaims wasted space. Recommended when wasted space > 30%.

Example output:
```
✓ Database compacted
Size before: 524288 bytes (0.50 MB)
Size after:  380000 bytes (0.36 MB)
Saved:       144288 bytes (0.14 MB, 27.5%)
```

## Help

Get general help:
```bash
skv help
```

## Examples

### Example 1: User Management
```bash
# Create database with user data
skv put users.skv user:1 '{"name":"Alice","role":"admin"}'
skv put users.skv user:2 '{"name":"Bob","role":"user"}'

# Retrieve user
skv get users.skv user:1

# Update user
skv update users.skv user:1 '{"name":"Alice","role":"superadmin"}'

# List all users
skv foreach users.skv

# Delete user
skv delete users.skv user:2
```

### Example 2: Configuration Management
```bash
# Store configuration files
skv putfile config.skv app:config config.ini
skv putfile config.skv app:env .env
skv putfile config.skv nginx:config nginx.conf

# List all configs
skv keys config.skv

# Retrieve config
skv getfile config.skv app:config restored_config.ini
```

### Example 3: File Archive
```bash
# Archive large files using streaming
skv putstream archive.skv video:intro intro.mp4
skv putstream archive.skv backup:daily backup_2024.tar.gz
skv putstream archive.skv logs:app app.log

# Extract files
skv getstream archive.skv video:intro extracted_intro.mp4

# Check archive stats
skv verify archive.skv

# Optimize archive
skv compact archive.skv
```

### Example 4: Batch Operations
```bash
# Store multiple settings at once
skv putbatch settings.skv \
  theme "dark" \
  language "en" \
  timezone "UTC" \
  notifications "enabled"

# Retrieve multiple settings
skv getbatch settings.skv theme language timezone
```

### Example 5: Backup & Restore
```bash
# Create backup
skv backup production.skv backup_$(date +%Y%m%d).json

# Restore from backup
skv restore production.skv backup_20241206.json

# Verify after restore
skv verify production.skv
```

## Use Cases

### 1. **Configuration Storage**
Store application configs, environment variables, feature flags.

### 2. **Cache System**
Quick key-value cache with persistence.

### 3. **Session Storage**
Store user sessions with string keys and JSON values.

### 4. **File Archive**
Archive and retrieve files using streaming for large files.

### 5. **Template Management**
Store HTML templates, email templates, etc.

### 6. **Settings & Preferences**
User settings, application preferences.

### 7. **Log Aggregation**
Store and retrieve log files.

### 8. **Asset Management**
Store CSS, JS, images for web applications.

## Performance Tips

1. **Use streaming for large files** (> 1MB): `putstream`/`getstream` instead of `putfile`/`getfile`
2. **Use batch operations** when working with multiple keys
3. **Monitor wasted space**: Run `verify` periodically
4. **Compact regularly**: When wasted space > 30%
5. **Backup important data**: Use `backup` before major operations

## Exit Codes

- `0` - Success
- `1` - Error (see stderr for details)

## Common Errors

### Key already exists
```
Error: Key 'username' already exists. Use 'update' to modify it.
```
**Solution**: Use `skv update` instead of `skv put`

### Key not found
```
Error: Key 'username' not found
```
**Solution**: Check key name with `skv keys` or use `skv put` to create it

### Database file issues
```
Error opening database: permission denied
```
**Solution**: Check file permissions

## License

See main SKV project license.
