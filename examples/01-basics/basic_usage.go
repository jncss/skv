// Basic usage example showing common operations
package main

import (
	"fmt"
	"log"

	"github.com/yourusername/skv"
)

func main() {
	// Open or create database
	db, err := skv.Open("mydata.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Basic SKV Operations ===\n")

	// Store string data
	fmt.Println("1. Storing data...")
	db.PutString("username", "alice")
	db.PutString("email", "alice@example.com")
	db.PutString("role", "admin")
	fmt.Println("✓ Data stored")

	// Retrieve data
	fmt.Println("\n2. Retrieving data...")
	username, _ := db.GetString("username")
	email, _ := db.GetString("email")
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Email: %s\n", email)

	// Update existing key
	fmt.Println("\n3. Updating data...")
	db.UpdateString("email", "alice.smith@example.com")
	email, _ = db.GetString("email")
	fmt.Printf("New email: %s\n", email)

	// Check if key exists
	fmt.Println("\n4. Checking existence...")
	if db.HasString("username") {
		fmt.Println("✓ Username exists")
	}
	if !db.HasString("phone") {
		fmt.Println("✓ Phone doesn't exist")
	}

	// Count keys
	fmt.Printf("\n5. Total keys: %d\n", db.Count())

	// List all keys
	fmt.Println("\n6. All keys:")
	keys, _ := db.KeysString()
	for _, key := range keys {
		value, _ := db.GetString(key)
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Delete a key
	fmt.Println("\n7. Deleting 'role'...")
	db.DeleteString("role")
	fmt.Printf("Keys remaining: %d\n", db.Count())

	// Get with default value
	fmt.Println("\n8. Get with default...")
	theme := db.GetOrDefaultString("theme", "dark")
	fmt.Printf("Theme: %s (default)\n", theme)

	fmt.Println("\n✓ Example completed successfully!")
}
