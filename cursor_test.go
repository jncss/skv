package skv

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestNewCursor(t *testing.T) {
	dbFile := "test_cursor.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert test data
	for i := 0; i < 10; i++ {
		key := []byte{byte('a' + i)}
		data := []byte{byte(i)}
		db.Put(key, data)
	}

	cursor := db.NewCursor(nil)
	if cursor == nil {
		t.Fatal("Expected cursor, got nil")
	}

	if cursor.Count() != 10 {
		t.Errorf("Expected 10 keys, got %d", cursor.Count())
	}
}

func TestCursorNext(t *testing.T) {
	dbFile := "test_cursor_next.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert in non-alphabetical order
	keys := []string{"c", "a", "e", "b", "d"}
	for _, k := range keys {
		db.Put([]byte(k), []byte("value_"+k))
	}

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	// Should iterate in sorted order
	expected := []string{"a", "b", "c", "d", "e"}
	for i, exp := range expected {
		key, value, err := cursor.Next()
		if err != nil {
			t.Fatalf("Unexpected error at position %d: %v", i, err)
		}

		if string(key) != exp {
			t.Errorf("Position %d: expected key %s, got %s", i, exp, string(key))
		}

		expectedValue := "value_" + exp
		if string(value) != expectedValue {
			t.Errorf("Position %d: expected value %s, got %s", i, expectedValue, string(value))
		}
	}

	// Next call should return EOF
	_, _, err := cursor.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestCursorReverse(t *testing.T) {
	dbFile := "test_cursor_reverse.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	keys := []string{"a", "b", "c", "d", "e"}
	for _, k := range keys {
		db.Put([]byte(k), []byte("value_"+k))
	}

	cursor := db.NewCursor(&CursorOptions{Reverse: true})
	defer cursor.Close()

	// Should iterate in reverse order
	expected := []string{"e", "d", "c", "b", "a"}
	for i, exp := range expected {
		key, _, err := cursor.Next()
		if err != nil {
			t.Fatalf("Unexpected error at position %d: %v", i, err)
		}

		if string(key) != exp {
			t.Errorf("Position %d: expected %s, got %s", i, exp, string(key))
		}
	}
}

func TestCursorRange(t *testing.T) {
	dbFile := "test_cursor_range.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert a-j
	for i := 0; i < 10; i++ {
		key := []byte{byte('a' + i)}
		db.Put(key, []byte{byte(i)})
	}

	// Range from 'c' to 'g'
	cursor := db.NewCursor(&CursorOptions{
		From: []byte("c"),
		To:   []byte("g"),
	})
	defer cursor.Close()

	expected := []string{"c", "d", "e", "f", "g"}
	count := 0
	for {
		key, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if string(key) != expected[count] {
			t.Errorf("Position %d: expected %s, got %s", count, expected[count], string(key))
		}
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 keys, got %d", count)
	}
}

func TestCursorRangeReverse(t *testing.T) {
	dbFile := "test_cursor_range_reverse.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	for i := 0; i < 10; i++ {
		key := []byte{byte('a' + i)}
		db.Put(key, []byte{byte(i)})
	}

	cursor := db.NewCursor(&CursorOptions{
		From:    []byte("c"),
		To:      []byte("g"),
		Reverse: true,
	})
	defer cursor.Close()

	expected := []string{"g", "f", "e", "d", "c"}
	count := 0
	for {
		key, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if string(key) != expected[count] {
			t.Errorf("Position %d: expected %s, got %s", count, expected[count], string(key))
		}
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 keys, got %d", count)
	}
}

func TestCursorSeek(t *testing.T) {
	dbFile := "test_cursor_seek.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	keys := []string{"a", "c", "e", "g", "i"}
	for _, k := range keys {
		db.Put([]byte(k), []byte("value_"+k))
	}

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	// Seek to 'd' (should find 'e')
	cursor.Seek([]byte("d"))
	key, _, err := cursor.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "e" {
		t.Errorf("Expected 'e', got %s", string(key))
	}

	// Next should be 'g'
	key, _, err = cursor.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "g" {
		t.Errorf("Expected 'g', got %s", string(key))
	}
}

