// Example demonstrating batch operations
package main

import (
	"fmt"
	"log"

	"github.com/jncss/skv"
)

func main() {
	db, err := skv.Open("batch_example.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Batch Operations ===\n")

	// Batch insert - insert multiple keys at once
	fmt.Println("1. Batch insert users...")
	users := map[string]string{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
		"user:4": "Diana",
		"user:5": "Eve",
	}

	if err := db.PutBatchString(users); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Inserted %d users\n", len(users))

	// Batch get - retrieve multiple keys at once
	fmt.Println("\n2. Batch get specific users...")
	keys := []string{"user:1", "user:3", "user:5", "user:99"} // user:99 doesn't exist

	results, err := db.GetBatchString(keys)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Retrieved %d out of %d requested keys:\n", len(results), len(keys))
	for key, value := range results {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Demonstrate atomic behavior of batch insert
	fmt.Println("\n3. Attempting batch insert with duplicate...")
	duplicateUsers := map[string]string{
		"user:1": "Alice Updated", // This already exists!
		"user:6": "Frank",
	}

	err = db.PutBatchString(duplicateUsers)
	if err != nil {
		fmt.Printf("✗ Batch insert failed (as expected): %v\n", err)
		fmt.Println("  None of the keys were inserted (atomic operation)")
	}

	// Verify user:6 was not inserted
	if !db.HasString("user:6") {
		fmt.Println("✓ Confirmed: user:6 was not inserted")
	}

	// Verify user:1 still has original value
	value, _ := db.GetString("user:1")
	if value == "Alice" {
		fmt.Println("✓ Confirmed: user:1 unchanged")
	}

	fmt.Printf("\n✓ Final count: %d users\n", db.Count())
}
