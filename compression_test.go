package skv

import (
	"bytes"
	"os"
	"testing"
)

func TestCompressionNone(t *testing.T) {
	data := []byte("hello world")
	compressed, compressionType, err := compress(data, CompressionNone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionNone {
		t.Errorf("expected CompressionNone, got %v", compressionType)
	}
	if !bytes.Equal(compressed, data) {
		t.Errorf("expected data unchanged, got different data")
	}
}

func TestCompressionSmallData(t *testing.T) {
	// Data below threshold should not be compressed
	data := make([]byte, CompressionThreshold-1)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Test with Snappy
	compressed, compressionType, err := compress(data, CompressionSnappy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionNone {
		t.Errorf("small data should not be compressed, got %v", compressionType)
	}
	if !bytes.Equal(compressed, data) {
		t.Errorf("expected data unchanged")
	}

	// Test with LZ4
	compressed, compressionType, err = compress(data, CompressionLZ4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionNone {
		t.Errorf("small data should not be compressed, got %v", compressionType)
	}
	if !bytes.Equal(compressed, data) {
		t.Errorf("expected data unchanged")
	}
}

func TestCompressionSnappy(t *testing.T) {
	// Create compressible data (repeating pattern)
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 10) // Highly compressible pattern
	}

	compressed, compressionType, err := compress(data, CompressionSnappy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionSnappy {
		t.Errorf("expected CompressionSnappy, got %v", compressionType)
	}
	if len(compressed) >= len(data) {
		t.Errorf("compressed data should be smaller than original: %d >= %d", len(compressed), len(data))
	}

	// Decompress and verify
	decompressed, err := decompress(compressed, CompressionSnappy, len(data))
	if err != nil {
		t.Fatalf("decompression error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestCompressionLZ4(t *testing.T) {
	// Create compressible data (repeating pattern)
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 10) // Highly compressible pattern
	}

	compressed, compressionType, err := compress(data, CompressionLZ4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionLZ4 {
		t.Errorf("expected CompressionLZ4, got %v", compressionType)
	}
	if len(compressed) >= len(data) {
		t.Errorf("compressed data should be smaller than original: %d >= %d", len(compressed), len(data))
	}

	// Decompress and verify
	decompressed, err := decompress(compressed, CompressionLZ4, len(data))
	if err != nil {
		t.Fatalf("decompression error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestCompressionIncompressible(t *testing.T) {
	// Create random-like data (not compressible)
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	// Snappy should return original data if compression doesn't help
	_, compressionType, err := compress(data, CompressionSnappy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fallback to CompressionNone if compression doesn't reduce size
	if compressionType != CompressionNone && compressionType != CompressionSnappy {
		t.Errorf("expected CompressionNone or CompressionSnappy, got %v", compressionType)
	}

	// LZ4 similar behavior
	_, compressionType, err = compress(data, CompressionLZ4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressionType != CompressionNone && compressionType != CompressionLZ4 {
		t.Errorf("expected CompressionNone or CompressionLZ4, got %v", compressionType)
	}
}

func TestSKVCompressionSnappy(t *testing.T) {
	dbPath := "test_compression_snappy.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Open with Snappy compression
	db, err := OpenWithOptions(dbPath, &Options{
		Compression: CompressionSnappy,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create compressible data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 10)
	}

	// Put data
	key := []byte("test_key")
	if err := db.Put(key, data); err != nil {
		t.Fatalf("failed to put data: %v", err)
	}

	// Get data and verify
	retrieved, err := db.Get(key)
	if err != nil {
		t.Fatalf("failed to get data: %v", err)
	}
	if !bytes.Equal(retrieved, data) {
		t.Errorf("retrieved data doesn't match original")
	}

	// Close and reopen to test persistence
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify data after reopen
	retrieved, err = db.Get(key)
	if err != nil {
		t.Fatalf("failed to get data after reopen: %v", err)
	}
	if !bytes.Equal(retrieved, data) {
		t.Errorf("retrieved data doesn't match original after reopen")
	}
}

func TestSKVCompressionLZ4(t *testing.T) {
	dbPath := "test_compression_lz4.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Open with LZ4 compression
	db, err := OpenWithOptions(dbPath, &Options{
		Compression: CompressionLZ4,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create compressible data
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 20)
	}

	// Put data
	key := []byte("test_key")
	if err := db.Put(key, data); err != nil {
		t.Fatalf("failed to put data: %v", err)
	}

	// Get data and verify
	retrieved, err := db.Get(key)
	if err != nil {
		t.Fatalf("failed to get data: %v", err)
	}
	if !bytes.Equal(retrieved, data) {
		t.Errorf("retrieved data doesn't match original")
	}

	// Close and reopen to test persistence
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify data after reopen
	retrieved, err = db.Get(key)
	if err != nil {
		t.Fatalf("failed to get data after reopen: %v", err)
	}
	if !bytes.Equal(retrieved, data) {
		t.Errorf("retrieved data doesn't match original after reopen")
	}
}

func TestSKVCompressionMixed(t *testing.T) {
	dbPath := "test_compression_mixed.skv"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	// Create database with compression
	db, err := OpenWithOptions(dbPath, &Options{
		Compression: CompressionSnappy,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Insert data of various sizes
	testCases := []struct {
		key  string
		size int
	}{
		{"small", 50},    // Below threshold, won't compress
		{"medium", 512},  // Above threshold, compressible
		{"large", 10240}, // Large and compressible
		{"random", 256},  // Above threshold, less compressible
	}

	for _, tc := range testCases {
		data := make([]byte, tc.size)
		for i := range data {
			if tc.key == "random" {
				data[i] = byte(i) // Less compressible
			} else {
				data[i] = byte(i % 10) // Highly compressible
			}
		}

		if err := db.Put([]byte(tc.key), data); err != nil {
			t.Fatalf("failed to put %s: %v", tc.key, err)
		}

		// Verify immediate retrieval
		retrieved, err := db.Get([]byte(tc.key))
		if err != nil {
			t.Fatalf("failed to get %s: %v", tc.key, err)
		}
		if !bytes.Equal(retrieved, data) {
			t.Errorf("data mismatch for %s", tc.key)
		}
	}

	// Close and reopen
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db.Close()

	// Verify all data after reopen
	for _, tc := range testCases {
		data := make([]byte, tc.size)
		for i := range data {
			if tc.key == "random" {
				data[i] = byte(i)
			} else {
				data[i] = byte(i % 10)
			}
		}

		retrieved, err := db.Get([]byte(tc.key))
		if err != nil {
			t.Fatalf("failed to get %s after reopen: %v", tc.key, err)
		}
		if !bytes.Equal(retrieved, data) {
			t.Errorf("data mismatch for %s after reopen", tc.key)
		}
	}
}

func TestCompressionFlags(t *testing.T) {
	tests := []struct {
		name             string
		recordType       byte
		expectedCompType CompressionType
	}{
		{"None", Type1Byte, CompressionNone},
		{"Snappy", Type1Byte | CompressedSnappy, CompressionSnappy},
		{"LZ4", Type2Bytes | CompressedLZ4, CompressionLZ4},
		{"Deleted+Snappy", Type4Bytes | DeletedFlag | CompressedSnappy, CompressionSnappy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compType := getCompressionType(tt.recordType)
			if compType != tt.expectedCompType {
				t.Errorf("expected %v, got %v", tt.expectedCompType, compType)
			}

			// Test setCompressionType
			baseType := getBaseType(tt.recordType)
			newType := setCompressionType(baseType, tt.expectedCompType)
			if getCompressionType(newType) != tt.expectedCompType {
				t.Errorf("setCompressionType failed to set correct type")
			}
		})
	}
}
