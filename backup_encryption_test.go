package skv

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestBackupEncryptionPreservation verifies that backup preserves encrypted data
func TestBackupEncryptionPreservation(t *testing.T) {
	dbFile := "test_backup_enc.skv"
	backupFile := "test_backup_enc.json"
	defer os.Remove(dbFile)
	defer os.Remove(backupFile)

	// Create encrypted database
	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "test-password-123",
	}

	db, err := OpenWithOptions(dbFile, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Put sensitive data
	sensitiveKey := []byte("api_key")
	sensitiveValue := []byte("sk_live_secret_key_12345")

	if err := db.Put(sensitiveKey, sensitiveValue); err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Create backup
	if err := db.Backup(backupFile); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	db.Close()

	// Read backup file and verify data is encrypted
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	backupStr := string(backupData)

	// Verify sensitive data is NOT in plaintext
	if strings.Contains(backupStr, "api_key") {
		t.Error("Backup contains plaintext key 'api_key' - should be encrypted!")
	}
	if strings.Contains(backupStr, "sk_live_secret_key") {
		t.Error("Backup contains plaintext value - should be encrypted!")
	}

	// Parse JSON to verify structure
	var records []BackupRecord
	if err := json.Unmarshal(backupData, &records); err != nil {
		t.Fatalf("Failed to parse backup JSON: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	record := records[0]

	// Verify key is encrypted (should be base64-like gibberish)
	if record.Key == "api_key" {
		t.Error("Key in backup is plaintext - should be encrypted!")
	}
	if len(record.Key) < 10 {
		t.Error("Encrypted key seems too short")
	}

	// Verify value is encrypted
	valueToCheck := record.Value
	if record.IsBinary {
		valueToCheck = record.ValueB64
	}
	if valueToCheck == "sk_live_secret_key_12345" {
		t.Error("Value in backup is plaintext - should be encrypted!")
	}
	if len(valueToCheck) < 20 {
		t.Error("Encrypted value seems too short")
	}

	t.Logf("Backup verification passed:")
	t.Logf("  Encrypted key: %s", record.Key)
	t.Logf("  Encrypted value: %s", valueToCheck)
}

// TestBackupRestoreEncrypted verifies backup/restore with encryption works correctly
func TestBackupRestoreEncrypted(t *testing.T) {
	dbFile1 := "test_backup_enc1.skv"
	dbFile2 := "test_backup_enc2.skv"
	backupFile := "test_backup_enc_restore.json"
	defer os.Remove(dbFile1)
	defer os.Remove(dbFile2)
	defer os.Remove(backupFile)

	password := "super-secret-password"
	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: password,
	}

	// Create database with encrypted data
	db1, err := OpenWithOptions(dbFile1, opts)
	if err != nil {
		t.Fatalf("Failed to open db1: %v", err)
	}

	testData := map[string]string{
		"user":  "alice@example.com",
		"token": "bearer_xyz123456",
		"key":   "secret-api-key-789",
	}

	for k, v := range testData {
		if err := db1.Put([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Failed to put %s: %v", k, err)
		}
	}

	// Create backup
	if err := db1.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}
	db1.Close()

	// Restore to new database with SAME password
	db2, err := OpenWithOptions(dbFile2, opts)
	if err != nil {
		t.Fatalf("Failed to open db2: %v", err)
	}

	if err := db2.Restore(backupFile); err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify all data restored correctly
	for k, expectedV := range testData {
		actualV, err := db2.Get([]byte(k))
		if err != nil {
			t.Errorf("Failed to get %s after restore: %v", k, err)
			continue
		}
		if string(actualV) != expectedV {
			t.Errorf("Data mismatch for %s: expected %s, got %s", k, expectedV, string(actualV))
		}
	}

	db2.Close()
}

// TestBackupRestoreWrongPassword verifies that wrong password corrupts data
func TestBackupRestoreWrongPassword(t *testing.T) {
	dbFile1 := "test_backup_wrong1.skv"
	dbFile2 := "test_backup_wrong2.skv"
	backupFile := "test_backup_wrong.json"
	defer os.Remove(dbFile1)
	defer os.Remove(dbFile2)
	defer os.Remove(backupFile)

	// Create with password A
	optsA := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "password-A",
	}

	db1, err := OpenWithOptions(dbFile1, optsA)
	if err != nil {
		t.Fatalf("Failed to open db1: %v", err)
	}

	if err := db1.Put([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	if err := db1.Backup(backupFile); err != nil {
		t.Fatalf("Failed to backup: %v", err)
	}
	db1.Close()

	// Try to restore with password B (WRONG!)
	optsB := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "password-B",
	}

	db2, err := OpenWithOptions(dbFile2, optsB)
	if err != nil {
		t.Fatalf("Failed to open db2: %v", err)
	}

	// Restore will succeed (it just writes encrypted bytes)
	if err := db2.Restore(backupFile); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// But trying to read will fail or return garbage
	value, err := db2.Get([]byte("key"))

	// The data will be corrupted because we're trying to decrypt
	// data encrypted with password-A using password-B
	if err == nil && string(value) == "value" {
		t.Error("Data should be corrupted with wrong password, but it's correct!")
	}

	db2.Close()

	t.Log("Confirmed: wrong password results in corrupted/unreadable data")
}
