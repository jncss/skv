package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	// Create data directory if it doesn't exist
	os.MkdirAll("data", 0755)

	// Open database
	db, err := skv.Open("data/put_update_demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Understanding Put vs Update ===\n")

	// --- PUT: Only for NEW keys ---
	fmt.Println("1. PUT - Creating new keys:")
	fmt.Println("   Put() only works when the key does NOT exist yet\n")

	err = db.PutString("product", "laptop")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Put('product', 'laptop') - Success!")

	// Trying to Put the same key again will fail
	err = db.PutString("product", "desktop")
	if err == skv.ErrKeyExists {
		fmt.Println("✗ Put('product', 'desktop') - Failed: key already exists")
	}

	value, _ := db.GetString("product")
	fmt.Printf("   Current value: %s (unchanged)\n", value)

	// --- UPDATE: Only for EXISTING keys ---
	fmt.Println("\n2. UPDATE - Modifying existing keys:")
	fmt.Println("   Update() only works when the key EXISTS\n")

	err = db.UpdateString("product", "desktop")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Update('product', 'desktop') - Success!")

	value, _ = db.GetString("product")
	fmt.Printf("   Current value: %s (updated)\n", value)

	// Trying to Update a non-existent key will fail
	err = db.UpdateString("category", "electronics")
	if err == skv.ErrKeyNotFound {
		fmt.Println("\n✗ Update('category', 'electronics') - Failed: key not found")
	}

	// --- BEST PRACTICES ---
	fmt.Println("\n3. BEST PRACTICES:\n")

	// Pattern 1: Check before deciding
	fmt.Println("Pattern 1 - Check if key exists first:")
	key := "price"
	if db.HasString(key) {
		db.UpdateString(key, "999")
		fmt.Printf("✓ Updated existing key '%s'\n", key)
	} else {
		db.PutString(key, "999")
		fmt.Printf("✓ Created new key '%s'\n", key)
	}

	// Pattern 2: Handle errors explicitly
	fmt.Println("\nPattern 2 - Handle errors explicitly:")
	err = db.PutString("stock", "50")
	if err == skv.ErrKeyExists {
		// Key exists, use Update instead
		db.UpdateString("stock", "50")
		fmt.Println("✓ Key existed, used Update instead")
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("✓ New key created with Put")
	}

	// --- CLEAR and START FRESH ---
	fmt.Println("\n4. CLEARING database:\n")

	beforeCount := db.Count()
	fmt.Printf("Keys before Clear: %d\n", beforeCount)

	err = db.Clear()
	if err != nil {
		log.Fatal(err)
	}

	afterCount := db.Count()
	fmt.Printf("Keys after Clear: %d\n", afterCount)

	// Now we can Put new keys again
	db.PutString("status", "empty")
	fmt.Println("✓ Database cleared and new key added")

	fmt.Println("\n=== Summary ===")
	fmt.Println("• Use Put() to create NEW keys")
	fmt.Println("• Use Update() to modify EXISTING keys")
	fmt.Println("• Use Has() to check existence before operations")
	fmt.Println("• Handle ErrKeyExists and ErrKeyNotFound appropriately")

	fmt.Println("\n✅ Demonstration completed!")
}
