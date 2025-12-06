# Backup and Restore

This example demonstrates how to backup and restore SKV databases using JSON format.

## Features

- **JSON format** - Human-readable backups
- **Smart encoding** - Small UTF-8 strings stored as text, binary data as base64
- **Automatic detection** - Values ≤256 bytes are tested for UTF-8 validity
- **Partial restore** - Existing keys not in backup remain untouched
- **Overwrite on restore** - Backup values overwrite existing keys

## Encoding Rules

The backup automatically chooses the best encoding:

- **String format**: Values ≤ 256 bytes that are valid UTF-8
- **Base64 format**: Values > 256 bytes OR binary data

## Usage

```bash
cd examples/05-backup
go run backup_demo.go
```

This will:
1. Create a database with various data types
2. Backup to JSON
3. Modify the database
4. Restore from backup
5. Show the differences
