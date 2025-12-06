package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	os.MkdirAll("data", 0755)

	db, err := skv.Open("data/maintenance_demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Database Maintenance ===\n")

	// Setup: Create, update, and delete some data to generate fragmentation
	fmt.Println("Setting up test data with fragmentation...")

	// Insert initial data
	for i := 1; i <= 20; i++ {
		key := fmt.Sprintf("item:%d", i)
		value := fmt.Sprintf("value_%d", i)
		db.PutString(key, value)
	}

	// Update some items (creates new versions, old ones marked deleted)
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("item:%d", i)
		value := fmt.Sprintf("updated_value_%d", i)
		db.UpdateString(key, value)
	}

	// Delete some items
	for i := 11; i <= 15; i++ {
		key := fmt.Sprintf("item:%d", i)
		db.DeleteString(key)
	}

	fmt.Printf("✓ Data setup complete\n")
	fmt.Printf("  Active keys: %d\n", db.Count())

	// --- Verify: Check database statistics ---
	fmt.Println("\n1. Verify - Database Statistics:")

	stats, err := db.Verify()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n  Total Records:    %d\n", stats.TotalRecords)
	fmt.Printf("  Active Records:   %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted Records:  %d\n", stats.DeletedRecords)
	fmt.Printf("  File Size:        %d bytes\n", stats.FileSize)
	fmt.Printf("  Data Size:        %d bytes\n", stats.DataSize)
	fmt.Printf("  Wasted Space:     %d bytes\n", stats.WastedSpace)
	fmt.Printf("  Wasted %%:         %.2f%%\n", stats.WastedPercent)

	// --- Compact: Remove deleted records and duplicates ---
	fmt.Println("\n2. Compact - Optimize database:")

	fmt.Println("\n  Before compaction:")
	fmt.Printf("    File size: %d bytes\n", stats.FileSize)
	fmt.Printf("    Wasted space: %d bytes (%.2f%%)\n", stats.WastedSpace, stats.WastedPercent)

	err = db.Compact()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n  ✓ Compaction completed")

	// Verify again after compaction
	statsAfter, err := db.Verify()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n  After compaction:")
	fmt.Printf("    File size: %d bytes\n", statsAfter.FileSize)
	fmt.Printf("    Space saved: %d bytes\n", stats.FileSize-statsAfter.FileSize)
	fmt.Printf("    Wasted space: %d bytes (%.2f%%)\n", statsAfter.WastedSpace, statsAfter.WastedPercent)

	if stats.FileSize > 0 {
		savingsPercent := float64(stats.FileSize-statsAfter.FileSize) / float64(stats.FileSize) * 100
		fmt.Printf("    Reduction: %.2f%%\n", savingsPercent)
	}

	// --- When to compact ---
	fmt.Println("\n3. When to Compact:")
	fmt.Println("   • After many updates or deletes")
	fmt.Println("   • When wasted space exceeds a threshold (e.g., 30%)")
	fmt.Println("   • During scheduled maintenance windows")
	fmt.Println("   • Before backups to reduce file size")

	// --- Example: Conditional compaction ---
	fmt.Println("\n4. Conditional Compaction Pattern:")

	currentStats, _ := db.Verify()
	threshold := 0.30 // 30% wasted space

	fmt.Printf("\n   Current wasted space: %.2f%%\n", currentStats.WastedPercent)
	fmt.Printf("   Threshold: %.2f%%\n", threshold*100)

	if currentStats.WastedPercent > threshold*100 {
		fmt.Println("   ➜ Triggering compaction (threshold exceeded)")
		db.Compact()
	} else {
		fmt.Println("   ➜ No compaction needed (below threshold)")
	}

	// --- CloseWithCompact ---
	fmt.Println("\n5. Close with Compact:")
	fmt.Println("   Use CloseWithCompact() instead of Close() to:")
	fmt.Println("   • Automatically compact before closing")
	fmt.Println("   • Ensure optimal file size on disk")
	fmt.Println("   • Useful for long-running applications")

	fmt.Println("\n   Example:")
	fmt.Println("   defer db.CloseWithCompact() // Instead of db.Close()")

	fmt.Println("\n✅ Maintenance operations completed!")
}
