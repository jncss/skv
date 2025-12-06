package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	os.MkdirAll("data", 0755)

	db, err := skv.Open("data/iteration_demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Iteration with ForEach ===\n")

	// Setup: Add some sample data
	fmt.Println("Setting up sample data...")
	products := map[string]string{
		"product:laptop":     "Dell XPS 13",
		"product:mouse":      "Logitech MX Master",
		"product:keyboard":   "Keychron K8",
		"product:monitor":    "LG UltraWide",
		"product:headphones": "Sony WH-1000XM4",
	}

	db.PutBatchString(products)
	fmt.Printf("✓ Added %d products\n\n", len(products))

	// --- ForEachString: Iterate over all key-value pairs ---
	fmt.Println("1. Iterate with ForEachString:")

	count := 0
	err = db.ForEachString(func(key string, value string) error {
		count++
		fmt.Printf("  [%d] %s: %s\n", count, key, value)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✓ Iterated over %d items\n", count)

	// --- ForEach with filtering ---
	fmt.Println("\n2. Filtered iteration (keys starting with 'product:'):")

	filtered := 0
	err = db.ForEachString(func(key string, value string) error {
		// Filter: only process keys starting with "product:"
		if len(key) >= 8 && key[:8] == "product:" {
			filtered++
			productName := key[8:] // Remove prefix
			fmt.Printf("  %s: %s\n", productName, value)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✓ Found %d products\n", filtered)

	// --- ForEach with early termination ---
	fmt.Println("\n3. Early termination (stop after 3 items):")

	itemsProcessed := 0
	maxItems := 3

	err = db.ForEachString(func(key string, value string) error {
		if itemsProcessed >= maxItems {
			return fmt.Errorf("reached limit") // Return error to stop iteration
		}
		itemsProcessed++
		fmt.Printf("  [%d] %s: %s\n", itemsProcessed, key, value)
		return nil
	})

	// ForEach returns the error that stopped iteration
	if err != nil {
		fmt.Printf("\n✓ Stopped early: %v\n", err)
	}

	// --- ForEach with byte slices ---
	fmt.Println("\n4. ForEach with byte data:")

	// Add some binary data
	binaryData := map[string][]byte{
		"config:version": {1, 0, 0},
		"config:flags":   {0xFF, 0x00},
	}
	db.PutBatch(binaryData)

	err = db.ForEach(func(key []byte, value []byte) error {
		keyStr := string(key)
		if len(keyStr) >= 7 && keyStr[:7] == "config:" {
			fmt.Printf("  %s: %v\n", keyStr, value)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// --- Collecting results during iteration ---
	fmt.Println("\n5. Collecting results (building a list):")

	var productNames []string
	err = db.ForEachString(func(key string, value string) error {
		if len(key) >= 8 && key[:8] == "product:" {
			productNames = append(productNames, value)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Collected product names: %v\n", productNames)

	// --- Keys() alternative ---
	fmt.Println("\n6. Alternative: Get all keys first with KeysString():")

	allKeys, err := db.KeysString()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Total keys: %d\n", len(allKeys))
	fmt.Printf("  Keys: %v\n", allKeys)

	fmt.Println("\n=== Iteration Summary ===")
	fmt.Println("• ForEach/ForEachString: Iterate over all key-value pairs")
	fmt.Println("• Return error to stop iteration early")
	fmt.Println("• Use filtering logic inside the callback")
	fmt.Println("• Keys/KeysString: Get all keys as a slice")

	fmt.Println("\n✅ Iteration examples completed!")
}
