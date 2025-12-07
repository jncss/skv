package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/jncss/skv"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}

func main() {
	// Open database
	db, err := skv.Open("users.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Secondary Indexes Example ===")
	fmt.Println()

	// Insert some users
	users := []User{
		{ID: "u001", Email: "alice@example.com", Name: "Alice Johnson", Age: 30},
		{ID: "u002", Email: "bob@example.com", Name: "Bob Smith", Age: 25},
		{ID: "u003", Email: "charlie@example.com", Name: "Charlie Brown", Age: 35},
		{ID: "u004", Email: "diana@example.com", Name: "Diana Prince", Age: 28},
	}

	fmt.Println("1. Inserting users...")
	for _, user := range users {
		data, _ := json.Marshal(user)
		if err := db.Put([]byte(user.ID), data); err != nil {
			if err == skv.ErrKeyExists {
				fmt.Printf("   User %s already exists, skipping\n", user.ID)
			} else {
				log.Fatal(err)
			}
		} else {
			fmt.Printf("   Inserted: %s (%s)\n", user.Name, user.Email)
		}
	}

	// Create index by email
	fmt.Println("\n2. Creating index by email...")
	err = db.CreateIndex("by_email", func(data []byte) []byte {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil
		}
		return []byte(user.Email)
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Index created: %d entries\n", db.IndexSize("by_email"))

	// Get user by email
	fmt.Println("\n3. Looking up user by email...")
	email := "charlie@example.com"
	data, err := db.GetByIndexString("by_email", email)
	if err != nil {
		log.Fatal(err)
	}

	var user User
	json.Unmarshal(data, &user)
	fmt.Printf("   Found: %s (ID: %s, Age: %d)\n", user.Name, user.ID, user.Age)

	// Check if email exists
	fmt.Println("\n4. Checking if emails exist...")
	emails := []string{"alice@example.com", "nonexistent@example.com"}
	for _, email := range emails {
		exists := db.HasIndexString("by_email", email)
		fmt.Printf("   %s: %v\n", email, exists)
	}

	// Update user email
	fmt.Println("\n5. Updating user email...")
	data, _ = db.Get([]byte("u001"))
	var alice User
	json.Unmarshal(data, &alice)
	fmt.Printf("   Original: %s\n", alice.Email)

	alice.Email = "alice.johnson@newdomain.com"
	updatedData, _ := json.Marshal(alice)
	db.Update([]byte("u001"), updatedData)
	fmt.Printf("   Updated: %s\n", alice.Email)

	// Verify old email not in index
	if !db.HasIndexString("by_email", "alice@example.com") {
		fmt.Println("   ✓ Old email removed from index")
	}

	// Verify new email in index
	if db.HasIndexString("by_email", "alice.johnson@newdomain.com") {
		fmt.Println("   ✓ New email added to index")
	}

	// Delete user
	fmt.Println("\n6. Deleting user...")
	db.Delete([]byte("u002"))
	fmt.Println("   Deleted: u002 (Bob Smith)")

	if !db.HasIndexString("by_email", "bob@example.com") {
		fmt.Println("   ✓ Email removed from index")
	}

	// Save index for later use
	fmt.Println("\n7. Saving index to file...")
	if err := db.SaveIndex("by_email", "email_index.json"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Index saved to email_index.json")

	// List all indexes
	fmt.Println("\n8. Listing all indexes:")
	indexes := db.ListIndexes()
	for _, name := range indexes {
		fmt.Printf("   - %s (%d entries)\n", name, db.IndexSize(name))
	}

	// Rebuild index (useful if index gets out of sync)
	fmt.Println("\n9. Rebuilding index...")
	if err := db.RebuildIndex("by_email"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Index rebuilt: %d entries\n", db.IndexSize("by_email"))

	fmt.Println("\n=== Example completed successfully ===")
}
