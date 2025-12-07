package skv

import (
	"io"
	"os"
	"testing"
)

// TestCursorStringFunctions tests string convenience functions for cursors
func TestCursorStringFunctions(t *testing.T) {
	db, err := Open("test_cursor_string.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_cursor_string.skv")
	defer db.Close()

	// Add test data
	testData := map[string]string{
		"user:1":  "Alice",
		"user:2":  "Bob",
		"user:3":  "Charlie",
		"admin:1": "Dave",
		"admin:2": "Eve",
	}

	for k, v := range testData {
		if err := db.PutString(k, v); err != nil {
			t.Fatalf("Failed to put %s: %v", k, err)
		}
	}

	// Test NewCursorString
	t.Run("NewCursorString", func(t *testing.T) {
		cursor := db.NewCursorString("user:1", "user:3", false)
		defer cursor.Close()

		count := 0
		for {
			key, value, err := cursor.NextString()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextString failed: %v", err)
			}

			expected, exists := testData[key]
			if !exists {
				t.Errorf("Unexpected key: %s", key)
			}
			if value != expected {
				t.Errorf("Value mismatch for %s: got %s, want %s", key, value, expected)
			}
			count++
		}

		if count != 3 {
			t.Errorf("Expected 3 records, got %d", count)
		}
	})

	// Test PrefixCursorString
	t.Run("PrefixCursorString", func(t *testing.T) {
		cursor := db.PrefixCursorString("user:", false)
		defer cursor.Close()

		keys := cursor.KeysString()
		if len(keys) != 3 {
			t.Errorf("Expected 3 keys with user: prefix, got %d", len(keys))
		}

		for _, key := range keys {
			if len(key) < 5 || key[:5] != "user:" {
				t.Errorf("Key %s doesn't have user: prefix", key)
			}
		}
	})

	// Test KeyString and ValueString
	t.Run("KeyString_ValueString", func(t *testing.T) {
		cursor := db.AllCursorString(false)
		defer cursor.Close()

		cursor.NextString() // Move to first record

		key := cursor.KeyString()
		value, err := cursor.ValueString()
		if err != nil {
			t.Fatalf("ValueString failed: %v", err)
		}

		if key == "" || value == "" {
			t.Error("KeyString or ValueString returned empty")
		}

		expected, exists := testData[key]
		if !exists || value != expected {
			t.Errorf("Mismatch: %s = %s, want %s", key, value, expected)
		}
	})

	// Test SeekString
	t.Run("SeekString", func(t *testing.T) {
		cursor := db.AllCursorString(false)
		defer cursor.Close()

		if err := cursor.SeekString("user:2"); err != nil {
			t.Fatalf("SeekString failed: %v", err)
		}

		key, value, err := cursor.NextString()
		if err != nil {
			t.Fatalf("NextString after Seek failed: %v", err)
		}

		if key != "user:2" {
			t.Errorf("Seek didn't position correctly: got %s, want user:2", key)
		}
		if value != "Bob" {
			t.Errorf("Value mismatch: got %s, want Bob", value)
		}
	})

	// Test HasPrefixString
	t.Run("HasPrefixString", func(t *testing.T) {
		cursor := db.AllCursorString(false)
		defer cursor.Close()

		cursor.NextString() // Move to first key (admin:1 or user:1)

		// Check if it has either prefix
		hasAdmin := cursor.HasPrefixString("admin:")
		hasUser := cursor.HasPrefixString("user:")

		if !hasAdmin && !hasUser {
			t.Error("Current key should have either admin: or user: prefix")
		}
	})

	// Test CollectString
	t.Run("CollectString", func(t *testing.T) {
		cursor := db.PrefixCursorString("admin:", false)
		defer cursor.Close()

		keys, values, err := cursor.CollectString()
		if err != nil {
			t.Fatalf("CollectString failed: %v", err)
		}

		if len(keys) != 2 || len(values) != 2 {
			t.Errorf("Expected 2 admin entries, got %d keys, %d values", len(keys), len(values))
		}

		for i := range keys {
			if values[i] != testData[keys[i]] {
				t.Errorf("Mismatch at index %d: %s = %s, want %s", i, keys[i], values[i], testData[keys[i]])
			}
		}
	})

	// Test AllCursorString with reverse
	t.Run("AllCursorString_Reverse", func(t *testing.T) {
		cursor := db.AllCursorString(true)
		defer cursor.Close()

		keys := cursor.KeysString()
		if len(keys) != 5 {
			t.Errorf("Expected 5 total keys, got %d", len(keys))
		}

		// Keys should be in descending order when reverse=true
		// But KeysString() returns the internal keys slice which is always sorted ascending
		// The reverse flag affects iteration order (Next), not the Keys() result
		// So we need to iterate to verify reverse order
		cursor.Reset()
		var iteratedKeys []string
		for {
			key, _, err := cursor.NextString()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextString failed: %v", err)
			}
			iteratedKeys = append(iteratedKeys, key)
		}

		// Verify reverse order in iteration
		for i := 0; i < len(iteratedKeys)-1; i++ {
			if iteratedKeys[i] < iteratedKeys[i+1] {
				t.Errorf("Keys not in reverse order during iteration at index %d: %s should be > %s", i, iteratedKeys[i], iteratedKeys[i+1])
			}
		}
	})
}

