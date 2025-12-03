// Example demonstrating Put vs Update behavior
package main

import (
	"fmt"
	"log"

	"github.com/jncss/skv"
)

func main() {
	// Open database
	db, err := skv.Open("example.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Put: Create a new key
	fmt.Println("Creating new key 'user1'...")
	if err := db.Put([]byte("user1"), []byte("Alice")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Key created")

	// Try to Put the same key again - will fail
	fmt.Println("\nTrying to Put 'user1' again...")
	if err := db.Put([]byte("user1"), []byte("Bob")); err != nil {
		if err == skv.ErrKeyExists {
			fmt.Println("✗ Error: Key already exists (as expected)")
		} else {
			log.Fatal(err)
		}
	}

	// Verify original value is preserved
	value, _ := db.Get([]byte("user1"))
	fmt.Printf("Current value: %s\n", value)

	// Update: Modify existing key
	fmt.Println("\nUpdating 'user1' with Update()...")
	if err := db.Update([]byte("user1"), []byte("Alice Smith")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Key updated")

	// Verify new value
	value, _ = db.Get([]byte("user1"))
	fmt.Printf("New value: %s\n", value)

	// Try to Update a non-existent key - will fail
	fmt.Println("\nTrying to Update non-existent key 'user2'...")
	if err := db.Update([]byte("user2"), []byte("Charlie")); err != nil {
		if err == skv.ErrKeyNotFound {
			fmt.Println("✗ Error: Key not found (as expected)")
		} else {
			log.Fatal(err)
		}
	}

	// Delete and re-add
	fmt.Println("\nDeleting 'user1'...")
	db.Delete([]byte("user1"))
	fmt.Println("✓ Key deleted")

	// Now Put works again
	fmt.Println("\nPutting 'user1' again after delete...")
	if err := db.Put([]byte("user1"), []byte("Bob")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Key created")

	value, _ = db.Get([]byte("user1"))
	fmt.Printf("Final value: %s\n", value)
}
