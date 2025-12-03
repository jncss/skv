package skv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/gofrs/flock"
)

// Record type based on the size of the data field
const (
	Type1Byte  byte = 0x01 // Data size in 1 byte (max 255 bytes)
	Type2Bytes byte = 0x02 // Data size in 2 bytes (max 64KB)
	Type4Bytes byte = 0x04 // Data size in 4 bytes (max 4GB)
	Type8Bytes byte = 0x08 // Data size in 8 bytes

	// Deleted flag (bit 7)
	DeletedFlag byte = 0x80 // When this bit is set, the record is deleted
)

// isDeleted checks if a type has the deleted bit set
func isDeleted(recordType byte) bool {
	return (recordType & DeletedFlag) != 0
}

// getBaseType returns the base type without the deleted bit
func getBaseType(recordType byte) byte {
	return recordType & ^DeletedFlag
}

// SKV represents a key/value database
type SKV struct {
	file     *os.File
	filePath string
	cache    map[string]int64 // Cache: key -> file position
	mu       sync.RWMutex     // Mutex for thread-safe operations
	fileLock *flock.Flock     // Cross-platform file lock
}

// Open opens or creates a .skv file and returns an SKV object
func Open(name string) (*SKV, error) {
	// Add .skv extension if it doesn't have it
	if len(name) < 4 || name[len(name)-4:] != ".skv" {
		name += ".skv"
	}

	// Open or create the file with read/write permissions
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", name, err)
	}

	skv := &SKV{
		file:     file,
		filePath: name,
		cache:    make(map[string]int64),
		fileLock: flock.New(name),
	}

	// Build cache by scanning the file
	if err := skv.rebuildCache(); err != nil {
		file.Close()
		return nil, fmt.Errorf("error building cache: %w", err)
	}

	return skv, nil
}

// Close closes the database file
func (s *SKV) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// CloseWithCompact compacts the database before closing to remove deleted records
// This is useful to optimize the file size when closing the database
func (s *SKV) CloseWithCompact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	// Compact the database to remove deleted records
	// Note: compactInternal is called without lock since we already have it
	if err := s.compactInternal(); err != nil {
		// Even if compact fails, try to close the file
		s.file.Close()
		return fmt.Errorf("error compacting before close: %w", err)
	}

	return s.file.Close()
}

// writeRecord writes a complete record (type, key, data) to the end of the file
// Returns the position where the record was written
func (s *SKV) writeRecord(key []byte, data []byte) (int64, error) {
	// Determine the type based on the data size
	var recordType byte
	dataSize := uint64(len(data))

	switch {
	case dataSize <= 0xFF: // 255 bytes
		recordType = Type1Byte
	case dataSize <= 0xFFFF: // 64KB
		recordType = Type2Bytes
	case dataSize <= 0xFFFFFFFF: // 4GB
		recordType = Type4Bytes
	default:
		recordType = Type8Bytes
	}

	// Move to the end of the file
	if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
		return 0, fmt.Errorf("error seeking to end of file: %w", err)
	}

	// Save position before writing
	recordPos, _ := s.file.Seek(0, io.SeekCurrent)

	// Write the type
	if _, err := s.file.Write([]byte{recordType}); err != nil {
		return 0, fmt.Errorf("error writing type: %w", err)
	}

	// Write the key size
	keySize := byte(len(key))
	if _, err := s.file.Write([]byte{keySize}); err != nil {
		return 0, fmt.Errorf("error writing key size: %w", err)
	}

	// Write the key
	if _, err := s.file.Write(key); err != nil {
		return 0, fmt.Errorf("error writing key: %w", err)
	}

	// Write the data size according to the type
	switch recordType {
	case Type1Byte:
		if _, err := s.file.Write([]byte{byte(dataSize)}); err != nil {
			return 0, fmt.Errorf("error writing data size: %w", err)
		}
	case Type2Bytes:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return 0, fmt.Errorf("error writing data size: %w", err)
		}
	case Type4Bytes:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return 0, fmt.Errorf("error writing data size: %w", err)
		}
	case Type8Bytes:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, dataSize)
		if _, err := s.file.Write(buf); err != nil {
			return 0, fmt.Errorf("error writing data size: %w", err)
		}
	}

	// Write the data
	if len(data) > 0 {
		if _, err := s.file.Write(data); err != nil {
			return 0, fmt.Errorf("error writing data: %w", err)
		}
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return 0, fmt.Errorf("error syncing to disk: %w", err)
	}

	return recordPos, nil
}

