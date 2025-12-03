package skv

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestOpen(t *testing.T) {
	// Test file name
	testFile := "test_db.skv"

	// Clean up file if it exists
	os.Remove(testFile)
	defer os.Remove(testFile)

	// Test 1: Create new file
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Verify that the file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Close the database
	if err := db.Close(); err != nil {
		t.Errorf("Error closing database: %v", err)
	}

	// Test 2: Open existing file
	db2, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening existing file: %v", err)
	}
	defer db2.Close()
}

func TestOpenAutoExtension(t *testing.T) {
	// Test automatic .skv extension
	testFile := "test_auto"
	fullPath := testFile + ".skv"

	os.Remove(fullPath)
	defer os.Remove(fullPath)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Verify that the file with extension was created
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("File with .skv extension was not created")
	}
}

func TestPut(t *testing.T) {
	testFile := "test_put.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Store small data (Type1Byte)
	err = db.Put([]byte("key1"), []byte("small value"))
	if err != nil {
		t.Errorf("Error storing small data: %v", err)
	}

	// Test 2: Store empty data
	err = db.Put([]byte("key2"), []byte{})
	if err != nil {
		t.Errorf("Error storing empty data: %v", err)
	}

	// Test 3: Store medium-sized data (Type2Bytes - >255 bytes)
	largeData := make([]byte, 300)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	err = db.Put([]byte("key3"), largeData)
	if err != nil {
		t.Errorf("Error storing medium data: %v", err)
	}

	// Test 4: Empty key (should fail)
	err = db.Put([]byte{}, []byte("value"))
	if err == nil {
		t.Error("Should return error with empty key")
	}

	// Test 5: Key too long (should fail)
	longKey := make([]byte, 256)
	for i := range longKey {
		longKey[i] = 'a'
	}
	err = db.Put(longKey, []byte("value"))
	if err == nil {
		t.Error("Should return error with key > 255 bytes")
	}

	// Verify that file has content
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Error reading file info: %v", err)
	}
	if info.Size() == 0 {
		t.Error("File is empty after Put")
	}
}

func TestPutDifferentTypes(t *testing.T) {
	testFile := "test_types.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test Type1Byte (≤ 255 bytes)
	data1 := bytes.Repeat([]byte("a"), 200)
	if err := db.Put([]byte("type1"), data1); err != nil {
		t.Errorf("Error with Type1Byte: %v", err)
	}

	// Test Type2Bytes (256-65535 bytes)
	data2 := bytes.Repeat([]byte("b"), 1000)
	if err := db.Put([]byte("type2"), data2); err != nil {
		t.Errorf("Error with Type2Bytes: %v", err)
	}

	// Test Type4Bytes (>65535 bytes)
	data4 := bytes.Repeat([]byte("c"), 100000)
	if err := db.Put([]byte("type4"), data4); err != nil {
		t.Errorf("Error with Type4Bytes: %v", err)
	}

	// Verify file size
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Error reading file info: %v", err)
	}

	// Expected size approximately:
	// type1: 1(type)+1(keySize)+5(key)+1(dataSize)+200(data) = 208
	// type2: 1+1+5+2+1000 = 1009
	// type4: 1+1+5+4+100000 = 100011
	// Total ≈ 101228 bytes
	expectedSize := int64(208 + 1009 + 100011)
	if info.Size() != expectedSize {
		t.Logf("File size: %d, expected: %d", info.Size(), expectedSize)
	}
}

