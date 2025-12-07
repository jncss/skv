package skv

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

// errorReader simulates a reader that fails after N bytes
type errorReader struct {
	data      []byte
	failAfter int
	readSoFar int
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.readSoFar >= r.failAfter {
		return 0, fmt.Errorf("simulated read error after %d bytes", r.readSoFar)
	}

	remaining := len(r.data) - r.readSoFar
	if remaining == 0 {
		return 0, io.EOF
	}

	toRead := len(p)
	if toRead > remaining {
		toRead = remaining
	}
	if r.readSoFar+toRead > r.failAfter {
		toRead = r.failAfter - r.readSoFar
	}

	copy(p, r.data[r.readSoFar:r.readSoFar+toRead])
	r.readSoFar += toRead

	if r.readSoFar >= r.failAfter {
		return toRead, fmt.Errorf("simulated read error after %d bytes", r.readSoFar)
	}

	return toRead, nil
}

// TestPutStreamRollbackOnError verifies that PutStream rolls back on write errors
func TestPutStreamRollbackOnError(t *testing.T) {
	// Create a test database
	db, err := Open("test_putstream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_putstream_rollback.skv")
	defer db.Close()

	// Put a valid key first
	validData := []byte("valid data for key1")
	err = db.Put([]byte("key1"), validData)
	if err != nil {
		t.Fatalf("Failed to put valid key: %v", err)
	}

	// Verify the key exists
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Failed to get key1: %v", err)
	}
	if !bytes.Equal(value, validData) {
		t.Fatalf("Data mismatch for key1")
	}

	// Get file size before failed operation
	fileInfo, err := os.Stat("test_putstream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	sizeBefore := fileInfo.Size()

	// Try to put a key with a failing reader
	failingData := bytes.Repeat([]byte("x"), 1000)
	failingReader := &errorReader{
		data:      failingData,
		failAfter: 100, // Fail after 100 bytes
	}

	err = db.PutStream([]byte("key2"), failingReader, int64(len(failingData)))
	if err == nil {
		t.Fatal("Expected PutStream to fail, but it succeeded")
	}

	// Verify the file was rolled back to original size
	fileInfo, err = os.Stat("test_putstream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to stat file after rollback: %v", err)
	}
	sizeAfter := fileInfo.Size()

	if sizeAfter != sizeBefore {
		t.Errorf("File size changed after rollback: before=%d, after=%d", sizeBefore, sizeAfter)
	}

	// Verify key2 was not added to cache
	_, err = db.Get([]byte("key2"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for key2, got: %v", err)
	}

	// Verify original key still works
	value, err = db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Failed to get key1 after rollback: %v", err)
	}
	if !bytes.Equal(value, validData) {
		t.Fatalf("Data corrupted for key1 after rollback")
	}

	// Verify we can still write successfully after rollback
	newData := []byte("new data after rollback")
	err = db.PutStream([]byte("key3"), bytes.NewReader(newData), int64(len(newData)))
	if err != nil {
		t.Fatalf("Failed to put key3 after rollback: %v", err)
	}

	value, err = db.Get([]byte("key3"))
	if err != nil {
		t.Fatalf("Failed to get key3: %v", err)
	}
	if !bytes.Equal(value, newData) {
		t.Fatalf("Data mismatch for key3")
	}
}

