package skv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
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

// cacheEntry stores cached key information
type cacheEntry struct {
	position int64  // Position in file where the record starts
	data     []byte // Cached data value
}

// SKV represents a key/value database
type SKV struct {
	file     *os.File
	filePath string
	cache    map[string]*cacheEntry // Cache for active keys
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
		cache:    make(map[string]*cacheEntry),
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
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// Put stores a new key with its value
// Returns ErrKeyExists if the key already exists
func (s *SKV) Put(key []byte, data []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key cannot be longer than 255 bytes")
	}

	// Check if the key already exists
	_, err := s.Get(key)
	if err == nil {
		// Key exists
		return ErrKeyExists
	} else if err != ErrKeyNotFound {
		// Error reading the file
		return err
	}
	// Key doesn't exist, proceed with insertion

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
		return fmt.Errorf("error seeking to end of file: %w", err)
	}

	// Prepare the buffer to write
	keySize := byte(len(key))

	// Write the type
	if _, err := s.file.Write([]byte{recordType}); err != nil {
		return fmt.Errorf("error writing type: %w", err)
	}

	// Write the key size
	if _, err := s.file.Write([]byte{keySize}); err != nil {
		return fmt.Errorf("error writing key size: %w", err)
	}

	// Write the key
	if _, err := s.file.Write(key); err != nil {
		return fmt.Errorf("error writing key: %w", err)
	}

	// Write the data size according to the type
	switch recordType {
	case Type1Byte:
		if _, err := s.file.Write([]byte{byte(dataSize)}); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type2Bytes:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type4Bytes:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type8Bytes:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, dataSize)
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	}

	// Write the data
	if len(data) > 0 {
		if _, err := s.file.Write(data); err != nil {
			return fmt.Errorf("error writing data: %w", err)
		}
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("error syncing to disk: %w", err)
	}

	// Update cache
	keyStr := string(key)
	s.cache[keyStr] = &cacheEntry{
		position: -1,
		data:     data,
	}

	return nil
}

// Update modifies the value of an existing key
// Returns ErrKeyNotFound if the key doesn't exist
func (s *SKV) Update(key []byte, data []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if the key exists
	_, err := s.Get(key)
	if err == ErrKeyNotFound {
		// Key doesn't exist
		return ErrKeyNotFound
	} else if err != nil {
		// Error reading the file
		return err
	}

	// Key exists, delete it first
	if err := s.Delete(key); err != nil {
		return err
	}

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
	_, err = s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("error seeking to end of file: %w", err)
	}

	// Prepare the buffer to write
	keySize := byte(len(key))

	// Write type
	if _, err := s.file.Write([]byte{recordType}); err != nil {
		return fmt.Errorf("error writing type: %w", err)
	}

	// Write key size
	if _, err := s.file.Write([]byte{keySize}); err != nil {
		return fmt.Errorf("error writing key size: %w", err)
	}

	// Write the key
	if _, err := s.file.Write(key); err != nil {
		return fmt.Errorf("error writing key: %w", err)
	}

	// Write the data size according to the type
	switch recordType {
	case Type1Byte:
		if _, err := s.file.Write([]byte{byte(dataSize)}); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type2Bytes:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type4Bytes:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataSize))
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	case Type8Bytes:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, dataSize)
		if _, err := s.file.Write(buf); err != nil {
			return fmt.Errorf("error writing data size: %w", err)
		}
	}

	// Write the data
	if len(data) > 0 {
		if _, err := s.file.Write(data); err != nil {
			return fmt.Errorf("error writing data: %w", err)
		}
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("error syncing to disk: %w", err)
	}

	// Update cache
	s.cache[string(key)] = &cacheEntry{
		position: -1, // Position will be set when needed
		data:     data,
	}

	return nil
}