// readRecord reads a complete record from the current file position
// Returns the key and data. Assumes file is already positioned at the start of a record.
// readRecord reads a complete record from the current file position
// If readData is false, the data portion is skipped for efficiency
func (s *SKV) readRecord(readData bool) (recordType byte, key []byte, data []byte, err error) {
	// Read type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, typeBuf); err != nil {
		if err == io.EOF {
			return 0, nil, nil, io.EOF // Return EOF directly
		}
		return 0, nil, nil, fmt.Errorf("error reading type: %w", err)
	}
	recordType = typeBuf[0]

	// Read key size
	keySizeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
		return 0, nil, nil, fmt.Errorf("error reading key size: %w", err)
	}
	keySize := keySizeBuf[0]

	// Read key
	key = make([]byte, keySize)
	if _, err := io.ReadFull(s.file, key); err != nil {
		return 0, nil, nil, fmt.Errorf("error reading key: %w", err)
	}

	// Read data size
	baseType := getBaseType(recordType)
	var dataSize uint64
	switch baseType {
	case Type1Byte:
		buf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(buf[0])
	case Type2Bytes:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint16(buf))
	case Type4Bytes:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint32(buf))
	case Type8Bytes:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = binary.LittleEndian.Uint64(buf)
	default:
		return 0, nil, nil, fmt.Errorf("unknown record type: 0x%02X", recordType)
	}

	// Read or skip data depending on readData parameter
	if readData {
		data = make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := io.ReadFull(s.file, data); err != nil {
				return 0, nil, nil, fmt.Errorf("error reading data: %w", err)
			}
		}
	} else {
		// Skip data by seeking forward for efficiency
		if dataSize > 0 {
			if _, err := s.file.Seek(int64(dataSize), io.SeekCurrent); err != nil {
				return 0, nil, nil, fmt.Errorf("error skipping data: %w", err)
			}
		}
	}

	return recordType, key, data, nil
}

// Put stores a new key with its value
// Returns ErrKeyExists if the key already exists
func (s *SKV) Put(key []byte, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long (max 255 bytes)")
	}

	// Check if the key already exists in cache
	if _, exists := s.cache[string(key)]; exists {
		return ErrKeyExists
	}

	// Write the record
	recordPos, err := s.writeRecord(key, data)
	if err != nil {
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	return nil
}

// Update modifies the value of an existing key
// Returns ErrKeyNotFound if the key doesn't exist
func (s *SKV) Update(key []byte, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if the key exists in cache
	if _, exists := s.cache[string(key)]; !exists {
		return ErrKeyNotFound
	}

	// Key exists, delete it first (internal version without lock)
	if err := s.deleteInternal(key); err != nil {
		return err
	}

	// Write the record
	recordPos, err := s.writeRecord(key, data)
	if err != nil {
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	return nil
}

// rebuildCache scans the entire file and builds the cache
func (s *SKV) rebuildCache() error {
	// Clear existing cache
	s.cache = make(map[string]int64)

	// Move to the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start of file: %w", err)
	}

	// Read all records
	for {
		// Save current position
		currentPos, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("error getting current position: %w", err)
		}

		// Read only record metadata (type and key), skip data for efficiency
		recordType, key, _, err := s.readRecord(false)
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			return fmt.Errorf("error reading record metadata: %w", err)
		}

		// Update cache (last occurrence wins)
		keyStr := string(key)
		if isDeleted(recordType) {
			// Remove from cache if deleted
			delete(s.cache, keyStr)
		} else {
			// Add or update in cache
			s.cache[keyStr] = currentPos
		}
	}

	return nil
}

// ErrKeyNotFound is returned when the key is not found
var ErrKeyNotFound = errors.New("key not found")

// ErrKeyExists is returned when trying to insert a key that already exists
var ErrKeyExists = errors.New("key already exists")

// Get retrieves the value associated with a key
// Returns ErrKeyNotFound if the key doesn't exist or is deleted
func (s *SKV) Get(key []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire shared lock for read operation (allows multiple readers)
	if err := s.fileLock.RLock(); err != nil {
		return nil, fmt.Errorf("error acquiring read lock: %w", err)
	}
	defer s.fileLock.Unlock()

	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	// Check cache for position
	position, found := s.cache[string(key)]
	if !found {
		return nil, ErrKeyNotFound
	}

	// Read from file at cached position
	if _, err := s.file.Seek(position, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking to position: %w", err)
	}

	// Read the record
	_, _, data, err := s.readRecord(true)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Delete deletes a key by setting the deleted bit in its record
func (s *SKV) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	return s.deleteInternal(key)
}

// deleteInternal is the internal implementation of Delete without locking
// Used by Update to avoid deadlock
func (s *SKV) deleteInternal(key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if key exists in cache and get its position
	keyStr := string(key)
	position, found := s.cache[keyStr]
	if !found {
		return ErrKeyNotFound
	}

	// Move to the record position (start of record)
	if _, err := s.file.Seek(position, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to record position: %w", err)
	}

	// Read the current type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, typeBuf); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}

	// Set the deleted bit
	deletedType := typeBuf[0] | DeletedFlag

	// Go back to overwrite the type
	if _, err := s.file.Seek(position, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to type position: %w", err)
	}

	// Write the type with the deleted bit
	if _, err := s.file.Write([]byte{deletedType}); err != nil {
		return fmt.Errorf("error marking record as deleted: %w", err)
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("error syncing to disk: %w", err)
	}

	// Remove from cache
	delete(s.cache, keyStr)

	return nil
}

