package skv

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestVerifyExtendedStats(t *testing.T) {
	dbFile := "test_verify_extended.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add some records with known sizes
	db.PutString("key1", "value1") // key=4, value=6
	db.PutString("key2", "value2") // key=4, value=6
	db.PutString("key3", "value3") // key=4, value=6

	// Verify initial stats
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Check basic counts
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 3 {
		t.Errorf("Expected 3 active records, got %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records, got %d", stats.DeletedRecords)
	}

	// Check file size
	if stats.FileSize <= 0 {
		t.Error("FileSize should be positive")
	}
	if stats.HeaderSize != HeaderSize {
		t.Errorf("Expected HeaderSize %d, got %d", HeaderSize, stats.HeaderSize)
	}

	// Check averages
	if stats.AverageKeySize != 4.0 {
		t.Errorf("Expected average key size 4.0, got %.2f", stats.AverageKeySize)
	}
	if stats.AverageDataSize != 6.0 {
		t.Errorf("Expected average data size 6.0, got %.2f", stats.AverageDataSize)
	}

	// Check efficiency (should be high with no deleted records)
	if stats.Efficiency < 90.0 {
		t.Errorf("Expected efficiency > 90%%, got %.2f%%", stats.Efficiency)
	}
	if stats.WastedPercent > 10.0 {
		t.Errorf("Expected wasted space < 10%%, got %.2f%%", stats.WastedPercent)
	}

	// Delete one record
	db.DeleteString("key2")

	// Verify after deletion
	stats, err = db.Verify()
	if err != nil {
		t.Fatalf("Verify after delete failed: %v", err)
	}

	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records after delete, got %d", stats.TotalRecords)
	}
	if stats.ActiveRecords != 2 {
		t.Errorf("Expected 2 active records after delete, got %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 1 {
		t.Errorf("Expected 1 deleted record, got %d", stats.DeletedRecords)
	}

	// Wasted space should now be positive
	if stats.WastedSpace <= 0 {
		t.Error("WastedSpace should be positive after deletion")
	}

	// Efficiency should decrease
	if stats.Efficiency >= 90.0 {
		t.Errorf("Expected efficiency to decrease after deletion, got %.2f%%", stats.Efficiency)
	}
}

func TestVerifyWithUpdates(t *testing.T) {
	dbFile := "test_verify_updates.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add and update records with progressively larger values to avoid reuse
	db.PutString("key1", "initial")
	db.UpdateString("key1", "updated_value_longer")
	db.UpdateString("key1", "final_value_even_much_longer")

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Should have 3 total records (1 active + 2 deleted old versions)
	// Note: depending on space reuse, some may be overwritten
	if stats.ActiveRecords != 1 {
		t.Errorf("Expected 1 active record, got %d", stats.ActiveRecords)
	}

	// Total should be at least 1 (may have more if updates created new records)
	if stats.TotalRecords < 1 {
		t.Errorf("Expected at least 1 total record, got %d", stats.TotalRecords)
	}

	// If there are deleted records, wasted space should be positive
	if stats.DeletedRecords > 0 && stats.WastedSpace <= 0 {
		t.Error("WastedSpace should be positive when there are deleted records")
	}
}

func TestVerifyAfterCompact(t *testing.T) {
	dbFile := "test_verify_compact.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add records
	for i := 0; i < 10; i++ {
		db.PutString("key"+string(rune('0'+i)), "value"+string(rune('0'+i)))
	}

	// Delete half of them
	for i := 0; i < 5; i++ {
		db.DeleteString("key" + string(rune('0'+i)))
	}

	// Verify before compact
	statsBefore, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify before compact failed: %v", err)
	}

	// Compact
	if err := db.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Verify after compact
	statsAfter, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify after compact failed: %v", err)
	}

	// After compact, should have no deleted records
	if statsAfter.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got %d", statsAfter.DeletedRecords)
	}

	// File should be smaller
	if statsAfter.FileSize >= statsBefore.FileSize {
		t.Errorf("Expected smaller file after compact: before=%d, after=%d",
			statsBefore.FileSize, statsAfter.FileSize)
	}

	// Wasted space should be minimal
	if statsAfter.WastedPercent > 5.0 {
		t.Errorf("Expected minimal waste after compact, got %.2f%%", statsAfter.WastedPercent)
	}

	// Efficiency should be very high
	if statsAfter.Efficiency < 95.0 {
		t.Errorf("Expected high efficiency after compact, got %.2f%%", statsAfter.Efficiency)
	}

	// Should still have 5 active records
	if statsAfter.ActiveRecords != 5 {
		t.Errorf("Expected 5 active records after compact, got %d", statsAfter.ActiveRecords)
	}
}

