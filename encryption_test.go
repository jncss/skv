package skv

import (
	"bytes"
	"os"
	"testing"
)

// TestEncryptionAES tests basic encryption with AES
func TestEncryptionAES(t *testing.T) {
	dbPath := "test_encryption_aes.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "test-password-123",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test data
	testKey := []byte("test-key")
	testValue := []byte("test-value-with-some-data")

	// Put
	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	// Get
	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if !bytes.Equal(retrieved, testValue) {
		t.Errorf("Retrieved value doesn't match. Expected %s, got %s", testValue, retrieved)
	}
}

// TestEncryptionSimpleCipher tests basic encryption with SimpleCipher
func TestEncryptionSimpleCipher(t *testing.T) {
	dbPath := "test_encryption_simplecipher.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionSimpleCipher,
		EncryptionPassword: "another-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	testKey := []byte("key123")
	testValue := []byte("value456")

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if !bytes.Equal(retrieved, testValue) {
		t.Errorf("Retrieved value doesn't match. Expected %s, got %s", testValue, retrieved)
	}
}

// TestEncryptionWithCompression tests encryption combined with compression
func TestEncryptionWithCompression(t *testing.T) {
	dbPath := "test_encryption_compression.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "compress-password",
		Compression:        CompressionSnappy,
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Large compressible data
	testKey := []byte("large-key")
	testValue := make([]byte, 10000)
	for i := range testValue {
		testValue[i] = byte('A' + (i % 10))
	}

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if !bytes.Equal(retrieved, testValue) {
		t.Errorf("Retrieved value doesn't match (length: expected %d, got %d)", len(testValue), len(retrieved))
	}
}

// TestEncryptionMultipleKeys tests multiple encrypted keys
func TestEncryptionMultipleKeys(t *testing.T) {
	dbPath := "test_encryption_multiple.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "multi-key-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store multiple key-value pairs
	testData := map[string]string{
		"user1":    "alice",
		"user2":    "bob",
		"password": "secret123",
		"token":    "abc-def-ghi",
		"apikey":   "sk-12345678",
	}

	for k, v := range testData {
		err = db.Put([]byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Failed to put %s: %v", k, err)
		}
	}

	// Retrieve and verify
	for k, expectedV := range testData {
		retrieved, err := db.Get([]byte(k))
		if err != nil {
			t.Fatalf("Failed to get %s: %v", k, err)
		}

		if string(retrieved) != expectedV {
			t.Errorf("Key %s: expected %s, got %s", k, expectedV, string(retrieved))
		}
	}
}

// TestEncryptionPersistence tests that encrypted data persists across reopens
func TestEncryptionPersistence(t *testing.T) {
	dbPath := "test_encryption_persistence.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	password := "persistence-password"
	testKey := []byte("persistent-key")
	testValue := []byte("persistent-value")

	// First: create and write
	{
		opts := &Options{
			Encryption:         EncryptionAES,
			EncryptionPassword: password,
			Logger:             NullLogger(),
		}

		db, err := OpenWithOptions(dbPath, opts)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		err = db.Put(testKey, testValue)
		if err != nil {
			t.Fatalf("Failed to put: %v", err)
		}

		db.Close()
	}

	// Second: reopen and read
	{
		opts := &Options{
			Encryption:         EncryptionAES,
			EncryptionPassword: password,
			Logger:             NullLogger(),
		}

		db, err := OpenWithOptions(dbPath, opts)
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}
		defer db.Close()

		retrieved, err := db.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to get after reopen: %v", err)
		}

		if !bytes.Equal(retrieved, testValue) {
			t.Errorf("Value not persisted correctly. Expected %s, got %s", testValue, retrieved)
		}
	}
}

// TestEncryptionUpdate tests updating encrypted values
func TestEncryptionUpdate(t *testing.T) {
	dbPath := "test_encryption_update.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionSimpleCipher,
		EncryptionPassword: "update-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	testKey := []byte("updateable-key")
	value1 := []byte("initial-value")
	value2 := []byte("updated-value")

	// Initial put
	err = db.Put(testKey, value1)
	if err != nil {
		t.Fatalf("Failed to put initial value: %v", err)
	}

	// Verify initial
	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get initial value: %v", err)
	}
	if !bytes.Equal(retrieved, value1) {
		t.Errorf("Initial value incorrect. Expected %s, got %s", value1, retrieved)
	}

	// Update
	err = db.Update(testKey, value2)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify updated
	retrieved, err = db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get updated value: %v", err)
	}
	if !bytes.Equal(retrieved, value2) {
		t.Errorf("Updated value incorrect. Expected %s, got %s", value2, retrieved)
	}
}

// TestEncryptionDelete tests deleting encrypted keys
func TestEncryptionDelete(t *testing.T) {
	dbPath := "test_encryption_delete.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "delete-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	testKey := []byte("deletable-key")
	testValue := []byte("deletable-value")

	// Put
	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	// Delete
	err = db.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deleted
	_, err = db.Get(testKey)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got: %v", err)
	}
}

// TestEncryptionNoPassword tests that encryption requires a password
func TestEncryptionNoPassword(t *testing.T) {
	dbPath := "test_encryption_nopassword.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "", // Empty password
		Logger:             NullLogger(),
	}

	_, err := OpenWithOptions(dbPath, opts)
	if err == nil {
		t.Fatal("Expected error when opening with encryption but no password")
	}
}

// TestEncryptionLargeKeys tests encryption with keys near the 255 byte limit
func TestEncryptionLargeKeys(t *testing.T) {
	dbPath := "test_encryption_largekeys.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionAES,
		EncryptionPassword: "large-key-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a key that when encrypted might be close to 255 bytes
	// EasyAES with Base64 encoding typically expands data
	testKey := make([]byte, 100) // Start with 100 bytes
	for i := range testKey {
		testKey[i] = byte('A' + (i % 26))
	}
	testValue := []byte("value for large key")

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put large key: %v", err)
	}

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get large key: %v", err)
	}

	if !bytes.Equal(retrieved, testValue) {
		t.Errorf("Value mismatch for large key")
	}
}

// TestEncryptionBinaryData tests encryption with binary data
func TestEncryptionBinaryData(t *testing.T) {
	dbPath := "test_encryption_binary.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	opts := &Options{
		Encryption:         EncryptionSimpleCipher,
		EncryptionPassword: "binary-password",
		Logger:             NullLogger(),
	}

	db, err := OpenWithOptions(dbPath, opts)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Binary data with all byte values
	testKey := []byte("binary-key")
	testValue := make([]byte, 256)
	for i := range testValue {
		testValue[i] = byte(i)
	}

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put binary data: %v", err)
	}

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get binary data: %v", err)
	}

	if !bytes.Equal(retrieved, testValue) {
		t.Errorf("Binary data mismatch")
	}
}