func TestGet(t *testing.T) {
	testFile := "test_get.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Try to retrieve a non-existent key
	_, err = db.Get([]byte("nonexistent"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got: %v", err)
	}

	// Test 2: Store and retrieve a key
	key1 := []byte("key1")
	value1 := []byte("value1")
	if err := db.Put(key1, value1); err != nil {
		t.Fatalf("Error storing key1: %v", err)
	}

	result, err := db.Get(key1)
	if err != nil {
		t.Fatalf("Error retrieving key1: %v", err)
	}
	if !bytes.Equal(result, value1) {
		t.Errorf("Incorrect value. Expected: %s, Got: %s", value1, result)
	}

	// Test 3: Multiple keys
	key2 := []byte("key2")
	value2 := []byte("value2 different")
	if err := db.Put(key2, value2); err != nil {
		t.Fatalf("Error storing key2: %v", err)
	}

	result2, err := db.Get(key2)
	if err != nil {
		t.Fatalf("Error retrieving key2: %v", err)
	}
	if !bytes.Equal(result2, value2) {
		t.Errorf("Incorrect value for key2. Expected: %s, Got: %s", value2, result2)
	}

	// Verify that key1 still exists
	result1Again, err := db.Get(key1)
	if err != nil {
		t.Fatalf("Error retrieving key1 again: %v", err)
	}
	if !bytes.Equal(result1Again, value1) {
		t.Errorf("Incorrect value for key1. Expected: %s, Got: %s", value1, result1Again)
	}

	// Test 4: Empty data
	key3 := []byte("key3")
	emptyValue := []byte{}
	if err := db.Put(key3, emptyValue); err != nil {
		t.Fatalf("Error storing key with empty value: %v", err)
	}

	result3, err := db.Get(key3)
	if err != nil {
		t.Fatalf("Error retrieving key with empty value: %v", err)
	}
	if len(result3) != 0 {
		t.Errorf("Expected empty value, got: %v", result3)
	}

	// Test 5: Empty key (should fail)
	_, err = db.Get([]byte{})
	if err == nil {
		t.Error("Should return error with empty key")
	}
}

func TestGetDifferentTypes(t *testing.T) {
	testFile := "test_get_types.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test Type1Byte
	data1 := bytes.Repeat([]byte("a"), 200)
	if err := db.Put([]byte("type1"), data1); err != nil {
		t.Fatalf("Error storing Type1Byte: %v", err)
	}

	result1, err := db.Get([]byte("type1"))
	if err != nil {
		t.Fatalf("Error retrieving Type1Byte: %v", err)
	}
	if !bytes.Equal(result1, data1) {
		t.Error("Type1Byte data does not match")
	}

	// Test Type2Bytes
	data2 := bytes.Repeat([]byte("b"), 1000)
	if err := db.Put([]byte("type2"), data2); err != nil {
		t.Fatalf("Error storing Type2Bytes: %v", err)
	}

	result2, err := db.Get([]byte("type2"))
	if err != nil {
		t.Fatalf("Error retrieving Type2Bytes: %v", err)
	}
	if !bytes.Equal(result2, data2) {
		t.Error("Type2Bytes data does not match")
	}

	// Test Type4Bytes
	data4 := bytes.Repeat([]byte("c"), 100000)
	if err := db.Put([]byte("type4"), data4); err != nil {
		t.Fatalf("Error storing Type4Bytes: %v", err)
	}

	result4, err := db.Get([]byte("type4"))
	if err != nil {
		t.Fatalf("Error retrieving Type4Bytes: %v", err)
	}
	if !bytes.Equal(result4, data4) {
		t.Error("Type4Bytes data does not match")
	}
}

func TestPutAndGet(t *testing.T) {
	testFile := "test_put_get.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Store several keys
	testData := map[string]string{
		"name":    "John",
		"surname": "Garcia",
		"city":    "Barcelona",
		"country": "Catalonia",
		"code":    "08001",
	}

	for k, v := range testData {
		if err := db.Put([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Error storing %s: %v", k, err)
		}
	}

	// Retrieve and verify all keys
	for k, expectedValue := range testData {
		gotValue, err := db.Get([]byte(k))
		if err != nil {
			t.Fatalf("Error retrieving %s: %v", k, err)
		}
		if !bytes.Equal(gotValue, []byte(expectedValue)) {
			t.Errorf("Incorrect value for %s. Expected: %s, Got: %s", k, expectedValue, gotValue)
		}
	}
}

func TestDelete(t *testing.T) {
	testFile := "test_delete.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Try to delete a non-existent key
	err = db.Delete([]byte("nonexistent"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got: %v", err)
	}

	// Test 2: Store and delete a key
	key1 := []byte("key1")
	value1 := []byte("value1")
	if err := db.Put(key1, value1); err != nil {
		t.Fatalf("Error storing key1: %v", err)
	}

	// Verify that it exists
	_, err = db.Get(key1)
	if err != nil {
		t.Fatalf("Error retrieving key1 before delete: %v", err)
	}

	// Delete the key
	if err := db.Delete(key1); err != nil {
		t.Fatalf("Error deleting key1: %v", err)
	}

	// Verify that it no longer exists
	_, err = db.Get(key1)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after delete, got: %v", err)
	}

	// Test 3: Delete with multiple keys
	key2 := []byte("key2")
	value2 := []byte("value2")
	key3 := []byte("key3")
	value3 := []byte("value3")

	db.Put(key2, value2)
	db.Put(key3, value3)

	// Delete key2
	if err := db.Delete(key2); err != nil {
		t.Fatalf("Error deleting key2: %v", err)
	}

	// Verify that key2 does not exist
	_, err = db.Get(key2)
	if err != ErrKeyNotFound {
		t.Errorf("key2 should be deleted")
	}

	// Verify that key3 still exists
	value, err := db.Get(key3)
	if err != nil {
		t.Fatalf("Error retrieving key3: %v", err)
	}
	if !bytes.Equal(value, value3) {
		t.Errorf("key3 should still exist")
	}

	// Test 4: Empty key (should fail)
	err = db.Delete([]byte{})
	if err == nil {
		t.Error("Should return error with empty key")
	}
}

func TestDeleteWithUpdate(t *testing.T) {
	testFile := "test_delete_update.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Store a key
	key := []byte("key")
	value1 := []byte("value1")
	if err := db.Put(key, value1); err != nil {
		t.Fatalf("Error storing value1: %v", err)
	}

	// Update the key (stored at the end)
	value2 := []byte("value2")
	if err := db.Update(key, value2); err != nil {
		t.Fatalf("Error storing value2: %v", err)
	}

	// Delete the key (should delete the last occurrence)
	if err := db.Delete(key); err != nil {
		t.Fatalf("Error deleting key: %v", err)
	}

	// Verify that it does not exist
	_, err = db.Get(key)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after deleting last occurrence")
	}
}

func TestDeleteAndReAdd(t *testing.T) {
	testFile := "test_delete_readd.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	key := []byte("key")
	value1 := []byte("value1")
	value2 := []byte("value2_different")

	// Store
	if err := db.Put(key, value1); err != nil {
		t.Fatalf("Error storing: %v", err)
	}

	// Delete
	if err := db.Delete(key); err != nil {
		t.Fatalf("Error deleting: %v", err)
	}

	// Re-add with a different value
	if err := db.Put(key, value2); err != nil {
		t.Fatalf("Error re-adding: %v", err)
	}

	// Verify that the new value is correct
	result, err := db.Get(key)
	if err != nil {
		t.Fatalf("Error retrieving after re-adding: %v", err)
	}
	if !bytes.Equal(result, value2) {
		t.Errorf("Incorrect value. Expected: %s, Got: %s", value2, result)
	}
}

func TestVerify(t *testing.T) {
	testFile := "test_verify.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Empty file
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying empty file: %v", err)
	}
	if stats.TotalRecords != 0 || stats.ActiveRecords != 0 || stats.DeletedRecords != 0 {
		t.Errorf("Incorrect statistics for empty file: %+v", stats)
	}

	// Test 2: Add some records
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))

	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying: %v", err)
	}
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 3 {
		t.Errorf("Expected 3 active records, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records, got: %d", stats.DeletedRecords)
	}

	// Test 3: Delete a record
	db.Delete([]byte("key2"))

	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after delete: %v", err)
	}
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 1 {
		t.Errorf("Expected 1 deleted record, got: %d", stats.DeletedRecords)
	}

	// Test 4: Update a key (stored at the end)
	db.Update([]byte("key1"), []byte("value1_new"))

	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after update: %v", err)
	}
	if stats.TotalRecords != 4 {
		t.Errorf("Expected 4 total records, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 2 {
		t.Errorf("Expected 2 deleted records, got: %d", stats.DeletedRecords)
	}
}

