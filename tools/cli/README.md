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
skv [options] <command> [arguments]
```

### Global Options

- `--hex` or `-x`: Display keys and values as hexadecimal dump
  - Format: `offset + hex bytes + ASCII representation`
  - Useful for binary data or debugging encoding issues
  - Works with: `get`, `keys`, `foreach`, `getbatch`

- `--compression` or `-c <type>`: Set compression algorithm for the database
  - Values: `none` (default), `snappy`, `lz4`
  - Applies to all write operations (put, update, putfile, etc.)
  - Data below 128 bytes is never compressed
  - Compression is transparent: reads work regardless of write compression settings
  - Example: `skv -c lz4 put mydb.skv key value`

- `--encryption` or `-e <type>`: Set encryption algorithm for the database
  - Values: `none` (default), `aes` (AES-256), `simplecipher` (custom XOR)
  - Requires `--password` option when not `none`
  - Both keys and values are encrypted separately
  - Encryption is applied BEFORE compression
  - Example: `skv -e aes -p mypassword put secure.skv key value`

- `--password` or `-p <password>`: Encryption/decryption password
  - Required when `--encryption` is not `none`
  - Must be the same for all operations on the same database
  - Example: `skv --encryption aes --password secret123 get secure.skv key`

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

# View as hexdump
skv --hex get mydb.skv username
# Output:
# 00000000  6a 6f 68 6e 5f 64 6f 65                           |john_doe|
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

# View as hexdump
skv --hex keys mydb.skv
# Output:
# 00000000  75 73 65 72 6e 61 6d 65                           |username|
# 00000000  65 6d 61 69 6c                                    |email|
# 00000000  63 6f 6e 66 69 67                                 |config|
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

# View as hexdump
skv -x foreach mydb.skv
# Output:
# Key:
# 00000000  75 73 65 72 6e 61 6d 65                           |username|
# Value:
# 00000000  6a 6f 68 6e 5f 64 6f 65                           |john_doe|
#
# Key:
# 00000000  65 6d 61 69 6c                                    |email|
# Value:
# 00000000  6a 6f 68 6e 40 65 78 61  6d 70 6c 65 2e 63 6f 6d  |john@example.com|
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

# View as hexdump
skv --hex getbatch mydb.skv username email
# Output:
# Key:
# 00000000  75 73 65 72 6e 61 6d 65                           |username|
# Value:
# 00000000  6a 6f 68 6e                                       |john|
#
# Key:
# 00000000  65 6d 61 69 6c                                    |email|
# Value:
# 00000000  6a 6f 68 6e 40 65 78 61  6d 70 6c 65 2e 63 6f 6d  |john@example.com|
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

#### recover - Recover valid records from corrupted database
```bash
skv recover corrupted.skv repaired.skv
```

Scans a corrupted SKV file byte-by-byte looking for valid records. When a potential record is found (type byte 0x01, 0x02, 0x04, or 0x08), it attempts to parse and verify the CRC. Valid records are saved to a new database file.

**Features:**
- Byte-by-byte scanning for maximum recovery
- CRC verification ensures only valid records are recovered
- Skips deleted records and corrupted data
- Progress reporting for large files

**Use cases:**
- Database corruption detected by verify
- Disk errors or file system corruption
- Interrupted writes or power failures

Example output:
```
Attempting to recover records from 'corrupted.skv'...
Found valid SKV header, skipping...
Scanning 15438 bytes for valid records...
  Recovered 100 records...
  Recovered 200 records...

✓ Recovery complete
  Total records recovered: 247
  Invalid bytes skipped: 1834
  Recovered database: repaired.skv
```

**See [RECOVERY.md](RECOVERY.md) for detailed recovery strategies and best practices.**

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

### Example 6: Compression
```bash
# Store large data with LZ4 compression (fastest)
skv -c lz4 put logs.skv log:2024-12-07 "$(cat large_log.txt)"

# Store with Snappy compression (balanced)
skv -c snappy put archive.skv data:20241207 "$(cat data.json)"

# Compression is transparent - no flag needed for reading
skv get logs.skv log:2024-12-07
```

### Example 7: Encryption

The CLI supports AES-256 and SimpleCipher encryption for secure storage.

```bash
# Store data with AES encryption
skv -e aes -p mySecretPassword put secure.skv api:key "sk_live_abc123..."
skv -e aes -p mySecretPassword put secure.skv token "bearer_xyz..."

# Store file with AES encryption
skv --encryption aes --password mySecretPassword putfile secure.skv credentials creds.json

# Retrieve encrypted data (must provide correct password)
skv -e aes -p mySecretPassword get secure.skv api:key

# Use SimpleCipher for custom encryption
skv -e simplecipher -p secret123 put private.skv data "sensitive info"
skv -e simplecipher -p secret123 get private.skv data

# Combine compression and encryption
skv -c lz4 -e aes -p pass123 put hybrid.skv large:data "$(cat big_file.txt)"

# List keys (encrypted, requires password)
skv -e aes -p mySecretPassword keys secure.skv

# Iterate over encrypted data
skv --encryption aes --password mySecretPassword foreach secure.skv

# Backup encrypted database
# IMPORTANT: Backup preserves encryption - data stays encrypted in JSON!
skv -e aes -p mySecretPassword backup secure.skv backup.json
# The backup.json contains ENCRYPTED data (secure even if stolen)

