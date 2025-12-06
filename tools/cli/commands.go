package main

import (
	"fmt"
	"os"

	"github.com/jncss/skv"
)

// handlePut stores a new key-value pair
func handlePut() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv put <database> <key> <value>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	value := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.PutString(key, value)
	if err != nil {
		if err == skv.ErrKeyExists {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' already exists. Use 'update' to modify it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Stored key '%s'\n", key)
}

// handleGet retrieves a value
func handleGet() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv get <database> <key>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	value, err := db.GetString(key)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Print(value)
}

// handleUpdate updates an existing key
func handleUpdate() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv update <database> <key> <value>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	value := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.UpdateString(key, value)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found. Use 'put' to create it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Updated key '%s'\n", key)
}

// handleDelete deletes a key
func handleDelete() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv delete <database> <key>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.DeleteString(key)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Deleted key '%s'\n", key)
}

// handleExists checks if a key exists
func handleExists() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv exists <database> <key>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	exists := db.ExistsString(key)
	fmt.Println(exists)
}

// handleCount counts active keys
func handleCount() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv count <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	count := db.Count()
	fmt.Println(count)
}

// handleKeys lists all keys
func handleKeys() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv keys <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	keys, err := db.KeysString()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting keys: %v\n", err)
		os.Exit(1)
	}

	for _, key := range keys {
		fmt.Println(key)
	}
}

// handleClear removes all keys
func handleClear() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv clear <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Clear()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Database cleared")
}

// handleForEach iterates over all key-value pairs
func handleForEach() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv foreach <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.ForEachString(func(key string, value string) error {
		fmt.Printf("%s=%s\n", key, value)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error iterating: %v\n", err)
		os.Exit(1)
	}
}

// handlePutFile stores file contents
func handlePutFile() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv putfile <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.PutFile(key, filePath)
	if err != nil {
		if err == skv.ErrKeyExists {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' already exists. Use 'updatefile' to modify it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Get file size for feedback
	info, _ := os.Stat(filePath)
	fmt.Printf("✓ Stored file '%s' under key '%s' (%d bytes)\n", filePath, key, info.Size())
}

// handleGetFile retrieves value to file
func handleGetFile() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv getfile <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.GetFile(key, filePath)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	info, _ := os.Stat(filePath)
	fmt.Printf("✓ Retrieved to '%s' (%d bytes)\n", filePath, info.Size())
}

// handleUpdateFile updates with file contents
func handleUpdateFile() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv updatefile <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.UpdateFile(key, filePath)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found. Use 'putfile' to create it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	info, _ := os.Stat(filePath)
	fmt.Printf("✓ Updated key '%s' with file '%s' (%d bytes)\n", key, filePath, info.Size())
}

// handlePutStream streams file to database
func handlePutStream() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv putstream <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
		os.Exit(1)
	}

	err = db.PutStreamString(key, file, info.Size())
	if err != nil {
		if err == skv.ErrKeyExists {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' already exists. Use 'updatestream' to modify it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Streamed file '%s' to key '%s' (%d bytes)\n", filePath, key, info.Size())
}

// handleGetStream streams value to file
func handleGetStream() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv getstream <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create output file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	n, err := db.GetStreamString(key, file)
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Streamed to '%s' (%d bytes)\n", filePath, n)
}

// handleUpdateStream updates via streaming
func handleUpdateStream() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv updatestream <database> <key> <filepath>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	filePath := os.Args[4]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
		os.Exit(1)
	}

	err = db.UpdateStreamString(key, file, info.Size())
	if err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Fprintf(os.Stderr, "Error: Key '%s' not found. Use 'putstream' to create it.\n", key)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Updated key '%s' via streaming (%d bytes)\n", key, info.Size())
}

