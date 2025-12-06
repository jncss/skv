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

	// Open or create a new SKV database
	// The .skv extension is automatically added if not present
	db, err := skv.Open("data/mydb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Basic CRUD Operations ===\n")

	// --- PUT: Add new key-value pairs ---
	// PutString is convenient for string data
	err = db.PutString("username", "alice")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Created key 'username' = 'alice'")

	err = db.PutString("email", "alice@example.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Created key 'email' = 'alice@example.com'")

	// Put with byte slices for binary data
	err = db.Put([]byte("age"), []byte{25})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Created key 'age' = 25")

	// --- GET: Retrieve values ---
	fmt.Println("\n=== Reading Values ===\n")

	username, err := db.GetString("username")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("username: %s\n", username)

	email, err := db.GetString("email")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("email: %s\n", email)

	// --- UPDATE: Modify existing values ---
	fmt.Println("\n=== Updating Values ===\n")

	err = db.UpdateString("username", "alice_smith")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Updated 'username' to 'alice_smith'")

	updatedUsername, _ := db.GetString("username")
	fmt.Printf("New username: %s\n", updatedUsername)

	// --- EXISTS: Check if key exists ---
	fmt.Println("\n=== Checking Key Existence ===\n")

	if db.HasString("username") {
		fmt.Println("✓ Key 'username' exists")
	}

	if !db.HasString("nonexistent") {
		fmt.Println("✗ Key 'nonexistent' does not exist")
	}

	// --- GET WITH DEFAULT: Safe retrieval ---
	fmt.Println("\n=== Get with Default Value ===\n")

	theme := db.GetOrDefaultString("theme", "dark")
	fmt.Printf("Theme (with default): %s\n", theme)

	// --- COUNT: Number of keys ---
	fmt.Println("\n=== Database Statistics ===\n")

	count := db.Count()
	fmt.Printf("Total keys in database: %d\n", count)

	// --- DELETE: Remove a key ---
	fmt.Println("\n=== Deleting Keys ===\n")

	err = db.DeleteString("age")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Deleted key 'age'")

	// Verify deletion
	if !db.HasString("age") {
		fmt.Println("✓ Key 'age' no longer exists")
	}

	fmt.Printf("\nFinal key count: %d\n", db.Count())

	// --- ERROR HANDLING ---
	fmt.Println("\n=== Error Handling Examples ===\n")

	// Trying to Put a key that already exists
	err = db.PutString("username", "bob")
	if err == skv.ErrKeyExists {
		fmt.Println("✗ Cannot Put - key 'username' already exists (use Update instead)")
	}

	// Trying to Update a key that doesn't exist
	err = db.UpdateString("nonexistent", "value")
	if err == skv.ErrKeyNotFound {
		fmt.Println("✗ Cannot Update - key 'nonexistent' not found (use Put instead)")
	}

	// Trying to Get a key that doesn't exist
	_, err = db.GetString("missing")
	if err == skv.ErrKeyNotFound {
		fmt.Println("✗ Cannot Get - key 'missing' not found")
	}

	fmt.Println("\n✅ Basic operations completed successfully!")
}
