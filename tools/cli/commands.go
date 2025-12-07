package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/jncss/skv"
)

// Global flags
var hexDump bool
var compressionType string = "none"

// parseFlags parses global flags from args
func parseFlags() []string {
	remaining := []string{os.Args[0]}
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--hex" || arg == "-x" {
			hexDump = true
		} else if arg == "--compression" || arg == "-c" {
			if i+1 < len(os.Args) {
				i++ // Move to next arg
				compressionType = os.Args[i]
			} else {
				fmt.Fprintln(os.Stderr, "Error: --compression requires an argument (none, snappy, lz4)")
				os.Exit(1)
			}
		} else {
			remaining = append(remaining, arg)
		}
	}
	return remaining
}

// getCompressionOption returns the compression option based on the flag
func getCompressionOption() skv.CompressionType {
	switch compressionType {
	case "none":
		return skv.CompressionNone
	case "snappy":
		return skv.CompressionSnappy
	case "lz4":
		return skv.CompressionLZ4
	default:
		fmt.Fprintf(os.Stderr, "Invalid compression type: %s (use none, snappy, or lz4)\n", compressionType)
		os.Exit(1)
		return skv.CompressionNone
	}
}

// openDatabase opens a database with the configured compression
func openDatabase(path string) (*skv.SKV, error) {
	return skv.OpenWithOptions(path, &skv.Options{
		Compression: getCompressionOption(),
	})
}

// formatHex formats a string as hexdump
func formatHex(s string) string {
	data := []byte(s)
	if len(data) == 0 {
		return "(empty)"
	}

	var builder strings.Builder
	for i := 0; i < len(data); i += 16 {
		// Offset
		builder.WriteString(fmt.Sprintf("%08x  ", i))

		// Hex bytes
		end := i + 16
		if end > len(data) {
			end = len(data)
		}

		for j := i; j < i+16; j++ {
			if j < end {
				builder.WriteString(fmt.Sprintf("%02x ", data[j]))
			} else {
				builder.WriteString("   ")
			}
			if j == i+7 {
				builder.WriteString(" ")
			}
		}

		// ASCII representation
		builder.WriteString(" |")
		for j := i; j < end; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				builder.WriteByte(data[j])
			} else {
				builder.WriteByte('.')
			}
		}
		builder.WriteString("|")

		if i+16 < len(data) {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// formatHexCompact formats a string as compact hex (like "48656c6c6f")
func formatHexCompact(s string) string {
	return hex.EncodeToString([]byte(s))
}

// handlePut stores a new key-value pair
func handlePut() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: skv put <database> <key> <value>")
		os.Exit(1)
	}

	dbPath := os.Args[2]
	key := os.Args[3]
	value := os.Args[4]

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	if hexDump {
		fmt.Println(formatHex(value))
	} else {
		fmt.Print(value)
	}
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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
		if hexDump {
			fmt.Println(formatHex(key))
		} else {
			fmt.Println(key)
		}
	}
}

