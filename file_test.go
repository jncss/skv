package skv

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPutFile(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	testData := []byte("Hello, World! This is test data.")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Open database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test PutFile
	err = db.PutFile("testkey", testFile)
	if err != nil {
		t.Fatalf("PutFile failed: %v", err)
	}

	// Verify data was stored correctly
	value, err := db.Get([]byte("testkey"))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !bytes.Equal(value, testData) {
		t.Errorf("Expected %q, got %q", testData, value)
	}
}

func TestPutFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to put non-existent file
	err = db.PutFile("key", "/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestPutFileDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	testFile := filepath.Join(tmpDir, "test.txt")

	testData := []byte("test data")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// First put should succeed
	err = db.PutFile("key", testFile)
	if err != nil {
		t.Fatalf("First PutFile failed: %v", err)
	}

	// Second put should fail (duplicate key)
	err = db.PutFile("key", testFile)
	if err != ErrKeyExists {
		t.Errorf("Expected ErrKeyExists, got %v", err)
	}
}

func TestGetFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	outputFile := filepath.Join(tmpDir, "output.txt")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store data
	testData := []byte("Test data for GetFile")
	err = db.Put([]byte("filekey"), testData)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Get file
	err = db.GetFile("filekey", outputFile)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Verify file contents
	readData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("Expected %q, got %q", testData, readData)
	}
}

func TestGetFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	outputFile := filepath.Join(tmpDir, "output.txt")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to get non-existent key
	err = db.GetFile("nonexistent", outputFile)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestGetFileInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store data
	err = db.Put([]byte("key"), []byte("data"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Try to write to invalid path
	err = db.GetFile("key", "/nonexistent/directory/file.txt")
	if err == nil {
		t.Error("Expected error for invalid file path, got nil")
	}
}

func TestUpdateFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	testFile := filepath.Join(tmpDir, "update.txt")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create initial key
	err = db.Put([]byte("updatekey"), []byte("initial data"))
	if err != nil {
		t.Fatalf("Failed to put initial data: %v", err)
	}

	// Create file with new data
	newData := []byte("Updated data from file")
	err = os.WriteFile(testFile, newData, 0644)
	if err != nil {
		t.Fatalf("Failed to create update file: %v", err)
	}

	// Update with file
	err = db.UpdateFile("updatekey", testFile)
	if err != nil {
		t.Fatalf("UpdateFile failed: %v", err)
	}

	// Verify update
	value, err := db.Get([]byte("updatekey"))
	if err != nil {
		t.Fatalf("Failed to get updated value: %v", err)
	}

	if !bytes.Equal(value, newData) {
		t.Errorf("Expected %q, got %q", newData, value)
	}
}

func TestUpdateFileNonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	testFile := filepath.Join(tmpDir, "test.txt")

	testData := []byte("test data")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to update non-existent key
	err = db.UpdateFile("nonexistent", testFile)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestUpdateFileNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create key
	err = db.Put([]byte("key"), []byte("data"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	// Try to update with non-existent file
	err = db.UpdateFile("key", "/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestFileOperationsBinary(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	inputFile := filepath.Join(tmpDir, "binary.dat")
	outputFile := filepath.Join(tmpDir, "output.dat")

	// Create binary test data
	binaryData := make([]byte, 1000)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	err := os.WriteFile(inputFile, binaryData, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put binary file
	err = db.PutFile("binary", inputFile)
	if err != nil {
		t.Fatalf("PutFile failed: %v", err)
	}

	// Get binary file
	err = db.GetFile("binary", outputFile)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Verify binary data
	readData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.Equal(readData, binaryData) {
		t.Error("Binary data mismatch")
	}
}

func TestFileOperationsLarge(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	largeFile := filepath.Join(tmpDir, "large.dat")
	outputFile := filepath.Join(tmpDir, "large_out.dat")

	// Create large file (1 MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := os.WriteFile(largeFile, largeData, 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put large file
	err = db.PutFile("largefile", largeFile)
	if err != nil {
		t.Fatalf("PutFile failed: %v", err)
	}

	// Get large file
	err = db.GetFile("largefile", outputFile)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Verify size
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if info.Size() != int64(len(largeData)) {
		t.Errorf("Expected size %d, got %d", len(largeData), info.Size())
	}
}

func TestGetStream(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store test data
	testData := []byte("Hello, this is streaming test data!")
	err = db.Put([]byte("streamkey"), testData)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Stream to buffer
	var buf bytes.Buffer
	n, err := db.GetStream([]byte("streamkey"), &buf)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}

	if !bytes.Equal(buf.Bytes(), testData) {
		t.Errorf("Expected %q, got %q", testData, buf.Bytes())
	}
}

func TestGetStreamString(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store test data
	testData := "Hello, streaming world!"
	err = db.PutString("key", testData)
	if err != nil {
		t.Fatalf("PutString failed: %v", err)
	}

	// Stream to buffer
	var buf bytes.Buffer
	n, err := db.GetStreamString("key", &buf)
	if err != nil {
		t.Fatalf("GetStreamString failed: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}

	if buf.String() != testData {
		t.Errorf("Expected %q, got %q", testData, buf.String())
	}
}

func TestGetStreamLargeData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create 5MB of test data
	largeData := make([]byte, 5*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Store large data
	err = db.Put([]byte("large"), largeData)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Stream to buffer
	var buf bytes.Buffer
	n, err := db.GetStream([]byte("large"), &buf)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if n != int64(len(largeData)) {
		t.Errorf("Expected %d bytes written, got %d", len(largeData), n)
	}

	if !bytes.Equal(buf.Bytes(), largeData) {
		t.Errorf("Data mismatch in large stream")
	}
}

func TestGetStreamToFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	outputFile := filepath.Join(tmpDir, "output.dat")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Store test data
	testData := []byte("This will be streamed to a file!")
	err = db.Put([]byte("filestream"), testData)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Stream to file
	file, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	n, err := db.GetStream([]byte("filestream"), file)
	file.Close()
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}

	// Verify file contents
	readData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("Expected %q, got %q", testData, readData)
	}
}

func TestGetStreamNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var buf bytes.Buffer
	_, err = db.GetStream([]byte("nonexistent"), &buf)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestPutStream(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test data
	testData := []byte("Hello, this is streaming test data for PutStream!")
	reader := bytes.NewReader(testData)

	// Store using PutStream
	err = db.PutStream([]byte("streamkey"), reader, int64(len(testData)))
	if err != nil {
		t.Fatalf("PutStream failed: %v", err)
	}

	// Verify data was stored correctly
	value, err := db.Get([]byte("streamkey"))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !bytes.Equal(value, testData) {
		t.Errorf("Expected %q, got %q", testData, value)
	}
}

func TestPutStreamString(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test data
	testData := "Streaming string data!"
	reader := bytes.NewReader([]byte(testData))

	// Store using PutStreamString
	err = db.PutStreamString("key", reader, int64(len(testData)))
	if err != nil {
		t.Fatalf("PutStreamString failed: %v", err)
	}

	// Verify
	value, err := db.GetString("key")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if value != testData {
		t.Errorf("Expected %q, got %q", testData, value)
	}
}

func TestPutStreamLargeData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create 5MB of test data
	largeData := make([]byte, 5*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	reader := bytes.NewReader(largeData)

	// Store using PutStream
	err = db.PutStream([]byte("large"), reader, int64(len(largeData)))
	if err != nil {
		t.Fatalf("PutStream failed: %v", err)
	}

	// Verify
	value, err := db.Get([]byte("large"))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !bytes.Equal(value, largeData) {
		t.Errorf("Data mismatch in large stream")
	}
}

func TestPutStreamFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")
	testFile := filepath.Join(tmpDir, "input.dat")

	// Create test file
	testData := []byte("This is file data to be streamed!")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Open file and stream it
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}

	info, _ := file.Stat()
	err = db.PutStream([]byte("filedata"), file, info.Size())
	file.Close()

	if err != nil {
		t.Fatalf("PutStream failed: %v", err)
	}

	// Verify
	value, err := db.Get([]byte("filedata"))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !bytes.Equal(value, testData) {
		t.Errorf("Expected %q, got %q", testData, value)
	}
}

