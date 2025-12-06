package skv

import (
	"bytes"
	"encoding/json"
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

func TestBackupRestore(t *testing.T) {
	dbFile := "test_backup.skv"
	backupFile := "test_backup.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	// Create database and add some data
	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Add various types of data
	testData := map[string][]byte{
		"simple":    []byte("hello world"),
		"unicode":   []byte("Hello 世界 🌍"),
		"binary":    {0x00, 0x01, 0x02, 0xFF, 0xFE},
		"large":     make([]byte, 300), // Larger than 256 bytes
		"empty":     []byte(""),
		"multiline": []byte("line1\nline2\nline3"),
	}

	// Fill the large array with some pattern
	for i := range testData["large"] {
		testData["large"][i] = byte(i % 256)
	}

	// Insert all test data
	for key, value := range testData {
		if err := db.Put([]byte(key), value); err != nil {
			t.Fatalf("Failed to put key %s: %v", key, err)
		}
	}

	// Backup the database
	if err := db.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup database: %v", err)
	}

	// Verify backup file exists and is valid JSON
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	var records []BackupRecord
	if err := json.Unmarshal(backupData, &records); err != nil {
		t.Fatalf("Backup file is not valid JSON: %v", err)
	}

	if len(records) != len(testData) {
		t.Errorf("Expected %d records in backup, got %d", len(testData), len(records))
	}

	// Close and delete the database
	db.Close()
	os.Remove(dbFile)

	// Create a new empty database
	db2, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open new database: %v", err)
	}
	defer db2.Close()

	// Restore from backup
	if err := db2.Restore(backupFile); err != nil {
		t.Fatalf("Failed to restore database: %v", err)
	}

	// Verify all data was restored correctly
	for key, expectedValue := range testData {
		value, err := db2.Get([]byte(key))
		if err != nil {
			t.Errorf("Failed to get key %s after restore: %v", key, err)
			continue
		}

		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Value mismatch for key %s after restore.\nExpected: %v\nGot: %v", key, expectedValue, value)
		}
	}
}

func TestBackupStringVsBinary(t *testing.T) {
	dbFile := "test_backup_encoding.skv"
	backupFile := "test_backup_encoding.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test cases: key, value, expected encoding (true = binary/base64, false = string)
	testCases := []struct {
		key          string
		value        []byte
		expectBinary bool
		description  string
	}{
		{"small_text", []byte("Hello"), false, "Small valid UTF-8 text should be stored as string"},
		{"small_unicode", []byte("Hello 世界"), false, "Small valid UTF-8 with unicode should be stored as string"},
		{"small_binary", []byte{0x00, 0xFF, 0x80}, true, "Small binary data should be stored as base64"},
		{"large_text", bytes.Repeat([]byte("a"), 300), true, "Large text (>256 bytes) should be stored as base64"},
		{"exactly_256", bytes.Repeat([]byte("b"), 256), false, "Exactly 256 bytes of valid UTF-8 should be stored as string"},
		{"257_bytes", bytes.Repeat([]byte("c"), 257), true, "257 bytes should be stored as base64"},
	}

	// Insert test data
	for _, tc := range testCases {
		if err := db.Put([]byte(tc.key), tc.value); err != nil {
			t.Fatalf("Failed to put key %s: %v", tc.key, err)
		}
	}

	// Create backup
	if err := db.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}

	// Parse backup and verify encoding
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	var records []BackupRecord
	if err := json.Unmarshal(backupData, &records); err != nil {
		t.Fatalf("Failed to parse backup JSON: %v", err)
	}

	// Create a map for easy lookup
	recordMap := make(map[string]BackupRecord)
	for _, record := range records {
		recordMap[record.Key] = record
	}

	// Verify encoding for each test case
	for _, tc := range testCases {
		record, found := recordMap[tc.key]
		if !found {
			t.Errorf("Key %s not found in backup", tc.key)
			continue
		}

		if record.IsBinary != tc.expectBinary {
			t.Errorf("%s: expected IsBinary=%v, got %v", tc.description, tc.expectBinary, record.IsBinary)
		}

		if tc.expectBinary {
			if record.ValueB64 == "" {
				t.Errorf("%s: expected ValueB64 to be set, but it's empty", tc.description)
			}
			if record.Value != "" {
				t.Errorf("%s: expected Value to be empty when IsBinary=true, but got %q", tc.description, record.Value)
			}
		} else {
			if record.Value == "" && len(tc.value) > 0 {
				t.Errorf("%s: expected Value to be set, but it's empty", tc.description)
			}
			if record.ValueB64 != "" {
				t.Errorf("%s: expected ValueB64 to be empty when IsBinary=false, but got %q", tc.description, record.ValueB64)
			}
		}
	}
}

func TestRestoreOverwrite(t *testing.T) {
	dbFile := "test_restore_overwrite.skv"
	backupFile := "test_restore_overwrite.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	// Create database with initial data
	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.Put([]byte("key1"), []byte("original1"))
	db.Put([]byte("key2"), []byte("original2"))
	db.Put([]byte("key3"), []byte("original3"))

	// Backup
	if err := db.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}

	// Modify the database
	db.Put([]byte("key1"), []byte("modified1"))
	db.Put([]byte("key2"), []byte("modified2"))
	db.Put([]byte("key4"), []byte("new_key"))

	// Restore (should overwrite key1 and key2, leave key4 untouched)
	if err := db.Restore(backupFile); err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify
	tests := []struct {
		key      string
		expected string
	}{
		{"key1", "original1"}, // Restored
		{"key2", "original2"}, // Restored
		{"key3", "original3"}, // Already existed
		{"key4", "new_key"},   // Not in backup, should remain
	}

	for _, tt := range tests {
		value, err := db.Get([]byte(tt.key))
		if err != nil {
			t.Errorf("Failed to get key %s: %v", tt.key, err)
			continue
		}

		if string(value) != tt.expected {
			t.Errorf("Key %s: expected %q, got %q", tt.key, tt.expected, string(value))
		}
	}

	db.Close()
}

func TestBackupEmptyDatabase(t *testing.T) {
	dbFile := "test_backup_empty.skv"
	backupFile := "test_backup_empty.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Backup empty database
	if err := db.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup empty database: %v", err)
	}

	// Verify backup contains empty array
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	var records []BackupRecord
	if err := json.Unmarshal(backupData, &records); err != nil {
		t.Fatalf("Failed to parse backup JSON: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("Expected empty backup, got %d records", len(records))
	}
}

func TestRestoreInvalidFile(t *testing.T) {
	dbFile := "test_restore_invalid.skv"
	invalidFile := "test_restore_invalid.json"
	defer os.Remove(dbFile)
	defer os.Remove(invalidFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create invalid JSON file
	os.WriteFile(invalidFile, []byte("not valid json"), 0644)

	// Restore should fail
	if err := db.Restore(invalidFile); err == nil {
		t.Error("Expected error when restoring from invalid JSON file, got nil")
	}

	// Restore from non-existent file should fail
	if err := db.Restore("non_existent_file.json"); err == nil {
		t.Error("Expected error when restoring from non-existent file, got nil")
	}
}