func TestVerifyWithDifferentTypes(t *testing.T) {
	testFile := "test_verify_types.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Store different data types
	db.Put([]byte("type1"), bytes.Repeat([]byte("a"), 100))    // Type1Byte
	db.Put([]byte("type2"), bytes.Repeat([]byte("b"), 1000))   // Type2Bytes
	db.Put([]byte("type4"), bytes.Repeat([]byte("c"), 100000)) // Type4Bytes

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying different types: %v", err)
	}
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 3 {
		t.Errorf("Expected 3 active records, got: %d", stats.ActiveRecords)
	}

	// Delete a large record
	db.Delete([]byte("type4"))

	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after delete: %v", err)
	}
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 1 {
		t.Errorf("Expected 1 deleted record, got: %d", stats.DeletedRecords)
	}
}

func TestCompact(t *testing.T) {
	testFile := "test_compact.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Compact empty file
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting empty file: %v", err)
	}

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}
	if stats.TotalRecords != 0 {
		t.Errorf("Expected 0 records after compacting empty file, got: %d", stats.TotalRecords)
	}

	// Test 2: Add records, delete some, then compact
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))
	db.Put([]byte("key4"), []byte("value4"))

	// Delete some keys
	db.Delete([]byte("key2"))
	db.Delete([]byte("key4"))

	// Verify before compact
	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying before compact: %v", err)
	}
	if stats.TotalRecords != 4 {
		t.Errorf("Expected 4 total records before compact, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records before compact, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 2 {
		t.Errorf("Expected 2 deleted records before compact, got: %d", stats.DeletedRecords)
	}

	// Compact
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Verify after compact
	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}
	if stats.TotalRecords != 2 {
		t.Errorf("Expected 2 total records after compact, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records after compact, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got: %d", stats.DeletedRecords)
	}

	// Verify that active keys still exist
	value1, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1 after compact: %v", err)
	}
	if !bytes.Equal(value1, []byte("value1")) {
		t.Errorf("Incorrect value for key1 after compact")
	}

	value3, err := db.Get([]byte("key3"))
	if err != nil {
		t.Fatalf("Error retrieving key3 after compact: %v", err)
	}
	if !bytes.Equal(value3, []byte("value3")) {
		t.Errorf("Incorrect value for key3 after compact")
	}

	// Verify that deleted keys don't exist
	_, err = db.Get([]byte("key2"))
	if err != ErrKeyNotFound {
		t.Errorf("key2 should not exist after compact")
	}

	_, err = db.Get([]byte("key4"))
	if err != ErrKeyNotFound {
		t.Errorf("key4 should not exist after compact")
	}
}

