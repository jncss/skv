package skv

import (
	"os"
	"testing"
)

// Test Exists and Has functions
func TestExists(t *testing.T) {
	testFile := "test_exists.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Initially key should not exist
	if db.Exists([]byte("key1")) {
		t.Error("Key should not exist initially")
	}

	// Add key
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting key: %v", err)
	}

	// Now it should exist
	if !db.Exists([]byte("key1")) {
		t.Error("Key should exist after Put")
	}

	// Test Has (alias)
	if !db.Has([]byte("key1")) {
		t.Error("Has should return true for existing key")
	}

	// Delete key
	if err := db.Delete([]byte("key1")); err != nil {
		t.Fatalf("Error deleting key: %v", err)
	}

	// Should not exist after delete
	if db.Exists([]byte("key1")) {
		t.Error("Key should not exist after Delete")
	}
}

func TestExistsString(t *testing.T) {
	testFile := "test_exists_string.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.PutString("user", "alice"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	if !db.ExistsString("user") {
		t.Error("ExistsString should return true")
	}

	if !db.HasString("user") {
		t.Error("HasString should return true")
	}

	if db.ExistsString("nonexistent") {
		t.Error("ExistsString should return false for missing key")
	}
}

// Test Count function
func TestCount(t *testing.T) {
	testFile := "test_count.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Initially should be 0
	if count := db.Count(); count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Add keys
	for i := 0; i < 5; i++ {
		key := []byte{byte('a' + i)}
		if err := db.Put(key, []byte("value")); err != nil {
			t.Fatalf("Error putting key: %v", err)
		}
	}

	if count := db.Count(); count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	// Delete one
	if err := db.Delete([]byte("a")); err != nil {
		t.Fatalf("Error deleting key: %v", err)
	}

	if count := db.Count(); count != 4 {
		t.Errorf("Expected count 4 after delete, got %d", count)
	}
}

// Test Clear function
func TestClear(t *testing.T) {
	testFile := "test_clear.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add some keys
	for i := 0; i < 10; i++ {
		key := []byte{byte('a' + i)}
		if err := db.Put(key, []byte("value")); err != nil {
			t.Fatalf("Error putting key: %v", err)
		}
	}

	if count := db.Count(); count != 10 {
		t.Errorf("Expected 10 keys, got %d", count)
	}

	// Clear all
	if err := db.Clear(); err != nil {
		t.Fatalf("Error clearing database: %v", err)
	}

	// Should be empty
	if count := db.Count(); count != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", count)
	}

	// Should not be able to get any keys
	_, err = db.Get([]byte("a"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after clear, got %v", err)
	}

	// Should be able to add new keys after clear
	if err := db.Put([]byte("new"), []byte("value")); err != nil {
		t.Fatalf("Error putting key after clear: %v", err)
	}

	if count := db.Count(); count != 1 {
		t.Errorf("Expected 1 key after adding post-clear, got %d", count)
	}
}

// Test GetOrDefault function
func TestGetOrDefault(t *testing.T) {
	testFile := "test_getordefault.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add a key
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting key: %v", err)
	}

	// Get existing key - should return actual value
	value := db.GetOrDefault([]byte("key1"), []byte("default"))
	if string(value) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Get missing key - should return default
	value = db.GetOrDefault([]byte("missing"), []byte("default"))
	if string(value) != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
}

func TestGetOrDefaultString(t *testing.T) {
	testFile := "test_getordefault_string.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.PutString("user", "alice"); err != nil {
		t.Fatalf("Error putting string: %v", err)
	}

	// Existing key
	value := db.GetOrDefaultString("user", "guest")
	if value != "alice" {
		t.Errorf("Expected 'alice', got '%s'", value)
	}

	// Missing key
	value = db.GetOrDefaultString("missing", "guest")
	if value != "guest" {
		t.Errorf("Expected 'guest', got '%s'", value)
	}
}

