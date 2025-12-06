package skv

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

// TestStress10000Records tests intensive usage with 10,000 records
func TestStress10000Records(t *testing.T) {
	testFile := "test_stress_10k.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	const numRecords = 10000
	t.Logf("Starting stress test with %d records", numRecords)

	// Phase 1: Insert 10,000 records
	t.Run("Phase1_Insert10000", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < numRecords; i++ {
			key := []byte(fmt.Sprintf("key_%06d", i))
			value := []byte(fmt.Sprintf("value_%06d_%s", i, randomString(50)))

			if err := db.Put(key, value); err != nil {
				t.Fatalf("Error inserting record %d: %v", i, err)
			}

			if i > 0 && i%1000 == 0 {
				t.Logf("Inserted %d records", i)
			}
		}
		elapsed := time.Since(start)
		t.Logf("Inserted %d records in %v (%.0f records/sec)", numRecords, elapsed, float64(numRecords)/elapsed.Seconds())
	})

	// Phase 2: Verify all records exist and are correct
	t.Run("Phase2_VerifyAll", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < numRecords; i++ {
			key := []byte(fmt.Sprintf("key_%06d", i))
			value, err := db.Get(key)

			if err != nil {
				t.Errorf("Error getting record %d: %v", i, err)
				continue
			}

			expectedPrefix := []byte(fmt.Sprintf("value_%06d_", i))
			if !bytes.HasPrefix(value, expectedPrefix) {
				t.Errorf("Wrong value for key %d", i)
			}
		}
		elapsed := time.Since(start)
		t.Logf("Verified %d records in %v (%.0f reads/sec)", numRecords, elapsed, float64(numRecords)/elapsed.Seconds())
	})

	// Phase 3: Update 30% of records
	t.Run("Phase3_Update3000", func(t *testing.T) {
		start := time.Now()
		updateCount := numRecords * 3 / 10 // 30%
		for i := 0; i < updateCount; i++ {
			keyNum := rand.Intn(numRecords)
			key := []byte(fmt.Sprintf("key_%06d", keyNum))
			value := []byte(fmt.Sprintf("updated_value_%06d_%s", keyNum, randomString(60)))

			if err := db.Update(key, value); err != nil {
				t.Errorf("Error updating record %d: %v", keyNum, err)
			}

			if i > 0 && i%1000 == 0 {
				t.Logf("Updated %d records", i)
			}
		}
		elapsed := time.Since(start)
		t.Logf("Updated %d records in %v (%.0f updates/sec)", updateCount, elapsed, float64(updateCount)/elapsed.Seconds())
	})

	// Phase 4: Delete 20% of records
	t.Run("Phase4_Delete2000", func(t *testing.T) {
		start := time.Now()
		deleteCount := numRecords * 2 / 10 // 20%
		for i := 0; i < deleteCount; i++ {
			keyNum := rand.Intn(numRecords)
			key := []byte(fmt.Sprintf("key_%06d", keyNum))

			// Ignore ErrKeyNotFound (may have been deleted already)
			err := db.Delete(key)
			if err != nil && err != ErrKeyNotFound {
				t.Errorf("Error deleting record %d: %v", keyNum, err)
			}
		}
		elapsed := time.Since(start)
		t.Logf("Deleted up to %d records in %v", deleteCount, elapsed)
	})

	// Phase 5: Random mixed operations
	t.Run("Phase5_MixedOperations", func(t *testing.T) {
		start := time.Now()
		operations := 5000

		for i := 0; i < operations; i++ {
			keyNum := rand.Intn(numRecords)
			key := []byte(fmt.Sprintf("key_%06d", keyNum))

			op := rand.Intn(100)
			switch {
			case op < 50: // 50% reads
				db.Get(key)
			case op < 75: // 25% updates
				value := []byte(fmt.Sprintf("mixed_value_%06d_%s", keyNum, randomString(40)))
				db.Update(key, value)
			case op < 90: // 15% puts
				value := []byte(fmt.Sprintf("new_value_%06d_%s", keyNum, randomString(40)))
				db.Put(key, value)
			default: // 10% deletes
				db.Delete(key)
			}
		}
		elapsed := time.Since(start)
		t.Logf("Completed %d mixed operations in %v (%.0f ops/sec)", operations, elapsed, float64(operations)/elapsed.Seconds())
	})

	// Phase 6: Verify integrity
	t.Run("Phase6_VerifyIntegrity", func(t *testing.T) {
		stats, err := db.Verify()
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		t.Logf("Database stats:")
		t.Logf("  Total records: %d", stats.TotalRecords)
		t.Logf("  Active records: %d", stats.ActiveRecords)
		t.Logf("  Deleted records: %d", stats.DeletedRecords)

		if stats.ActiveRecords < 1000 {
			t.Errorf("Too few active records: %d", stats.ActiveRecords)
		}
	})

	// Phase 7: Test compaction
	t.Run("Phase7_Compact", func(t *testing.T) {
		fileInfo, _ := os.Stat(testFile)
		sizeBefore := fileInfo.Size()

		start := time.Now()
		if err := db.Compact(); err != nil {
			t.Fatalf("Compact failed: %v", err)
		}
		elapsed := time.Since(start)

		fileInfo, _ = os.Stat(testFile)
		sizeAfter := fileInfo.Size()

		reduction := float64(sizeBefore-sizeAfter) / float64(sizeBefore) * 100
		t.Logf("Compaction completed in %v", elapsed)
		t.Logf("  Size before: %d bytes", sizeBefore)
		t.Logf("  Size after: %d bytes", sizeAfter)
		t.Logf("  Reduction: %.1f%%", reduction)

		// Verify data is still accessible after compaction
		stats, _ := db.Verify()
		t.Logf("  Active records after compact: %d", stats.ActiveRecords)
		t.Logf("  Deleted records after compact: %d", stats.DeletedRecords)

		if stats.DeletedRecords > 0 {
			t.Errorf("Should have 0 deleted records after compact, got %d", stats.DeletedRecords)
		}
	})

	// Phase 8: Re-verify all active records after compact
	t.Run("Phase8_VerifyAfterCompact", func(t *testing.T) {
		keys, err := db.Keys()
		if err != nil {
			t.Fatalf("Error getting keys: %v", err)
		}

		t.Logf("Verifying %d active keys after compaction", len(keys))

		for i, key := range keys {
			value, err := db.Get(key)
			if err != nil {
				t.Errorf("Error reading key %s: %v", key, err)
				continue
			}
			if len(value) == 0 {
				t.Errorf("Empty value for key %s", key)
			}

			if i > 0 && i%1000 == 0 {
				t.Logf("Verified %d keys", i)
			}
		}
	})
}

