package skv

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode/utf8"
)

// File header constants
const (
	HeaderMagic  = "SKV" // Magic bytes to identify SKV files
	HeaderSize   = 6     // Total header size: 3 bytes magic + 3 bytes version
	VersionMajor = 0     // Major version number
	VersionMinor = 1     // Minor version number
	VersionPatch = 0     // Patch version number
)

// Record type based on the size of the data field
const (
	Type1Byte  byte = 0x01 // Data size in 1 byte (max 255 bytes)
	Type2Bytes byte = 0x02 // Data size in 2 bytes (max 64KB)
	Type4Bytes byte = 0x04 // Data size in 4 bytes (max 4GB)
	Type8Bytes byte = 0x08 // Data size in 8 bytes

	// Deleted flag (bit 7)
	DeletedFlag byte = 0x80 // When this bit is set, the record is deleted

	// Padding byte for filling small gaps
	PaddingByte byte = 0x80 // Used to fill gaps too small for a deleted record

	// Minimum record size (type + key_size + key(1) + data_size)
	MinRecordSize = 4 // Minimum size for a valid record
)

// isDeleted checks if a type has the deleted bit set
func isDeleted(recordType byte) bool {
	return (recordType & DeletedFlag) != 0
}

// getBaseType returns the base type without the deleted bit
func getBaseType(recordType byte) byte {
	return recordType & ^DeletedFlag
}

// getRecordType determines the record type based on data size
func getRecordType(dataSize uint64) byte {
	switch {
	case dataSize <= 0xFF: // 255 bytes
		return Type1Byte
	case dataSize <= 0xFFFF: // 64KB
		return Type2Bytes
	case dataSize <= 0xFFFFFFFF: // 4GB
		return Type4Bytes
	default:
		return Type8Bytes
	}
}

// calculateRecordSize calculates the total size of a record
// Returns: total size including type, key_size, key, data_size, and data
func calculateRecordSize(keySize byte, dataSize uint64, recordType byte) uint64 {
	baseType := getBaseType(recordType)
	var dataSizeFieldSize uint64
	switch baseType {
	case Type1Byte:
		dataSizeFieldSize = 1
	case Type2Bytes:
		dataSizeFieldSize = 2
	case Type4Bytes:
		dataSizeFieldSize = 4
	case Type8Bytes:
		dataSizeFieldSize = 8
	default:
		dataSizeFieldSize = 1
	}

	// type (1) + key_size (1) + key + data_size_field + data
	return 1 + 1 + uint64(keySize) + dataSizeFieldSize + dataSize
}

// skipPaddingBytes skips any padding bytes (0x80) at the current file position
// Returns the number of padding bytes skipped
func (s *SKV) skipPaddingBytes() (int64, error) {
	var paddingCount int64

	for {
		// Read one byte
		buf := make([]byte, 1)
		n, err := s.file.Read(buf)
		if err != nil {
			if err == io.EOF && paddingCount > 0 {
				return paddingCount, io.EOF
			}
			if err == io.EOF {
				return 0, io.EOF
			}
			return paddingCount, err
		}
		if n == 0 {
			break
		}

		// If it's not a padding byte, seek back and return
		if buf[0] != PaddingByte {
			if _, err := s.file.Seek(-1, io.SeekCurrent); err != nil {
				return paddingCount, fmt.Errorf("error seeking back: %w", err)
			}
			break
		}

		paddingCount++
	}

	return paddingCount, nil
}

// findBestFreeSpace finds the best free space for a record of the given size
// Returns the index in freeSpace slice, or -1 if no suitable space found
// Strategy: find smallest space that fits (best fit)
func (s *SKV) findBestFreeSpace(neededSize uint64) int {
	bestIdx := -1
	var bestSize uint64 = ^uint64(0) // Max uint64

	for i, free := range s.freeSpace {
		if free.size >= neededSize && free.size < bestSize {
			bestIdx = i
			bestSize = free.size
		}
	}

	return bestIdx
}

// FreeSpace represents a deleted record that can be reused
type FreeSpace struct {
	position int64  // File position of the free space
	size     uint64 // Total size of the free space (including padding)
}

