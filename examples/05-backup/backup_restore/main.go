package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	fmt.Println("=== Backup and Restore ===\n")

	os.MkdirAll("data", 0755)

	// --- Step 1: Create a database with sample data ---
	fmt.Println("1. Creating database with sample data:\n")

	db, err := skv.Open("data/production")
	if err != nil {
		log.Fatal(err)
	}

	// Add sample data
	sampleData := map[string]string{
		"user:1":       `{"name": "Alice", "email": "alice@example.com"}`,
		"user:2":       `{"name": "Bob", "email": "bob@example.com"}`,
		"user:3":       `{"name": "Charlie", "email": "charlie@example.com"}`,
		"config:theme": "dark",
		"config:lang":  "en",
		"session:abc":  `{"user_id": "1", "expires": "2024-12-31"}`,
	}

	err = db.PutBatchString(sampleData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Created database with %d keys\n", db.Count())

	// Show database contents
	fmt.Println("\nDatabase contents:")
	db.ForEachString(func(key string, value string) error {
		fmt.Printf("  %s: %s\n", key, value)
		return nil
	})

	// --- Step 2: Create a backup ---
	fmt.Println("\n2. Creating backup:\n")

	backupFile := "data/backup.json"
	err = db.Backup(backupFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Backup created: %s\n", backupFile)

	// Show backup file info
	info, _ := os.Stat(backupFile)
	fmt.Printf("  Backup size: %d bytes\n", info.Size())

	// Peek at backup format
	fmt.Println("\n  Backup format (JSON):")
	backupContent, _ := os.ReadFile(backupFile)
	var backupData map[string]interface{}
	json.Unmarshal(backupContent, &backupData)

	fmt.Printf("  - Format version: %v\n", backupData["version"])
	fmt.Printf("  - Created at: %v\n", backupData["created_at"])
	fmt.Printf("  - Total records: %v\n", backupData["total_records"])

	db.Close()

	// --- Step 3: Simulate data loss ---
	fmt.Println("\n3. Simulating data loss:\n")

	// Delete the original database file
	err = os.Remove("data/production.skv")
	if err != nil {
		log.Printf("Warning: %v\n", err)
	}

	fmt.Println("✓ Original database deleted (simulating data loss)")

	// --- Step 4: Restore from backup ---
	fmt.Println("\n4. Restoring from backup:\n")

	// Open a new database (will be empty)
	restoredDB, err := skv.Open("data/restored")
	if err != nil {
		log.Fatal(err)
	}
	defer restoredDB.Close()

	fmt.Printf("  New database has %d keys (empty)\n", restoredDB.Count())

	// Restore from backup
	err = restoredDB.Restore(backupFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✓ Restored from backup\n")
	fmt.Printf("  Database now has %d keys\n", restoredDB.Count())

	// Verify restored data
	fmt.Println("\n5. Verifying restored data:\n")

	restored := 0
	restoredDB.ForEachString(func(key string, value string) error {
		restored++
		fmt.Printf("  ✓ %s: %s\n", key, value)
		return nil
	})

	fmt.Printf("\n✓ Verified %d restored records\n", restored)

	// --- Step 5: Backup best practices ---
	fmt.Println("\n6. Backup Best Practices:\n")

	fmt.Println("Regular backups:")
	fmt.Println("  • Create backups before major operations")
	fmt.Println("  • Schedule periodic backups (daily, weekly)")
	fmt.Println("  • Store backups in a different location")
	fmt.Println("  • Keep multiple backup versions")

	fmt.Println("\nBackup format:")
	fmt.Println("  • JSON format for portability")
	fmt.Println("  • UTF-8 text for human-readable keys/values")
	fmt.Println("  • Base64 for binary data")
	fmt.Println("  • Includes metadata (version, timestamp)")

	fmt.Println("\nRestore scenarios:")
	fmt.Println("  • Data corruption recovery")
	fmt.Println("  • Accidental deletion recovery")
	fmt.Println("  • Database migration")
	fmt.Println("  • Testing with production data")

	// --- Step 6: Example backup workflow ---
	fmt.Println("\n7. Example: Automated backup workflow:\n")

	// Create a timestamped backup
	timestamp := "2024-12-06_120000"
	backupPath := fmt.Sprintf("data/backup_%s.json", timestamp)

	err = restoredDB.Backup(backupPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Created timestamped backup: %s\n", backupPath)

	// List all backups
	fmt.Println("\nAvailable backups:")
	files, _ := os.ReadDir("data")
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 5 {
			ext := file.Name()[len(file.Name())-5:]
			if ext == ".json" {
				info, _ := file.Info()
				fmt.Printf("  • %s (%d bytes)\n", file.Name(), info.Size())
			}
		}
	}

	// --- Step 7: Compact before backup ---
	fmt.Println("\n8. Optimization: Compact before backup:\n")

	// Check stats
	stats, _ := restoredDB.Verify()
	fmt.Printf("  Before compact:\n")
	fmt.Printf("    Total size: %d bytes\n", stats.FileSize)
	fmt.Printf("    Wasted space: %d bytes\n", stats.WastedSpace)

	// Compact to optimize
	err = restoredDB.Compact()
	if err != nil {
		log.Fatal(err)
	}

	// Check stats after
	statsAfter, _ := restoredDB.Verify()
	fmt.Printf("\n  After compact:\n")
	fmt.Printf("    Total size: %d bytes\n", statsAfter.FileSize)
	fmt.Printf("    Space saved: %d bytes\n", stats.FileSize-statsAfter.FileSize)

	// Create optimized backup
	optimizedBackup := "data/backup_optimized.json"
	err = restoredDB.Backup(optimizedBackup)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✓ Created optimized backup: %s\n", optimizedBackup)

	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ Backup and restore operations completed successfully")
	fmt.Println("✓ Data integrity verified")
	fmt.Println("✓ Best practices demonstrated")

	fmt.Println("\n✅ Backup/Restore demonstration completed!")
}