// TestStressConcurrent tests concurrent access with multiple goroutines
func TestStressConcurrent(t *testing.T) {
	testFile := "test_stress_concurrent.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	const numGoroutines = 10
	const opsPerGoroutine = 1000

	t.Logf("Starting concurrent stress test: %d goroutines, %d ops each", numGoroutines, opsPerGoroutine)

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*opsPerGoroutine)

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < opsPerGoroutine; i++ {
				key := []byte(fmt.Sprintf("g%d_key_%d", goroutineID, i))
				value := []byte(fmt.Sprintf("g%d_value_%d_%s", goroutineID, i, randomString(30)))

				// Random operation
				op := rand.Intn(100)
				var err error

				switch {
				case op < 40: // 40% Put
					err = db.Put(key, value)
					if err != nil && err != ErrKeyExists {
						errors <- fmt.Errorf("goroutine %d: Put error: %v", goroutineID, err)
					}
				case op < 70: // 30% Get
					_, err = db.Get(key)
					if err != nil && err != ErrKeyNotFound {
						errors <- fmt.Errorf("goroutine %d: Get error: %v", goroutineID, err)
					}
				case op < 90: // 20% Update
					err = db.Update(key, value)
					if err != nil && err != ErrKeyNotFound {
						errors <- fmt.Errorf("goroutine %d: Update error: %v", goroutineID, err)
					}
				default: // 10% Delete
					err = db.Delete(key)
					if err != nil && err != ErrKeyNotFound {
						errors <- fmt.Errorf("goroutine %d: Delete error: %v", goroutineID, err)
					}
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)
	totalOps := numGoroutines * opsPerGoroutine

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			t.Error("Too many errors, stopping error reporting")
			break
		}
	}

	if errorCount > 0 {
		t.Fatalf("Concurrent test had %d errors", errorCount)
	}

	t.Logf("Completed %d concurrent operations in %v (%.0f ops/sec)", totalOps, elapsed, float64(totalOps)/elapsed.Seconds())

	// Verify final state
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	t.Logf("Final stats:")
	t.Logf("  Total records: %d", stats.TotalRecords)
	t.Logf("  Active records: %d", stats.ActiveRecords)
	t.Logf("  Deleted records: %d", stats.DeletedRecords)
}

