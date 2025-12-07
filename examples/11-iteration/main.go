package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")
	defer os.Remove("demo.skv")
	defer os.Remove("demo.skv.wal")

	// Open database
	db, err := skv.Open("demo.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Insert sample data
	data := map[string]string{
		"user:alice":   "Alice Johnson",
		"user:bob":     "Bob Smith",
		"user:charlie": "Charlie Brown",
		"product:001":  "Laptop",
		"product:002":  "Mouse",
		"product:003":  "Keyboard",
	}

	for key, value := range data {
		if err := db.Put([]byte(key), []byte(value)); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("=== Iteration Methods Comparison ===")
	fmt.Println()

	// Method 1: ForEach - Fast but unordered
	fmt.Println("1. ForEach (fast, unordered iteration)")
	fmt.Println("   - Memory: O(1) for values (read on-demand)")
	fmt.Println("   - Order: NOT guaranteed (iterates over map)")
	fmt.Println("   - Use case: Process all records, don't care about order")
	fmt.Println()

	count := 0
	db.ForEachString(func(key, value string) error {
		count++
		fmt.Printf("   [%d] %s = %s\n", count, key, value)
		return nil
	})

	// Method 2: NewCursor - Ordered iteration (all keys)
	fmt.Println()
	fmt.Println("2. NewCursor (ordered iteration, sorted by key)")
	fmt.Println("   - Memory: O(n) for keys (must sort all keys)")
	fmt.Println("   - Order: Guaranteed (sorted)")
	fmt.Println("   - Use case: Need sorted output")
	fmt.Println()

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	count = 0
	for {
		key, value, err := cursor.Next()
		if err != nil {
			break // End of cursor
		}
		count++
		fmt.Printf("   [%d] %s = %s\n", count, string(key), string(value))
	}

	// Method 3: NewCursor with Range - Ordered subset
	fmt.Println()
	fmt.Println("3. NewCursor with Range (ordered, filtered)")
	fmt.Println("   - Memory: O(n) for keys, but can limit range")
	fmt.Println("   - Use case: Iterate only a subset of keys")
	fmt.Println()

	cursor2 := db.NewCursor(&skv.CursorOptions{
		From: []byte("user:"),
		To:   []byte("user:\xff"), // \xff is higher than any printable char
	})
	defer cursor2.Close()

	count = 0
	for {
		key, value, err := cursor2.Next()
		if err != nil {
			break
		}
		count++
		fmt.Printf("   [%d] %s = %s\n", count, string(key), string(value))
	}

	// Method 4: PrefixCursor - Most efficient for prefix queries
	fmt.Println()
	fmt.Println("4. PrefixCursor (ordered, prefix match)")
	fmt.Println("   - Memory: O(n) for keys matching prefix")
	fmt.Println("   - Use case: All keys with specific prefix")
	fmt.Println()

	cursor3 := db.PrefixCursor([]byte("product:"), false)
	defer cursor3.Close()

	count = 0
	for {
		key, value, err := cursor3.Next()
		if err != nil {
			break
		}
		count++
		fmt.Printf("   [%d] %s = %s\n", count, string(key), string(value))
	}

	// Method 5: Reverse iteration
	fmt.Println()
	fmt.Println("5. Reverse Cursor (descending order)")
	fmt.Println("   - Order: Reverse sorted")
	fmt.Println("   - Use case: Iterate from end to start")
	fmt.Println()

	cursor4 := db.NewCursor(&skv.CursorOptions{
		Reverse: true,
	})
	defer cursor4.Close()

	count = 0
	for {
		key, value, err := cursor4.Next()
		if err != nil {
			break
		}
		count++
		fmt.Printf("   [%d] %s = %s\n", count, string(key), string(value))
		if count >= 3 { // Show only first 3
			break
		}
	}

	// Performance comparison
	fmt.Println()
	fmt.Println("=== Performance Guidelines ===")
	fmt.Println("ForEach:       Best for: Processing all records, don't care about order")
	fmt.Println("               Memory: O(1) for values")
	fmt.Println()
	fmt.Println("NewCursor:     Best for: Need sorted output of all records")
	fmt.Println("               Memory: O(n) for keys")
	fmt.Println()
	fmt.Println("RangeCursor:   Best for: Iterate subset (from..to)")
	fmt.Println("               Memory: O(n) for keys in range")
	fmt.Println()
	fmt.Println("PrefixCursor:  Best for: All keys starting with prefix")
	fmt.Println("               Memory: O(n) for matching keys")
}