// handleClear removes all keys
func handleClear() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: skv clear <database>")
		os.Exit(1)
	}

	dbPath := os.Args[2]

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.ForEachString(func(key string, value string) error {
		if hexDump {
			fmt.Println("Key:")
			fmt.Println(formatHex(key))
			fmt.Println("Value:")
			fmt.Println(formatHex(value))
			fmt.Println()
		} else {
			fmt.Printf("%s=%s\n", key, value)
		}
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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
		if hexDump {
			fmt.Println("Key:")
			fmt.Println(formatHex(key))
			fmt.Println("Value:")
			fmt.Println(formatHex(value))
			fmt.Println()
		} else {
			fmt.Printf("%s=%s\n", key, value)
		}
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

	db, err := openDatabase(dbPath)
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

func handleRecover() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: skv recover <corrupted.skv> <recovered.skv>\n")
		os.Exit(1)
	}

	corruptedFile := os.Args[2]
	recoveredFile := os.Args[3]

	// Check if corrupted file exists
	if _, err := os.Stat(corruptedFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' does not exist\n", corruptedFile)
		os.Exit(1)
	}

	// Check if recovered file already exists
	if _, err := os.Stat(recoveredFile); err == nil {
		fmt.Fprintf(os.Stderr, "Error: Output file '%s' already exists\n", recoveredFile)
		os.Exit(1)
	}

	fmt.Printf("Attempting to recover records from '%s'...\n", corruptedFile)

	// Open corrupted file for reading
	inFile, err := os.Open(corruptedFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening corrupted file: %v\n", err)
		os.Exit(1)
	}
	defer inFile.Close()

	// Get file size
	fileInfo, err := inFile.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
		os.Exit(1)
	}
	fileSize := fileInfo.Size()

	// Read entire file into memory
	fileData := make([]byte, fileSize)
	_, err = inFile.Read(fileData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	inFile.Close()

	// Create new database for recovered records
	recoveredDB, err := openDatabase(recoveredFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating recovered database: %v\n", err)
		os.Exit(1)
	}
	defer recoveredDB.Close()

	recoveredCount := 0
	skippedCount := 0
	position := int64(0)

	// Skip header if present (6 bytes: "SKV" + version)
	if fileSize >= 6 && string(fileData[0:3]) == "SKV" {
		position = 6
		fmt.Println("Found valid SKV header, skipping...")
	}

	fmt.Printf("Scanning %d bytes for valid records...\n", fileSize)

	// Scan byte by byte looking for potential record starts
	for position < fileSize {
		typeByte := fileData[position]
		baseType := typeByte & 0x0F // Remove deleted flag

		// Check if this could be a valid type byte
		if baseType != 0x01 && baseType != 0x02 && baseType != 0x04 && baseType != 0x08 {
			position++
			continue
		}

		// Try to parse as a record (pass fileSize for sanity checking)
		record, recordSize, err := tryParseRecord(fileData, position, fileSize)
		if err != nil {
			// Not a valid record, continue scanning
			position++
			skippedCount++
			continue
		}

		// Valid record found! Save it to recovered database
		if record.deleted {
			// Skip deleted records
			position += recordSize
			skippedCount++
			continue
		}

		err = recoveredDB.Put(record.key, record.data)
		if err != nil {
			// If key exists, try to update
			err = recoveredDB.Update(record.key, record.data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not save record at position %d: %v\n", position, err)
			}
		}

		recoveredCount++
		if recoveredCount%100 == 0 {
			fmt.Printf("  Recovered %d records...\n", recoveredCount)
		}

		position += recordSize
	}

	fmt.Printf("\n✓ Recovery complete\n")
	fmt.Printf("  Total records recovered: %d\n", recoveredCount)
	fmt.Printf("  Invalid bytes skipped: %d\n", skippedCount)
	fmt.Printf("  Recovered database: %s\n", recoveredFile)
}

// recordInfo holds information about a recovered record
type recordInfo struct {
	key     []byte
	data    []byte
	deleted bool
}