func TestCompactWithUpdates(t *testing.T) {
	testFile := "test_compact_updates.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add a key, update it multiple times
	db.Put([]byte("key1"), []byte("value1"))
	db.Update([]byte("key1"), []byte("value2"))
	db.Update([]byte("key1"), []byte("value3"))

	// Before compact, there should be 3 records (2 deleted + 1 current)
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying before compact: %v", err)
	}
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records before compact, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 1 {
		t.Errorf("Expected 1 active record before compact, got: %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 2 {
		t.Errorf("Expected 2 deleted records before compact, got: %d", stats.DeletedRecords)
	}

	// Compact
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// After compact, there should be only 1 record
	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}
	if stats.TotalRecords != 1 {
		t.Errorf("Expected 1 total record after compact, got: %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 1 {
		t.Errorf("Expected 1 active record after compact, got: %d", stats.ActiveRecords)
	}

	// Verify that the latest value is preserved
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1 after compact: %v", err)
	}
	if !bytes.Equal(value, []byte("value3")) {
		t.Errorf("Expected value3, got: %s", value)
	}
}

func TestCompactWithDifferentTypes(t *testing.T) {
	testFile := "test_compact_types.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add different data types
	data1 := bytes.Repeat([]byte("a"), 100)    // Type1Byte
	data2 := bytes.Repeat([]byte("b"), 1000)   // Type2Bytes
	data4 := bytes.Repeat([]byte("c"), 100000) // Type4Bytes

	db.Put([]byte("type1"), data1)
	db.Put([]byte("type2"), data2)
	db.Put([]byte("type4"), data4)

	// Delete one
	db.Delete([]byte("type2"))

	// Compact
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Verify
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Error verifying after compact: %v", err)
	}
	if stats.TotalRecords != 2 {
		t.Errorf("Expected 2 total records after compact, got: %d", stats.TotalRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got: %d", stats.DeletedRecords)
	}

	// Verify that remaining data is intact
	result1, err := db.Get([]byte("type1"))
	if err != nil {
		t.Fatalf("Error retrieving type1: %v", err)
	}
	if !bytes.Equal(result1, data1) {
		t.Error("Type1Byte data corrupted after compact")
	}

	result4, err := db.Get([]byte("type4"))
	if err != nil {
		t.Fatalf("Error retrieving type4: %v", err)
	}
	if !bytes.Equal(result4, data4) {
		t.Error("Type4Bytes data corrupted after compact")
	}

	// Verify deleted key doesn't exist
	_, err = db.Get([]byte("type2"))
	if err != ErrKeyNotFound {
		t.Errorf("type2 should not exist after compact")
	}
}

