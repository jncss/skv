package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	os.MkdirAll("data", 0755)

	db, err := skv.Open("data/batch_demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Batch Operations ===\n")

	// --- PutBatchString: Insert multiple keys at once ---
	fmt.Println("1. Batch Insert with PutBatchString:")

	users := map[string]string{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
		"user:4": "Diana",
		"user:5": "Eve",
	}

	err = db.PutBatchString(users)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Inserted %d users in a single batch operation\n", len(users))
	fmt.Printf("Total keys in database: %d\n", db.Count())

	// --- GetBatchString: Retrieve multiple keys at once ---
	fmt.Println("\n2. Batch Retrieve with GetBatchString:")

	keysToGet := []string{"user:1", "user:3", "user:5"}
	results, err := db.GetBatchString(keysToGet)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Retrieved %d values:\n", len(results))
	for key, value := range results {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// --- PutBatch with byte slices ---
	fmt.Println("\n3. Batch Insert with byte data (PutBatch):")

	settings := map[string][]byte{
		"max_users":      []byte{100},
		"timeout":        []byte{30},
		"retry_attempts": []byte{3},
	}

	err = db.PutBatch(settings)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Inserted %d settings\n", len(settings))

	// --- GetBatch with byte slices ---
	fmt.Println("\n4. Batch Retrieve with byte data (GetBatch):")

	settingKeys := [][]byte{
		[]byte("max_users"),
		[]byte("timeout"),
		[]byte("retry_attempts"),
	}

	settingsData, err := db.GetBatch(settingKeys)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Retrieved %d settings:\n", len(settingsData))
	for key, value := range settingsData {
		fmt.Printf("  %s: %v\n", key, value[0])
	}

	// --- Performance comparison ---
	fmt.Println("\n5. Performance Benefits:")
	fmt.Println("   Batch operations are more efficient because:")
	fmt.Println("   • Single lock acquisition for multiple operations")
	fmt.Println("   • Reduced function call overhead")
	fmt.Println("   • Better cache locality")

	fmt.Printf("\n   Total keys: %d\n", db.Count())

	fmt.Println("\n✅ Batch operations completed!")
}
