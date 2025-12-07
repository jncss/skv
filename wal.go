package skv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// WAL (Write-Ahead Log) provides durability guarantees by logging operations
// before they are applied to the main data file. In case of a crash, the WAL
// can be replayed to recover uncommitted operations.

// WAL operation types
const (
	WALOpPut    byte = 0x01 // Put operation
	WALOpDelete byte = 0x02 // Delete operation
	WALOpCommit byte = 0x03 // Commit marker (checkpoint)
)

// WAL file format:
// Header: "WAL" (3 bytes) + version (3 bytes) = 6 bytes
// Entry: opType (1) + keySize (2) + key + dataSize (4) + data + crc32 (4)

const (
	WALMagic      = "WAL"
	WALHeaderSize = 6
	WALEntryCRC   = 4 // CRC-32 size
)

// WALEntry represents a single operation in the write-ahead log
type WALEntry struct {
	OpType byte   // Operation type (Put, Delete, Commit)
	Key    []byte // Key
	Data   []byte // Data (empty for Delete operations)
}

// WAL manages the write-ahead log file
type WAL struct {
	file     *os.File
	filePath string
	logger   Logger
	mu       sync.Mutex
	enabled  bool
}

// OpenWAL opens or creates a WAL file
func OpenWAL(path string, logger Logger) (*WAL, error) {
	if logger == nil {
		logger = NullLogger()
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	// Open file with read-write, create if not exists
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	wal := &WAL{
		file:     file,
		filePath: path,
		logger:   logger,
		enabled:  true,
	}

	// Check if file is empty (newly created)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat WAL file: %w", err)
	}

	if info.Size() == 0 {
		// New file - write header
		if err := wal.writeHeader(); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write WAL header: %w", err)
		}
	} else {
		// Existing file - verify header
		if err := wal.verifyHeader(); err != nil {
			file.Close()
			return nil, fmt.Errorf("invalid WAL header: %w", err)
		}
		// Seek to end for appending
		if _, err := file.Seek(0, io.SeekEnd); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to seek to end: %w", err)
		}
	}

	return wal, nil
}

// writeHeader writes the WAL file header
func (w *WAL) writeHeader() error {
	header := make([]byte, WALHeaderSize)
	copy(header[0:3], WALMagic)
	header[3] = VersionMajor
	header[4] = VersionMinor
	header[5] = VersionPatch

	if _, err := w.file.WriteAt(header, 0); err != nil {
		return err
	}

	if err := w.file.Sync(); err != nil {
		return err
	}

	// Seek to end of header for subsequent writes
	if _, err := w.file.Seek(WALHeaderSize, io.SeekStart); err != nil {
		return err
	}

	return nil
}

// verifyHeader verifies the WAL file header
func (w *WAL) verifyHeader() error {
	header := make([]byte, WALHeaderSize)
	if _, err := w.file.ReadAt(header, 0); err != nil {
		return err
	}

	if string(header[0:3]) != WALMagic {
		return errors.New("invalid WAL magic bytes")
	}

	// Version check could be added here if needed
	return nil
}

// LogPut logs a Put operation to the WAL
func (w *WAL) LogPut(key, data []byte) error {
	if !w.enabled {
		return nil
	}

	return w.logEntry(&WALEntry{
		OpType: WALOpPut,
		Key:    key,
		Data:   data,
	})
}

// LogDelete logs a Delete operation to the WAL
func (w *WAL) LogDelete(key []byte) error {
	if !w.enabled {
		return nil
	}

	return w.logEntry(&WALEntry{
		OpType: WALOpDelete,
		Key:    key,
		Data:   nil,
	})
}

// LogCommit writes a commit marker to the WAL
func (w *WAL) LogCommit() error {
	if !w.enabled {
		return nil
	}

	err := w.logEntry(&WALEntry{
		OpType: WALOpCommit,
		Key:    nil,
		Data:   nil,
	})

	if err == nil {
		w.logger.Debug("WAL commit logged")
	}

	return err
}

// logEntry writes a single entry to the WAL
func (w *WAL) logEntry(entry *WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Calculate sizes
	keySize := len(entry.Key)
	dataSize := len(entry.Data)

	// Validate sizes
	if keySize > 0xFFFF {
		return errors.New("key too large for WAL")
	}

	// Build entry buffer
	// opType (1) + keySize (2) + key + dataSize (4) + data
	bufferSize := 1 + 2 + keySize + 4 + dataSize
	buffer := make([]byte, bufferSize)

	offset := 0

	// Write opType
	buffer[offset] = entry.OpType
	offset++

	// Write keySize (2 bytes)
	binary.LittleEndian.PutUint16(buffer[offset:], uint16(keySize))
	offset += 2

	// Write key
	copy(buffer[offset:], entry.Key)
	offset += keySize

	// Write dataSize (4 bytes)
	binary.LittleEndian.PutUint32(buffer[offset:], uint32(dataSize))
	offset += 4

	// Write data
	copy(buffer[offset:], entry.Data)

	// Calculate CRC-32 for the entire entry
	crc := crc32.ChecksumIEEE(buffer)
	crcBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(crcBytes, crc)

	// Write entry + CRC to file
	if _, err := w.file.Write(buffer); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	if _, err := w.file.Write(crcBytes); err != nil {
		return fmt.Errorf("failed to write WAL CRC: %w", err)
	}

	// Sync to disk for durability
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL: %w", err)
	}

	return nil
}

