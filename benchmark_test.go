package skv

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkPutWithWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	value := make([]byte, 100) // 100 bytes value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		if err := db.Put(keyWithID, value); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}

func BenchmarkPutWithoutWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Disable WAL
	db.wal.Disable()

	value := make([]byte, 100) // 100 bytes value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		if err := db.Put(keyWithID, value); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}

func BenchmarkGetCached(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Populate with data
	key := []byte("benchmark_key")
	value := []byte("benchmark_value")
	db.Put(key, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Get(key); err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkUpdateWithWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Populate with data
	for i := 0; i < 100; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		db.Put(keyWithID, []byte("initial"))
	}

	value := make([]byte, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i%100))
		if err := db.Update(keyWithID, value); err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

func BenchmarkUpdateWithoutWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Disable WAL
	db.wal.Disable()

	// Populate with data
	for i := 0; i < 100; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		db.Put(keyWithID, []byte("initial"))
	}

	value := make([]byte, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i%100))
		if err := db.Update(keyWithID, value); err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

func BenchmarkDeleteWithWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Populate with data
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		db.Put(keyWithID, []byte("value"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		if err := db.Delete(keyWithID); err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}

func BenchmarkDeleteWithoutWAL(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.skv")

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Disable WAL
	db.wal.Disable()

	// Populate with data
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		db.Put(keyWithID, []byte("value"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyWithID := []byte(fmt.Sprintf("key_%d", i))
		if err := db.Delete(keyWithID); err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}

// Performance comparison test
func TestPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()

	// Test with WAL
	dbPath1 := filepath.Join(tmpDir, "wal.skv")
	db1, _ := Open(dbPath1)

	t1 := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := []byte(fmt.Sprintf("key_%d", i))
			db1.Put(key, []byte("value"))
		}
	})
	db1.Close()
	os.Remove(dbPath1)
	os.Remove(dbPath1 + ".wal")

	// Test without WAL
	dbPath2 := filepath.Join(tmpDir, "nowal.skv")
	db2, _ := Open(dbPath2)
	db2.wal.Disable()

	t2 := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := []byte(fmt.Sprintf("key_%d", i))
			db2.Put(key, []byte("value"))
		}
	})
	db2.Close()
	os.Remove(dbPath2)
	os.Remove(dbPath2 + ".wal")

	opsWithWAL := float64(t1.N) / t1.T.Seconds()
	opsWithoutWAL := float64(t2.N) / t2.T.Seconds()
	overhead := ((opsWithoutWAL - opsWithWAL) / opsWithoutWAL) * 100

	t.Logf("Performance Comparison:")
	t.Logf("  With WAL:    %.0f ops/sec", opsWithWAL)
	t.Logf("  Without WAL: %.0f ops/sec", opsWithoutWAL)
	t.Logf("  WAL Overhead: %.1f%%", overhead)
}
