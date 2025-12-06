package skv

import (
	"os"
	"testing"
)

// Test error cases in Open
func TestOpenErrors(t *testing.T) {
	// Test opening file in non-existent directory with no permissions
	_, err := Open("/root/nonexistent/test.skv")
	if err == nil {
		t.Error("Expected error when opening file in inaccessible directory")
	}
}

// Test error in writeHeader
func TestWriteHeaderError(t *testing.T) {
	dbFile := "test_write_header.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Close the file to cause sync error
	db.file.Close()

	// Try to write header (should fail)
	err = db.writeHeader()
	if err == nil {
		t.Error("Expected error when writing header to closed file")
	}
}

// Test error in verifyHeader with corrupted file
func TestVerifyHeaderCorrupted(t *testing.T) {
	dbFile := "test_corrupted.skv"
	defer os.Remove(dbFile)

	// Create file with wrong magic bytes
	file, _ := os.Create(dbFile)
	file.Write([]byte("BAD"))
	file.Write([]byte{0, 1, 0})
	file.Close()

	_, err := Open(dbFile)
	if err == nil {
		t.Error("Expected error when opening file with wrong magic bytes")
	}
}

// Test error in verifyHeader with truncated file
func TestVerifyHeaderTruncated(t *testing.T) {
	dbFile := "test_truncated.skv"
	defer os.Remove(dbFile)

	// Create file with incomplete header
	file, _ := os.Create(dbFile)
	file.Write([]byte("SK")) // Only 2 bytes instead of 6
	file.Close()

	_, err := Open(dbFile)
	if err == nil {
		t.Error("Expected error when opening file with truncated header")
	}
}

// Test Close error handling
func TestCloseError(t *testing.T) {
	dbFile := "test_close_error.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Close once normally
	if err := db.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	// Close again should fail
	if err := db.Close(); err == nil {
		t.Error("Expected error when closing already closed database")
	}
}

// Test CloseWithCompact error handling
func TestCloseWithCompactError(t *testing.T) {
	dbFile := "test_compact_close_error.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Add some data
	db.PutString("key1", "value1")

	// Close the file to cause error during compact
	db.file.Close()

	// CloseWithCompact should fail
	if err := db.CloseWithCompact(); err == nil {
		t.Error("Expected error when compacting with closed file")
	}
}

// Test getRecordType with various sizes
func TestGetRecordType(t *testing.T) {
	tests := []struct {
		size     uint64
		expected byte
	}{
		{0, Type1Byte},
		{255, Type1Byte},
		{256, Type2Bytes},
		{65535, Type2Bytes},
		{65536, Type4Bytes},
		{4294967295, Type4Bytes},
		{4294967296, Type8Bytes},
	}

	for _, tt := range tests {
		result := getRecordType(tt.size)
		if result != tt.expected {
			t.Errorf("getRecordType(%d) = %v, want %v", tt.size, result, tt.expected)
		}
	}
}

// Test calculateRecordSize with different types
func TestCalculateRecordSize(t *testing.T) {
	tests := []struct {
		keySize    byte
		dataSize   uint64
		recordType byte
		expected   uint64
	}{
		{5, 10, Type1Byte, 1 + 1 + 5 + 1 + 10},                   // 18
		{5, 10, Type2Bytes, 1 + 1 + 5 + 2 + 10},                  // 19
		{5, 10, Type4Bytes, 1 + 1 + 5 + 4 + 10},                  // 21
		{5, 10, Type8Bytes, 1 + 1 + 5 + 8 + 10},                  // 25
		{10, 100, DeletedFlag | Type1Byte, 1 + 1 + 10 + 1 + 100}, // With deleted flag
	}

	for _, tt := range tests {
		result := calculateRecordSize(tt.keySize, tt.dataSize, tt.recordType)
		if result != tt.expected {
			t.Errorf("calculateRecordSize(%d, %d, %v) = %d, want %d",
				tt.keySize, tt.dataSize, tt.recordType, result, tt.expected)
		}
	}
}