// rebuildCache scans the entire file and builds the cache
func (s *SKV) rebuildCache() error {
	// Clear existing cache
	s.cache = make(map[string]*cacheEntry)

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

		// Read type (1 byte)
		typeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, typeBuf); err != nil {
			if err == io.EOF {
				break // End of file
			}
			return fmt.Errorf("error reading type: %w", err)
		}
		recordType := typeBuf[0]

		// Read key size (1 byte)
		keySizeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
			return fmt.Errorf("error reading key size: %w", err)
		}
		keySize := keySizeBuf[0]

		if keySize == 0 {
			return fmt.Errorf("invalid record: key size = 0")
		}

		// Read the key
		key := make([]byte, keySize)
		if _, err := io.ReadFull(s.file, key); err != nil {
			return fmt.Errorf("error reading key: %w", err)
		}

		// Read the data size according to the base type
		baseType := getBaseType(recordType)
		var dataSize uint64
		switch baseType {
		case Type1Byte:
			buf := make([]byte, 1)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(buf[0])
		case Type2Bytes:
			buf := make([]byte, 2)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint16(buf))
		case Type4Bytes:
			buf := make([]byte, 4)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint32(buf))
		case Type8Bytes:
			buf := make([]byte, 8)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = binary.LittleEndian.Uint64(buf)
		default:
			return fmt.Errorf("unknown record type: 0x%02X", recordType)
		}

		// Read the data
		data := make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := io.ReadFull(s.file, data); err != nil {
				return fmt.Errorf("error reading data: %w", err)
			}
		}

		// Update cache (last occurrence wins)
		keyStr := string(key)
		if isDeleted(recordType) {
			// Remove from cache if deleted
			delete(s.cache, keyStr)
		} else {
			// Add or update in cache
			s.cache[keyStr] = &cacheEntry{
				position: currentPos,
				data:     data,
			}
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
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	// Check cache first
	keyStr := string(key)
	if entry, found := s.cache[keyStr]; found {
		// Return a copy of the cached data to prevent modification
		dataCopy := make([]byte, len(entry.data))
		copy(dataCopy, entry.data)
		return dataCopy, nil
	}

	return nil, ErrKeyNotFound
}