// Test ForEach function
func TestForEach(t *testing.T) {
	testFile := "test_foreach.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		if err := db.PutString(key, value); err != nil {
			t.Fatalf("Error putting key: %v", err)
		}
	}

	// Collect all keys and values
	collected := make(map[string]string)
	err = db.ForEach(func(key []byte, value []byte) error {
		collected[string(key)] = string(value)
		return nil
	})

	if err != nil {
		t.Fatalf("Error in ForEach: %v", err)
	}

	// Verify all data was collected
	if len(collected) != len(testData) {
		t.Errorf("Expected %d items, got %d", len(testData), len(collected))
	}

	for key, expectedValue := range testData {
		if value, ok := collected[key]; !ok {
			t.Errorf("Key %q not found in iteration", key)
		} else if value != expectedValue {
			t.Errorf("For key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestForEachString(t *testing.T) {
	testFile := "test_foreach_string.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.PutString("a", "1"); err != nil {
		t.Fatalf("Error putting: %v", err)
	}
	if err := db.PutString("b", "2"); err != nil {
		t.Fatalf("Error putting: %v", err)
	}

	count := 0
	err = db.ForEachString(func(key string, value string) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Error in ForEachString: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected to iterate 2 times, got %d", count)
	}
}

// Test PutBatch function
func TestPutBatch(t *testing.T) {
	testFile := "test_putbatch.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Batch insert
	items := map[string][]byte{
		"user1": []byte("alice"),
		"user2": []byte("bob"),
		"user3": []byte("charlie"),
	}

	if err := db.PutBatch(items); err != nil {
		t.Fatalf("Error in PutBatch: %v", err)
	}

	// Verify all were inserted
	if count := db.Count(); count != 3 {
		t.Errorf("Expected 3 keys, got %d", count)
	}

	// Verify values
	for key, expectedValue := range items {
		value, err := db.Get([]byte(key))
		if err != nil {
			t.Errorf("Error getting key %q: %v", key, err)
			continue
		}
		if string(value) != string(expectedValue) {
			t.Errorf("For key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestPutBatchDuplicate(t *testing.T) {
	testFile := "test_putbatch_dup.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add one key first
	if err := db.PutString("user1", "alice"); err != nil {
		t.Fatalf("Error putting key: %v", err)
	}

	// Try to batch insert with a duplicate
	items := map[string][]byte{
		"user1": []byte("bob"), // Duplicate!
		"user2": []byte("charlie"),
	}

	err = db.PutBatch(items)
	if err == nil {
		t.Error("Expected error for duplicate key in batch")
	}

	// user1 should still have original value
	value, _ := db.GetString("user1")
	if value != "alice" {
		t.Errorf("Expected original value 'alice', got '%s'", value)
	}

	// user2 should not have been inserted (batch should be atomic)
	if db.Exists([]byte("user2")) {
		t.Error("user2 should not exist after failed batch")
	}
}

func TestPutBatchString(t *testing.T) {
	testFile := "test_putbatch_string.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	items := map[string]string{
		"name":  "Alice",
		"city":  "London",
		"email": "alice@example.com",
	}

	if err := db.PutBatchString(items); err != nil {
		t.Fatalf("Error in PutBatchString: %v", err)
	}

	if count := db.Count(); count != 3 {
		t.Errorf("Expected 3 keys, got %d", count)
	}
}

// Test GetBatch function
func TestGetBatch(t *testing.T) {
	testFile := "test_getbatch.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		if err := db.PutString(key, value); err != nil {
			t.Fatalf("Error putting key: %v", err)
		}
	}

	// Get batch
	keys := [][]byte{
		[]byte("key1"),
		[]byte("key2"),
		[]byte("missing"), // This one doesn't exist
	}

	result, err := db.GetBatch(keys)
	if err != nil {
		t.Fatalf("Error in GetBatch: %v", err)
	}

	// Should have 2 results (missing key excluded)
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	// Verify values
	if value, ok := result["key1"]; !ok {
		t.Error("key1 not in results")
	} else if string(value) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	if value, ok := result["key2"]; !ok {
		t.Error("key2 not in results")
	} else if string(value) != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}

	if _, ok := result["missing"]; ok {
		t.Error("missing key should not be in results")
	}
}

func TestGetBatchString(t *testing.T) {
	testFile := "test_getbatch_string.skv"
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.PutString("a", "1"); err != nil {
		t.Fatalf("Error putting: %v", err)
	}
	if err := db.PutString("b", "2"); err != nil {
		t.Fatalf("Error putting: %v", err)
	}

	keys := []string{"a", "b", "c"}
	result, err := db.GetBatchString(keys)
	if err != nil {
		t.Fatalf("Error in GetBatchString: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	if result["a"] != "1" {
		t.Errorf("Expected '1', got '%s'", result["a"])
	}

	if result["b"] != "2" {
		t.Errorf("Expected '2', got '%s'", result["b"])
	}
}
