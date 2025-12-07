# Write-Ahead Log (WAL) Example

This example demonstrates how SKV's Write-Ahead Log (WAL) ensures durability and crash recovery.

## What is WAL?

A Write-Ahead Log is a technique that logs all operations **before** they are applied to the main data file. If the system crashes before an operation completes, the WAL can be replayed during startup to recover the uncommitted operations.

## How it Works in SKV

1. **Every operation is logged first**: When you call `Put()` or `Delete()`, the operation is written to the WAL file (`.skv.wal`) before modifying the main data file
2. **Operation is applied**: After logging, the operation is applied to the main `.skv` file
3. **Commit marker**: A commit marker is written to the WAL
4. **WAL is truncated**: After successful commit, the WAL is cleared
5. **Recovery on startup**: If the database crashed before step 4, the WAL entries are replayed when you next open the database

## Example

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jncss/skv"
)

func main() {
	tmpDir := os.TempDir()
	dbPath := filepath.Join(tmpDir, "wal_demo.skv")
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	fmt.Println("=== Write-Ahead Log (WAL) Demo ===\n")

	// Scenario 1: Normal operation
	fmt.Println("1. Normal operation with WAL")
	fmt.Println("   Opening database...")
	db, err := skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Writing data (logged to WAL first)...")
	db.Put([]byte("user:1"), []byte(`{"name":"Alice","email":"alice@example.com"}`))
	db.Put([]byte("user:2"), []byte(`{"name":"Bob","email":"bob@example.com"}`))
	db.Put([]byte("user:3"), []byte(`{"name":"Carol","email":"carol@example.com"}`))

	fmt.Println("   Reading data...")
	val, _ := db.Get([]byte("user:1"))
	fmt.Printf("   user:1 = %s\n", val)

	fmt.Println("   Closing database cleanly...")
	db.Close()

	// Scenario 2: Crash recovery simulation
	fmt.Println("\n2. Crash recovery simulation")
	fmt.Println("   Opening database again...")
	db, err = skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Data persisted correctly:")
	for _, key := range []string{"user:1", "user:2", "user:3"} {
		val, err := db.Get([]byte(key))
		if err != nil {
			fmt.Printf("   %s: ERROR - %v\n", key, err)
		} else {
			fmt.Printf("   %s = %s\n", key, val)
		}
	}

	// Scenario 3: Delete operation
	fmt.Println("\n3. Delete operation (also logged to WAL)")
	fmt.Println("   Deleting user:2...")
	db.Delete([]byte("user:2"))

	fmt.Println("   Verifying deletion...")
	_, err = db.Get([]byte("user:2"))
	if err == skv.ErrKeyNotFound {
		fmt.Println("   ✓ user:2 successfully deleted")
	}

	db.Close()

	// Scenario 4: Reopen to verify delete persisted
	fmt.Println("\n4. Reopen to verify delete was persisted")
	db, err = skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Get([]byte("user:2"))
	if err == skv.ErrKeyNotFound {
		fmt.Println("   ✓ Delete persisted correctly (user:2 still deleted)")
	}

	// Remaining users
	fmt.Println("\n5. Remaining users:")
	for _, key := range []string{"user:1", "user:3"} {
		val, err := db.Get([]byte(key))
		if err == nil {
			fmt.Printf("   %s = %s\n", key, val)
		}
	}

	db.Close()

	fmt.Println("\n=== WAL ensures durability! ===")
	fmt.Println("All operations are safe even if the system crashes")
	fmt.Println("between writes, thanks to the Write-Ahead Log.")
}
```

## Running the Example

```bash
cd examples/10-wal
go run main.go
```

## Expected Output

```
=== Write-Ahead Log (WAL) Demo ===

1. Normal operation with WAL
   Opening database...
   Writing data (logged to WAL first)...
   Reading data...
   user:1 = {"name":"Alice","email":"alice@example.com"}
   Closing database cleanly...

2. Crash recovery simulation
   Opening database again...
   Data persisted correctly:
   user:1 = {"name":"Alice","email":"alice@example.com"}
   user:2 = {"name":"Bob","email":"bob@example.com"}
   user:3 = {"name":"Carol","email":"carol@example.com"}

3. Delete operation (also logged to WAL)
   Deleting user:2...
   Verifying deletion...
   ✓ user:2 successfully deleted

4. Reopen to verify delete was persisted
   ✓ Delete persisted correctly (user:2 still deleted)

5. Remaining users:
   user:1 = {"name":"Alice","email":"alice@example.com"}
   user:3 = {"name":"Carol","email":"carol@example.com"}

=== WAL ensures durability! ===
All operations are safe even if the system crashes
between writes, thanks to the Write-Ahead Log.
```

## Key Points

- **Automatic**: WAL is enabled by default. You don't need to do anything special
- **Transparent**: All `Put()`, `Update()`, and `Delete()` operations automatically use WAL
- **Crash-safe**: If your program crashes, uncommitted operations in the WAL are replayed on next startup
- **Performance**: WAL adds minimal overhead because entries are small and writes are sequential
- **File**: The WAL file is `<database>.skv.wal` (e.g., `mydb.skv.wal`)

## Technical Details

### WAL File Format

```
Header (6 bytes):
  - Magic: "WAL" (3 bytes)
  - Version: major.minor.patch (3 bytes)

Entry (variable size):
  - OpType: 1 byte (0x01=Put, 0x02=Delete, 0x03=Commit)
  - KeySize: 2 bytes (little-endian)
  - Key: variable bytes
  - DataSize: 4 bytes (little-endian)
  - Data: variable bytes (empty for Delete)
  - CRC32: 4 bytes (checksum)
```

### Recovery Process

On database startup (during `Open()`):
1. Check if WAL file exists and has content beyond the header
2. Read all entries from the WAL
3. Replay Put and Delete operations
4. Stop at commit marker (or EOF)
5. Truncate WAL after successful recovery

### When WAL is Updated

Each operation follows this sequence:
1. `wal.LogPut(key, data)` or `wal.LogDelete(key)` - Write to WAL
2. Apply operation to main `.skv` file
3. `wal.LogCommit()` - Mark as committed
4. `wal.Truncate()` - Clear WAL (operation complete)

This ensures that if a crash occurs at any point, the operation can be replayed.
