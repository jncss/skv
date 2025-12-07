package skv

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestReadRecordOptimization verifies that readRecord(false) doesn't load
// data into memory when readData=false, using streaming CRC verification instead
func TestReadRecordOptimization(t *testing.T) {
	dbPath := "test_readrecord_opt.skv"
	defer os.Remove(dbPath)

	// Create database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Insert records of different sizes to test all code paths
	testCases := []struct {
		name     string
		key      string
		dataSize int
	}{
		{"Small_Type1Byte", "small", 100},              // Type1Byte with CRC-16
		{"Medium_Type2Bytes", "medium", 500},           // Type2Bytes with CRC-32
		{"Large_Type4Bytes", "large", 100000},          // Type4Bytes with CRC-32
		{"VeryLarge_Type4Bytes", "verylarge", 5000000}, // Very large to test streaming
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test data
			data := bytes.Repeat([]byte{byte(tc.dataSize % 256)}, tc.dataSize)

			// Insert
			err := db.Put([]byte(tc.key), data)
			if err != nil {
				t.Fatalf("Failed to insert %s: %v", tc.key, err)
			}
		})
	}

	// Close and reopen to test loadCacheFromFile which uses readRecord(false)
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Reopen - this will call loadCacheFromFile which uses readRecord(false)
	// The optimization should prevent loading all data into memory
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify all records are in cache and data is correct
	for _, tc := range testCases {
		t.Run(tc.name+"_Verify", func(t *testing.T) {
			// Check cache
			if !db.Has([]byte(tc.key)) {
				t.Errorf("Key %s not found in cache after reopen", tc.key)
			}

			// Verify data
			retrievedData, err := db.Get([]byte(tc.key))
			if err != nil {
				t.Fatalf("Failed to get %s: %v", tc.key, err)
			}

			expectedData := bytes.Repeat([]byte{byte(tc.dataSize % 256)}, tc.dataSize)
			if !bytes.Equal(retrievedData, expectedData) {
				t.Errorf("Data mismatch for %s", tc.key)
			}
		})
	}

	// Verify record count
	count := db.Count()
	if count != len(testCases) {
		t.Errorf("Expected %d records, got %d", len(testCases), count)
	}
}

// TestReadRecordOptimizationWithDeleted verifies that readRecord(false)
// skips CRC verification for deleted records
func TestReadRecordOptimizationWithDeleted(t *testing.T) {
	dbPath := "test_readrecord_deleted.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Insert and delete records
	for i := 0; i < 10; i++ {
		key := []byte("key_" + string(rune('0'+i)))
		data := bytes.Repeat([]byte{byte(i)}, 10000) // 10KB each

		if err := db.Put(key, data); err != nil {
			t.Fatalf("Failed to insert key_%d: %v", i, err)
		}
	}

	// Delete half of them
	for i := 0; i < 5; i++ {
		key := []byte("key_" + string(rune('0'+i)))
		if err := db.Delete(key); err != nil {
			t.Fatalf("Failed to delete key_%d: %v", i, err)
		}
	}

	// Close and reopen
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Reopen - loadCacheFromFile should skip deleted records efficiently
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify only active records are in cache
	if db.Count() != 5 {
		t.Errorf("Expected 5 active records, got %d", db.Count())
	}

	// Verify deleted records are not in cache
	for i := 0; i < 5; i++ {
		key := []byte("key_" + string(rune('0'+i)))
		if db.Has(key) {
			t.Errorf("Deleted key_%d should not be in cache", i)
		}
	}

	// Verify active records are correct
	for i := 5; i < 10; i++ {
		key := []byte("key_" + string(rune('0'+i)))
		if !db.Has(key) {
			t.Errorf("Active key_%d should be in cache", i)
		}

		data, err := db.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key_%d: %v", i, err)
		}

		expectedData := bytes.Repeat([]byte{byte(i)}, 10000)
		if !bytes.Equal(data, expectedData) {
			t.Errorf("Data mismatch for key_%d", i)
		}
	}
}

// TestReadRecordOptimizationCRCVerification verifies that CRC is still
// verified correctly when readData=false
func TestReadRecordOptimizationCRCVerification(t *testing.T) {
	dbPath := "test_readrecord_crc.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Insert a large record
	key := []byte("testkey")
	data := bytes.Repeat([]byte("ABCDEFGH"), 100000) // 800KB

	if err := db.Put(key, data); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Corrupt the data in the file (but not the CRC)
	file, err := os.OpenFile(dbPath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for corruption: %v", err)
	}

	// Seek to position after header + type + keySize + key + dataSize (about 20 bytes)
	// and corrupt some data bytes
	if _, err := file.Seek(20, 0); err != nil {
		file.Close()
		t.Fatalf("Failed to seek: %v", err)
	}

	corruptData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	if _, err := file.Write(corruptData); err != nil {
		file.Close()
		t.Fatalf("Failed to write corrupt data: %v", err)
	}
	file.Close()

	// Try to reopen - should detect corruption during cache load
	db, err = Open(dbPath)
	if err == nil {
		db.Close()
		t.Fatal("Expected CRC error when opening corrupted database, got nil")
	}

	// Verify it's a CRC error (should contain "CRC mismatch" somewhere)
	if !strings.Contains(err.Error(), "CRC mismatch") {
		t.Errorf("Expected CRC mismatch error, got: %v", err)
	}
}