// tryParseRecord attempts to parse a record starting at the given position
func tryParseRecord(fileData []byte, pos int64, fileSize int64) (*recordInfo, int64, error) {
	if pos >= int64(len(fileData)) {
		return nil, 0, fmt.Errorf("position beyond file end")
	}

	originalPos := pos
	typeByte := fileData[pos]
	baseType := typeByte & 0x0F
	deleted := (typeByte & 0x80) != 0
	pos++

	// Read key size
	if pos >= int64(len(fileData)) {
		return nil, 0, fmt.Errorf("incomplete key size")
	}
	keySize := int(fileData[pos])
	pos++

	// Read key
	if pos+int64(keySize) > int64(len(fileData)) {
		return nil, 0, fmt.Errorf("incomplete key")
	}
	key := make([]byte, keySize)
	copy(key, fileData[pos:pos+int64(keySize)])
	pos += int64(keySize)

	// Check if compressed (bits 5-6 in type byte)
	compressionBits := typeByte & 0x60
	isCompressed := compressionBits != 0

	// If compressed, skip original size field (we don't decompress during recovery)
	if isCompressed {
		switch baseType {
		case 0x01:
			if pos >= int64(len(fileData)) {
				return nil, 0, fmt.Errorf("incomplete original size")
			}
			pos++
		case 0x02:
			if pos+2 > int64(len(fileData)) {
				return nil, 0, fmt.Errorf("incomplete original size")
			}
			pos += 2
		case 0x04:
			if pos+4 > int64(len(fileData)) {
				return nil, 0, fmt.Errorf("incomplete original size")
			}
			pos += 4
		case 0x08:
			if pos+8 > int64(len(fileData)) {
				return nil, 0, fmt.Errorf("incomplete original size")
			}
			pos += 8
		default:
			return nil, 0, fmt.Errorf("invalid type byte")
		}
	}

	// Read data size based on type (this is compressed size if compressed)
	var dataSize uint64
	switch baseType {
	case 0x01:
		if pos >= int64(len(fileData)) {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = uint64(fileData[pos])
		pos++
	case 0x02:
		if pos+2 > int64(len(fileData)) {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = uint64(fileData[pos]) | uint64(fileData[pos+1])<<8
		pos += 2
	case 0x04:
		if pos+4 > int64(len(fileData)) {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = uint64(fileData[pos]) | uint64(fileData[pos+1])<<8 |
			uint64(fileData[pos+2])<<16 | uint64(fileData[pos+3])<<24
		pos += 4
	case 0x08:
		if pos+8 > int64(len(fileData)) {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = uint64(fileData[pos]) | uint64(fileData[pos+1])<<8 |
			uint64(fileData[pos+2])<<16 | uint64(fileData[pos+3])<<24 |
			uint64(fileData[pos+4])<<32 | uint64(fileData[pos+5])<<40 |
			uint64(fileData[pos+6])<<48 | uint64(fileData[pos+7])<<56
		pos += 8
	default:
		return nil, 0, fmt.Errorf("invalid type byte")
	}

	// Sanity check: data size cannot exceed remaining file size
	// This prevents trying to allocate huge amounts of memory for corrupted data
	remainingBytes := fileSize - pos
	if int64(dataSize) > remainingBytes {
		return nil, 0, fmt.Errorf("data size %d exceeds remaining file size %d", dataSize, remainingBytes)
	}

	// Read data
	if pos+int64(dataSize) > int64(len(fileData)) {
		return nil, 0, fmt.Errorf("incomplete data")
	}
	data := make([]byte, dataSize)
	copy(data, fileData[pos:pos+int64(dataSize)])
	pos += int64(dataSize)

	// Read and verify CRC
	var crcSize int64
	if baseType == 0x01 {
		crcSize = 2
	} else {
		crcSize = 4
	}

	if pos+crcSize > int64(len(fileData)) {
		return nil, 0, fmt.Errorf("incomplete CRC")
	}

	// Build record for CRC calculation
	recordBuf := make([]byte, 0, pos-originalPos-crcSize)
	recordBuf = append(recordBuf, fileData[originalPos:pos]...)

	// Read stored CRC
	var storedCRC uint32
	if baseType == 0x01 {
		storedCRC = uint32(fileData[pos]) | uint32(fileData[pos+1])<<8
	} else {
		storedCRC = uint32(fileData[pos]) | uint32(fileData[pos+1])<<8 |
			uint32(fileData[pos+2])<<16 | uint32(fileData[pos+3])<<24
	}
	pos += crcSize

	// Calculate CRC
	var calculatedCRC uint32
	if baseType == 0x01 {
		calculatedCRC = uint32(calculateCRC16(recordBuf))
	} else {
		calculatedCRC = calculateCRC32(recordBuf)
	}

	// Verify CRC (only for non-deleted records)
	if !deleted && storedCRC != calculatedCRC {
		return nil, 0, fmt.Errorf("CRC mismatch")
	}

	recordSize := pos - originalPos
	return &recordInfo{
		key:     key,
		data:    data,
		deleted: deleted,
	}, recordSize, nil
}

// calculateCRC16 calculates CRC-16-CCITT
func calculateCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// calculateCRC32 calculates CRC-32-IEEE
func calculateCRC32(data []byte) uint32 {
	return crc32IEEE(data)
}

// crc32IEEE implements CRC-32-IEEE polynomial
func crc32IEEE(data []byte) uint32 {
	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}