// SKV represents a key/value database
type SKV struct {
	file      *os.File
	filePath  string
	cache     map[string]int64 // Cache: key -> file position
	freeSpace []FreeSpace      // List of free spaces (deleted records)
	mu        sync.RWMutex     // Mutex for thread-safe operations
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
		file:      file,
		filePath:  name,
		cache:     make(map[string]int64),
		freeSpace: make([]FreeSpace, 0),
	}

	// Check if file is new or existing
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error getting file info: %w", err)
	}

	if info.Size() == 0 {
		// New file - write header
		if err := skv.writeHeader(); err != nil {
			file.Close()
			return nil, fmt.Errorf("error writing header: %w", err)
		}
	} else {
		// Existing file - verify header
		if err := skv.verifyHeader(); err != nil {
			file.Close()
			return nil, fmt.Errorf("error verifying header: %w", err)
		}
	}

	// Build cache by scanning the file
	if err := skv.rebuildCache(); err != nil {
		file.Close()
		return nil, fmt.Errorf("error building cache: %w", err)
	}

	return skv, nil
}

// writeHeader writes the SKV file header (magic bytes + version)
func (s *SKV) writeHeader() error {
	header := make([]byte, HeaderSize)
	// Write magic bytes "SKV"
	copy(header[0:3], HeaderMagic)
	// Write version (3 bytes: major, minor, patch)
	header[3] = byte(VersionMajor)
	header[4] = byte(VersionMinor)
	header[5] = byte(VersionPatch)

	// Write header at the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start: %w", err)
	}
	if _, err := s.file.Write(header); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("error syncing header: %w", err)
	}
	return nil
}

// verifyHeader verifies the SKV file header
func (s *SKV) verifyHeader() error {
	header := make([]byte, HeaderSize)

	// Read header from the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start: %w", err)
	}

	if _, err := io.ReadFull(s.file, header); err != nil {
		return fmt.Errorf("error reading header: %w", err)
	}

	// Verify magic bytes
	if string(header[0:3]) != HeaderMagic {
		return fmt.Errorf("invalid SKV file: expected magic bytes %q, got %q", HeaderMagic, string(header[0:3]))
	}

	// Header is valid - file position is now after header, ready to read records
	return nil
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

// writeRecordAtPosition writes a complete record (type, key, data) at the current file position
// Returns the position where the record was written
func (s *SKV) writeRecordAtPosition(key []byte, data []byte) (int64, error) {
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

	// Save position before writing
	recordPos, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("error getting current position: %w", err)
	}

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

// writeRecord writes a complete record (type, key, data)
// Returns the position where the record was written
// Tries to reuse free space if available, otherwise appends to end of file
func (s *SKV) writeRecord(key []byte, data []byte) (int64, error) {
	// Calculate total size needed for this record
	recordType := getRecordType(uint64(len(data)))
	neededSize := calculateRecordSize(byte(len(key)), uint64(len(data)), recordType)

	// Try to find suitable free space
	freeIdx := s.findBestFreeSpace(neededSize)

	if freeIdx >= 0 {
		// Reuse free space
		freeSlot := s.freeSpace[freeIdx]
		recordPos := freeSlot.position

		// Seek to the free space position
		if _, err := s.file.Seek(recordPos, io.SeekStart); err != nil {
			return 0, fmt.Errorf("error seeking to free space: %w", err)
		}

		// Write the record
		if _, err := s.writeRecordAtPosition(key, data); err != nil {
			return 0, err
		}

		// If there's leftover space, fill with padding
		leftover := freeSlot.size - neededSize
		if leftover > 0 {
			padding := make([]byte, leftover)
			for i := range padding {
				padding[i] = PaddingByte
			}
			if _, err := s.file.Write(padding); err != nil {
				return 0, fmt.Errorf("error writing padding: %w", err)
			}
			if err := s.file.Sync(); err != nil {
				return 0, fmt.Errorf("error syncing padding: %w", err)
			}
		}

		// Remove this free space from the list
		s.freeSpace = append(s.freeSpace[:freeIdx], s.freeSpace[freeIdx+1:]...)

		return recordPos, nil
	}

	// No suitable free space, append to end of file
	if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
		return 0, fmt.Errorf("error seeking to end of file: %w", err)
	}

	// Check if we're at the beginning (just after header or empty file)
	currentPos, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("error getting current position: %w", err)
	}

	// If file only contains header, we're ready to write first record
	// If file is empty (shouldn't happen as Open writes header), write header first
	if currentPos == 0 {
		if err := s.writeHeader(); err != nil {
			return 0, fmt.Errorf("error writing header: %w", err)
		}
	}

	return s.writeRecordAtPosition(key, data)
}