func TestCursorReset(t *testing.T) {
	dbFile := "test_cursor_reset.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	// Read first item
	key, _, _ := cursor.Next()
	if string(key) != "a" {
		t.Errorf("Expected 'a', got %s", string(key))
	}

	// Reset and read again
	cursor.Reset()
	key, _, _ = cursor.Next()
	if string(key) != "a" {
		t.Errorf("After reset, expected 'a', got %s", string(key))
	}
}

func TestCursorForEach(t *testing.T) {
	dbFile := "test_cursor_foreach.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	for i := 0; i < 5; i++ {
		db.Put([]byte{byte('a' + i)}, []byte{byte(i)})
	}

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	count := 0
	err := cursor.ForEach(func(key, value []byte) bool {
		count++
		return true
	})

	if err != nil {
		t.Fatal(err)
	}

	if count != 5 {
		t.Errorf("Expected 5 iterations, got %d", count)
	}

	// Test early termination
	count = 0
	cursor.ForEach(func(key, value []byte) bool {
		count++
		return count < 3 // Stop after 3
	})

	if count != 3 {
		t.Errorf("Expected 3 iterations, got %d", count)
	}
}

func TestIndexCursor(t *testing.T) {
	dbFile := "test_index_cursor.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert users
	users := []User{
		{Email: "alice@example.com", Name: "Alice", Age: 30},
		{Email: "bob@example.com", Name: "Bob", Age: 25},
		{Email: "charlie@example.com", Name: "Charlie", Age: 35},
	}

	for i, user := range users {
		data, _ := json.Marshal(user)
		db.Put([]byte{byte('0' + i)}, data)
	}

	// Create index
	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Create cursor on index
	cursor, err := db.NewIndexCursor("by_email", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cursor.Close()

	// Should iterate in email order (sorted by secondary key, then primary key)
	expectedPrimaryKeys := []string{"0", "1", "2"}
	expectedEmails := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}
	count := 0
	for {
		key, value, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		// Key should be the primary key
		if string(key) != expectedPrimaryKeys[count] {
			t.Errorf("Position %d: expected primary key %s, got %s", count, expectedPrimaryKeys[count], string(key))
		}

		var user User
		json.Unmarshal(value, &user)
		if user.Email != expectedEmails[count] {
			t.Errorf("Position %d: user email mismatch, expected %s got %s", count, expectedEmails[count], user.Email)
		}

		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 users, got %d", count)
	}
}

func TestIndexCursorRange(t *testing.T) {
	dbFile := "test_index_cursor_range.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert users with emails a-e
	emails := []string{"alice@", "bob@", "charlie@", "diana@", "eve@"}
	for i, email := range emails {
		user := User{Email: email + "example.com", Name: string(rune('A' + i)), Age: 20 + i}
		data, _ := json.Marshal(user)
		db.Put([]byte{byte('0' + i)}, data)
	}

	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Range from bob@ to diana@ (inclusive)
	cursor, _ := db.NewIndexCursor("by_email", &CursorOptions{
		From: []byte("bob@example.com"),
		To:   []byte("diana@example.com"),
	})
	defer cursor.Close()

	count := 0
	for {
		_, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}

	if count != 3 { // bob, charlie, diana
		t.Errorf("Expected 3 results, got %d", count)
	}
}

func TestPrefixCursor(t *testing.T) {
	dbFile := "test_prefix_cursor.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert keys with different prefixes
	keys := []string{"user:1", "user:2", "user:3", "post:1", "post:2", "comment:1"}
	for _, k := range keys {
		db.Put([]byte(k), []byte("data_"+k))
	}

	// Get only user: keys
	cursor := db.PrefixCursor([]byte("user:"), false)
	defer cursor.Close()

	count := 0
	for {
		key, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.HasPrefix(key, []byte("user:")) {
			t.Errorf("Key %s doesn't have prefix 'user:'", string(key))
		}
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 user keys, got %d", count)
	}
}

func TestPrefixIndexCursor(t *testing.T) {
	dbFile := "test_prefix_index_cursor.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	// Insert users with different email domains
	users := []User{
		{Email: "alice@example.com", Name: "Alice"},
		{Email: "bob@example.com", Name: "Bob"},
		{Email: "charlie@test.com", Name: "Charlie"},
		{Email: "diana@example.com", Name: "Diana"},
	}

	for i, user := range users {
		data, _ := json.Marshal(user)
		db.Put([]byte{byte('0' + i)}, data)
	}

	db.CreateIndex("by_email", func(data []byte) []byte {
		var u User
		json.Unmarshal(data, &u)
		return []byte(u.Email)
	})

	// Get only users with prefix (alphabetically before 'c')
	cursor, _ := db.PrefixIndexCursor("by_email", []byte("a"), false)
	defer cursor.Close()

	count := 0
	for {
		_, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}

	// Should find alice@ and bob@ (not charlie@ or diana@)
	if count != 1 { // Only alice@
		t.Errorf("Expected 1 result with prefix 'a', got %d", count)
	}
}

