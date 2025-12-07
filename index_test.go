package skv

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}

func TestCreateIndex(t *testing.T) {
	dbFile := "test_create_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert some users
	users := []User{
		{Email: "alice@example.com", Name: "Alice", Age: 30},
		{Email: "bob@example.com", Name: "Bob", Age: 25},
		{Email: "charlie@example.com", Name: "Charlie", Age: 35},
	}

	for i, user := range users {
		data, _ := json.Marshal(user)
		key := []byte("user_" + string(rune('0'+i)))
		if err := db.Put(key, data); err != nil {
			t.Fatalf("Failed to put user: %v", err)
		}
	}

	// Create index by email
	err = db.CreateIndex("by_email", func(data []byte) []byte {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil
		}
		return []byte(user.Email)
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Verify index was created
	indexes := db.ListIndexes()
	if len(indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(indexes))
	}
	if indexes[0] != "by_email" {
		t.Errorf("Expected index name 'by_email', got '%s'", indexes[0])
	}

	// Verify index size
	size := db.IndexSize("by_email")
	if size != 3 {
		t.Errorf("Expected index size 3, got %d", size)
	}
}

func TestGetByIndex(t *testing.T) {
	dbFile := "test_get_by_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert users
	users := []User{
		{Email: "alice@example.com", Name: "Alice", Age: 30},
		{Email: "bob@example.com", Name: "Bob", Age: 25},
	}

	for i, user := range users {
		data, _ := json.Marshal(user)
		key := []byte("user_" + string(rune('0'+i)))
		if err := db.Put(key, data); err != nil {
			t.Fatalf("Failed to put user: %v", err)
		}
	}

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil
		}
		return []byte(user.Email)
	})

	// Get by email
	data, err := db.GetByIndex("by_email", []byte("alice@example.com"))
	if err != nil {
		t.Fatalf("Failed to get by index: %v", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if user.Name != "Alice" {
		t.Errorf("Expected Alice, got %s", user.Name)
	}
	if user.Age != 30 {
		t.Errorf("Expected age 30, got %d", user.Age)
	}
}

func TestGetByIndexString(t *testing.T) {
	dbFile := "test_get_by_index_string.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	user := User{Email: "test@example.com", Name: "Test", Age: 20}
	data, _ := json.Marshal(user)
	db.Put([]byte("user1"), data)

	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Use string convenience method
	retrievedData, err := db.GetByIndexString("by_email", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to get by index string: %v", err)
	}

	var retrieved User
	json.Unmarshal(retrievedData, &retrieved)

	if retrieved.Name != "Test" {
		t.Errorf("Expected Test, got %s", retrieved.Name)
	}
}

func TestIndexNotFound(t *testing.T) {
	dbFile := "test_index_not_found.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to get from non-existent index
	_, err = db.GetByIndex("nonexistent", []byte("key"))
	if err == nil {
		t.Error("Expected error for non-existent index")
	}
}

func TestKeyNotFoundInIndex(t *testing.T) {
	dbFile := "test_key_not_found_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	user := User{Email: "test@example.com", Name: "Test", Age: 20}
	data, _ := json.Marshal(user)
	db.Put([]byte("user1"), data)

	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Try to get non-existent email
	_, err = db.GetByIndex("by_email", []byte("nonexistent@example.com"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestIndexUpdateOnPut(t *testing.T) {
	dbFile := "test_index_update_put.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create index first
	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Insert user after index creation
	user := User{Email: "new@example.com", Name: "New User", Age: 40}
	data, _ := json.Marshal(user)
	db.Put([]byte("user_new"), data)

	// Verify it's in the index
	retrievedData, err := db.GetByIndex("by_email", []byte("new@example.com"))
	if err != nil {
		t.Fatalf("Failed to get newly inserted user from index: %v", err)
	}

	var retrieved User
	json.Unmarshal(retrievedData, &retrieved)

	if retrieved.Name != "New User" {
		t.Errorf("Expected New User, got %s", retrieved.Name)
	}
}

func TestIndexUpdateOnUpdate(t *testing.T) {
	dbFile := "test_index_update_update.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert user
	user := User{Email: "old@example.com", Name: "Old", Age: 30}
	data, _ := json.Marshal(user)
	db.Put([]byte("user1"), data)

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Update user email
	updatedUser := User{Email: "new@example.com", Name: "Old", Age: 30}
	updatedData, _ := json.Marshal(updatedUser)
	db.Update([]byte("user1"), updatedData)

	// Old email should not be found
	_, err = db.GetByIndex("by_email", []byte("old@example.com"))
	if err != ErrKeyNotFound {
		t.Error("Old email should not be found in index after update")
	}

	// New email should be found
	retrievedData, err := db.GetByIndex("by_email", []byte("new@example.com"))
	if err != nil {
		t.Fatalf("Failed to get updated user from index: %v", err)
	}

	var retrieved User
	json.Unmarshal(retrievedData, &retrieved)
	if retrieved.Email != "new@example.com" {
		t.Errorf("Expected new@example.com, got %s", retrieved.Email)
	}
}

func TestIndexRemoveOnDelete(t *testing.T) {
	dbFile := "test_index_remove_delete.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert user
	user := User{Email: "delete@example.com", Name: "ToDelete", Age: 25}
	data, _ := json.Marshal(user)
	db.Put([]byte("user1"), data)

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Verify it's in index
	if !db.HasIndex("by_email", []byte("delete@example.com")) {
		t.Error("User should be in index before delete")
	}

	// Delete user
	db.Delete([]byte("user1"))

	// Verify it's removed from index
	if db.HasIndex("by_email", []byte("delete@example.com")) {
		t.Error("User should be removed from index after delete")
	}

	_, err = db.GetByIndex("by_email", []byte("delete@example.com"))
	if err != ErrKeyNotFound {
		t.Error("Deleted user should not be found in index")
	}
}

func TestDropIndex(t *testing.T) {
	dbFile := "test_drop_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		return []byte("test")
	})

	// Verify exists
	if len(db.ListIndexes()) != 1 {
		t.Error("Index should exist")
	}

	// Drop index
	err = db.DropIndex("by_email")
	if err != nil {
		t.Fatalf("Failed to drop index: %v", err)
	}

	// Verify removed
	if len(db.ListIndexes()) != 0 {
		t.Error("Index should be dropped")
	}

	// Try to drop again
	err = db.DropIndex("by_email")
	if err == nil {
		t.Error("Expected error when dropping non-existent index")
	}
}

func TestSaveAndLoadIndex(t *testing.T) {
	dbFile := "test_save_load_index.skv"
	indexFile := "test_index.json"
	defer os.Remove(dbFile)
	defer os.Remove(indexFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Insert users
	users := []User{
		{Email: "alice@example.com", Name: "Alice", Age: 30},
		{Email: "bob@example.com", Name: "Bob", Age: 25},
	}

	for i, user := range users {
		data, _ := json.Marshal(user)
		key := []byte("user_" + string(rune('0'+i)))
		db.Put(key, data)
	}

	// Create and save index
	keyFunc := func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	}

	db.CreateIndex("by_email", keyFunc)

	if err := db.SaveIndex("by_email", indexFile); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	db.Close()

	// Reopen database
	db, err = Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	// Load index
	if err := db.LoadIndex("by_email", indexFile, keyFunc); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// Verify index works
	data, err := db.GetByIndex("by_email", []byte("alice@example.com"))
	if err != nil {
		t.Fatalf("Failed to get by loaded index: %v", err)
	}

	var user User
	json.Unmarshal(data, &user)
	if user.Name != "Alice" {
		t.Errorf("Expected Alice, got %s", user.Name)
	}
}

func TestRebuildIndex(t *testing.T) {
	dbFile := "test_rebuild_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert users
	user1 := User{Email: "alice@example.com", Name: "Alice", Age: 30}
	data1, _ := json.Marshal(user1)
	db.Put([]byte("user1"), data1)

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Add more users after index creation
	user2 := User{Email: "bob@example.com", Name: "Bob", Age: 25}
	data2, _ := json.Marshal(user2)
	db.Put([]byte("user2"), data2)

	// Manually corrupt index (simulate out-of-sync)
	db.indexes["by_email"].index = make(map[string][]string)

	// Rebuild index
	if err := db.RebuildIndex("by_email"); err != nil {
		t.Fatalf("Failed to rebuild index: %v", err)
	}

	// Verify both users are in index
	if _, err := db.GetByIndex("by_email", []byte("alice@example.com")); err != nil {
		t.Error("Alice should be in rebuilt index")
	}

	if _, err := db.GetByIndex("by_email", []byte("bob@example.com")); err != nil {
		t.Error("Bob should be in rebuilt index")
	}
}

func TestHasIndex(t *testing.T) {
	dbFile := "test_has_index.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	user := User{Email: "test@example.com", Name: "Test", Age: 20}
	data, _ := json.Marshal(user)
	db.Put([]byte("user1"), data)

	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Test HasIndex
	if !db.HasIndex("by_email", []byte("test@example.com")) {
		t.Error("Should find email in index")
	}

	if db.HasIndex("by_email", []byte("nonexistent@example.com")) {
		t.Error("Should not find non-existent email in index")
	}

	// Test HasIndexString
	if !db.HasIndexString("by_email", "test@example.com") {
		t.Error("Should find email in index using string method")
	}
}

func TestIndexWithBinaryData(t *testing.T) {
	dbFile := "test_index_binary.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert binary data
	data1 := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	data2 := []byte{0x0A, 0x0B, 0x0C, 0x0D, 0x0E}

	db.Put([]byte("bin1"), data1)
	db.Put([]byte("bin2"), data2)

	// Create index using first byte as secondary key
	db.CreateIndex("by_first_byte", func(data []byte) []byte {
		if len(data) > 0 {
			return []byte{data[0]}
		}
		return nil
	})

	// Get by first byte
	retrieved, err := db.GetByIndex("by_first_byte", []byte{0x01})
	if err != nil {
		t.Fatalf("Failed to get by index: %v", err)
	}

	if !bytes.Equal(retrieved, data1) {
		t.Errorf("Expected %v, got %v", data1, retrieved)
	}
}
