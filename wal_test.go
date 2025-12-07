package skv

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWALBasicOperations(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Write some entries
	if err := wal.LogPut([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("Failed to log put: %v", err)
	}

	if err := wal.LogPut([]byte("key2"), []byte("value2")); err != nil {
		t.Fatalf("Failed to log put: %v", err)
	}

	if err := wal.LogDelete([]byte("key1")); err != nil {
		t.Fatalf("Failed to log delete: %v", err)
	}

	// Close and reopen to recover
	wal.Close()

	wal, err = OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	// Recover entries
	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Verify first entry (Put)
	if entries[0].OpType != WALOpPut {
		t.Errorf("Expected Put operation, got %d", entries[0].OpType)
	}
	if !bytes.Equal(entries[0].Key, []byte("key1")) {
		t.Errorf("Expected key 'key1', got %s", entries[0].Key)
	}
	if !bytes.Equal(entries[0].Data, []byte("value1")) {
		t.Errorf("Expected data 'value1', got %s", entries[0].Data)
	}

	// Verify second entry (Put)
	if entries[1].OpType != WALOpPut {
		t.Errorf("Expected Put operation, got %d", entries[1].OpType)
	}
	if !bytes.Equal(entries[1].Key, []byte("key2")) {
		t.Errorf("Expected key 'key2', got %s", entries[1].Key)
	}

	// Verify third entry (Delete)
	if entries[2].OpType != WALOpDelete {
		t.Errorf("Expected Delete operation, got %d", entries[2].OpType)
	}
	if !bytes.Equal(entries[2].Key, []byte("key1")) {
		t.Errorf("Expected key 'key1', got %s", entries[2].Key)
	}
}

func TestWALTruncate(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Write entries
	wal.LogPut([]byte("key1"), []byte("value1"))
	wal.LogPut([]byte("key2"), []byte("value2"))

	// Truncate
	if err := wal.Truncate(); err != nil {
		t.Fatalf("Failed to truncate: %v", err)
	}

	// Check size (should be just header)
	size, err := wal.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != WALHeaderSize {
		t.Errorf("Expected size %d after truncate, got %d", WALHeaderSize, size)
	}

	// Recover should return no entries
	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after truncate, got %d", len(entries))
	}
}

func TestWALCommitMarker(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Write entries with commit
	wal.LogPut([]byte("key1"), []byte("value1"))
	wal.LogCommit()
	wal.LogPut([]byte("key2"), []byte("value2")) // This should not be recovered

	// Recover should stop at commit
	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	// Should have Put + Commit (stops at commit, doesn't include entries after)
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries (up to commit), got %d", len(entries))
	}

	if entries[len(entries)-1].OpType != WALOpCommit {
		t.Errorf("Expected last entry to be commit")
	}
}

func TestWALDisableEnable(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Disable WAL
	wal.Disable()

	if wal.IsEnabled() {
		t.Error("Expected WAL to be disabled")
	}

	// Operations should be no-ops
	if err := wal.LogPut([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("LogPut should succeed even when disabled: %v", err)
	}

	// Size should still be just header
	size, err := wal.Size()
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != WALHeaderSize {
		t.Errorf("Expected size %d with disabled WAL, got %d", WALHeaderSize, size)
	}

	// Re-enable
	wal.Enable()

	if !wal.IsEnabled() {
		t.Error("Expected WAL to be enabled")
	}

	// Now operations should work
	wal.LogPut([]byte("key1"), []byte("value1"))

	size, _ = wal.Size()
	if size <= WALHeaderSize {
		t.Error("Expected WAL to have entries after re-enabling")
	}
}

func TestWALLargeData(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	if err := wal.LogPut([]byte("large"), largeData); err != nil {
		t.Fatalf("Failed to log large data: %v", err)
	}

	// Recover and verify
	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if !bytes.Equal(entries[0].Data, largeData) {
		t.Error("Large data mismatch")
	}
}

func TestWALCorruptedEntry(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	// Write valid entry
	wal.LogPut([]byte("key1"), []byte("value1"))

	// Close WAL
	wal.Close()

	// Corrupt the file by appending garbage
	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for corruption: %v", err)
	}
	file.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // Garbage data
	file.Close()

	// Reopen and try to recover
	wal, err = OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	// Should recover the first valid entry, stop at corruption
	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 valid entry before corruption, got %d", len(entries))
	}

	if !bytes.Equal(entries[0].Key, []byte("key1")) {
		t.Error("First entry should be valid")
	}
}

func TestWALEmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Empty key should work
	if err := wal.LogPut([]byte{}, []byte("value")); err != nil {
		t.Fatalf("Failed to log empty key: %v", err)
	}

	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries[0].Key) != 0 {
		t.Error("Expected empty key")
	}
}

func TestWALConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	// Note: This is not a true concurrent test (no goroutines)
	// Just verifies multiple sequential writes work correctly
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		val := []byte{byte(i * 2)}
		if err := wal.LogPut(key, val); err != nil {
			t.Fatalf("Failed to log entry %d: %v", i, err)
		}
	}

	entries, err := wal.Recover()
	if err != nil {
		t.Fatalf("Failed to recover: %v", err)
	}

	if len(entries) != 100 {
		t.Errorf("Expected 100 entries, got %d", len(entries))
	}
}

func TestWALWithSKV(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	// Create database and write data
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))

	db.Close()

	// Reopen - should recover cleanly (WAL should be empty after normal close)
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify data persisted
	val, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Failed to get key1: %v", err)
	}

	if !bytes.Equal(val, []byte("value1")) {
		t.Errorf("Expected 'value1', got %s", val)
	}

	// Check WAL is empty
	size, err := db.wal.Size()
	if err != nil {
		t.Fatalf("Failed to get WAL size: %v", err)
	}

	if size != WALHeaderSize {
		t.Errorf("Expected empty WAL (size %d), got %d", WALHeaderSize, size)
	}
}