// Recover reads all entries from the WAL and returns them
// This is used during database startup to replay uncommitted operations
func (w *WAL) Recover() ([]*WALEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Seek to start of entries (after header)
	if _, err := w.file.Seek(WALHeaderSize, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek WAL: %w", err)
	}

	var entries []*WALEntry
	var corruptedEntries int

	for {
		entry, err := w.readEntry()
		if err != nil {
			if err == io.EOF {
				break
			}
			// If we encounter a corrupted entry, stop recovery at this point
			// This allows partial recovery up to the last valid entry
			corruptedEntries++
			w.logger.Warn("WAL recovery stopped at corrupted entry",
				Field{Key: "error", Value: err.Error()},
				Field{Key: "recovered_entries", Value: len(entries)},
			)
			break
		}

		entries = append(entries, entry)

		// Stop at commit marker
		if entry.OpType == WALOpCommit {
			break
		}
	}

	if len(entries) > 0 {
		w.logger.Info("WAL recovery completed",
			Field{Key: "recovered_entries", Value: len(entries)},
			Field{Key: "corrupted_entries", Value: corruptedEntries},
		)
	}

	return entries, nil
}

// readEntry reads a single entry from the WAL
func (w *WAL) readEntry() (*WALEntry, error) {
	// Read opType (1 byte)
	opTypeBuf := make([]byte, 1)
	if _, err := io.ReadFull(w.file, opTypeBuf); err != nil {
		return nil, err
	}
	opType := opTypeBuf[0]

	// Read keySize (2 bytes)
	keySizeBuf := make([]byte, 2)
	if _, err := io.ReadFull(w.file, keySizeBuf); err != nil {
		return nil, err
	}
	keySize := binary.LittleEndian.Uint16(keySizeBuf)

	// Read key
	key := make([]byte, keySize)
	if keySize > 0 {
		if _, err := io.ReadFull(w.file, key); err != nil {
			return nil, err
		}
	}

	// Read dataSize (4 bytes)
	dataSizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(w.file, dataSizeBuf); err != nil {
		return nil, err
	}
	dataSize := binary.LittleEndian.Uint32(dataSizeBuf)

	// Read data
	data := make([]byte, dataSize)
	if dataSize > 0 {
		if _, err := io.ReadFull(w.file, data); err != nil {
			return nil, err
		}
	}

	// Read CRC-32
	crcBuf := make([]byte, 4)
	if _, err := io.ReadFull(w.file, crcBuf); err != nil {
		return nil, err
	}
	storedCRC := binary.LittleEndian.Uint32(crcBuf)

	// Verify CRC
	entrySize := 1 + 2 + int(keySize) + 4 + int(dataSize)
	entryBuf := make([]byte, entrySize)
	entryBuf[0] = opType
	binary.LittleEndian.PutUint16(entryBuf[1:], keySize)
	copy(entryBuf[3:], key)
	binary.LittleEndian.PutUint32(entryBuf[3+keySize:], dataSize)
	copy(entryBuf[7+keySize:], data)

	calculatedCRC := crc32.ChecksumIEEE(entryBuf)
	if calculatedCRC != storedCRC {
		return nil, errors.New("WAL entry CRC mismatch")
	}

	return &WALEntry{
		OpType: opType,
		Key:    key,
		Data:   data,
	}, nil
}

// Truncate removes all entries from the WAL (called after successful commit)
func (w *WAL) Truncate() error {
	if !w.enabled {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Truncate to just the header
	if err := w.file.Truncate(WALHeaderSize); err != nil {
		w.logger.Error("WAL truncate failed",
			Field{Key: "error", Value: err.Error()},
		)
		return fmt.Errorf("failed to truncate WAL: %w", err)
	}

	// Seek to end of header
	if _, err := w.file.Seek(WALHeaderSize, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek WAL: %w", err)
	}

	w.logger.Debug("WAL truncated")
	return w.file.Sync()
}

// Close closes the WAL file
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Enable enables WAL logging
func (w *WAL) Enable() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.enabled = true
}

// Disable disables WAL logging (useful for bulk operations)
func (w *WAL) Disable() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.enabled = false
}

// IsEnabled returns whether WAL is currently enabled
func (w *WAL) IsEnabled() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enabled
}

// Size returns the current size of the WAL file
func (w *WAL) Size() (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := w.file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