// readRecord reads a complete record from the current file position
// If readData is false, the data portion is skipped for efficiency
// Returns: recordType, key, data, recordSize, error
func (s *SKV) readRecord(readData bool) (recordType byte, key []byte, data []byte, recordSize uint64, err error) {
	// Read type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, typeBuf); err != nil {
		if err == io.EOF {
			return 0, nil, nil, 0, io.EOF // Return EOF directly
		}
		return 0, nil, nil, 0, fmt.Errorf("error reading type: %w", err)
	}
	recordType = typeBuf[0]

	// Read key size
	keySizeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("error reading key size: %w", err)
	}
	keySize := keySizeBuf[0]

	// Read key
	key = make([]byte, keySize)
	if _, err := io.ReadFull(s.file, key); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("error reading key: %w", err)
	}

	// Read data size
	baseType := getBaseType(recordType)
	var dataSize uint64
	switch baseType {
	case Type1Byte:
		buf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(buf[0])
	case Type2Bytes:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint16(buf))
	case Type4Bytes:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint32(buf))
	case Type8Bytes:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = binary.LittleEndian.Uint64(buf)
	default:
		return 0, nil, nil, 0, fmt.Errorf("unknown record type: 0x%02X", recordType)
	}

	// Calculate total record size
	recordSize = calculateRecordSize(keySize, dataSize, recordType)

	// Read or skip data depending on readData parameter
	if readData {
		data = make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := io.ReadFull(s.file, data); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading data: %w", err)
			}
		}
	} else {
		// Skip data by seeking forward for efficiency
		if dataSize > 0 {
			if _, err := s.file.Seek(int64(dataSize), io.SeekCurrent); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error skipping data: %w", err)
			}
		}
	}

	return recordType, key, data, recordSize, nil
}

// Put stores a new key with its value
// Returns ErrKeyExists if the key already exists
func (s *SKV) Put(key []byte, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// putInternal writes or overwrites a key without acquiring the lock
// Used internally when the lock is already held (e.g., in Restore)
func (s *SKV) putInternal(key []byte, data []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long (max 255 bytes)")
	}

	keyStr := string(key)

	// If key exists, delete it first
	if _, exists := s.cache[keyStr]; exists {
		if err := s.deleteInternal(key); err != nil {
			return err
		}
	}

	// Write the record
	recordPos, err := s.writeRecord(key, data)
	if err != nil {
		return err
	}

	// Update cache with record start position
	s.cache[keyStr] = recordPos

	return nil
}

// Update modifies the value of an existing key
// Returns ErrKeyNotFound if the key doesn't exist
func (s *SKV) Update(key []byte, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	// Clear existing cache and free space list
	s.cache = make(map[string]int64)
	s.freeSpace = make([]FreeSpace, 0)

	// Move to the beginning of the file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start of file: %w", err)
	}

	// Skip the header (all SKV files must have a header)
	if _, err := s.file.Seek(HeaderSize, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking past header: %w", err)
	}

	// Read all records
	for {
		// Save current position
		currentPos, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("error getting current position: %w", err)
		}

		// Check for padding bytes
		paddingSize, err := s.skipPaddingBytes()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// If we found padding, update current position
		if paddingSize > 0 {
			currentPos, err = s.file.Seek(0, io.SeekCurrent)
			if err != nil {
				return fmt.Errorf("error getting current position after padding: %w", err)
			}
		}

		// Read only record metadata (type and key), skip data for efficiency
		recordType, key, _, recordSize, err := s.readRecord(false)
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

			// Check for padding bytes after this deleted record
			postPaddingSize, err := s.skipPaddingBytes()
			if err != nil && err != io.EOF {
				return err
			}

			// Add to free space list (record + padding)
			totalFreeSize := recordSize + uint64(postPaddingSize)
			s.freeSpace = append(s.freeSpace, FreeSpace{
				position: currentPos + int64(paddingSize),
				size:     totalFreeSize,
			})
		} else {
			// Add or update in cache
			s.cache[keyStr] = currentPos + int64(paddingSize)
		}
	}

	return nil
} // ErrKeyNotFound is returned when the key is not found
var ErrKeyNotFound = errors.New("key not found")

