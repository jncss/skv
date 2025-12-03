// Example demonstrating iteration with ForEach
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/yourusername/skv"
)

func main() {
	db, err := skv.Open("iteration_example.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any existing data
	db.Clear()

	fmt.Println("=== ForEach Iteration ===\n")

	// Add sample data
	fmt.Println("1. Adding sample products...")
	products := map[string]string{
		"product:laptop":   "899.99",
		"product:mouse":    "24.99",
		"product:keyboard": "79.99",
		"product:monitor":  "299.99",
		"product:webcam":   "89.99",
	}

	db.PutBatchString(products)
	fmt.Printf("✓ Added %d products\n", len(products))

	// Iterate over all items
	fmt.Println("\n2. Listing all products:")
	err = db.ForEachString(func(key string, value string) error {
		// Extract product name from key
		name := strings.TrimPrefix(key, "product:")
		fmt.Printf("  %s: $%s\n", name, value)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Calculate total using iteration
	fmt.Println("\n3. Calculating total inventory value:")
	var total float64
	var count int

	db.ForEachString(func(key string, value string) error {
		var price float64
		fmt.Sscanf(value, "%f", &price)
		total += price
		count++
		return nil
	})

	fmt.Printf("  Total items: %d\n", count)
	fmt.Printf("  Total value: $%.2f\n", total)

	// Filter items using iteration
	fmt.Println("\n4. Products under $100:")
	db.ForEachString(func(key string, value string) error {
		var price float64
		fmt.Sscanf(value, "%f", &price)

		if price < 100 {
			name := strings.TrimPrefix(key, "product:")
			fmt.Printf("  %s: $%.2f\n", name, price)
		}
		return nil
	})

	// Early termination example
	fmt.Println("\n5. Finding first product over $200:")
	db.ForEachString(func(key string, value string) error {
		var price float64
		fmt.Sscanf(value, "%f", &price)

		if price > 200 {
			name := strings.TrimPrefix(key, "product:")
			fmt.Printf("  Found: %s at $%.2f\n", name, price)
			return fmt.Errorf("STOP") // Return error to stop iteration
		}
		return nil
	})

	// Using byte version for binary data
	fmt.Println("\n6. Working with binary data:")
	db.Clear()

	// Store some binary data
	db.Put([]byte("config:max_users"), []byte{0, 0, 3, 232}) // 1000 in big-endian
	db.Put([]byte("config:timeout"), []byte{0, 0, 0, 60})    // 60 seconds

	db.ForEach(func(key []byte, value []byte) error {
		fmt.Printf("  %s: %v\n", string(key), value)
		return nil
	})

	fmt.Println("\n✓ Example completed!")
}
