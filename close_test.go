package skv

import (
	"bytes"
	"os"
	"testing"
)

func TestCloseWithCompact(t *testing.T) {
	testFile := "test_close_compact.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	// Create database and add some data
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	// Add some keys
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting key1: %v", err)
	}
	if err := db.Put([]byte("key2"), []byte("value2")); err != nil {
		t.Fatalf("Error putting key2: %v", err)
	}
	if err := db.Put([]byte("key3"), []byte("value3")); err != nil {
		t.Fatalf("Error putting key3: %v", err)
	}

	// Update and delete to create some deleted records
	if err := db.Update([]byte("key1"), []byte("value1_updated")); err != nil {
		t.Fatalf("Error updating key1: %v", err)
	}
	if err := db.Delete([]byte("key3")); err != nil {
		t.Fatalf("Error deleting key3: %v", err)
	}

	// Verify stats before compact
	statsBefore, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying before compact: %v", err)
	}
	// 3 puts + 1 update (new) + 1 delete = 5, but delete marks the record, doesn't add one
	// So: 3 puts + 1 update = 4 total
	// Active: key1 (updated), key2 = 2
	// Deleted: old key1, key3 = 2
	if statsBefore.TotalRecords != 4 {
		t.Errorf("Expected 4 total records before compact, got: %d", statsBefore.TotalRecords)
	}
	if statsBefore.ActiveRecords != 2 { // key1 (updated), key2
		t.Errorf("Expected 2 active records before compact, got: %d", statsBefore.ActiveRecords)
	}
	if statsBefore.DeletedRecords != 2 { // old key1, key3
		t.Errorf("Expected 2 deleted records before compact, got: %d", statsBefore.DeletedRecords)
	}

	// Close with compact
	if err := db.CloseWithCompact(); err != nil {
		t.Fatalf("Error closing with compact: %v", err)
	}

	// Reopen and verify stats after compact
	db2, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error reopening database: %v", err)
	}
	defer db2.Close()

	statsAfter, err := db2.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}

	if statsAfter.TotalRecords != 2 { // Only key1 and key2 remain
		t.Errorf("Expected 2 total records after compact, got: %d", statsAfter.TotalRecords)
	}
	if statsAfter.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records after compact, got: %d", statsAfter.ActiveRecords)
	}
	if statsAfter.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got: %d", statsAfter.DeletedRecords)
	}

	// Verify data integrity
	value1, err := db2.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error getting key1: %v", err)
	}
	if !bytes.Equal(value1, []byte("value1_updated")) {
		t.Errorf("Expected value1_updated, got: %s", value1)
	}

	value2, err := db2.Get([]byte("key2"))
	if err != nil {
		t.Fatalf("Error getting key2: %v", err)
	}
	if !bytes.Equal(value2, []byte("value2")) {
		t.Errorf("Expected value2, got: %s", value2)
	}

	_, err = db2.Get([]byte("key3"))
	if err != ErrKeyNotFound {
		t.Errorf("key3 should not exist after compact")
	}
}

func TestCloseWithCompactEmptyDatabase(t *testing.T) {
	testFile := "test_close_compact_empty.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	// Create empty database
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	// Close with compact on empty database
	if err := db.CloseWithCompact(); err != nil {
		t.Fatalf("Error closing empty database with compact: %v", err)
	}

	// Reopen and verify it's still empty
	db2, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error reopening database: %v", err)
	}
	defer db2.Close()

	keys, err := db2.Keys()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty database, got: %d", len(keys))
	}
}

func TestCloseNormal(t *testing.T) {
	testFile := "test_close_normal.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	// Create database and add data
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error putting key1: %v", err)
	}
	if err := db.Delete([]byte("key1")); err != nil {
		t.Fatalf("Error deleting key1: %v", err)
	}

	// Normal close (without compact)
	if err := db.Close(); err != nil {
		t.Fatalf("Error closing database: %v", err)
	}

	// Reopen and verify deleted record still exists
	db2, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error reopening database: %v", err)
	}
	defer db2.Close()

	stats, err := db2.Verify()
	if err != nil {
		t.Fatalf("Error verifying: %v", err)
	}

	// Normal close should keep deleted records
	// 1 put + delete marks it as deleted = 1 total record (marked as deleted)
	if stats.TotalRecords != 1 {
		t.Errorf("Expected 1 total record (not compacted), got: %d", stats.TotalRecords)
	}
	if stats.DeletedRecords != 1 {
		t.Errorf("Expected 1 deleted record (not compacted), got: %d", stats.DeletedRecords)
	}
}