// ErrKeyExists is returned when trying to insert a key that already exists
var ErrKeyExists = errors.New("key already exists")

// Get retrieves the value associated with a key
// Returns ErrKeyNotFound if the key doesn't exist or is deleted
func (s *SKV) Get(key []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	_, _, data, _, err := s.readRecord(true)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Delete deletes a key by setting the deleted bit in its record
func (s *SKV) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// Read the record to get its size
	recordType, _, _, recordSize, err := s.readRecord(true)
	if err != nil {
		return fmt.Errorf("error reading record: %w", err)
	}

	// Set the deleted bit
	deletedType := recordType | DeletedFlag

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

	// Check for padding after this record
	afterRecordPos := position + int64(recordSize)
	if _, err := s.file.Seek(afterRecordPos, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking after record: %w", err)
	}

	paddingSize, err := s.skipPaddingBytes()
	if err != nil && err != io.EOF {
		return fmt.Errorf("error checking padding: %w", err)
	}

	// Add to free space list (record + any trailing padding)
	totalFreeSize := recordSize + uint64(paddingSize)
	s.freeSpace = append(s.freeSpace, FreeSpace{
		position: position,
		size:     totalFreeSize,
	})

	return nil
}

// Stats contains statistics about the database
type Stats struct {
	TotalRecords    int     // Total number of records
	ActiveRecords   int     // Number of active records (not deleted)
	DeletedRecords  int     // Number of deleted records
	FileSize        int64   // Total file size in bytes
	HeaderSize      int64   // Size of file header in bytes
	DataSize        int64   // Size of all data (active + deleted records) in bytes
	WastedSpace     int64   // Space occupied by deleted records in bytes
	PaddingBytes    int64   // Space occupied by padding bytes
	WastedPercent   float64 // Percentage of wasted space (deleted + padding)
	Efficiency      float64 // Percentage of space used by active records
	AverageKeySize  float64 // Average key size in bytes
	AverageDataSize float64 // Average data value size in bytes
}

// Verify checks the file integrity and returns statistics
func (s *SKV) Verify() (*Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := &Stats{
		HeaderSize: HeaderSize,
	}

	// Get file size
	fileInfo, err := s.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}
	stats.FileSize = fileInfo.Size()

	// Skip the header (all SKV files must have a header)
	if _, err := s.file.Seek(HeaderSize, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking past header: %w", err)
	}

	var totalKeySize int64
	var totalDataSize int64
	var activeDataSize int64

	// Read all records in the file
	for {
		// Skip any padding bytes
		paddingCount, err := s.skipPaddingBytes()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error skipping padding: %w", err)
		}
		stats.PaddingBytes += paddingCount

		// Check if we're at EOF after skipping padding
		posAfterPadding, _ := s.file.Seek(0, io.SeekCurrent)
		if posAfterPadding >= stats.FileSize {
			break
		}

		// Read record metadata and data
		recordType, key, data, recordSize, err := s.readRecord(true)
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			return nil, fmt.Errorf("error reading record: %w", err)
		}

		// Count the record
		stats.TotalRecords++
		totalKeySize += int64(len(key))
		totalDataSize += int64(len(data))

		if isDeleted(recordType) {
			stats.DeletedRecords++
			stats.WastedSpace += int64(recordSize)
		} else {
			stats.ActiveRecords++
			activeDataSize += int64(recordSize)
		}
	}

	// Calculate data size (all records, excluding header and padding)
	stats.DataSize = stats.FileSize - stats.HeaderSize - stats.PaddingBytes

	// Calculate wasted space percentage
	usableSpace := stats.FileSize - stats.HeaderSize
	if usableSpace > 0 {
		totalWasted := stats.WastedSpace + stats.PaddingBytes
		stats.WastedPercent = (float64(totalWasted) / float64(usableSpace)) * 100.0
		stats.Efficiency = (float64(activeDataSize) / float64(usableSpace)) * 100.0
	}

	// Calculate averages
	if stats.TotalRecords > 0 {
		stats.AverageKeySize = float64(totalKeySize) / float64(stats.TotalRecords)
		stats.AverageDataSize = float64(totalDataSize) / float64(stats.TotalRecords)
	}

	return stats, nil
}

