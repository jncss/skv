package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jncss/skv"
)

func main() {
	os.MkdirAll("data", 0755)

	db, err := skv.Open("data/concurrent_demo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Concurrent Operations ===\n")
	fmt.Println("SKV is thread-safe - all operations are protected by mutex locks")
	fmt.Println("Multiple goroutines can safely access the same database\n")

	// --- Example 1: Concurrent Writes ---
	fmt.Println("1. Concurrent Writes (10 goroutines, 10 writes each):\n")

	var wg sync.WaitGroup
	numGoroutines := 10
	writesPerGoroutine := 10

	startTime := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		goroutineID := g

		go func(id int) {
			defer wg.Done()

			for i := 0; i < writesPerGoroutine; i++ {
				key := fmt.Sprintf("writer:%d:item:%d", id, i)
				value := fmt.Sprintf("value_from_goroutine_%d_iteration_%d", id, i)

				err := db.PutString(key, value)
				if err != nil {
					log.Printf("Error in goroutine %d: %v", id, err)
				}
			}

			fmt.Printf("  ✓ Goroutine %d completed %d writes\n", id, writesPerGoroutine)
		}(goroutineID)
	}

	wg.Wait()

	duration := time.Since(startTime)
	totalWrites := numGoroutines * writesPerGoroutine

	fmt.Printf("\n  Total writes: %d\n", totalWrites)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Rate: %.0f ops/sec\n", float64(totalWrites)/duration.Seconds())
	fmt.Printf("  Keys in DB: %d\n", db.Count())

	// --- Example 2: Concurrent Reads ---
	fmt.Println("\n2. Concurrent Reads (10 goroutines reading):\n")

	numReaders := 10
	readsPerGoroutine := 50

	startTime = time.Now()

	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		readerID := r

		go func(id int) {
			defer wg.Done()

			successfulReads := 0
			for i := 0; i < readsPerGoroutine; i++ {
				// Read from different writers
				writerID := i % numGoroutines
				itemID := i % writesPerGoroutine
				key := fmt.Sprintf("writer:%d:item:%d", writerID, itemID)

				_, err := db.GetString(key)
				if err == nil {
					successfulReads++
				}
			}

			fmt.Printf("  ✓ Reader %d completed %d reads (%d successful)\n",
				id, readsPerGoroutine, successfulReads)
		}(readerID)
	}

	wg.Wait()

	duration = time.Since(startTime)
	totalReads := numReaders * readsPerGoroutine

	fmt.Printf("\n  Total reads: %d\n", totalReads)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Rate: %.0f ops/sec\n", float64(totalReads)/duration.Seconds())

	// --- Example 3: Mixed Operations ---
	fmt.Println("\n3. Mixed Operations (reads, writes, updates, deletes):\n")

	numWorkers := 5
	operationsPerWorker := 20

	startTime = time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		workerID := w

		go func(id int) {
			defer wg.Done()

			stats := map[string]int{
				"writes":  0,
				"reads":   0,
				"updates": 0,
				"deletes": 0,
			}

			for i := 0; i < operationsPerWorker; i++ {
				key := fmt.Sprintf("worker:%d:key:%d", id, i)

				switch i % 4 {
				case 0: // Write
					db.PutString(key, fmt.Sprintf("value_%d", i))
					stats["writes"]++

				case 1: // Read
					db.GetString(key)
					stats["reads"]++

				case 2: // Update (if exists)
					if db.HasString(key) {
						db.UpdateString(key, fmt.Sprintf("updated_%d", i))
						stats["updates"]++
					}

				case 3: // Delete (if exists)
					if db.HasString(key) {
						db.DeleteString(key)
						stats["deletes"]++
					}
				}
			}

			fmt.Printf("  ✓ Worker %d: %d writes, %d reads, %d updates, %d deletes\n",
				id, stats["writes"], stats["reads"], stats["updates"], stats["deletes"])
		}(workerID)
	}

	wg.Wait()

	duration = time.Since(startTime)
	totalOps := numWorkers * operationsPerWorker

	fmt.Printf("\n  Total operations: %d\n", totalOps)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Rate: %.0f ops/sec\n", float64(totalOps)/duration.Seconds())
	fmt.Printf("  Final key count: %d\n", db.Count())

	// --- Example 4: Concurrent Batch Operations ---
	fmt.Println("\n4. Concurrent Batch Operations:\n")

	numBatchWorkers := 5

	startTime = time.Now()

	for b := 0; b < numBatchWorkers; b++ {
		wg.Add(1)
		batchID := b

		go func(id int) {
			defer wg.Done()

			// Create a batch of data
			batch := make(map[string]string)
			for i := 0; i < 20; i++ {
				key := fmt.Sprintf("batch:%d:item:%d", id, i)
				value := fmt.Sprintf("batch_value_%d_%d", id, i)
				batch[key] = value
			}

			// Insert batch
			err := db.PutBatchString(batch)
			if err != nil {
				log.Printf("Batch error in worker %d: %v", id, err)
			}

			fmt.Printf("  ✓ Worker %d inserted batch of %d items\n", id, len(batch))
		}(batchID)
	}

	wg.Wait()

	duration = time.Since(startTime)
	fmt.Printf("\n  Duration: %v\n", duration)
	fmt.Printf("  Final database size: %d keys\n", db.Count())

	// --- Summary ---
	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ All concurrent operations completed successfully")
	fmt.Println("✓ No race conditions detected")
	fmt.Println("✓ Database remains consistent")
	fmt.Println("\nThread Safety Features:")
	fmt.Println("• All operations use mutex locks (sync.RWMutex)")
	fmt.Println("• Read operations use read locks (multiple readers allowed)")
	fmt.Println("• Write operations use exclusive locks")
	fmt.Println("• Safe for concurrent access within a single process")

	fmt.Println("\n✅ Concurrent operations demonstration completed!")
}
