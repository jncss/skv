package main

import (
	"fmt"
	"log"
	"os"

	skv "github.com/jncss/skv"
)

func main() {
	// Create a database
	db, err := skv.Open("example_crc.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("example_crc.skv")

	// Insert some data
	db.PutString("user:1", "John Doe")
	db.PutString("user:2", "Jane Smith")
	db.PutString("user:3", "Bob Johnson")

	// Verify database integrity
	stats, err := db.Verify()
	if err != nil {
		log.Fatalf("Database corruption detected: %v", err)
	}

	fmt.Printf("Database verified successfully!\n")
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Total records: %d\n", stats.TotalRecords)
	fmt.Printf("  CRC checks: All passed ✓\n")

	// Try to read data (CRC is automatically verified)
	value, err := db.GetString("user:1")
	if err != nil {
		log.Fatalf("Error reading data: %v", err)
	}
	fmt.Printf("\nRead 'user:1': %s (CRC verified ✓)\n", value)

	// Update a record (old record is marked deleted)
	db.UpdateString("user:1", "John Doe Updated")
	value, _ = db.GetString("user:1")
	fmt.Printf("Updated 'user:1': %s (CRC verified ✓)\n", value)

	// Verify again - deleted records are skipped (CRC not checked)
	stats, err = db.Verify()
	if err != nil {
		log.Fatalf("Database corruption after update: %v", err)
	}
	fmt.Printf("\nDatabase verified after update:\n")
	fmt.Printf("  Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("  Deleted records: %d (CRC not verified)\n", stats.DeletedRecords)
}