// Compact removes deleted records by creating a new file with only active records
// For keys that appear multiple times, only the last occurrence is kept
func (s *SKV) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		_, key, data, _, err := s.readRecord(true)
		if err != nil {
			return fmt.Errorf("error reading record: %w", err)
		}

		activeData = append(activeData, keyData{key: key, data: data})
	}

	// Seek to beginning of file
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to beginning: %w", err)
	}

	// Write header first
	if err := s.writeHeader(); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	// Write all active records in-place using writeRecordAtPosition
	newCache := make(map[string]int64)
	for _, kd := range activeData {
		pos, err := s.writeRecordAtPosition(kd.key, kd.data)
		if err != nil {
			return fmt.Errorf("error writing record: %w", err)
		}
		newCache[string(kd.key)] = pos
	}

	// Get current position (end of compacted data)
	endPos, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("error getting current position: %w", err)
	}

	// Truncate file to new size
	if err := s.file.Truncate(endPos); err != nil {
		return fmt.Errorf("error truncating file: %w", err)
	}

	// Sync to ensure all data is written
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %w", err)
	}

	// Update cache with new positions
	s.cache = newCache

	// Clear free space list (compaction eliminates all deleted records)
	s.freeSpace = make([]FreeSpace, 0)

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

	// Truncate the file to 0 bytes
	if err := s.file.Truncate(0); err != nil {
		return fmt.Errorf("error truncating file: %w", err)
	}

	// Seek to the beginning
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start: %w", err)
	}

	// Write header to the empty file
	if err := s.writeHeader(); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	// Clear the cache and free space list
	s.cache = make(map[string]int64)
	s.freeSpace = make([]FreeSpace, 0)

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

	// Iterate over all cached keys
	for _, position := range s.cache {
		// Seek to the record position
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to position: %w", err)
		}

		// Read the record
		_, key, data, _, err := s.readRecord(true)
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
		_, _, data, _, err := s.readRecord(true)
		if err != nil {
			return nil, fmt.Errorf("error reading record: %w", err)
		}

		result[keyStr] = data
	}

	return result, nil
}