func TestCursorCollect(t *testing.T) {
	dbFile := "test_cursor_collect.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	for i := 0; i < 5; i++ {
		db.Put([]byte{byte('a' + i)}, []byte{byte(i)})
	}

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	keys, values, err := cursor.Collect()
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 5 {
		t.Errorf("Expected 5 keys, got %d", len(keys))
	}

	if len(values) != 5 {
		t.Errorf("Expected 5 values, got %d", len(values))
	}

	for i := 0; i < 5; i++ {
		expectedKey := string(byte('a' + i))
		if string(keys[i]) != expectedKey {
			t.Errorf("Position %d: expected key %s, got %s", i, expectedKey, string(keys[i]))
		}
	}
}

func TestCursorKeyValue(t *testing.T) {
	dbFile := "test_cursor_key_value.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	db.Put([]byte("test"), []byte("value"))

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	// Before Next, Key/Value should error
	_, err := cursor.Key()
	if err == nil {
		t.Error("Expected error for Key() before Next()")
	}

	// After Next, should work
	cursor.Next()

	key, err := cursor.Key()
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "test" {
		t.Errorf("Expected 'test', got %s", string(key))
	}

	value, err := cursor.Value()
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != "value" {
		t.Errorf("Expected 'value', got %s", string(value))
	}
}

func TestCursorIsFirstIsLast(t *testing.T) {
	dbFile := "test_cursor_is_first_last.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))
	db.Put([]byte("c"), []byte("3"))

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	cursor.Next() // Move to 'a'
	if !cursor.IsFirst() {
		t.Error("Should be at first position")
	}
	if cursor.IsLast() {
		t.Error("Should not be at last position")
	}

	cursor.Next() // Move to 'b'
	if cursor.IsFirst() {
		t.Error("Should not be at first position")
	}
	if cursor.IsLast() {
		t.Error("Should not be at last position")
	}

	cursor.Next() // Move to 'c'
	if cursor.IsFirst() {
		t.Error("Should not be at first position")
	}
	if !cursor.IsLast() {
		t.Error("Should be at last position")
	}
}

func TestCursorClosed(t *testing.T) {
	dbFile := "test_cursor_closed.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	db.Put([]byte("test"), []byte("value"))

	cursor := db.NewCursor(nil)
	cursor.Close()

	// All operations should fail after close
	_, _, err := cursor.Next()
	if err == nil {
		t.Error("Expected error for Next() on closed cursor")
	}

	_, err = cursor.Key()
	if err == nil {
		t.Error("Expected error for Key() on closed cursor")
	}

	err = cursor.Seek([]byte("test"))
	if err == nil {
		t.Error("Expected error for Seek() on closed cursor")
	}
}

func TestAllCursor(t *testing.T) {
	dbFile := "test_all_cursor.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	for i := 0; i < 5; i++ {
		db.Put([]byte{byte('a' + i)}, []byte{byte(i)})
	}

	// Forward
	cursor := db.AllCursor(false)
	defer cursor.Close()

	count := 0
	for {
		_, _, err := cursor.Next()
		if err == io.EOF {
			break
		}
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 items, got %d", count)
	}

	// Reverse
	cursor2 := db.AllCursor(true)
	defer cursor2.Close()

	firstKey, _, _ := cursor2.Next()
	if string(firstKey) != "e" {
		t.Errorf("Expected 'e' as first in reverse, got %s", string(firstKey))
	}
}

func TestCursorWithEmptyDatabase(t *testing.T) {
	dbFile := "test_cursor_empty.skv"
	defer os.Remove(dbFile)

	db, _ := Open(dbFile)
	defer db.Close()

	cursor := db.NewCursor(nil)
	defer cursor.Close()

	if cursor.Count() != 0 {
		t.Errorf("Expected 0 keys in empty database, got %d", cursor.Count())
	}

	_, _, err := cursor.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF for empty cursor, got %v", err)
	}
}
