package skv

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// TestMultiProcessCompact simulates the scenario where one process compacts
// while another process is reading, to verify that:
// 1. In-place compact doesn't invalidate file descriptors
// 2. Change detection mechanism (lastSize) works correctly
// 3. Cache is automatically rebuilt when file changes
func TestMultiProcessCompact(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test_multiprocess_*.skv")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Process 1: Initial writes
	db1, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database (db1): %v", err)
	}

	// Write some initial data
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 10)}
		if err := db1.Put(key, value); err != nil {
			t.Fatalf("Error putting key %d: %v", i, err)
		}
	}

	// Add some deletions to create fragmentation
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		if err := db1.Delete(key); err != nil {
			t.Fatalf("Error deleting key %d: %v", i, err)
		}
	}

	// Verify stats before compact
	stats, err := db1.Verify()
	if err != nil {
		t.Fatalf("Error verifying: %v", err)
	}
	t.Logf("Before compact - Total: %d, Active: %d, Deleted: %d", stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)

	// Process 2: Open the same file (simulating another process)
	db2, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database (db2): %v", err)
	}
	defer db2.Close()

	// Process 2 should see the same data
	for i := 5; i < 10; i++ {
		key := []byte{byte(i)}
		value, err := db2.Get(key)
		if err != nil {
			t.Fatalf("Error getting key %d from db2: %v", i, err)
		}
		expected := []byte{byte(i * 10)}
		if !bytes.Equal(value, expected) {
			t.Errorf("Key %d: expected %v, got %v", i, expected, value)
		}
	}

	t.Log("Process 2 can read data before compact")

	// Process 1: Perform compact
	if err := db1.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Verify stats after compact
	stats, err = db1.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}
	t.Logf("After compact - Total: %d, Active: %d, Deleted: %d", stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)

	if stats.TotalRecords != 5 {
		t.Errorf("Expected 5 total records after compact, got: %d", stats.TotalRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got: %d", stats.DeletedRecords)
	}

	// Give a small delay to ensure file changes are propagated
	time.Sleep(10 * time.Millisecond)

	// Process 2: Try to read data - should automatically detect file change and rebuild cache
	for i := 5; i < 10; i++ {
		key := []byte{byte(i)}
		value, err := db2.Get(key)
		if err != nil {
			t.Fatalf("Error getting key %d from db2 after compact: %v", i, err)
		}
		expected := []byte{byte(i * 10)}
		if !bytes.Equal(value, expected) {
			t.Errorf("Key %d after compact: expected %v, got %v", i, expected, value)
		}
	}

	t.Log("Process 2 can still read data after Process 1 compacted (cache was rebuilt)")

	// Process 2: Verify that deleted keys are not accessible
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		_, err := db2.Get(key)
		if err != ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound for deleted key %d, got: %v", i, err)
		}
	}

	// Process 2: Write new data to verify cache is in sync
	newKey := []byte{99}
	newValue := []byte{100}
	if err := db2.Put(newKey, newValue); err != nil {
		t.Fatalf("Error putting new key from db2: %v", err)
	}

	// Process 1: Should be able to read the new key (after detecting file change)
	time.Sleep(10 * time.Millisecond)
	readValue, err := db1.Get(newKey)
	if err != nil {
		t.Fatalf("Error getting new key from db1: %v", err)
	}
	if !bytes.Equal(readValue, newValue) {
		t.Errorf("Expected %v, got %v", newValue, readValue)
	}

	t.Log("Both processes in sync - multi-process compact test passed")

	db1.Close()
}

// TestConcurrentCompactAndRead verifies that concurrent operations work correctly
// when one goroutine compacts while others read
func TestConcurrentCompactAndRead(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test_concurrent_compact_*.skv")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	db, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Write initial data
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i)}
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error putting key %d: %v", i, err)
		}
	}

	// Delete half of them to create fragmentation
	for i := 0; i < 50; i++ {
		key := []byte{byte(i)}
		if err := db.Delete(key); err != nil {
			t.Fatalf("Error deleting key %d: %v", i, err)
		}
	}

	// Channel to signal when all operations are done
	done := make(chan bool)
	errors := make(chan error, 10)

	// Start readers
	for g := 0; g < 5; g++ {
		go func(id int) {
			for i := 0; i < 20; i++ {
				// Read active keys
				for k := 50; k < 100; k++ {
					key := []byte{byte(k)}
					value, err := db.Get(key)
					if err != nil {
						errors <- err
						return
					}
					expected := []byte{byte(k)}
					if !bytes.Equal(value, expected) {
						errors <- err
						return
					}
				}
				time.Sleep(time.Millisecond)
			}
			done <- true
		}(g)
	}

	// Compact in the middle of reads
	time.Sleep(50 * time.Millisecond)
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}
	t.Log("Compact completed while readers were active")

	// Wait for all readers to finish
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Reader finished successfully
		case err := <-errors:
			t.Fatalf("Reader error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for readers")
		}
	}

	t.Log("All readers completed successfully after concurrent compact")
}

// TestChangeDetectionMechanism specifically tests the checkAndRebuild functionality
func TestChangeDetectionMechanism(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_change_detection_*.skv")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	db1, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database (db1): %v", err)
	}
	defer db1.Close()

	// Write initial data
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 10)}
		if err := db1.Put(key, value); err != nil {
			t.Fatalf("Error putting key %d: %v", i, err)
		}
	}

	// Close db1 to flush all data
	db1.Close()

	// Reopen both databases to start from same state
	db1, err = Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error reopening database (db1): %v", err)
	}
	defer db1.Close()

	db2, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database (db2): %v", err)
	}
	defer db2.Close()

	// Get initial file size - both should be the same now
	initialSize := db1.lastSize
	db2InitialSize := db2.lastSize
	t.Logf("Initial file size - db1: %d, db2: %d bytes", initialSize, db2InitialSize)

	if db2InitialSize != initialSize {
		t.Errorf("db2 initial size (%d) should match db1 initial size (%d)", db2InitialSize, initialSize)
	}

	// Add deletions and compact from db1
	for i := 0; i < 3; i++ {
		if err := db1.Delete([]byte{byte(i)}); err != nil {
			t.Fatalf("Error deleting key %d: %v", i, err)
		}
	}

	if err := db1.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	compactedSize := db1.lastSize
	t.Logf("After compact file size: %d bytes", compactedSize)

	if compactedSize >= initialSize {
		t.Errorf("Compacted size (%d) should be less than initial size (%d)", compactedSize, initialSize)
	}

	// db2 should detect the change on next operation
	// Read a key - this should trigger checkAndRebuild
	value, err := db2.Get([]byte{3})
	if err != nil {
		t.Fatalf("Error getting key from db2: %v", err)
	}
	if !bytes.Equal(value, []byte{30}) {
		t.Errorf("Expected value 30, got %v", value)
	}

	// Verify db2's lastSize was updated
	if db2.lastSize != compactedSize {
		t.Errorf("db2.lastSize (%d) should have been updated to compacted size (%d)", db2.lastSize, compactedSize)
	}

	t.Log("Change detection mechanism working correctly")
}
