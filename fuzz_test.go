package skv

import (
	"bytes"
	"path/filepath"
	"testing"
)

// FuzzPutGet fuzzes basic Put/Get operations with random keys and values
func FuzzPutGet(f *testing.F) {
	// Add seed corpus - various edge cases
	f.Add([]byte("key"), []byte("value"))
	f.Add([]byte(""), []byte("data"))
	f.Add([]byte("k"), []byte(""))
	f.Add([]byte("very-long-key-with-many-characters-to-test-limits"), []byte("data"))
	f.Add([]byte("key"), make([]byte, 1000))                  // Large value
	f.Add([]byte{0x00, 0xFF, 0x01}, []byte{0xFF, 0x00, 0xAA}) // Binary data

	f.Fuzz(func(t *testing.T, key []byte, data []byte) {
		// Skip empty keys (expected error)
		if len(key) == 0 {
			return
		}
		// Skip keys that are too long (expected error)
		if len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_putget.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Put
		err = db.Put(key, data)
		if err != nil {
			t.Fatalf("Put failed with valid input: %v", err)
		}

		// Get
		retrieved, err := db.Get(key)
		if err != nil {
			t.Fatalf("Get failed after successful Put: %v", err)
		}

		// Verify
		if !bytes.Equal(retrieved, data) {
			t.Errorf("Data mismatch: expected %d bytes, got %d bytes", len(data), len(retrieved))
		}
	})
}

// FuzzUpdate fuzzes Update operations
func FuzzUpdate(f *testing.F) {
	f.Add([]byte("key"), []byte("original"), []byte("updated"))
	f.Add([]byte("k"), []byte(""), []byte("new"))
	f.Add([]byte("test"), []byte("small"), make([]byte, 10000)) // Small to large

	f.Fuzz(func(t *testing.T, key []byte, original []byte, updated []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_update.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Put original
		if err := db.Put(key, original); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Update
		if err := db.Update(key, updated); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify
		retrieved, err := db.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !bytes.Equal(retrieved, updated) {
			t.Errorf("Update verification failed")
		}
	})
}

// FuzzDelete fuzzes Delete operations
func FuzzDelete(f *testing.F) {
	f.Add([]byte("key"), []byte("value"))
	f.Add([]byte("test"), make([]byte, 5000))

	f.Fuzz(func(t *testing.T, key []byte, data []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_delete.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Put
		if err := db.Put(key, data); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Delete
		if err := db.Delete(key); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify not found
		_, err = db.Get(key)
		if err != ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound after delete, got: %v", err)
		}

		// Verify Has returns false
		if db.Has(key) {
			t.Errorf("Has() returned true for deleted key")
		}
	})
}

// FuzzMultipleOperations fuzzes sequences of operations
func FuzzMultipleOperations(f *testing.F) {
	f.Add(byte(0), []byte("key1"), []byte("data1"))
	f.Add(byte(1), []byte("key2"), []byte("data2"))
	f.Add(byte(2), []byte("key3"), []byte("data3"))

	f.Fuzz(func(t *testing.T, opType byte, key []byte, data []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_multi.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Insert initial data
		initialKey := []byte("initial")
		if err := db.Put(initialKey, []byte("initial-data")); err != nil {
			t.Fatalf("Initial put failed: %v", err)
		}

		// Perform random operation
		switch opType % 4 {
		case 0: // Put
			err = db.Put(key, data)
			if err == nil {
				// Verify
				retrieved, err := db.Get(key)
				if err != nil {
					t.Fatalf("Get failed after Put: %v", err)
				}
				if !bytes.Equal(retrieved, data) {
					t.Errorf("Data mismatch after Put")
				}
			}
		case 1: // Update (may fail if key doesn't exist)
			db.Update(key, data) // Ignore error, key might not exist
		case 2: // Delete (may fail if key doesn't exist)
			db.Delete(key) // Ignore error, key might not exist
		case 3: // Get (may fail if key doesn't exist)
			db.Get(key) // Ignore error, key might not exist
		}

		// Verify initial key still exists
		retrieved, err := db.Get(initialKey)
		if err != nil {
			t.Fatalf("Initial key was corrupted: %v", err)
		}
		if !bytes.Equal(retrieved, []byte("initial-data")) {
			t.Errorf("Initial data was corrupted")
		}

		// Verify database count is consistent
		count := db.Count()
		if count < 0 {
			t.Errorf("Invalid count: %d", count)
		}
	})
}