func TestVerifyEmptyDatabase(t *testing.T) {
	dbFile := "test_verify_empty.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Empty database should have only header
	if stats.TotalRecords != 0 {
		t.Errorf("Expected 0 total records, got %d", stats.TotalRecords)
	}
	if stats.FileSize != HeaderSize {
		t.Errorf("Expected file size %d, got %d", HeaderSize, stats.FileSize)
	}
	if stats.WastedSpace != 0 {
		t.Errorf("Expected 0 wasted space, got %d", stats.WastedSpace)
	}
}

func TestVerifyWithLargeValues(t *testing.T) {
	dbFile := "test_verify_large.skv"
	defer os.Remove(dbFile)

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add records with different sizes
	db.PutString("small", "x")                    // 1 byte
	db.Put([]byte("medium"), make([]byte, 1000))  // 1KB
	db.Put([]byte("large"), make([]byte, 100000)) // 100KB

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 records, got %d", stats.TotalRecords)
	}

	// Average data size should be skewed by the large value
	expectedAvg := (1.0 + 1000.0 + 100000.0) / 3.0
	if stats.AverageDataSize < expectedAvg-1 || stats.AverageDataSize > expectedAvg+1 {
		t.Errorf("Expected average data size ~%.1f, got %.1f", expectedAvg, stats.AverageDataSize)
	}

	// Average key size
	expectedKeyAvg := (5.0 + 6.0 + 5.0) / 3.0
	if stats.AverageKeySize < expectedKeyAvg-0.1 || stats.AverageKeySize > expectedKeyAvg+0.1 {
		t.Errorf("Expected average key size ~%.1f, got %.1f", expectedKeyAvg, stats.AverageKeySize)
	}
}

func TestFileHeader(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test_header_*.skv")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create a new database
	db, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	// Write some data
	if err := db.PutString("test", "value"); err != nil {
		t.Fatalf("Error putting data: %v", err)
	}

	db.Close()

	// Read the header manually
	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		t.Fatalf("Error reading header: %v", err)
	}

	// Verify magic bytes
	if string(header[0:3]) != HeaderMagic {
		t.Errorf("Expected magic bytes %q, got %q", HeaderMagic, string(header[0:3]))
	}

	// Verify version
	if header[3] != byte(VersionMajor) {
		t.Errorf("Expected major version %d, got %d", VersionMajor, header[3])
	}
	if header[4] != byte(VersionMinor) {
		t.Errorf("Expected minor version %d, got %d", VersionMinor, header[4])
	}
	if header[5] != byte(VersionPatch) {
		t.Errorf("Expected patch version %d, got %d", VersionPatch, header[5])
	}

	t.Logf("Header verified: %s version %d.%d.%d",
		string(header[0:3]), header[3], header[4], header[5])
}

func TestHeaderInNewFile(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test_new_header_*.skv")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Create a new database (should write header automatically)
	db, err := Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	db.Close()

	// Check that header was written
	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		t.Fatalf("Error getting file info: %v", err)
	}

	if info.Size() != HeaderSize {
		t.Errorf("New empty file should have size %d (header only), got %d", HeaderSize, info.Size())
	}

	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		t.Fatalf("Error reading header: %v", err)
	}

	if !bytes.Equal(header[0:3], []byte(HeaderMagic)) {
		t.Errorf("New file should have magic bytes, got: %v", header[0:3])
	}

	t.Log("New file correctly initialized with header")
}