func TestKeys(t *testing.T) {
	testFile := "test_keys.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Empty database
	keys, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys from empty database: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty database, got: %d", len(keys))
	}

	// Test 2: Add some keys
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))

	keys, err = db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got: %d", len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[string(k)] = true
	}
	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Errorf("Not all keys are present: %v", keys)
	}

	// Test 3: Delete a key
	db.Delete([]byte("key2"))

	keys, err = db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys after delete: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after delete, got: %d", len(keys))
	}

	// Verify key2 is not in the list
	keyMap = make(map[string]bool)
	for _, k := range keys {
		keyMap[string(k)] = true
	}
	if !keyMap["key1"] || !keyMap["key3"] {
		t.Errorf("Expected key1 and key3, got: %v", keys)
	}
	if keyMap["key2"] {
		t.Errorf("key2 should not be in the list after deletion")
	}

	// Test 4: Update a key (should not create duplicates)
	db.Update([]byte("key1"), []byte("value1_updated"))

	keys, err = db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys after update: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after update, got: %d", len(keys))
	}

	// Test 5: Delete and re-add a key
	db.Delete([]byte("key3"))
	db.Put([]byte("key3"), []byte("value3_new"))

	keys, err = db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys after re-add: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after re-add, got: %d", len(keys))
	}

	keyMap = make(map[string]bool)
	for _, k := range keys {
		keyMap[string(k)] = true
	}
	if !keyMap["key1"] || !keyMap["key3"] {
		t.Errorf("Expected key1 and key3 after re-add, got: %v", keys)
	}
}

func TestKeysWithDifferentTypes(t *testing.T) {
	testFile := "test_keys_types.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add different data types
	data1 := bytes.Repeat([]byte("a"), 100)    // Type1Byte
	data2 := bytes.Repeat([]byte("b"), 1000)   // Type2Bytes
	data4 := bytes.Repeat([]byte("c"), 100000) // Type4Bytes

	db.Put([]byte("small"), data1)
	db.Put([]byte("medium"), data2)
	db.Put([]byte("large"), data4)

	keys, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got: %d", len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[string(k)] = true
	}
	if !keyMap["small"] || !keyMap["medium"] || !keyMap["large"] {
		t.Errorf("Not all keys are present: %v", keys)
	}
}

func TestKeysAfterCompact(t *testing.T) {
	testFile := "test_keys_compact.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add keys, update some, delete some
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))
	db.Update([]byte("key1"), []byte("value1_updated")) // Update
	db.Delete([]byte("key2"))                           // Delete

	// Keys before compact
	keysBefore, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys before compact: %v", err)
	}
	if len(keysBefore) != 2 {
		t.Errorf("Expected 2 keys before compact, got: %d", len(keysBefore))
	}

	// Compact
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Keys after compact
	keysAfter, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys after compact: %v", err)
	}
	if len(keysAfter) != 2 {
		t.Errorf("Expected 2 keys after compact, got: %d", len(keysAfter))
	}

	// Verify same keys are present
	keyMap := make(map[string]bool)
	for _, k := range keysAfter {
		keyMap[string(k)] = true
	}
	if !keyMap["key1"] || !keyMap["key3"] {
		t.Errorf("Expected key1 and key3 after compact, got: %v", keysAfter)
	}
	if keyMap["key2"] {
		t.Errorf("key2 should not be present after compact")
	}
}

func TestPutDuplicateKey(t *testing.T) {
	testFile := "test_put_duplicate.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add a key
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error inserting key1: %v", err)
	}

	// Try to add the same key again - should fail
	err = db.Put([]byte("key1"), []byte("value2"))
	if err != ErrKeyExists {
		t.Errorf("Expected ErrKeyExists when inserting duplicate key, got: %v", err)
	}

	// Verify original value is preserved
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1: %v", err)
	}
	if !bytes.Equal(value, []byte("value1")) {
		t.Errorf("Expected value1, got: %s", value)
	}
}

