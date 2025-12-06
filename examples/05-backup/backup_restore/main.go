package main

import (
"fmt"
"log"
"os"

"github.com/jncss/skv"
)

func main() {
	dbFile := "demo.skv"
	backupFile := "demo_backup.json"

	// Clean up at the end
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	fmt.Println("=== SKV Backup/Restore Demo ===")
	fmt.Println()

	// 1. Create database and add some data
	fmt.Println("1. Creating database with sample data...")
	db, err := skv.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	// Add various types of data
	data := map[string]string{
		"user:1:name":  "Alice Johnson",
		"user:1:email": "alice@example.com",
		"user:1:bio":   "Software engineer interested in databases and Go",
		"user:2:name":  "Bob Smith",
		"user:2:email": "bob@example.com",
		"config:theme": "dark",
		"config:lang":  "en",
	}

	for key, value := range data {
		if err := db.PutString(key, value); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  Added: %s\n", key)
	}

	// Add some binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	if err := db.Put([]byte("binary:data"), binaryData); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Added binary data (7 bytes)\n")

	fmt.Printf("\nDatabase contains %d keys\n", db.Count())
	fmt.Println()

	// 2. Create backup
	fmt.Println("2. Creating JSON backup...")
	if err := db.Backup(backupFile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Backup saved to: %s\n", backupFile)
	fmt.Println()

	// Show a snippet of the backup file
	fmt.Println("3. Backup file preview:")
	backupContent, _ := os.ReadFile(backupFile)
	// Show first 500 characters
	preview := string(backupContent)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(preview)
	fmt.Println()

	// 4. Modify the database
	fmt.Println("4. Modifying database...")
	db.UpdateString("user:1:name", "Alice Williams")
	db.DeleteString("user:2:email")
	db.PutString("user:3:name", "Charlie Brown")
	fmt.Println("  Updated user:1:name")
	fmt.Println("  Deleted user:2:email")
	fmt.Println("  Added user:3:name")
	fmt.Printf("\nDatabase now contains %d keys\n", db.Count())
	fmt.Println()

	// 5. Show current state
	fmt.Println("5. Current database state:")
	db.ForEachString(func(key, value string) error {
if len(value) > 50 {
			value = value[:50] + "..."
		}
		fmt.Printf("  %s = %s\n", key, value)
		return nil
	})
	fmt.Println()

	// 6. Restore from backup
	fmt.Println("6. Restoring from backup...")
	if err := db.Restore(backupFile); err != nil {
		log.Fatal(err)
	}
	fmt.Println("  Restore completed")
	fmt.Printf("\nDatabase now contains %d keys\n", db.Count())
	fmt.Println()

	// 7. Show restored state
	fmt.Println("7. Database state after restore:")
	db.ForEachString(func(key, value string) error {
if len(value) > 50 {
			value = value[:50] + "..."
		}
		fmt.Printf("  %s = %s\n", key, value)
		return nil
	})
	fmt.Println()

	// 8. Verify specific changes
	fmt.Println("8. Verification:")

	name, _ := db.GetString("user:1:name")
	if name == "Alice Johnson" {
		fmt.Println("  user:1:name restored to original: Alice Johnson")
	}

	if db.ExistsString("user:2:email") {
		fmt.Println("  user:2:email was restored")
	}

	if db.ExistsString("user:3:name") {
		fmt.Println("  user:3:name (not in backup) still exists")
	}

	restoredBinary, _ := db.Get([]byte("binary:data"))
	if len(restoredBinary) == 7 {
		fmt.Println("  Binary data correctly restored (7 bytes)")
	}

	db.Close()

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
	fmt.Println()
	fmt.Println("Key Takeaways:")
	fmt.Println("- Backup creates a human-readable JSON file")
	fmt.Println("- Small text values stored as strings, binary as base64")
	fmt.Println("- Restore overwrites existing keys with backup values")
	fmt.Println("- Keys not in backup remain in the database")
	fmt.Println("- Perfect for database migration and disaster recovery")
}