// Stats contains statistics about the database
type Stats struct {
	TotalRecords   int // Total number of records
	ActiveRecords  int // Number of active records (not deleted)
	DeletedRecords int // Number of deleted records
}

// Verify checks the file integrity and returns statistics
func (s *SKV) Verify() (*Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire shared lock for read operation
	if err := s.fileLock.RLock(); err != nil {
		return nil, fmt.Errorf("error acquiring read lock: %w", err)
	}
	defer s.fileLock.Unlock()

	stats := &Stats{}

	// Move to the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking to start of file: %w", err)
	}

	// Read all records in the file
	for {
		// Read record metadata (skip data for efficiency)
		recordType, _, _, err := s.readRecord(false)
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			return nil, fmt.Errorf("error reading record: %w", err)
		}

		// Count the record
		stats.TotalRecords++
		if isDeleted(recordType) {
			stats.DeletedRecords++
		} else {
			stats.ActiveRecords++
		}
	}

	return stats, nil
}

// Compact removes deleted records by creating a new file with only active records
// For keys that appear multiple times, only the last occurrence is kept
func (s *SKV) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	return s.compactInternal()
}

// compactInternal is the internal implementation of Compact without locking
// Used by CloseWithCompact to avoid deadlock
func (s *SKV) compactInternal() error {
	// Collect all active keys and their data from cache
	type keyData struct {
		key  []byte
		data []byte
	}
	activeData := make([]keyData, 0, len(s.cache))

	// Read all active records using cache positions
	for _, position := range s.cache {
		// Seek to record position
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to position: %w", err)
		}

		// Read record
		_, key, data, err := s.readRecord(true)
		if err != nil {
			return fmt.Errorf("error reading record: %w", err)
		}

		activeData = append(activeData, keyData{key: key, data: data})
	}

	// Create new temporary file
	tempPath := s.filePath + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer tempFile.Close()

	// Swap file to write to temp file
	originalFile := s.file
	s.file = tempFile

	// Write all active records using writeRecord
	newCache := make(map[string]int64)
	for _, kd := range activeData {
		pos, err := s.writeRecord(kd.key, kd.data)
		if err != nil {
			s.file = originalFile
			return fmt.Errorf("error writing record: %w", err)
		}
		newCache[string(kd.key)] = pos
	}

	// Restore original file and close both
	s.file = originalFile

	if err := s.file.Close(); err != nil {
		return fmt.Errorf("error closing original file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("error closing temporary file: %w", err)
	}

	// Replace the original file with the temporary file
	if err := os.Rename(tempPath, s.filePath); err != nil {
		return fmt.Errorf("error replacing file: %w", err)
	}

	// Reopen the file
	file, err := os.OpenFile(s.filePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error reopening file: %w", err)
	}

	s.file = file

	// Update cache with new positions
	s.cache = newCache

	return nil
}

// Keys returns a list of all active keys in the database
func (s *SKV) Keys() ([][]byte, error) {
	// Convert cache keys to slice
	keys := make([][]byte, 0, len(s.cache))
	for keyStr := range s.cache {
		keys = append(keys, []byte(keyStr))
	}

	return keys, nil
}

// String-based convenience functions

// PutString stores a new key-value pair using strings
func (s *SKV) PutString(key string, value string) error {
	return s.Put([]byte(key), []byte(value))
}

// UpdateString updates an existing key with a new value using strings
func (s *SKV) UpdateString(key string, value string) error {
	return s.Update([]byte(key), []byte(value))
}

// GetString retrieves the value for a key using strings
func (s *SKV) GetString(key string) (string, error) {
	value, err := s.Get([]byte(key))
	if err != nil {
		return "", err
	}
	return string(value), nil
}

// DeleteString deletes a key using a string
func (s *SKV) DeleteString(key string) error {
	return s.Delete([]byte(key))
}

// KeysString returns a list of all active keys as strings
func (s *SKV) KeysString() ([]string, error) {
	keys := make([]string, 0, len(s.cache))
	for keyStr := range s.cache {
		keys = append(keys, keyStr)
	}
	return keys, nil
}

// Exists checks if a key exists in the database
func (s *SKV) Exists(key []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.cache[string(key)]
	return exists
}

// Has is an alias for Exists (more idiomatic name)
func (s *SKV) Has(key []byte) bool {
	return s.Exists(key)
}