# Restore encrypted backup (must use SAME password!)
skv -e aes -p mySecretPassword restore secure.skv backup.json

# To change encryption password, you need to:
# 1. Backup with old password
# 2. Create new database with new password  
# 3. For each record: read from backup → write to new database
```

**Important Notes:**
- Both keys and values are encrypted separately
- Encryption is applied BEFORE compression (encrypt → compress on write, decompress → decrypt on read)
- Password is required for all operations on encrypted databases
- Wrong password will cause decryption errors
- **Backup files preserve encryption** - data stays encrypted in JSON (secure!)
- Must use the **same password** to restore that was used for backup
- AES-256: More secure, industry standard
- SimpleCipher: Custom XOR-based cipher, faster but less secure

**Example 8: Encrypted Backup Security**
```bash
# Create encrypted database and backup
skv -e aes -p secret123 put secure.skv key1 "sensitive data"
skv -e aes -p secret123 backup secure.skv backup.json

# The backup.json contains ENCRYPTED data:
# {
#   "key": "OQ_DJIsZ2TXU2tUcNb-NANk6ATE=",
#   "value": "notqYE46bmB9po-cSPShhFR0VcvEqO9_..."
# }
# ☝️ Completely unreadable without password!

# Restore with correct password - works!
skv -e aes -p secret123 restore restored.skv backup.json
skv -e aes -p secret123 get restored.skv key1
# Output: sensitive data

# Restore with wrong password - corrupts data!
skv -e aes -p WRONG restore bad.skv backup.json
skv -e aes -p WRONG get bad.skv key1
# Output: garbage or error
```

**Example 9: Recovery with Encryption**
```bash
# Recover encrypted database
# Note: Specify encryption flags to indicate the file WAS encrypted
# Recovered file will preserve the encryption (data stays encrypted)
skv -e aes -p mySecretPassword recover corrupted_secure.skv repaired_secure.skv

# Access recovered encrypted database (use same password)
skv -e aes -p mySecretPassword get repaired_secure.skv mykey
```

**Important:** Recovery preserves the encryption state of the original file. If the corrupted file was encrypted, the recovered file will also be encrypted with the same data. You must use the same encryption password to access the recovered database.
skv get logs.skv log:2024-12-07

# Recover compressed database
skv recover corrupted_compressed.skv repaired.skv

# Verify shows actual stored size (compressed)
skv verify logs.skv
```

**Compression notes:**
- Data < 128 bytes is never compressed (not worth it)
- Compression is automatic based on size threshold
- Read operations don't need compression flag
- Both algorithms work with `recover` command
- LZ4: Faster compression/decompression, good ratio
- Snappy: Very fast, slightly lower ratio

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

## Hexdump Mode

The `--hex` (or `-x`) flag enables hexadecimal dump mode for viewing keys and values as raw bytes. This is particularly useful for:

### Use Cases

1. **Binary data inspection**: View non-text data stored in the database
2. **Encoding verification**: Check UTF-8 or other encodings
3. **Debugging**: Identify hidden characters, null bytes, or control characters
4. **Data validation**: Verify exact byte sequences

### Format

Hexdump output uses the classic `hexdump -C` format:

```
00000000  48 65 6c 6c 6f 20 57 6f  72 6c 64 21 20 54 68 69  |Hello World! Thi|
00000010  73 20 69 73 20 61 20 74  65 73 74 20 76 61 6c 75  |s is a test valu|
00000020  65 20 77 69 74 68 20 73  70 65 63 69 61 6c 20 63  |e with special c|
00000030  68 61 72 73 3a 20 c3 b1  c3 a1 c3 a9 c3 ad c3 b3  |hars: ..........|
00000040  c3 ba                                             |..|
```

**Format components:**
- **Offset** (left): Byte position in hexadecimal (8 digits)
- **Hex bytes** (middle): Raw bytes in hexadecimal (16 bytes per line, grouped by 8)
- **ASCII** (right): Printable ASCII characters (`.` for non-printable)

### Examples

#### Inspect binary data
```bash
# Store binary data
skv putfile mydb.skv image logo.png

# View first bytes in hex
skv --hex get mydb.skv image | head -5
# Output:
# 00000000  89 50 4e 47 0d 0a 1a 0a  00 00 00 0d 49 48 44 52  |.PNG........IHDR|
# ...
```

#### Check encoding issues
```bash
# Store text with special characters
skv put mydb.skv text "Héllo Wörld €"

# View encoding
skv -x get mydb.skv text
# Output shows UTF-8 encoding:
# 00000000  48 c3 a9 6c 6c 6f 20 57  c3 b6 72 6c 64 20 e2 82  |H..llo W..rld ..|
# 00000010  ac                                                |.|
```

#### Debug hidden characters
```bash
# Check for null bytes, newlines, etc.
skv --hex get mydb.skv suspect_data
```

#### Compare keys
```bash
# List all keys in hex to spot differences
skv -x keys mydb.skv
```

### Supported Commands

The `--hex` flag works with these commands:

- `get` - View single value as hexdump
- `keys` - View all keys as hexdump
- `foreach` - View all key-value pairs as hexdump
- `getbatch` - View multiple values as hexdump

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