// TestUpdateStreamRollbackOnError verifies that UpdateStream rolls back on write errors
func TestUpdateStreamRollbackOnError(t *testing.T) {
	// Create a test database
	db, err := Open("test_updatestream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_updatestream_rollback.skv")
	defer db.Close()

	// Put initial keys
	originalData := []byte("original data for key1")
	err = db.Put([]byte("key1"), originalData)
	if err != nil {
		t.Fatalf("Failed to put key1: %v", err)
	}

	validData := []byte("valid data for key2")
	err = db.Put([]byte("key2"), validData)
	if err != nil {
		t.Fatalf("Failed to put key2: %v", err)
	}

	// Get file size before failed operation
	fileInfo, err := os.Stat("test_updatestream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	sizeBefore := fileInfo.Size()

	// Try to update with a failing reader
	failingData := bytes.Repeat([]byte("y"), 2000)
	failingReader := &errorReader{
		data:      failingData,
		failAfter: 200, // Fail after 200 bytes
	}

	err = db.UpdateStream([]byte("key1"), failingReader, int64(len(failingData)))
	if err == nil {
		t.Fatal("Expected UpdateStream to fail, but it succeeded")
	}

	// Verify the file was rolled back (but won't be exactly the same size due to deletion)
	fileInfo, err = os.Stat("test_updatestream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to stat file after rollback: %v", err)
	}
	sizeAfter := fileInfo.Size()

	// The file should have grown (new record written but not old record deleted due to rollback)
	growth := sizeAfter - sizeBefore
	if growth < 0 {
		t.Errorf("File shrank after rollback: growth=%d bytes", growth)
	}

	// With the new implementation, if UpdateStream fails, the old value should still exist
	// because we write the new record first, and only delete the old one after sync succeeds

	// Close and reopen to ensure cache is rebuilt after rollback
	db.Close()
	db, err = Open("test_updatestream_rollback.skv")
	if err != nil {
		t.Fatalf("Failed to reopen database after rollback: %v", err)
	}
	defer db.Close()

	// Verify key1 still has its ORIGINAL value (update was rolled back)
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Failed to get key1 after rollback: %v (expected original value to be preserved)", err)
	}
	if !bytes.Equal(value, originalData) {
		t.Errorf("key1 value changed after failed update: got %s, want %s", value, originalData)
	}

	// Verify key2 still works (wasn't affected)
	value, err = db.Get([]byte("key2"))
	if err != nil {
		t.Fatalf("Failed to get key2 after rollback and reopen: %v", err)
	}
	if !bytes.Equal(value, validData) {
		t.Fatalf("Data corrupted for key2 after rollback")
	}

	// Verify we can still write successfully after rollback
	newData := []byte("new data after rollback")
	err = db.Put([]byte("key3"), newData)
	if err != nil {
		t.Fatalf("Failed to put key3 after rollback: %v", err)
	}

	value, err = db.Get([]byte("key3"))
	if err != nil {
		t.Fatalf("Failed to get key3: %v", err)
	}
	if !bytes.Equal(value, newData) {
		t.Fatalf("Data mismatch for key3")
	}
}

// TestStreamRollbackPreservesIntegrity verifies database integrity after rollback
func TestStreamRollbackPreservesIntegrity(t *testing.T) {
	// Create a test database
	db, err := Open("test_stream_integrity.skv")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer os.Remove("test_stream_integrity.skv")

	// Put several keys
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))
		err = db.Put(key, value)
		if err != nil {
			t.Fatalf("Failed to put key%d: %v", i, err)
		}
	}

	// Close and reopen to ensure data is persisted
	db.Close()
	db, err = Open("test_stream_integrity.skv")
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}

	// Verify all keys before rollback
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		expectedValue := []byte(fmt.Sprintf("value%d", i))
		value, err := db.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key%d before rollback: %v", i, err)
		}
		if !bytes.Equal(value, expectedValue) {
			t.Fatalf("Data mismatch for key%d before rollback", i)
		}
	}

	// Try to put a key with a failing reader
	failingData := bytes.Repeat([]byte("z"), 5000)
	failingReader := &errorReader{
		data:      failingData,
		failAfter: 500,
	}

	err = db.PutStream([]byte("failkey"), failingReader, int64(len(failingData)))
	if err == nil {
		t.Fatal("Expected PutStream to fail")
	}

	// Verify all original keys still work
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		expectedValue := []byte(fmt.Sprintf("value%d", i))
		value, err := db.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key%d after rollback: %v", i, err)
		}
		if !bytes.Equal(value, expectedValue) {
			t.Fatalf("Data corrupted for key%d after rollback", i)
		}
	}

	// Close and reopen to verify persistence
	db.Close()
	db, err = Open("test_stream_integrity.skv")
	if err != nil {
		t.Fatalf("Failed to reopen database after rollback: %v", err)
	}
	defer db.Close()

	// Verify all keys again after reopen
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		expectedValue := []byte(fmt.Sprintf("value%d", i))
		value, err := db.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key%d after reopen: %v", i, err)
		}
		if !bytes.Equal(value, expectedValue) {
			t.Fatalf("Data corrupted for key%d after reopen", i)
		}
	}

	// Verify failed key doesn't exist
	_, err = db.Get([]byte("failkey"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected failkey not to exist, got: %v", err)
	}
}