// Test ForEach with error in callback
func TestForEachError(t *testing.T) {
	dbFile := "test_foreach_error.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	db.PutString("key1", "value1")
	db.PutString("key2", "value2")

	// Callback returns error on second key
	count := 0
	err = db.ForEach(func(key []byte, value []byte) error {
		count++
		if count == 2 {
			return ErrKeyNotFound // Return any error
		}
		return nil
	})

	if err != ErrKeyNotFound {
		t.Error("Expected error from ForEach callback")
	}
}

// Test Clear with errors
func TestClearError(t *testing.T) {
	dbFile := "test_clear_error.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.PutString("key1", "value1")

	// Close file to cause error
	db.file.Close()

	// Clear should fail
	if err := db.Clear(); err == nil {
		t.Error("Expected error when clearing with closed file")
	}
}

// Test PutBatch with validation errors
func TestPutBatchValidation(t *testing.T) {
	dbFile := "test_putbatch_validation.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test with empty key
	items := map[string][]byte{
		"": []byte("value"),
	}
	if err := db.PutBatch(items); err == nil {
		t.Error("Expected error when batch putting empty key")
	}

	// Test with key too long
	longKey := string(make([]byte, 256))
	items = map[string][]byte{
		longKey: []byte("value"),
	}
	if err := db.PutBatch(items); err == nil {
		t.Error("Expected error when batch putting key > 255 bytes")
	}
}

// Test GetBatch with empty keys list
func TestGetBatchEmpty(t *testing.T) {
	dbFile := "test_getbatch_empty.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get batch with empty list
	result, err := db.GetBatch([][]byte{})
	if err != nil {
		t.Errorf("GetBatch with empty list failed: %v", err)
	}
	if len(result) != 0 {
		t.Error("Expected empty result from GetBatch with empty list")
	}
}

// Test readRecord with file seek errors
func TestReadRecordSeekError(t *testing.T) {
	dbFile := "test_read_seek.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.PutString("key1", "value1")

	// Close file to cause seek errors
	db.file.Close()

	_, _, _, _, err = db.readRecord(false)
	if err == nil {
		t.Error("Expected error when reading from closed file")
	}
}

// Test Backup with file write error
func TestBackupWriteError(t *testing.T) {
	dbFile := "test_backup_write.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	db.PutString("key1", "value1")

	// Try to backup to invalid path
	if err := db.Backup("/root/invalid/path.json"); err == nil {
		t.Error("Expected error when backing up to invalid path")
	}
}

// Test Restore with seek errors
func TestRestoreSeekError(t *testing.T) {
	dbFile := "test_restore_seek.skv"
	backupFile := "test_restore_seek.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.PutString("key1", "value1")
	db.Backup(backupFile)

	// Close file to cause errors
	db.file.Close()

	// Restore should fail
	if err := db.Restore(backupFile); err == nil {
		t.Error("Expected error when restoring to closed database")
	}
}

// Test Update with validation
func TestUpdateValidation(t *testing.T) {
	dbFile := "test_update_validation.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Update with empty key
	if err := db.Update([]byte(""), []byte("value")); err == nil {
		t.Error("Expected error when updating with empty key")
	}
}

// Test Get with seek errors
func TestGetSeekError(t *testing.T) {
	dbFile := "test_get_seek.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.PutString("key1", "value1")

	// Close file
	db.file.Close()

	// Get should fail
	if _, err := db.Get([]byte("key1")); err == nil {
		t.Error("Expected error when getting from closed file")
	}
}

// Test Verify with file errors
func TestVerifyFileError(t *testing.T) {
	dbFile := "test_verify_error.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.PutString("key1", "value1")

	// Close file
	db.file.Close()

	// Verify should fail
	if _, err := db.Verify(); err == nil {
		t.Error("Expected error when verifying closed file")
	}
}