// BackupRecord represents a single key-value pair in the backup
type BackupRecord struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`     // Used when data is valid UTF-8 string
	ValueB64 string `json:"value_b64,omitempty"` // Used when data is binary (base64 encoded)
	IsBinary bool   `json:"is_binary"`           // True if ValueB64 is used
}

// Backup creates a JSON backup of all key-value pairs in the database
// For values <= 256 bytes, it attempts to store them as strings if they are valid UTF-8,
// otherwise stores them as base64-encoded data
// For values > 256 bytes, always uses base64 encoding
func (s *SKV) Backup(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]BackupRecord, 0, len(s.cache))

	// Iterate through all cached keys
	for key, position := range s.cache {
		// Seek to the record position
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to position for key %q: %w", key, err)
		}

		// Read the record
		_, _, data, _, err := s.readRecord(true)
		if err != nil {
			return fmt.Errorf("error reading record for key %q: %w", key, err)
		}

		record := BackupRecord{
			Key: key,
		}

		// Decide how to encode the value
		if len(data) <= 256 && utf8.Valid(data) {
			// Try to store as string if it's valid UTF-8 and small enough
			record.Value = string(data)
			record.IsBinary = false
		} else {
			// Store as base64
			record.ValueB64 = base64.StdEncoding.EncodeToString(data)
			record.IsBinary = true
		}

		records = append(records, record)
	}

	// Create the backup file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating backup file: %w", err)
	}
	defer file.Close()

	// Encode to JSON with indentation for readability
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(records); err != nil {
		return fmt.Errorf("error encoding backup to JSON: %w", err)
	}

	return nil
}

// Restore loads key-value pairs from a JSON backup file
// This will overwrite existing keys with the same name
// The database is not cleared before restore - existing keys not in the backup remain
func (s *SKV) Restore(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Open the backup file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening backup file: %w", err)
	}
	defer file.Close()

	// Decode JSON
	var records []BackupRecord
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&records); err != nil {
		return fmt.Errorf("error decoding backup JSON: %w", err)
	}

	// Restore each record
	for _, record := range records {
		var data []byte

		if record.IsBinary {
			// Decode from base64
			data, err = base64.StdEncoding.DecodeString(record.ValueB64)
			if err != nil {
				return fmt.Errorf("error decoding base64 for key %q: %w", record.Key, err)
			}
		} else {
			// Use string value directly
			data = []byte(record.Value)
		}

		// Write the record to the database
		key := []byte(record.Key)
		if err := s.putInternal(key, data); err != nil {
			return fmt.Errorf("error restoring key %q: %w", record.Key, err)
		}
	}

	return nil
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

// PutFile stores a file from disk into the database
// The file contents are read and stored as the value for the given key
// Returns error if file cannot be read or if key already exists
func (s *SKV) PutFile(key string, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}
	return s.Put([]byte(key), data)
}

// GetFile retrieves a value from the database and writes it to a file
// Creates the file if it doesn't exist, overwrites if it does
// Returns error if key not found or if file cannot be written
func (s *SKV) GetFile(key string, filePath string) error {
	data, err := s.Get([]byte(key))
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing file %s: %w", filePath, err)
	}

	return nil
}

// UpdateFile updates an existing key with the contents of a file
// Returns error if file cannot be read or if key doesn't exist
func (s *SKV) UpdateFile(key string, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}
	return s.Update([]byte(key), data)
}

// PutStream stores a new key by reading its value from an io.Reader
// This is useful for large values that shouldn't be loaded entirely into memory
// The size parameter must be the exact number of bytes that will be read from the reader
// Returns ErrKeyExists if the key already exists
func (s *SKV) PutStream(key []byte, reader io.Reader, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long (max 255 bytes)")
	}
	if size < 0 {
		return fmt.Errorf("size cannot be negative")
	}

	// Check if the key already exists in cache
	if _, exists := s.cache[string(key)]; exists {
		return ErrKeyExists
	}

	// Write the record using streaming approach
	recordPos, err := s.writeRecordStream(key, reader, uint64(size))
	if err != nil {
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	return nil
}

// PutStreamString is a convenience wrapper for PutStream using string keys
func (s *SKV) PutStreamString(key string, reader io.Reader, size int64) error {
	return s.PutStream([]byte(key), reader, size)
}

// UpdateStream updates an existing key by reading its new value from an io.Reader
// This is useful for large values that shouldn't be loaded entirely into memory
// The size parameter must be the exact number of bytes that will be read from the reader
// Returns ErrKeyNotFound if the key doesn't exist
func (s *SKV) UpdateStream(key []byte, reader io.Reader, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if size < 0 {
		return fmt.Errorf("size cannot be negative")
	}

	// Check if the key exists in cache
	if _, exists := s.cache[string(key)]; !exists {
		return ErrKeyNotFound
	}

	// Key exists, delete it first (internal version without lock)
	if err := s.deleteInternal(key); err != nil {
		return err
	}

	// Write the record using streaming approach
	recordPos, err := s.writeRecordStream(key, reader, uint64(size))
	if err != nil {
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	return nil
}

// UpdateStreamString is a convenience wrapper for UpdateStream using string keys
func (s *SKV) UpdateStreamString(key string, reader io.Reader, size int64) error {
	return s.UpdateStream([]byte(key), reader, size)
}

// writeRecordStream writes a complete record by reading data from an io.Reader
// This is used internally by PutStream and UpdateStream
// Returns the position where the record was written
func (s *SKV) writeRecordStream(key []byte, reader io.Reader, dataSize uint64) (int64, error) {
	// Determine the type based on the data size
	recordType := getRecordType(dataSize)
	neededSize := calculateRecordSize(byte(len(key)), dataSize, recordType)

	// Try to find suitable free space
	freeIdx := s.findBestFreeSpace(neededSize)

	var recordPos int64
	if freeIdx >= 0 {
		// Reuse free space
		freeSlot := s.freeSpace[freeIdx]
		recordPos = freeSlot.position

		// Seek to the free space position
		if _, err := s.file.Seek(recordPos, io.SeekStart); err != nil {
			return 0, fmt.Errorf("error seeking to free space: %w", err)
		}

		// Remove this free space from the list
		defer func() {
			s.freeSpace = append(s.freeSpace[:freeIdx], s.freeSpace[freeIdx+1:]...)
		}()

		// Calculate leftover space for later
		leftover := freeSlot.size - neededSize
		defer func() {
			if leftover > 0 {
				padding := make([]byte, leftover)
				for i := range padding {
					padding[i] = PaddingByte
				}
				s.file.Write(padding)
				s.file.Sync()
			}
		}()
	} else {
		// No suitable free space, append to end of file
		if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
			return 0, fmt.Errorf("error seeking to end of file: %w", err)
		}

		// Check if we're at the beginning
		currentPos, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, fmt.Errorf("error getting current position: %w", err)
		}

		if currentPos == 0 {
			if err := s.writeHeader(); err != nil {
				return 0, fmt.Errorf("error writing header: %w", err)
			}
		}

		recordPos, err = s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, fmt.Errorf("error getting record position: %w", err)
		}
	}

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

	// Stream the data from reader in chunks
	const bufferSize = 64 * 1024 // 64KB buffer
	var totalRead int64
	remaining := dataSize

	for remaining > 0 {
		chunkSize := bufferSize
		if remaining < bufferSize {
			chunkSize = int(remaining)
		}

		chunk := make([]byte, chunkSize)
		n, err := io.ReadFull(reader, chunk)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return 0, fmt.Errorf("reader provided less data than specified size: expected %d, got %d", dataSize, totalRead+int64(n))
			}
			return 0, fmt.Errorf("error reading data chunk: %w", err)
		}

		written, err := s.file.Write(chunk[:n])
		if err != nil {
			return 0, fmt.Errorf("error writing data chunk: %w", err)
		}
		if written != n {
			return 0, fmt.Errorf("incomplete write: expected %d, wrote %d", n, written)
		}

		totalRead += int64(n)
		remaining -= uint64(n)
	}

	// Verify no extra data in reader (best effort check)
	extraCheck := make([]byte, 1)
	n, err := reader.Read(extraCheck)
	if err == nil && n > 0 {
		return 0, fmt.Errorf("reader provided more data than specified size: expected %d bytes", dataSize)
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return 0, fmt.Errorf("error syncing to disk: %w", err)
	}

	return recordPos, nil
}

// GetStream retrieves the value for a key and writes it to an io.Writer
// This is useful for large values that shouldn't be loaded entirely into memory
// Returns the number of bytes written and any error encountered
func (s *SKV) GetStream(key []byte, writer io.Writer) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(key) == 0 {
		return 0, fmt.Errorf("key cannot be empty")
	}

	// Check cache for position
	position, found := s.cache[string(key)]
	if !found {
		return 0, ErrKeyNotFound
	}

	// Seek to the record position
	if _, err := s.file.Seek(position, io.SeekStart); err != nil {
		return 0, fmt.Errorf("error seeking to position: %w", err)
	}

	// Read record type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, typeBuf); err != nil {
		return 0, fmt.Errorf("error reading type: %w", err)
	}
	recordType := typeBuf[0]

	// Read key size
	keySizeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
		return 0, fmt.Errorf("error reading key size: %w", err)
	}
	keySize := keySizeBuf[0]

	// Skip the key
	if _, err := s.file.Seek(int64(keySize), io.SeekCurrent); err != nil {
		return 0, fmt.Errorf("error skipping key: %w", err)
	}

	// Read data size
	baseType := getBaseType(recordType)
	var dataSize uint64
	switch baseType {
	case Type1Byte:
		buf := make([]byte, 1)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(buf[0])
	case Type2Bytes:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint16(buf))
	case Type4Bytes:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint32(buf))
	case Type8Bytes:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = binary.LittleEndian.Uint64(buf)
	default:
		return 0, fmt.Errorf("unknown record type: 0x%02X", recordType)
	}

	// Stream the data in chunks to avoid loading everything into memory
	const bufferSize = 64 * 1024 // 64KB buffer
	var totalWritten int64
	remaining := dataSize

	for remaining > 0 {
		chunkSize := bufferSize
		if remaining < bufferSize {
			chunkSize = int(remaining)
		}

		chunk := make([]byte, chunkSize)
		n, err := io.ReadFull(s.file, chunk)
		if err != nil {
			return totalWritten, fmt.Errorf("error reading data chunk: %w", err)
		}

		written, err := writer.Write(chunk[:n])
		if err != nil {
			return totalWritten, fmt.Errorf("error writing to stream: %w", err)
		}

		totalWritten += int64(written)
		remaining -= uint64(n)
	}

	return totalWritten, nil
}

// GetStreamString is a convenience wrapper for GetStream using string keys
func (s *SKV) GetStreamString(key string, writer io.Writer) (int64, error) {
	return s.GetStream([]byte(key), writer)
}
