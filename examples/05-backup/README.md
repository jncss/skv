# 05 - Backup and Restore

Learn how to protect your data with backups and recover from data loss.

## Example

### `backup_restore/`
Complete backup and restore workflow:
- **Creating backups**: Export database to JSON format
- **Backup format**: Human-readable JSON with metadata
- **Restoring data**: Import backup into a database
- **Data verification**: Ensure all data was restored correctly
- **Best practices**: Timestamped backups, regular schedules
- **Optimization**: Compact before backup to reduce size

**Run:**
```bash
cd examples/05-backup/backup_restore
go run main.go
```

## Key Concepts

### Creating a Backup
```go
db, _ := skv.Open("mydata")

// Create backup (JSON format)
err := db.Backup("backup.json")
if err != nil {
    log.Fatal(err)
}
```

### Restoring from Backup
```go
db, _ := skv.Open("restored")

// Restore from backup file
err := db.Restore("backup.json")
if err != nil {
    log.Fatal(err)
}

// All data from backup is now in the database
```

### Backup Format
The backup is stored as JSON with the following structure:
```json
{
    "version": "1.0",
    "created_at": "2024-12-06T12:00:00Z",
    "total_records": 10,
    "records": [
        {
            "key": "username",
            "value": "alice",
            "encoding": "text"
        },
        {
            "key": "binary_data",
            "value": "AQIDBA==",
            "encoding": "base64"
        }
    ]
}
```

**Encoding:**
- `text`: UTF-8 text for keys and values that are valid UTF-8
- `base64`: Base64 encoding for binary data

### Timestamped Backups
```go
import "time"

// Create backup with timestamp
timestamp := time.Now().Format("2006-01-02_150405")
backupFile := fmt.Sprintf("backup_%s.json", timestamp)

db.Backup(backupFile)
// Creates: backup_2024-12-06_120000.json
```

### Optimized Backups
```go
// Compact before backup to reduce file size
db.Compact()

// Create backup (smaller file, only active data)
db.Backup("backup_optimized.json")
```

## Best Practices

### Regular Backups
```go
// Example: Daily backup function
func createDailyBackup(db *skv.SKV) error {
    date := time.Now().Format("2006-01-02")
    filename := fmt.Sprintf("backups/daily_%s.json", date)
    
    return db.Backup(filename)
}
```

### Backup Before Major Operations
```go
// Backup before risky operations
db.Backup("pre_migration.json")

// Perform migration
err := migrateData(db)
if err != nil {
    // Restore from backup if migration fails
    db.Restore("pre_migration.json")
}
```

### Keep Multiple Versions
```go
// Rotate backups (keep last N days)
const maxBackups = 7

func rotateBackups() {
    files, _ := filepath.Glob("backups/daily_*.json")
    
    if len(files) > maxBackups {
        // Delete oldest backups
        sort.Strings(files)
        for i := 0; i < len(files)-maxBackups; i++ {
            os.Remove(files[i])
        }
    }
}
```

### Backup Storage
- Store backups in a **different location** than the database
- Use **cloud storage** (S3, Google Cloud Storage, etc.)
- Keep backups **off-site** for disaster recovery
- **Encrypt** backups containing sensitive data

## Recovery Scenarios

### Scenario 1: Accidental Deletion
```go
// Accidentally deleted important data
db.DeleteString("important_key")

// Restore from latest backup
db.Restore("backup_latest.json")

// Data is recovered
```

### Scenario 2: Data Corruption
```go
// Database file corrupted
db, err := skv.Open("corrupted.skv")
if err != nil {
    // Create new database
    db, _ = skv.Open("recovered.skv")
    
    // Restore from backup
    db.Restore("backup_latest.json")
}
```

### Scenario 3: Database Migration
```go
// Migrate data to new location
oldDB, _ := skv.Open("old_location/data")
oldDB.Backup("migration.json")
oldDB.Close()

// Restore in new location
newDB, _ := skv.Open("new_location/data")
newDB.Restore("migration.json")
```

### Scenario 4: Testing with Production Data
```go
// Create backup of production database
prodDB, _ := skv.Open("production")
prodDB.Backup("prod_snapshot.json")

// Restore to test environment
testDB, _ := skv.Open("test")
testDB.Restore("prod_snapshot.json")

// Test with real data without affecting production
```

## Backup File Size

Factors affecting backup size:
- **Number of records**: More records = larger file
- **Value sizes**: Large values increase backup size
- **Encoding**: Base64 encoding adds ~33% overhead for binary data
- **JSON overhead**: Keys, structure, metadata

**Optimization:**
1. Compact before backup to remove deleted/old records
2. Consider selective backups (specific key prefixes)
3. Compress backup files (gzip, zip)
4. Filter out temporary or cache data

## Limitations

- **No incremental backups**: Always full backup
- **No streaming**: Entire backup loaded into memory
- **No encryption**: JSON is plain text (encrypt separately if needed)
- **Single file**: Not suitable for very large databases (> 1GB)

For very large databases, consider:
- External backup tools
- Custom selective backup logic
- Database replication/mirroring

## Next Steps

- **01-basics**: Review fundamental operations
- **02-advanced**: Explore maintenance and compaction
- **04-usecases**: See real-world applications