// ExistsString checks if a key exists using a string
func (s *SKV) ExistsString(key string) bool {
	return s.Exists([]byte(key))
}

// HasString is an alias for ExistsString
func (s *SKV) HasString(key string) bool {
	return s.ExistsString(key)
}

// Count returns the number of active keys in the database
func (s *SKV) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.cache)
}

// Clear removes all keys from the database by truncating the file
func (s *SKV) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	// Truncate the file to 0 bytes
	if err := s.file.Truncate(0); err != nil {
		return fmt.Errorf("error truncating file: %w", err)
	}

	// Seek to the beginning
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start: %w", err)
	}

	// Clear the cache
	s.cache = make(map[string]int64)

	return nil
}

// GetOrDefault retrieves the value for a key, returning a default value if not found
func (s *SKV) GetOrDefault(key []byte, defaultValue []byte) []byte {
	value, err := s.Get(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetOrDefaultString retrieves the value for a key as string, returning a default if not found
func (s *SKV) GetOrDefaultString(key string, defaultValue string) string {
	value, err := s.GetString(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// ForEach iterates over all active keys and values in the database
// The callback function receives each key-value pair
// If the callback returns an error, iteration stops and the error is returned
func (s *SKV) ForEach(fn func(key []byte, value []byte) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire shared lock for read operation
	if err := s.fileLock.RLock(); err != nil {
		return fmt.Errorf("error acquiring read lock: %w", err)
	}
	defer s.fileLock.Unlock()

	// Iterate over all cached keys
	for _, position := range s.cache {
		// Seek to the record position
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to position: %w", err)
		}

		// Read the record
		_, key, data, err := s.readRecord(true)
		if err != nil {
			return fmt.Errorf("error reading record: %w", err)
		}

		// Call the callback function
		if err := fn(key, data); err != nil {
			return err
		}
	}

	return nil
}

// ForEachString iterates over all active keys and values as strings
func (s *SKV) ForEachString(fn func(key string, value string) error) error {
	return s.ForEach(func(key []byte, value []byte) error {
		return fn(string(key), string(value))
	})
}

// PutBatch stores multiple key-value pairs in a single operation
// If any key already exists, the entire operation fails and returns ErrKeyExists
func (s *SKV) PutBatch(items map[string][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire exclusive lock for write operation
	if err := s.fileLock.Lock(); err != nil {
		return fmt.Errorf("error acquiring write lock: %w", err)
	}
	defer s.fileLock.Unlock()

	// Check if any key already exists
	for key := range items {
		if _, exists := s.cache[key]; exists {
			return fmt.Errorf("key %q already exists: %w", key, ErrKeyExists)
		}
	}

	// Write all records
	for key, data := range items {
		keyBytes := []byte(key)

		if len(keyBytes) == 0 {
			return fmt.Errorf("key cannot be empty")
		}
		if len(keyBytes) > 255 {
			return fmt.Errorf("key %q too long (max 255 bytes)", key)
		}

		recordPos, err := s.writeRecord(keyBytes, data)
		if err != nil {
			return fmt.Errorf("error writing key %q: %w", key, err)
		}

		s.cache[key] = recordPos
	}

	return nil
}

// PutBatchString stores multiple key-value pairs using strings
func (s *SKV) PutBatchString(items map[string]string) error {
	byteItems := make(map[string][]byte, len(items))
	for key, value := range items {
		byteItems[key] = []byte(value)
	}
	return s.PutBatch(byteItems)
}

// GetBatch retrieves multiple keys at once
// Returns a map with the values for existing keys
// Missing keys are not included in the result map
func (s *SKV) GetBatch(keys [][]byte) (map[string][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire shared lock for read operation
	if err := s.fileLock.RLock(); err != nil {
		return nil, fmt.Errorf("error acquiring read lock: %w", err)
	}
	defer s.fileLock.Unlock()

	result := make(map[string][]byte, len(keys))

	for _, key := range keys {
		keyStr := string(key)
		position, found := s.cache[keyStr]
		if !found {
			continue // Skip missing keys
		}

		// Seek to the record position
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			return nil, fmt.Errorf("error seeking to position: %w", err)
		}

		// Read the record
		_, _, data, err := s.readRecord(true)
		if err != nil {
			return nil, fmt.Errorf("error reading record: %w", err)
		}

		result[keyStr] = data
	}

	return result, nil
}

// GetBatchString retrieves multiple keys using strings
// Returns a map with the values for existing keys
// Missing keys are not included in the result map
func (s *SKV) GetBatchString(keys []string) (map[string]string, error) {
	byteKeys := make([][]byte, len(keys))
	for i, key := range keys {
		byteKeys[i] = []byte(key)
	}

	byteResult, err := s.GetBatch(byteKeys)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(byteResult))
	for key, value := range byteResult {
		result[key] = string(value)
	}

	return result, nil
}
