// Example showing database maintenance operations
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yourusername/skv"
)

func printFileSize(filename string) {
	info, err := os.Stat(filename)
	if err != nil {
		return
	}
	fmt.Printf("  File size: %d bytes\n", info.Size())
}

func main() {
	dbFile := "maintenance_example.skv"

	db, err := skv.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Database Maintenance ===\n")

	// Start fresh
	db.Clear()

	// Add initial data
	fmt.Println("1. Adding 100 records...")
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%03d", i)
		value := fmt.Sprintf("value_%03d", i)
		db.PutString(key, value)
	}
	printFileSize(dbFile)

	// Check database stats
	fmt.Println("\n2. Database statistics:")
	stats, _ := db.Verify()
	fmt.Printf("  Total records: %d\n", stats.TotalRecords)
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted records: %d\n", stats.DeletedRecords)

	// Update half of the records
	fmt.Println("\n3. Updating 50 records...")
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key:%03d", i)
		value := fmt.Sprintf("updated_value_%03d", i)
		db.UpdateString(key, value)
	}
	printFileSize(dbFile)

	// Check stats after updates
	fmt.Println("\n4. After updates:")
	stats, _ = db.Verify()
	fmt.Printf("  Total records: %d (old + new versions)\n", stats.TotalRecords)
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted records: %d (old versions marked deleted)\n", stats.DeletedRecords)

	// Delete some records
	fmt.Println("\n5. Deleting 25 records...")
	for i := 50; i < 75; i++ {
		key := fmt.Sprintf("key:%03d", i)
		db.DeleteString(key)
	}

	fmt.Println("\n6. After deletions:")
	stats, _ = db.Verify()
	fmt.Printf("  Total records: %d\n", stats.TotalRecords)
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted records: %d\n", stats.DeletedRecords)
	printFileSize(dbFile)

	// Compact to reclaim space
	fmt.Println("\n7. Compacting database...")
	if err := db.Compact(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n8. After compaction:")
	stats, _ = db.Verify()
	fmt.Printf("  Total records: %d (only active records)\n", stats.TotalRecords)
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted records: %d\n", stats.DeletedRecords)
	printFileSize(dbFile)

	// Verify data integrity
	fmt.Println("\n9. Verifying updated values are preserved...")
	value, _ := db.GetString("key:000")
	if value == "updated_value_000" {
		fmt.Println("✓ Updated values preserved")
	}

	value, _ = db.GetString("key:075")
	if value == "value_075" {
		fmt.Println("✓ Non-updated values preserved")
	}

	if !db.HasString("key:050") {
		fmt.Println("✓ Deleted keys removed")
	}

	fmt.Printf("\n✓ Final count: %d active keys\n", db.Count())

	// Close with compact option
	fmt.Println("\n10. Demonstrating CloseWithCompact...")
	fmt.Println("  (This compacts automatically on close)")

	// Add and delete some more to create garbage
	for i := 0; i < 10; i++ {
		db.PutString(fmt.Sprintf("temp:%d", i), "temporary")
		db.DeleteString(fmt.Sprintf("temp:%d", i))
	}

	stats, _ = db.Verify()
	fmt.Printf("  Before close: %d total, %d deleted\n", stats.TotalRecords, stats.DeletedRecords)

	if err := db.CloseWithCompact(); err != nil {
		log.Fatal(err)
	}

	// Reopen to verify
	db, _ = skv.Open(dbFile)
	defer db.Close()

	stats, _ = db.Verify()
	fmt.Printf("  After CloseWithCompact: %d total, %d deleted\n", stats.TotalRecords, stats.DeletedRecords)

	fmt.Println("\n✓ Maintenance example completed!")
}