// TestStressLargeValues tests with larger values
func TestStressLargeValues(t *testing.T) {
	testFile := "test_stress_large.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	const numRecords = 1000
	valueSizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	t.Logf("Testing with large values up to %d bytes", valueSizes[len(valueSizes)-1])

	start := time.Now()

	for i := 0; i < numRecords; i++ {
		sizeIdx := i % len(valueSizes)
		valueSize := valueSizes[sizeIdx]

		key := []byte(fmt.Sprintf("large_key_%04d", i))
		value := make([]byte, valueSize)
		for j := range value {
			value[j] = byte(j % 256)
		}

		if err := db.Put(key, value); err != nil {
			t.Fatalf("Error inserting large record %d: %v", i, err)
		}

		if (i+1)%100 == 0 {
			t.Logf("Inserted %d records", i+1)
		}
	}

	elapsed := time.Since(start)
	t.Logf("Inserted %d large records in %v", numRecords, elapsed)

	// Verify some records
	for i := 0; i < 100; i++ {
		keyNum := rand.Intn(numRecords)
		key := []byte(fmt.Sprintf("large_key_%04d", keyNum))

		value, err := db.Get(key)
		if err != nil {
			t.Errorf("Error getting large record %d: %v", keyNum, err)
			continue
		}

		sizeIdx := keyNum % len(valueSizes)
		expectedSize := valueSizes[sizeIdx]

		if len(value) != expectedSize {
			t.Errorf("Wrong size for record %d: expected %d, got %d", keyNum, expectedSize, len(value))
		}
	}

	t.Log("Large value verification completed")
}

// TestStressReopenAndRecover tests reopening database multiple times
func TestStressReopenAndRecover(t *testing.T) {
	testFile := "test_stress_reopen.skv"
	os.Remove(testFile)
	defer os.Remove(testFile)

	const cycles = 5
	const recordsPerCycle = 1000

	t.Logf("Testing database reopen/recovery: %d cycles", cycles)

	for cycle := 0; cycle < cycles; cycle++ {
		db, err := Open(testFile)
		if err != nil {
			t.Fatalf("Cycle %d: Error opening database: %v", cycle, err)
		}

		// Add records
		for i := 0; i < recordsPerCycle; i++ {
			key := []byte(fmt.Sprintf("cycle%d_key_%d", cycle, i))
			value := []byte(fmt.Sprintf("cycle%d_value_%d", cycle, i))

			err := db.Put(key, value)
			if err != nil && err != ErrKeyExists {
				t.Fatalf("Cycle %d: Error putting record: %v", cycle, err)
			}
		}

		// Verify some records from previous cycles
		if cycle > 0 {
			prevCycle := rand.Intn(cycle)
			key := []byte(fmt.Sprintf("cycle%d_key_0", prevCycle))
			value, err := db.Get(key)
			if err != nil {
				t.Errorf("Cycle %d: Could not read from previous cycle %d: %v", cycle, prevCycle, err)
			} else {
				expected := []byte(fmt.Sprintf("cycle%d_value_0", prevCycle))
				if !bytes.Equal(value, expected) {
					t.Errorf("Cycle %d: Wrong value from previous cycle", cycle)
				}
			}
		}

		db.Close()
		t.Logf("Completed cycle %d", cycle+1)
	}

	// Final verification
	db, err := Open(testFile)
	if err != nil {
		t.Fatalf("Error reopening for final check: %v", err)
	}
	defer db.Close()

	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Final verify failed: %v", err)
	}

	expectedMin := cycles * recordsPerCycle
	if stats.ActiveRecords < expectedMin {
		t.Errorf("Expected at least %d active records, got %d", expectedMin, stats.ActiveRecords)
	}

	t.Logf("Final stats: %d total, %d active, %d deleted", stats.TotalRecords, stats.ActiveRecords, stats.DeletedRecords)
}

// Helper function to generate random strings
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