func TestFreeSpaceReuse(t *testing.T) {
	// Test 1: Update with same size should reuse space
	t.Run("SameSize", func(t *testing.T) {
		testFile := "test_freespace_same.skv"
		os.Remove(testFile)
		defer os.Remove(testFile)

		db, err := Open(testFile)
		if err != nil {
			t.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()

		db.Put([]byte("key1"), []byte("value1"))

		_, _ = db.Verify()
		fileSize1, _ := db.file.Seek(0, 2) // Get file size

		// Update with same size
		db.Update([]byte("key1"), []byte("value2"))

		stats2, _ := db.Verify()
		fileSize2, _ := db.file.Seek(0, 2)

		// Should have 1 total record (space was reused)
		if stats2.TotalRecords != 1 {
			t.Errorf("Expected 1 total record after same-size update, got: %d", stats2.TotalRecords)
		}

		// File should not grow (or grow minimally due to padding)
		if fileSize2 > fileSize1+10 { // Allow small padding
			t.Errorf("File grew too much: before=%d, after=%d", fileSize1, fileSize2)
		}

		// Verify value
		value, _ := db.Get([]byte("key1"))
		if !bytes.Equal(value, []byte("value2")) {
			t.Errorf("Wrong value after reuse: got %s", value)
		}
	})

	// Test 2: Update with smaller size should reuse space with padding
	t.Run("SmallerSize", func(t *testing.T) {
		testFile := "test_freespace_smaller.skv"
		os.Remove(testFile)
		defer os.Remove(testFile)

		db, err := Open(testFile)
		if err != nil {
			t.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()

		db.Put([]byte("key2"), []byte("long-value-here"))

		fileSize1, _ := db.file.Seek(0, 2)

		// Update with smaller size
		db.Update([]byte("key2"), []byte("short"))

		fileSize2, _ := db.file.Seek(0, 2)

		// File should not grow
		if fileSize2 > fileSize1 {
			t.Errorf("File grew when updating to smaller value: before=%d, after=%d", fileSize1, fileSize2)
		}

		// Verify value
		value, _ := db.Get([]byte("key2"))
		if !bytes.Equal(value, []byte("short")) {
			t.Errorf("Wrong value: got %s", value)
		}
	})

	// Test 3: Update with larger size creates new record
	t.Run("LargerSize", func(t *testing.T) {
		testFile := "test_freespace_larger.skv"
		os.Remove(testFile)
		defer os.Remove(testFile)

		db, err := Open(testFile)
		if err != nil {
			t.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()

		db.Put([]byte("key3"), []byte("v"))

		stats1, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		// Update with much larger size
		db.Update([]byte("key3"), []byte("very-long-value-that-wont-fit"))

		stats2, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		// Should have 2 total records (old too small, new created)
		if stats2.TotalRecords != stats1.TotalRecords+1 {
			t.Errorf("Expected new record for larger update, total: %d", stats2.TotalRecords)
		}

		// Should have 1 deleted record
		if stats2.DeletedRecords != stats1.DeletedRecords+1 {
			t.Errorf("Expected 1 more deleted record, got: %d", stats2.DeletedRecords)
		}
	})

	// Test 4: Delete creates free space, next put should reuse it
	t.Run("DeleteAndReuse", func(t *testing.T) {
		testFile := "test_freespace_delete.skv"
		os.Remove(testFile)
		defer os.Remove(testFile)

		db, err := Open(testFile)
		if err != nil {
			t.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()

		db.Put([]byte("key4"), []byte("test-value"))

		stats1, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		fileSize1, _ := db.file.Seek(0, 2)

		// Delete the key
		db.Delete([]byte("key4"))

		// Add new key with similar size
		db.Put([]byte("key5"), []byte("test-value"))

		stats2, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		fileSize2, _ := db.file.Seek(0, 2)

		// Should have same or slightly more total records (reuse happened)
		if stats2.TotalRecords > stats1.TotalRecords+2 {
			t.Errorf("Too many records created, expected reuse")
		}

		// File should not grow much
		if fileSize2 > fileSize1+50 { // Allow some growth
			t.Errorf("File grew too much, expected reuse: before=%d, after=%d", fileSize1, fileSize2)
		}
	})
}

func TestFreeSpaceTracking(t *testing.T) {
	testFile := "test_freespace_tracking.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Add several keys
	for i := 0; i < 10; i++ {
		key := []byte{byte('a' + i)}
		value := bytes.Repeat([]byte("x"), 100)
		db.Put(key, value)
	}

	// Delete every other key
	for i := 0; i < 10; i += 2 {
		key := []byte{byte('a' + i)}
		db.Delete(key)
	}

	// Check that free space list has entries
	db.mu.RLock()
	freeSpaceCount := len(db.freeSpace)
	db.mu.RUnlock()

	if freeSpaceCount != 5 {
		t.Errorf("Expected 5 free space entries, got: %d", freeSpaceCount)
	}

	// Compact should clear free space list
	db.Compact()

	db.mu.RLock()
	freeSpaceCount = len(db.freeSpace)
	db.mu.RUnlock()

	if freeSpaceCount != 0 {
		t.Errorf("Expected 0 free space entries after compact, got: %d", freeSpaceCount)
	}
}

func TestRebuildCacheWithFreeSpace(t *testing.T) {
	testFile := "test_rebuild_freespace.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	// Add and delete some keys
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.Put([]byte("key3"), []byte("value3"))
	db.Delete([]byte("key2"))

	db.Close()

	// Reopen - should rebuild cache and free space list
	db, err = Open(testFile)
	if err != nil {
		t.Fatalf("Error reopening database: %v", err)
	}
	defer db.Close()

	// Check cache has 2 entries
	db.mu.RLock()
	cacheSize := len(db.cache)
	freeSpaceSize := len(db.freeSpace)
	db.mu.RUnlock()

	if cacheSize != 2 {
		t.Errorf("Expected 2 cache entries, got: %d", cacheSize)
	}

	if freeSpaceSize != 1 {
		t.Errorf("Expected 1 free space entry, got: %d", freeSpaceSize)
	}

	// Verify we can still get the keys
	value, err := db.Get([]byte("key1"))
	if err != nil || !bytes.Equal(value, []byte("value1")) {
		t.Errorf("Failed to get key1 after rebuild")
	}

	value, err = db.Get([]byte("key3"))
	if err != nil || !bytes.Equal(value, []byte("value3")) {
		t.Errorf("Failed to get key3 after rebuild")
	}

	_, err = db.Get([]byte("key2"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for deleted key2")
	}
}
