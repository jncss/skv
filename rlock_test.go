package skv

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// BenchmarkConcurrentReads benchmarks concurrent read operations
func BenchmarkConcurrentReads(b *testing.B) {
	dbPath := "bench_concurrent_reads.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Setup: create database with 1000 keys
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			b.Fatalf("failed to put: %v", err)
		}
	}

	b.ResetTimer()

	// Benchmark: concurrent reads
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := []byte(fmt.Sprintf("key:%d", i%1000))
			_, err := db.Get(key)
			if err != nil {
				b.Errorf("failed to get: %v", err)
			}
			i++
		}
	})

	b.StopTimer()
	db.Close()
}

// BenchmarkSequentialReads benchmarks sequential read operations
func BenchmarkSequentialReads(b *testing.B) {
	dbPath := "bench_sequential_reads.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Setup
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			b.Fatalf("failed to put: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := []byte(fmt.Sprintf("key:%d", i%1000))
		_, err := db.Get(key)
		if err != nil {
			b.Errorf("failed to get: %v", err)
		}
	}

	b.StopTimer()
	db.Close()
}

// BenchmarkKeysOperation benchmarks the Keys() operation with RLock
func BenchmarkKeysOperation(b *testing.B) {
	dbPath := "bench_keys.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Setup
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			b.Fatalf("failed to put: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := db.Keys()
		if err != nil {
			b.Errorf("failed to get keys: %v", err)
		}
	}

	b.StopTimer()
	db.Close()
}

// BenchmarkExistsOperation benchmarks the Exists() operation with RLock
func BenchmarkExistsOperation(b *testing.B) {
	dbPath := "bench_exists.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Setup
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			b.Fatalf("failed to put: %v", err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := []byte(fmt.Sprintf("key:%d", i%1000))
			_ = db.Exists(key)
			i++
		}
	})

	b.StopTimer()
	db.Close()
}

// BenchmarkCountOperation benchmarks the Count() operation with RLock
func BenchmarkCountOperation(b *testing.B) {
	dbPath := "bench_count.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Setup
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			b.Fatalf("failed to put: %v", err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = db.Count()
		}
	})

	b.StopTimer()
	db.Close()
}

// TestConcurrentKeysAndWrites tests Keys() with RLock during concurrent writes
func TestConcurrentKeysAndWrites(t *testing.T) {
	dbPath := "test_concurrent_keys.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Add initial keys
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatalf("failed to put: %v", err)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Start readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				keys, err := db.Keys()
				if err != nil {
					errors <- fmt.Errorf("Keys() error: %v", err)
					return
				}
				if len(keys) < 100 {
					errors <- fmt.Errorf("expected at least 100 keys, got %d", len(keys))
					return
				}
			}
		}()
	}

	// Start writers (they should not block readers)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := []byte(fmt.Sprintf("new_key:%d:%d", id, j))
				value := []byte(fmt.Sprintf("new_value:%d:%d", id, j))
				if err := db.Put(key, value); err != nil {
					errors <- fmt.Errorf("Put() error: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