func TestPutStreamKeyExists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	err = db.Put([]byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Try to PutStream with same key
	reader := bytes.NewReader([]byte("new data"))
	err = db.PutStream([]byte("key"), reader, 8)
	if err != ErrKeyExists {
		t.Errorf("Expected ErrKeyExists, got %v", err)
	}
}

func TestUpdateStream(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	err = db.Put([]byte("key"), []byte("initial value"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Update using UpdateStream
	newData := []byte("Updated value via stream!")
	reader := bytes.NewReader(newData)
	err = db.UpdateStream([]byte("key"), reader, int64(len(newData)))
	if err != nil {
		t.Fatalf("UpdateStream failed: %v", err)
	}

	// Verify
	value, err := db.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if !bytes.Equal(value, newData) {
		t.Errorf("Expected %q, got %q", newData, value)
	}
}

func TestUpdateStreamString(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	err = db.PutString("key", "initial")
	if err != nil {
		t.Fatalf("PutString failed: %v", err)
	}

	// Update using UpdateStreamString
	newData := "Updated via stream string!"
	reader := bytes.NewReader([]byte(newData))
	err = db.UpdateStreamString("key", reader, int64(len(newData)))
	if err != nil {
		t.Fatalf("UpdateStreamString failed: %v", err)
	}

	// Verify
	value, err := db.GetString("key")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if value != newData {
		t.Errorf("Expected %q, got %q", newData, value)
	}
}

func TestUpdateStreamNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to update non-existent key
	reader := bytes.NewReader([]byte("data"))
	err = db.UpdateStream([]byte("nonexistent"), reader, 4)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestPutStreamSizeMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create reader with 10 bytes but claim 5
	testData := []byte("1234567890")
	reader := bytes.NewReader(testData)

	err = db.PutStream([]byte("key"), reader, 5)
	if err == nil {
		t.Error("Expected error for size mismatch, got nil")
	}
}

func TestStreamRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create 1MB test data
	testData := make([]byte, 1*1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// PutStream
	putReader := bytes.NewReader(testData)
	err = db.PutStream([]byte("roundtrip"), putReader, int64(len(testData)))
	if err != nil {
		t.Fatalf("PutStream failed: %v", err)
	}

	// GetStream
	var getBuf bytes.Buffer
	n, err := db.GetStream([]byte("roundtrip"), &getBuf)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("Expected %d bytes, got %d", len(testData), n)
	}

	if !bytes.Equal(getBuf.Bytes(), testData) {
		t.Errorf("Round-trip data mismatch")
	}
}

func TestPutStreamDifferentTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test Type1Byte (< 256 bytes)
	data1 := make([]byte, 100)
	for i := range data1 {
		data1[i] = byte(i)
	}
	err = db.PutStream([]byte("type1"), bytes.NewReader(data1), int64(len(data1)))
	if err != nil {
		t.Fatalf("PutStream Type1Byte failed: %v", err)
	}

	// Test Type2Bytes (256 bytes to 64KB)
	data2 := make([]byte, 1000)
	for i := range data2 {
		data2[i] = byte(i % 256)
	}
	err = db.PutStream([]byte("type2"), bytes.NewReader(data2), int64(len(data2)))
	if err != nil {
		t.Fatalf("PutStream Type2Bytes failed: %v", err)
	}

	// Test Type4Bytes (> 64KB)
	data4 := make([]byte, 100000)
	for i := range data4 {
		data4[i] = byte(i % 256)
	}
	err = db.PutStream([]byte("type4"), bytes.NewReader(data4), int64(len(data4)))
	if err != nil {
		t.Fatalf("PutStream Type4Bytes failed: %v", err)
	}

	// Verify all
	val1, _ := db.Get([]byte("type1"))
	if !bytes.Equal(val1, data1) {
		t.Error("Type1Byte data mismatch")
	}

	val2, _ := db.Get([]byte("type2"))
	if !bytes.Equal(val2, data2) {
		t.Error("Type2Bytes data mismatch")
	}

	val4, _ := db.Get([]byte("type4"))
	if !bytes.Equal(val4, data4) {
		t.Error("Type4Bytes data mismatch")
	}
}

