package skv

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestConcurrentReads(t *testing.T) {
	testFile := "test_concurrent_reads.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	// Create database and add test data
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add 100 keys
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error putting key%d: %v", i, err)
		}
	}

	// Concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := []byte(fmt.Sprintf("key%d", j))
				expected := []byte(fmt.Sprintf("value%d", j))
				value, err := db.Get(key)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: error getting key%d: %v", id, j, err)
					return
				}
				if !bytes.Equal(value, expected) {
					errors <- fmt.Errorf("goroutine %d: key%d expected %s, got %s", id, j, expected, value)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentWrites(t *testing.T) {
	testFile := "test_concurrent_writes.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Concurrent writes
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := []byte(fmt.Sprintf("key_%d_%d", id, j))
				value := []byte(fmt.Sprintf("value_%d_%d", id, j))
				if err := db.Put(key, value); err != nil {
					errors <- fmt.Errorf("goroutine %d: error putting key: %v", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify all keys were written
	keys, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}
	if len(keys) != 100 {
		t.Errorf("Expected 100 keys, got %d", len(keys))
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	testFile := "test_concurrent_readwrite.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add initial data
	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error putting initial key%d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Start readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				keyNum := j % 50
				key := []byte(fmt.Sprintf("key%d", keyNum))
				_, err := db.Get(key)
				if err != nil && err != ErrKeyNotFound {
					errors <- fmt.Errorf("reader %d: error getting key%d: %v", id, keyNum, err)
					return
				}
			}
		}(i)
	}

	// Start writers (updating existing keys)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				keyNum := j % 50
				key := []byte(fmt.Sprintf("key%d", keyNum))
				value := []byte(fmt.Sprintf("updated_%d_%d", id, j))
				if err := db.Update(key, value); err != nil {
					errors <- fmt.Errorf("writer %d: error updating key%d: %v", id, keyNum, err)
					return
				}
			}
		}(i)
	}

	// Start deleters
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				keyNum := 40 + j // Delete keys 40-49
				key := []byte(fmt.Sprintf("key%d", keyNum))
				err := db.Delete(key)
				if err != nil && err != ErrKeyNotFound {
					errors <- fmt.Errorf("deleter %d: error deleting key%d: %v", id, keyNum, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify database is still consistent
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying database after concurrent operations: %v", err)
	}
	t.Logf("After concurrent ops - Total: %d, Active: %d, Deleted: %d",
		stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)
}

func TestConcurrentCompact(t *testing.T) {
	testFile := "test_concurrent_compact.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add data
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error putting key%d: %v", i, err)
		}
	}

	// Update some keys to create deleted records
	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("updated_value%d", i))
		if err := db.Update(key, value); err != nil {
			t.Fatalf("Error updating key%d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Concurrent compacts (only one should succeed at a time due to lock)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := db.Compact(); err != nil {
				errors <- fmt.Errorf("compact %d: error: %v", id, err)
			}
		}(i)
	}

	// Concurrent reads during compact
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := []byte(fmt.Sprintf("key%d", j))
				_, err := db.Get(key)
				if err != nil && err != ErrKeyNotFound {
					errors <- fmt.Errorf("reader %d during compact: error getting key%d: %v", id, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify final state
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}

	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got %d", stats.DeletedRecords)
	}
	if stats.ActiveRecords != 100 {
		t.Errorf("Expected 100 active records, got %d", stats.ActiveRecords)
	}
}

func TestConcurrentKeys(t *testing.T) {
	testFile := "test_concurrent_keys.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add initial data
	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error putting key%d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Concurrent Keys() calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			keys, err := db.Keys()
			if err != nil {
				errors <- fmt.Errorf("keys %d: error: %v", id, err)
				return
			}
			if len(keys) < 50 {
				errors <- fmt.Errorf("keys %d: expected at least 50 keys, got %d", id, len(keys))
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