// Delete deletes a key by setting the deleted bit in its record
func (s *SKV) Delete(key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if key exists in cache
	keyStr := string(key)
	_, found := s.cache[keyStr]
	if !found {
		return ErrKeyNotFound
	}

	// Move to the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start of file: %w", err)
	}

	var lastPosition int64 = -1

	// Search for the last occurrence of the key
	for {
		// Save the current position (start of record)
		recordStart, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("error getting position: %w", err)
		}

		// Read type (1 byte)
		typeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, typeBuf); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading type: %w", err)
		}
		recordType := typeBuf[0]

		// Read key size (1 byte)
		keySizeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
			return fmt.Errorf("error reading key size: %w", err)
		}
		keySize := keySizeBuf[0]

		// Read the key
		currentKey := make([]byte, keySize)
		if _, err := io.ReadFull(s.file, currentKey); err != nil {
			return fmt.Errorf("error reading key: %w", err)
		}

		// Read the data size according to the base type
		baseType := getBaseType(recordType)
		var dataSize uint64
		switch baseType {
		case Type1Byte:
			buf := make([]byte, 1)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(buf[0])
		case Type2Bytes:
			buf := make([]byte, 2)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint16(buf))
		case Type4Bytes:
			buf := make([]byte, 4)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint32(buf))
		case Type8Bytes:
			buf := make([]byte, 8)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = binary.LittleEndian.Uint64(buf)
		default:
			return fmt.Errorf("unknown record type: %d", baseType)
		}

		// Compare the key
		if bytes.Equal(currentKey, key) && !isDeleted(recordType) {
			// Find the last non-deleted occurrence
			lastPosition = recordStart
		}

		// Skip the data
		if dataSize > 0 {
			if _, err := s.file.Seek(int64(dataSize), io.SeekCurrent); err != nil {
				return fmt.Errorf("error skipping data: %w", err)
			}
		}
	}

	if lastPosition == -1 {
		return ErrKeyNotFound
	}

	// Move to the type position (start of record)
	if _, err := s.file.Seek(lastPosition, io.SeekStart); err != nil {
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
	if _, err := s.file.Seek(lastPosition, io.SeekStart); err != nil {
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
	stats := &Stats{}

	// Move to the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking to start of file: %w", err)
	}

	// Read all records in the file
	for {
		// Read type (1 byte)
		typeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, typeBuf); err != nil {
			if err == io.EOF {
				break // End of file
			}
			return nil, fmt.Errorf("error reading type: %w", err)
		}
		recordType := typeBuf[0]

		// Read key size (1 byte)
		keySizeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
			return nil, fmt.Errorf("error reading key size: %w", err)
		}
		keySize := keySizeBuf[0]

		if keySize == 0 {
			return nil, fmt.Errorf("invalid record: key size = 0")
		}

		// Read the key
		key := make([]byte, keySize)
		if _, err := io.ReadFull(s.file, key); err != nil {
			return nil, fmt.Errorf("error reading key: %w", err)
		}

		// Read the data size according to the base type
		baseType := getBaseType(recordType)
		var dataSize uint64
		switch baseType {
		case Type1Byte:
			buf := make([]byte, 1)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return nil, fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(buf[0])
		case Type2Bytes:
			buf := make([]byte, 2)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return nil, fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint16(buf))
		case Type4Bytes:
			buf := make([]byte, 4)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return nil, fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint32(buf))
		case Type8Bytes:
			buf := make([]byte, 8)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return nil, fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = binary.LittleEndian.Uint64(buf)
		default:
			return nil, fmt.Errorf("unknown record type: 0x%02X (base: 0x%02X)", recordType, baseType)
		}

		// Skip the data
		if dataSize > 0 {
			if _, err := s.file.Seek(int64(dataSize), io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("error skipping data: %w", err)
			}
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
	// First pass: collect all active keys with their last occurrence
	type recordInfo struct {
		position int64
		size     int
	}
	activeRecords := make(map[string]recordInfo)

	// Move to the beginning of the original file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start of file: %w", err)
	}

	// First pass: find the last occurrence of each active key
	for {
		// Save the current position
		currentPos, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("error getting current position: %w", err)
		}

		// Read type (1 byte)
		typeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, typeBuf); err != nil {
			if err == io.EOF {
				break // End of file
			}
			return fmt.Errorf("error reading type: %w", err)
		}
		recordType := typeBuf[0]

		// Read key size (1 byte)
		keySizeBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
			return fmt.Errorf("error reading key size: %w", err)
		}
		keySize := keySizeBuf[0]

		if keySize == 0 {
			return fmt.Errorf("invalid record: key size = 0")
		}

		// Read the key
		key := make([]byte, keySize)
		if _, err := io.ReadFull(s.file, key); err != nil {
			return fmt.Errorf("error reading key: %w", err)
		}

		// Read the data size according to the base type
		baseType := getBaseType(recordType)
		var dataSize uint64
		var dataSizeBytes int
		switch baseType {
		case Type1Byte:
			buf := make([]byte, 1)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(buf[0])
			dataSizeBytes = 1
		case Type2Bytes:
			buf := make([]byte, 2)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint16(buf))
			dataSizeBytes = 2
		case Type4Bytes:
			buf := make([]byte, 4)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = uint64(binary.LittleEndian.Uint32(buf))
			dataSizeBytes = 4
		case Type8Bytes:
			buf := make([]byte, 8)
			if _, err := io.ReadFull(s.file, buf); err != nil {
				return fmt.Errorf("error reading data size: %w", err)
			}
			dataSize = binary.LittleEndian.Uint64(buf)
			dataSizeBytes = 8
		default:
			return fmt.Errorf("unknown record type: 0x%02X", recordType)
		}

		// Calculate total record size
		recordSize := 1 + 1 + int(keySize) + dataSizeBytes + int(dataSize)

		// Skip the data
		if dataSize > 0 {
			if _, err := s.file.Seek(int64(dataSize), io.SeekCurrent); err != nil {
				return fmt.Errorf("error skipping data: %w", err)
			}
		}

		// If the record is not deleted, store its position (last occurrence wins)
		if !isDeleted(recordType) {
			activeRecords[string(key)] = recordInfo{
				position: currentPos,
				size:     recordSize,
			}
		}
	}

	// Second pass: create new file with only the active records
	tempPath := s.filePath + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer tempFile.Close()

	// Write each active record to the temporary file
	for _, info := range activeRecords {
		// Seek to the record position
		if _, err := s.file.Seek(info.position, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to record: %w", err)
		}

		// Read the full record
		fullRecord := make([]byte, info.size)
		if _, err := io.ReadFull(s.file, fullRecord); err != nil {
			return fmt.Errorf("error reading full record: %w", err)
		}

		// Write to the temporary file
		if _, err := tempFile.Write(fullRecord); err != nil {
			return fmt.Errorf("error writing to temporary file: %w", err)
		}
	}

	// Close both files
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

	// Rebuild cache
	if err := s.rebuildCache(); err != nil {
		return fmt.Errorf("error rebuilding cache: %w", err)
	}

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
