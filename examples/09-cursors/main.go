package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jncss/skv"
)

type Product struct {
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}

func main() {
	// Clean up from previous runs
	os.Remove("products.skv")
	os.Remove("products_by_category.json")

	db, err := skv.Open("products.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Cursor Example ===")
	fmt.Println()

	// Insert some products
	products := []Product{
		{SKU: "LAPTOP-001", Name: "ThinkPad X1", Category: "electronics", Price: 1299.99},
		{SKU: "LAPTOP-002", Name: "MacBook Pro", Category: "electronics", Price: 2399.99},
		{SKU: "BOOK-001", Name: "Go Programming", Category: "books", Price: 49.99},
		{SKU: "BOOK-002", Name: "Database Design", Category: "books", Price: 59.99},
		{SKU: "BOOK-003", Name: "System Architecture", Category: "books", Price: 69.99},
		{SKU: "DESK-001", Name: "Standing Desk", Category: "furniture", Price: 599.99},
		{SKU: "DESK-002", Name: "Office Chair", Category: "furniture", Price: 399.99},
		{SKU: "MOUSE-001", Name: "Wireless Mouse", Category: "electronics", Price: 29.99},
	}

	fmt.Println("1. Inserting products...")
	for _, p := range products {
		data, _ := json.Marshal(p)
		if err := db.Put([]byte(p.SKU), data); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Inserted %d products\n\n", len(products))

	// Create secondary index by category
	fmt.Println("2. Creating index by category...")
	db.CreateIndex("by_category", func(data []byte) []byte {
		var p Product
		json.Unmarshal(data, &p)
		return []byte(p.Category)
	})
	fmt.Println("Index created\n")

	// Example 1: Iterate all products (primary keys)
	fmt.Println("3. All products (sorted by SKU):")
	fmt.Println("   ------------------------------------")
	cursor := db.AllCursor(false)
	for {
		key, value, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("   %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	cursor.Close()
	fmt.Println()

	// Example 2: Prefix cursor (all books)
	fmt.Println("4. Products with SKU prefix 'BOOK-':")
	fmt.Println("   ------------------------------------")
	bookCursor := db.PrefixCursor([]byte("BOOK-"), false)
	for {
		key, value, err := bookCursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("   %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	bookCursor.Close()
	fmt.Println()

	// Example 3: Range query (BOOK-001 to DESK-001)
	fmt.Println("5. Products from BOOK-001 to DESK-001 (inclusive):")
	fmt.Println("   ------------------------------------")
	rangeCursor := db.NewCursor(&skv.CursorOptions{
		From: []byte("BOOK-001"),
		To:   []byte("DESK-001"),
	})
	for {
		key, value, err := rangeCursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("   %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	rangeCursor.Close()
	fmt.Println()

	// Example 4: Index cursor (all electronics)
	fmt.Println("6. All electronics (using index prefix cursor):")
	fmt.Println("   ------------------------------------")
	indexCursor, _ := db.PrefixIndexCursor("by_category", []byte("electronics"), false)
	for {
		key, value, err := indexCursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("   %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	indexCursor.Close()
	fmt.Println()

	// Example 5: Reverse iteration
	fmt.Println("7. All products in reverse order:")
	fmt.Println("   ------------------------------------")
	reverseCursor := db.AllCursor(true)
	for {
		key, value, err := reverseCursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("   %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	reverseCursor.Close()
	fmt.Println()

	// Example 6: ForEach method
	fmt.Println("8. Calculate total value using ForEach:")
	fmt.Println("   ------------------------------------")
	total := 0.0
	foreachCursor := db.AllCursor(false)
	foreachCursor.ForEach(func(key, value []byte) bool {
		var p Product
		json.Unmarshal(value, &p)
		total += p.Price
		return true
	})
	foreachCursor.Close()
	fmt.Printf("   Total inventory value: $%.2f\n", total)
	fmt.Println()

	// Example 7: Collect all keys
	fmt.Println("9. Collect all SKUs:")
	fmt.Println("   ------------------------------------")
	collectCursor := db.AllCursor(false)
	keys := collectCursor.Keys()
	collectCursor.Close()
	for i, key := range keys {
		fmt.Printf("   %d. %s\n", i+1, string(key))
	}
	fmt.Println()

	// Example 8: Seek to specific position
	fmt.Println("10. Seek to 'DESK-001' and continue:")
	fmt.Println("    ------------------------------------")
	seekCursor := db.AllCursor(false)
	seekCursor.Seek([]byte("DESK-001"))
	for {
		key, value, err := seekCursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var p Product
		json.Unmarshal(value, &p)
		fmt.Printf("    %-15s %-20s $%.2f\n", string(key), p.Name, p.Price)
	}
	seekCursor.Close()
	fmt.Println()

	fmt.Println("Example completed successfully!")

	// Cleanup
	os.Remove("products.skv")
	os.Remove("products_by_category.json")
}
