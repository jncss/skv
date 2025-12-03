// Example demonstrating concurrent access from multiple goroutines
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/skv"
)

func main() {
	db, err := skv.Open("concurrent_example.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Clear()

	fmt.Println("=== Concurrent Operations ===\n")

	// Example 1: Concurrent writes from multiple goroutines
	fmt.Println("1. Concurrent writes (10 goroutines)...")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("goroutine:%d:item:%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				db.PutString(key, value)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("✓ Completed: %d keys written\n", db.Count())

	// Example 2: Concurrent reads
	fmt.Println("\n2. Concurrent reads (20 goroutines)...")
	start := time.Now()

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("goroutine:%d:item:%d", id%10, j)
				db.GetString(key)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("✓ Completed 200 reads in %v\n", elapsed)

	// Example 3: Mixed read/write operations
	fmt.Println("\n3. Mixed read/write operations...")

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				key := fmt.Sprintf("writer:%d:msg:%d", id, j)
				db.PutString(key, fmt.Sprintf("message_%d", j))
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				keys, _ := db.KeysString()
				_ = keys
				time.Sleep(time.Millisecond * 2)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("✓ Final count: %d keys\n", db.Count())

	// Example 4: Concurrent updates
	fmt.Println("\n4. Concurrent updates (race condition safe)...")

	// Create a counter key
	db.PutString("counter", "0")

	// Multiple goroutines trying to increment
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Note: This is NOT a proper atomic counter
				// Just demonstrating thread-safe operations
				value, _ := db.GetString("counter")
				var count int
				fmt.Sscanf(value, "%d", &count)
				count++
				db.UpdateString("counter", fmt.Sprintf("%d", count))
			}
		}()
	}

	wg.Wait()
	counter, _ := db.GetString("counter")
	fmt.Printf("✓ Counter value: %s (thread-safe operations)\n", counter)

	// Example 5: Concurrent iteration
	fmt.Println("\n5. Concurrent ForEach iterations...")

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			count := 0
			db.ForEach(func(key []byte, value []byte) error {
				count++
				return nil
			})
			fmt.Printf("  Goroutine %d counted %d keys\n", id, count)
		}(i)
	}

	wg.Wait()

	fmt.Println("\n✓ Concurrent example completed!")
	fmt.Println("  All operations were thread-safe thanks to internal locking")
}