// FuzzReopenPersistence fuzzes persistence across close/reopen
func FuzzReopenPersistence(f *testing.F) {
	f.Add([]byte("persistent-key"), []byte("persistent-data"))
	f.Add([]byte("test"), make([]byte, 1000))

	f.Fuzz(func(t *testing.T, key []byte, data []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_reopen.skv")

		// First session: Put data
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		if err := db.Put(key, data); err != nil {
			db.Close()
			t.Fatalf("Put failed: %v", err)
		}

		if err := db.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Second session: Verify data persisted
		db, err = Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}
		defer db.Close()

		retrieved, err := db.Get(key)
		if err != nil {
			t.Fatalf("Get failed after reopen: %v", err)
		}

		if !bytes.Equal(retrieved, data) {
			t.Errorf("Data not persisted correctly across reopen")
		}
	})
}

// FuzzCompaction fuzzes compaction with random data
func FuzzCompaction(f *testing.F) {
	f.Add(byte(5), []byte("key"), []byte("data"))

	f.Fuzz(func(t *testing.T, numOps byte, key []byte, data []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}
		// Limit operations to avoid timeouts
		if numOps > 50 {
			numOps = 50
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_compact.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Perform random operations
		for i := byte(0); i < numOps; i++ {
			testKey := append(key, i)
			if len(testKey) > 255 {
				continue
			}

			switch i % 3 {
			case 0: // Put
				db.Put(testKey, data)
			case 1: // Update
				db.Update(testKey, data)
			case 2: // Delete
				db.Delete(testKey)
			}
		}

		// Get state before compaction
		countBefore := db.Count()

		// Compact
		if err := db.Compact(); err != nil {
			t.Fatalf("Compact failed: %v", err)
		}

		// Verify state after compaction
		countAfter := db.Count()
		if countBefore != countAfter {
			t.Errorf("Count changed after compaction: before=%d, after=%d", countBefore, countAfter)
		}

		// Verify file is valid
		stats, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed after compaction: %v", err)
		}

		if stats.DeletedRecords != 0 {
			t.Errorf("Deleted records remain after compaction: %d", stats.DeletedRecords)
		}
	})
}

// FuzzBinaryKeys fuzzes with binary key data that might contain special characters
func FuzzBinaryKeys(f *testing.F) {
	f.Add([]byte{0x00}, []byte("value"))
	f.Add([]byte{0xFF}, []byte("value"))
	f.Add([]byte{0x00, 0xFF, 0x7F}, []byte("value"))
	f.Add([]byte{0x80}, []byte("value")) // Deleted flag bit

	f.Fuzz(func(t *testing.T, key []byte, data []byte) {
		if len(key) == 0 || len(key) > 255 {
			return
		}

		dbPath := filepath.Join(t.TempDir(), "fuzz_binary.skv")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Test with binary key
		if err := db.Put(key, data); err != nil {
			t.Fatalf("Put failed with binary key: %v", err)
		}

		retrieved, err := db.Get(key)
		if err != nil {
			t.Fatalf("Get failed with binary key: %v", err)
		}

		if !bytes.Equal(retrieved, data) {
			t.Errorf("Binary key data mismatch")
		}

		// Test persistence
		if err := db.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		db, err = Open(dbPath)
		if err != nil {
			t.Fatalf("Reopen failed: %v", err)
		}
		defer db.Close()

		retrieved, err = db.Get(key)
		if err != nil {
			t.Fatalf("Get failed after reopen with binary key: %v", err)
		}

		if !bytes.Equal(retrieved, data) {
			t.Errorf("Binary key data not persisted correctly")
		}
	})
}
