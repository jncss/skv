package skv

import (
	"os"
	"testing"
)

func TestPutString(t *testing.T) {
	testFile := "test_putstring.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test basic put
	if err := db.PutString("name", "Alice"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	// Verify it was stored
	value, err := db.GetString("name")
	if err != nil {
		t.Fatalf("Error getting string: %v", err)
	}
	if value != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", value)
	}
}

func TestGetString(t *testing.T) {
	testFile := "test_getstring.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Store using byte version
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting: %v", err)
	}

	// Retrieve using string version
	value, err := db.GetString("key1")
	if err != nil {
		t.Fatalf("Error getting string: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test non-existent key
	_, err = db.GetString("nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestUpdateString(t *testing.T) {
	testFile := "test_updatestring.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create initial value
	if err := db.PutString("user", "john"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	// Update it
	if err := db.UpdateString("user", "john_doe"); err != nil {
		t.Fatalf("Error updating string: %v", err)
	}

	// Verify update
	value, err := db.GetString("user")
	if err != nil {
		t.Fatalf("Error getting string: %v", err)
	}
	if value != "john_doe" {
		t.Errorf("Expected 'john_doe', got '%s'", value)
	}

	// Test updating non-existent key
	err = db.UpdateString("nonexistent", "value")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestDeleteString(t *testing.T) {
	testFile := "test_deletestring.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create a key
	if err := db.PutString("temp", "data"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	// Delete it
	if err := db.DeleteString("temp"); err != nil {
		t.Fatalf("Error deleting string: %v", err)
	}

	// Verify it's deleted
	_, err = db.GetString("temp")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after delete, got %v", err)
	}

	// Test deleting non-existent key
	err = db.DeleteString("nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestKeysString(t *testing.T) {
	testFile := "test_keysstring.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add several keys
	keys := []string{"apple", "banana", "cherry"}
	for _, key := range keys {
		if err := db.PutString(key, "value_"+key); err != nil {
			t.Fatalf("Error putting string: %v", err)
		}
	}

	// Get all keys
	retrievedKeys, err := db.KeysString()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}

	// Verify count
	if len(retrievedKeys) != len(keys) {
		t.Errorf("Expected %d keys, got %d", len(keys), len(retrievedKeys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, key := range retrievedKeys {
		keyMap[key] = true
	}

	for _, expectedKey := range keys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key '%s' not found in results", expectedKey)
		}
	}
}

func TestStringMixedWithBytes(t *testing.T) {
	testFile := "test_mixed.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Store with string method
	if err := db.PutString("key1", "value1"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	// Store with byte method
	if err := db.Put([]byte("key2"), []byte("value2")); err != nil {
		t.Fatalf("Error putting bytes: %v", err)
	}

	// Retrieve with opposite methods
	value1, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error getting key1: %v", err)
	}
	if string(value1) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value1)
	}

	value2, err := db.GetString("key2")
	if err != nil {
		t.Fatalf("Error getting key2: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value2)
	}

	// Check all keys appear in both methods
	bytesKeys, _ := db.Keys()
	stringKeys, _ := db.KeysString()

	if len(bytesKeys) != 2 || len(stringKeys) != 2 {
		t.Errorf("Expected 2 keys in both methods, got %d bytes and %d strings",
			len(bytesKeys), len(stringKeys))
	}
}

func TestPutStringDuplicate(t *testing.T) {
	testFile := "test_putstring_dup.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// First put should succeed
	if err := db.PutString("duplicate", "first"); err != nil {
		t.Fatalf("Error on first put: %v", err)
	}

	// Second put should fail
	err = db.PutString("duplicate", "second")
	if err != ErrKeyExists {
		t.Errorf("Expected ErrKeyExists on duplicate put, got %v", err)
	}

	// Verify original value is unchanged
	value, _ := db.GetString("duplicate")
	if value != "first" {
		t.Errorf("Expected 'first', got '%s'", value)
	}
}