// TestIndexCursorStringFunctions tests string convenience functions for index cursors
func TestIndexCursorStringFunctions(t *testing.T) {
	db, err := Open("test_index_cursor_string.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_index_cursor_string.skv")
	defer db.Close()

	// Add test data with categories
	testData := map[string]string{
		"prod1": `{"name":"Laptop","category":"electronics"}`,
		"prod2": `{"name":"Phone","category":"electronics"}`,
		"prod3": `{"name":"Book","category":"books"}`,
		"prod4": `{"name":"Pen","category":"stationery"}`,
	}

	for k, v := range testData {
		if err := db.PutString(k, v); err != nil {
			t.Fatalf("Failed to put %s: %v", k, err)
		}
	}

	// Create index
	err = db.CreateIndex("by_category", func(data []byte) []byte {
		// Extract category from JSON - look for "category":"value"
		s := string(data)
		catKey := `"category":"`
		idx := 0
		for i := 0; i < len(s)-len(catKey); i++ {
			if s[i:i+len(catKey)] == catKey {
				idx = i + len(catKey)
				break
			}
		}
		if idx > 0 {
			for i := idx; i < len(s); i++ {
				if s[i] == '"' {
					return []byte(s[idx:i])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Test NewIndexCursorString
	t.Run("NewIndexCursorString", func(t *testing.T) {
		cursor, err := db.NewIndexCursorString("by_category", "a", "f", false)
		if err != nil {
			t.Fatalf("NewIndexCursorString failed: %v", err)
		}
		defer cursor.Close()

		keys := cursor.KeysString()
		// "books" and "electronics" are in range a-f, "stationery" is not
		// books has 1 product, electronics has 2 products = 3 total
		if len(keys) != 3 {
			t.Errorf("Expected 3 products in range a-f (1 book + 2 electronics), got %d", len(keys))
		}
	})

	// Test PrefixIndexCursorString
	t.Run("PrefixIndexCursorString", func(t *testing.T) {
		cursor, err := db.PrefixIndexCursorString("by_category", "elect", false)
		if err != nil {
			t.Fatalf("PrefixIndexCursorString failed: %v", err)
		}
		defer cursor.Close()

		count := 0
		for {
			_, _, err := cursor.NextString()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextString failed: %v", err)
			}
			count++
		}

		if count != 2 { // 2 electronics products
			t.Errorf("Expected 2 electronics products, got %d", count)
		}
	})

	// Test AllIndexCursorString
	t.Run("AllIndexCursorString", func(t *testing.T) {
		cursor, err := db.AllIndexCursorString("by_category", false)
		if err != nil {
			t.Fatalf("AllIndexCursorString failed: %v", err)
		}
		defer cursor.Close()

		keys := cursor.KeysString()
		if len(keys) != 4 {
			t.Errorf("Expected 4 total products, got %d", len(keys))
		}
	})
}

// TestIndexStringFunctions tests string convenience functions for index operations
func TestIndexStringFunctions(t *testing.T) {
	db, err := Open("test_index_string.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_index_string.skv")
	defer db.Close()

	// Add test data
	db.PutString("user1", `{"email":"alice@example.com"}`)
	db.PutString("user2", `{"email":"bob@example.com"}`)
	db.PutString("user3", `{"email":"alice@example.com"}`) // Duplicate email

	// Create index
	err = db.CreateIndex("by_email", func(data []byte) []byte {
		s := string(data)
		start := len(`{"email":"`)
		if len(s) > start {
			for i := start; i < len(s); i++ {
				if s[i] == '"' {
					return []byte(s[start:i])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Test GetByIndexString
	t.Run("GetByIndexString", func(t *testing.T) {
		value, err := db.GetByIndexString("by_email", "bob@example.com")
		if err != nil {
			t.Fatalf("GetByIndexString failed: %v", err)
		}

		if string(value) != `{"email":"bob@example.com"}` {
			t.Errorf("Value mismatch: got %s", value)
		}
	})

	// Test GetAllByIndexString
	t.Run("GetAllByIndexString", func(t *testing.T) {
		values, err := db.GetAllByIndexString("by_email", "alice@example.com")
		if err != nil {
			t.Fatalf("GetAllByIndexString failed: %v", err)
		}

		if len(values) != 2 {
			t.Errorf("Expected 2 values for alice@example.com, got %d", len(values))
		}
	})

	// Test HasIndexString
	t.Run("HasIndexString", func(t *testing.T) {
		if !db.HasIndexString("by_email", "bob@example.com") {
			t.Error("HasIndexString should return true for bob@example.com")
		}

		if db.HasIndexString("by_email", "nonexistent@example.com") {
			t.Error("HasIndexString should return false for nonexistent email")
		}
	})
}
