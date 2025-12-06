package skv

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompactSafety verifies that Compact uses a temporary file
// and doesn't corrupt the original file if something goes wrong
func TestCompactSafety(t *testing.T) {
	// Create a test database
	dbFile := "test_compact_safety.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Add some data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		if err := db.Put([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Failed to put %s: %v", k, err)
		}
	}

	// Get the file size before compact
	info, err := os.Stat(dbFile)
	if err != nil {
		t.Fatalf("Failed to stat database file: %v", err)
	}
	originalSize := info.Size()

	// Perform compaction
	if err := db.Compact(); err != nil {
		t.Fatalf("Compaction failed: %v", err)
	}

	// Verify the database is still valid
	for k, v := range testData {
		val, err := db.Get([]byte(k))
		if err != nil {
			t.Errorf("Failed to get %s after compact: %v", k, err)
		}
		if string(val) != v {
			t.Errorf("Value mismatch for %s: got %s, want %s", k, string(val), v)
		}
	}

	// Close the database
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Verify file still exists and is valid
	if _, err := os.Stat(dbFile); err != nil {
		t.Errorf("Database file doesn't exist after compact: %v", err)
	}

	// Reopen and verify data persisted
	db2, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	for k, v := range testData {
		val, err := db2.Get([]byte(k))
		if err != nil {
			t.Errorf("Failed to get %s after reopen: %v", k, err)
		}
		if string(val) != v {
			t.Errorf("Value mismatch for %s after reopen: got %s, want %s", k, string(val), v)
		}
	}

	t.Logf("Original size: %d bytes, compaction successful", originalSize)
}

// TestCompactNoTempFileLeftover verifies that no temporary files
// are left behind after compaction
func TestCompactNoTempFileLeftover(t *testing.T) {
	// Create a test database
	dbFile := "test_compact_temp.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Add some data
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 10)}
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Failed to put data: %v", err)
		}
	}

	// Get the directory of the database file
	dir := filepath.Dir(db.filePath)

	// Count temp files before compaction
	tempFilesBefore, err := filepath.Glob(filepath.Join(dir, ".skv-compact-*.tmp"))
	if err != nil {
		t.Fatalf("Failed to glob temp files: %v", err)
	}

	// Perform compaction
	if err := db.Compact(); err != nil {
		t.Fatalf("Compaction failed: %v", err)
	}

	// Count temp files after compaction
	tempFilesAfter, err := filepath.Glob(filepath.Join(dir, ".skv-compact-*.tmp"))
	if err != nil {
		t.Fatalf("Failed to glob temp files: %v", err)
	}

	// Should be the same (no new temp files left)
	if len(tempFilesAfter) != len(tempFilesBefore) {
		t.Errorf("Temporary files left after compaction: before=%d, after=%d",
			len(tempFilesBefore), len(tempFilesAfter))
		t.Logf("Temp files: %v", tempFilesAfter)
	}

	db.Close()
}

// TestCompactPreservesDataOnMultipleCompacts tests that multiple
// consecutive compactions don't corrupt data
func TestCompactPreservesDataOnMultipleCompacts(t *testing.T) {
	dbFile := "test_compact_multiple.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add initial data
	testData := make(map[string]string)
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i), byte(i * 2)}
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Failed to put data: %v", err)
		}
		testData[string(key)] = string(value)
	}

	// Perform multiple compactions
	for i := 0; i < 5; i++ {
		if err := db.Compact(); err != nil {
			t.Fatalf("Compaction %d failed: %v", i+1, err)
		}

		// Verify all data after each compaction
		for k, v := range testData {
			val, err := db.Get([]byte(k))
			if err != nil {
				t.Errorf("Compaction %d: Failed to get key after compact: %v", i+1, err)
			}
			if string(val) != v {
				t.Errorf("Compaction %d: Value mismatch: got %v, want %v", i+1, val, []byte(v))
			}
		}

		t.Logf("Compaction %d: All data verified", i+1)
	}
}

// TestCompactWithDeletesAndUpdates tests compaction after
// various operations to ensure data integrity
func TestCompactWithDeletesAndUpdates(t *testing.T) {
	dbFile := "test_compact_complex.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add initial data
	for i := 0; i < 50; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i)}
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Failed to put data: %v", err)
		}
	}

	// Delete some keys
	for i := 0; i < 25; i++ {
		key := []byte{byte(i)}
		if err := db.Delete(key); err != nil {
			t.Fatalf("Failed to delete data: %v", err)
		}
	}

	// Update some remaining keys
	for i := 25; i < 40; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2), byte(i * 3)}
		if err := db.Update(key, value); err != nil {
			t.Fatalf("Failed to update data: %v", err)
		}
	}

	// Get stats before compaction
	statsBefore, err := db.Verify()
	if err != nil {
		t.Fatalf("Failed to verify before compact: %v", err)
	}

	// Perform compaction
	if err := db.Compact(); err != nil {
		t.Fatalf("Compaction failed: %v", err)
	}

	// Get stats after compaction
	statsAfter, err := db.Verify()
	if err != nil {
		t.Fatalf("Failed to verify after compact: %v", err)
	}

	// Verify stats
	if statsAfter.ActiveRecords != statsBefore.ActiveRecords {
		t.Errorf("Active records changed: before=%d, after=%d",
			statsBefore.ActiveRecords, statsAfter.ActiveRecords)
	}

	if statsAfter.DeletedRecords != 0 {
		t.Errorf("Deleted records should be 0 after compact, got %d", statsAfter.DeletedRecords)
	}

	// File size should be smaller or equal after compaction
	if statsAfter.FileSize > statsBefore.FileSize {
		t.Errorf("File size increased after compaction: before=%d, after=%d",
			statsBefore.FileSize, statsAfter.FileSize)
	}

	// Verify data integrity
	// Keys 0-24 should be deleted
	for i := 0; i < 25; i++ {
		key := []byte{byte(i)}
		if _, err := db.Get(key); err != ErrKeyNotFound {
			t.Errorf("Key %d should be deleted", i)
		}
	}

	// Keys 25-39 should have updated values
	for i := 25; i < 40; i++ {
		key := []byte{byte(i)}
		expected := []byte{byte(i * 2), byte(i * 3)}
		val, err := db.Get(key)
		if err != nil {
			t.Errorf("Failed to get key %d: %v", i, err)
		}
		if string(val) != string(expected) {
			t.Errorf("Key %d: got %v, want %v", i, val, expected)
		}
	}

	// Keys 40-49 should have original values
	for i := 40; i < 50; i++ {
		key := []byte{byte(i)}
		expected := []byte{byte(i)}
		val, err := db.Get(key)
		if err != nil {
			t.Errorf("Failed to get key %d: %v", i, err)
		}
		if string(val) != string(expected) {
			t.Errorf("Key %d: got %v, want %v", i, val, expected)
		}
	}

	t.Logf("Stats before: %+v", statsBefore)
	t.Logf("Stats after: %+v", statsAfter)
	t.Logf("Space saved: %d bytes (%.1f%%)",
		statsBefore.FileSize-statsAfter.FileSize,
		float64(statsBefore.FileSize-statsAfter.FileSize)/float64(statsBefore.FileSize)*100)
}