// handlePutBatch stores multiple key-value pairs
func handlePutBatch() {
	if len(os.Args) < 4 || (len(os.Args)-3)%2 != 0 {
		fmt.Fprintln(os.Stderr, "Usage: skv putbatch <database> <key1> <value1> <key2> <value2> ...")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Build batch map
	batch := make(map[string]string)
	for i := 3; i < len(os.Args); i += 2 {
		key := os.Args[i]
		value := os.Args[i+1]
		batch[key] = value
	}

	err = db.PutBatchString(batch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Stored %d key-value pairs\n", len(batch))
}

// handleGetBatch retrieves multiple keys
func handleGetBatch() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv getbatch <database> <key1> <key2> <key3> ...")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	keys := os.Args[3:]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	result, err := db.GetBatchString(keys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for key, value := range result {
		fmt.Printf("%s=%s\n", key, value)
	}
}

// handleBackup creates JSON backup
func handleBackup() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv backup <database> <json-file>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	backupPath := os.Args[3]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Backup(backupPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating backup: %v\n", err)
		os.Exit(1)
	}

	info, _ := os.Stat(backupPath)
	count := db.Count()
	fmt.Printf("✓ Backup created: %s (%d keys, %d bytes)\n", backupPath, count, info.Size())
}

// handleRestore restores from JSON backup
func handleRestore() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: skv restore <database> <json-file>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	backupPath := os.Args[3]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Restore(backupPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error restoring backup: %v\n", err)
		os.Exit(1)
	}

	count := db.Count()
	fmt.Printf("✓ Restored from backup (%d keys)\n", count)
}

// handleVerify checks database integrity
func handleVerify() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv verify <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats, err := db.Verify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database Statistics:")
	fmt.Println("====================")
	fmt.Printf("Total Records:    %d\n", stats.TotalRecords)
	fmt.Printf("Active Records:   %d\n", stats.ActiveRecords)
	fmt.Printf("Deleted Records:  %d\n", stats.DeletedRecords)
	fmt.Println()
	fmt.Printf("File Size:        %d bytes (%.2f MB)\n", stats.FileSize, float64(stats.FileSize)/1024/1024)
	fmt.Printf("Header Size:      %d bytes\n", stats.HeaderSize)
	fmt.Printf("Data Size:        %d bytes (%.2f MB)\n", stats.DataSize, float64(stats.DataSize)/1024/1024)
	fmt.Printf("Wasted Space:     %d bytes (%.2f MB)\n", stats.WastedSpace, float64(stats.WastedSpace)/1024/1024)
	fmt.Printf("Padding Bytes:    %d bytes\n", stats.PaddingBytes)
	fmt.Println()
	fmt.Printf("Wasted Percent:   %.2f%%\n", stats.WastedPercent)
	fmt.Printf("Efficiency:       %.2f%%\n", stats.Efficiency)
	fmt.Println()
	fmt.Printf("Avg Key Size:     %.2f bytes\n", stats.AverageKeySize)
	fmt.Printf("Avg Data Size:    %.2f bytes\n", stats.AverageDataSize)
	fmt.Println()

	if stats.WastedPercent > 30 {
		fmt.Println("⚠ Warning: Wasted space > 30%. Consider running 'skv compact' to optimize.")
	} else {
		fmt.Println("✓ Database health: Good")
	}
}

// handleCompact removes deleted records
func handleCompact() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv compact <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := skv.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}

	// Get stats before compaction
	statsBefore, _ := db.Verify()
	sizeBefore := statsBefore.FileSize

	err = db.Compact()
	if err != nil {
		db.Close()
		fmt.Fprintf(os.Stderr, "Error compacting database: %v\n", err)
		os.Exit(1)
	}

	// Get stats after compaction
	statsAfter, _ := db.Verify()
	sizeAfter := statsAfter.FileSize

	db.Close()

	saved := sizeBefore - sizeAfter
	savedPercent := float64(saved) / float64(sizeBefore) * 100

	fmt.Println("✓ Database compacted")
	fmt.Printf("Size before: %d bytes (%.2f MB)\n", sizeBefore, float64(sizeBefore)/1024/1024)
	fmt.Printf("Size after:  %d bytes (%.2f MB)\n", sizeAfter, float64(sizeAfter)/1024/1024)
	fmt.Printf("Saved:       %d bytes (%.2f MB, %.1f%%)\n", saved, float64(saved)/1024/1024, savedPercent)
}