func TestUpdate(t *testing.T) {
	testFile := "test_update.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test 1: Update non-existent key - should fail
	err = db.Update([]byte("nonexistent"), []byte("value"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound when updating non-existent key, got: %v", err)
	}

	// Test 2: Add a key and update it
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error inserting key1: %v", err)
	}

	if err := db.Update([]byte("key1"), []byte("value2")); err != nil {
		t.Fatalf("Error updating key1: %v", err)
	}

	// Verify updated value
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1: %v", err)
	}
	if !bytes.Equal(value, []byte("value2")) {
		t.Errorf("Expected value2 after update, got: %s", value)
	}

	// Test 3: Update multiple times
	if err := db.Update([]byte("key1"), []byte("value3")); err != nil {
		t.Fatalf("Error updating key1 second time: %v", err)
	}

	value, err = db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1: %v", err)
	}
	if !bytes.Equal(value, []byte("value3")) {
		t.Errorf("Expected value3 after second update, got: %s", value)
	}

	// Test 4: Empty key - should fail
	err = db.Update([]byte{}, []byte("value"))
	if err == nil {
		t.Error("Should return error with empty key")
	}
}

func TestUpdateAfterDelete(t *testing.T) {
	testFile := "test_update_after_delete.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add a key
	if err := db.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Error inserting key1: %v", err)
	}

	// Delete the key
	if err := db.Delete([]byte("key1")); err != nil {
		t.Fatalf("Error deleting key1: %v", err)
	}

	// Try to update deleted key - should fail
	err = db.Update([]byte("key1"), []byte("value2"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound when updating deleted key, got: %v", err)
	}

	// Can still Put the key again
	if err := db.Put([]byte("key1"), []byte("value3")); err != nil {
		t.Fatalf("Error re-inserting key1 after delete: %v", err)
	}

	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error retrieving key1: %v", err)
	}
	if !bytes.Equal(value, []byte("value3")) {
		t.Errorf("Expected value3, got: %s", value)
	}
}

func TestCachePerformance(t *testing.T) {
	testFile := "test_cache_performance.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add 1000 keys
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("key%04d", i))
		value := []byte(fmt.Sprintf("value%04d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error inserting key %d: %v", i, err)
		}
	}

	// Test cache: Get should be very fast (no file scan)
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key%04d", i*10))
		expectedValue := []byte(fmt.Sprintf("value%04d", i*10))
		value, err := db.Get(key)
		if err != nil {
			t.Fatalf("Error retrieving key%04d: %v", i*10, err)
		}
		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Cache returned wrong value for key%04d", i*10)
		}
	}

	// Test Keys: should use cache
	keys, err := db.Keys()
	if err != nil {
		t.Fatalf("Error getting keys: %v", err)
	}
	if len(keys) != numKeys {
		t.Errorf("Expected %d keys, got: %d", numKeys, len(keys))
	}

	// Delete some keys and verify cache is updated
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%04d", i))
		if err := db.Delete(key); err != nil {
			t.Fatalf("Error deleting key%04d: %v", i, err)
		}
	}

	// Verify deleted keys are not in cache
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%04d", i))
		_, err := db.Get(key)
		if err != ErrKeyNotFound {
			t.Errorf("Deleted key%04d should not be in cache", i)
		}
	}

	// Verify cache size is correct
	keys, _ = db.Keys()
	if len(keys) != numKeys-10 {
		t.Errorf("Expected %d keys after deletion, got: %d", numKeys-10, len(keys))
	}
}

func TestCacheRebuildAfterCompact(t *testing.T) {
	testFile := "test_cache_rebuild.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add and update keys
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))
	db.Update([]byte("key1"), []byte("value1_v2"))
	db.Update([]byte("key2"), []byte("value2_v2"))
	db.Delete([]byte("key3"))

	// Compact (should rebuild cache)
	if err := db.Compact(); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Verify cache has correct data
	value1, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Error getting key1: %v", err)
	}
	if !bytes.Equal(value1, []byte("value1_v2")) {
		t.Errorf("Expected value1_v2, got: %s", value1)
	}

	value2, err := db.Get([]byte("key2"))
	if err != nil {
		t.Fatalf("Error getting key2: %v", err)
	}
	if !bytes.Equal(value2, []byte("value2_v2")) {
		t.Errorf("Expected value2_v2, got: %s", value2)
	}

	_, err = db.Get([]byte("key3"))
	if err != ErrKeyNotFound {
		t.Errorf("key3 should not exist after compact")
	}

	// Verify Keys uses updated cache
	keys, _ := db.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after compact, got: %d", len(keys))
	}
}