func TestStreamWithFreeSpaceReuse(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial data
	data1 := make([]byte, 5000)
	for i := range data1 {
		data1[i] = byte(i % 256)
	}
	err = db.PutStream([]byte("key1"), bytes.NewReader(data1), int64(len(data1)))
	if err != nil {
		t.Fatalf("PutStream failed: %v", err)
	}

	// Delete to create free space
	err = db.Delete([]byte("key1"))
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Put new data that should reuse the free space
	data2 := make([]byte, 4000)
	for i := range data2 {
		data2[i] = byte((i * 2) % 256)
	}
	err = db.PutStream([]byte("key2"), bytes.NewReader(data2), int64(len(data2)))
	if err != nil {
		t.Fatalf("PutStream with free space reuse failed: %v", err)
	}

	// Verify
	val, err := db.Get([]byte("key2"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(val, data2) {
		t.Error("Data mismatch after free space reuse")
	}
}

func TestGetStreamDifferentTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test different data sizes for different record types
	tests := []struct {
		name string
		size int
	}{
		{"small", 100},      // Type1Byte
		{"medium", 1000},    // Type2Bytes
		{"large", 100000},   // Type4Bytes
		{"xlarge", 1000000}, // Type4Bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// Store
			err := db.Put([]byte(tt.name), data)
			if err != nil {
				t.Fatalf("Put failed: %v", err)
			}

			// GetStream
			var buf bytes.Buffer
			n, err := db.GetStream([]byte(tt.name), &buf)
			if err != nil {
				t.Fatalf("GetStream failed: %v", err)
			}

			if n != int64(tt.size) {
				t.Errorf("Expected %d bytes, got %d", tt.size, n)
			}

			if !bytes.Equal(buf.Bytes(), data) {
				t.Error("Data mismatch")
			}
		})
	}
}

func TestPutStreamEmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	reader := bytes.NewReader([]byte("data"))
	err = db.PutStream([]byte(""), reader, 4)
	if err == nil || err.Error() != "key cannot be empty" {
		t.Errorf("Expected 'key cannot be empty' error, got %v", err)
	}
}

func TestPutStreamNegativeSize(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	reader := bytes.NewReader([]byte("data"))
	err = db.PutStream([]byte("key"), reader, -1)
	if err == nil || err.Error() != "size cannot be negative" {
		t.Errorf("Expected 'size cannot be negative' error, got %v", err)
	}
}

func TestUpdateStreamNegativeSize(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	db.Put([]byte("key"), []byte("value"))

	reader := bytes.NewReader([]byte("data"))
	err = db.UpdateStream([]byte("key"), reader, -1)
	if err == nil || err.Error() != "size cannot be negative" {
		t.Errorf("Expected 'size cannot be negative' error, got %v", err)
	}
}

func TestGetStreamEmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var buf bytes.Buffer
	_, err = db.GetStream([]byte(""), &buf)
	if err == nil || err.Error() != "key cannot be empty" {
		t.Errorf("Expected 'key cannot be empty' error, got %v", err)
	}
}

func TestPutStreamZeroSize(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.skv")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Empty data
	reader := bytes.NewReader([]byte{})
	err = db.PutStream([]byte("empty"), reader, 0)
	if err != nil {
		t.Fatalf("PutStream with zero size failed: %v", err)
	}

	// Verify
	val, err := db.Get([]byte("empty"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(val) != 0 {
		t.Errorf("Expected empty value, got %d bytes", len(val))
	}
}
