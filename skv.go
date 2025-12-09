package skv

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

// File header constants
const (
	HeaderMagic  = "SKV" // Magic bytes to identify SKV files
	HeaderSize   = 6     // Total header size: 3 bytes magic + 3 bytes version
	VersionMajor = 0     // Major version number
	VersionMinor = 7     // Minor version number
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

	// Compression flags (bits 5-6)
	// Bit pattern: 0x00 = none, 0x20 = snappy, 0x40 = lz4
	CompressedFlag   byte = 0x60 // Mask for compression bits (bits 5-6)
	CompressedNone   byte = 0x00 // No compression
	CompressedSnappy byte = 0x20 // Snappy compression (bit 5)
	CompressedLZ4    byte = 0x40 // LZ4 compression (bit 6)

	// Padding byte for filling small gaps
	PaddingByte byte = 0x80 // Used to fill gaps too small for a deleted record

	// Minimum record size (type + key_size + key(1) + data_size)
	MinRecordSize = 4 // Minimum size for a valid record
)

// isDeleted checks if a type has the deleted bit set
func isDeleted(recordType byte) bool {
	return (recordType & DeletedFlag) != 0
}

// getBaseType returns the base type without the deleted bit and compression bits
func getBaseType(recordType byte) byte {
	return recordType & ^(DeletedFlag | CompressedFlag)
}

// getCompressionType extracts the compression type from the record type byte
func getCompressionType(recordType byte) CompressionType {
	compressionBits := recordType & CompressedFlag
	switch compressionBits {
	case CompressedSnappy:
		return CompressionSnappy
	case CompressedLZ4:
		return CompressionLZ4
	default:
		return CompressionNone
	}
}

// setCompressionType sets the compression bits in the record type byte
func setCompressionType(recordType byte, compressionType CompressionType) byte {
	// Clear compression bits first
	recordType = recordType & ^CompressedFlag
	// Set new compression bits
	switch compressionType {
	case CompressionSnappy:
		return recordType | CompressedSnappy
	case CompressionLZ4:
		return recordType | CompressedLZ4
	default:
		return recordType
	}
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
	compressionType := getCompressionType(recordType)
	var dataSizeFieldSize uint64
	var crcSize uint64

	switch baseType {
	case Type1Byte:
		dataSizeFieldSize = 1
		crcSize = 2 // CRC-16
	case Type2Bytes:
		dataSizeFieldSize = 2
		crcSize = 4 // CRC-32
	case Type4Bytes:
		dataSizeFieldSize = 4
		crcSize = 4 // CRC-32
	case Type8Bytes:
		dataSizeFieldSize = 8
		crcSize = 4 // CRC-32
	default:
		dataSizeFieldSize = 1
		crcSize = 2 // CRC-16
	}

	// type (1) + key_size (1) + key + [original_size if compressed] + data_size_field + data + crc
	totalSize := 1 + 1 + uint64(keySize)
	if compressionType != CompressionNone {
		totalSize += dataSizeFieldSize // Add original_size field
	}
	totalSize += dataSizeFieldSize + dataSize + crcSize

	return totalSize
}

// crc16Hash implements hash.Hash for CRC-16-CCITT
type crc16Hash struct {
	crc uint16
}

// newCRC16Hash creates a new CRC-16 hash
func newCRC16Hash() *crc16Hash {
	return &crc16Hash{crc: 0xFFFF}
}

func (h *crc16Hash) Write(p []byte) (n int, err error) {
	for _, b := range p {
		h.crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if h.crc&0x8000 != 0 {
				h.crc = (h.crc << 1) ^ 0x1021
			} else {
				h.crc = h.crc << 1
			}
		}
	}
	return len(p), nil
}

func (h *crc16Hash) Sum(b []byte) []byte {
	s := h.Sum16()
	return append(b, byte(s>>8), byte(s))
}

func (h *crc16Hash) Sum16() uint16 {
	return h.crc
}

func (h *crc16Hash) Reset() {
	h.crc = 0xFFFF
}

func (h *crc16Hash) Size() int {
	return 2
}

func (h *crc16Hash) BlockSize() int {
	return 1
}

// calculateCRC16 calculates CRC-16 for the given data
func calculateCRC16(data []byte) uint16 {
	h := newCRC16Hash()
	h.Write(data)
	return h.Sum16()
}

// calculateCRC32 calculates CRC-32 for the given data using IEEE polynomial
func calculateCRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
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
	file            *os.File
	filePath        string
	cache           map[string]int64  // Cache: key -> file position
	freeSpace       []FreeSpace       // List of free spaces (deleted records)
	indexes         map[string]*Index // Secondary indexes
	wal             *WAL              // Write-ahead log for durability
	compressionType CompressionType   // Compression algorithm to use for new records
	encryptor       encryptor         // Encryptor for encrypting keys and values
	logger          Logger            // Logger for structured logging
	txCounter       uint64            // Transaction counter for generating unique transaction IDs
	mu              sync.RWMutex      // Mutex for thread-safe operations
}

// Options configures the behavior of an SKV database
type Options struct {
	// Compression specifies the compression algorithm to use for new records
	// Default: CompressionNone (no compression)
	// Available: CompressionSnappy, CompressionLZ4
	Compression CompressionType

	// Encryption specifies the encryption algorithm to use for keys and values
	// Default: EncryptionNone (no encryption)
	// Available: EncryptionAES, EncryptionSimpleCipher
	Encryption EncryptionType

	// EncryptionPassword is the password used for encryption/decryption
	// Required if Encryption is not EncryptionNone
	EncryptionPassword string

	// Logger specifies the logger to use for structured logging
	// Default: NullLogger() (no logging, zero overhead)
	// Available: NewJSONLogger(), NewTextLogger()
	Logger Logger
}

// DefaultOptions returns the default options for opening an SKV database
func DefaultOptions() *Options {
	return &Options{
		Compression:        CompressionNone,
		Encryption:         EncryptionNone,
		EncryptionPassword: "",
		Logger:             NullLogger(),
	}
}

// Open opens or creates a .skv file and returns an SKV object
func Open(name string) (*SKV, error) {
	return OpenWithOptions(name, DefaultOptions())
}

// OpenWithOptions opens or creates a .skv file with custom options
func OpenWithOptions(name string, opts *Options) (*SKV, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	// Fill in missing fields with defaults
	if opts.Logger == nil {
		opts.Logger = NullLogger()
	}
	// Add .skv extension if it doesn't have it
	if len(name) < 4 || name[len(name)-4:] != ".skv" {
		name += ".skv"
	}

	// Open or create the file with read/write permissions
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", name, err)
	}

	// Open WAL file (same name with .wal extension)
	walPath := name + ".wal"
	wal, err := OpenWAL(walPath, opts.Logger)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error opening WAL: %w", err)
	}

	// Create encryptor
	enc, err := createEncryptor(opts.Encryption, opts.EncryptionPassword)
	if err != nil {
		file.Close()
		wal.Close()
		return nil, fmt.Errorf("error creating encryptor: %w", err)
	}

	skv := &SKV{
		file:            file,
		filePath:        name,
		cache:           make(map[string]int64),
		freeSpace:       make([]FreeSpace, 0),
		indexes:         make(map[string]*Index),
		wal:             wal,
		compressionType: opts.Compression,
		encryptor:       enc,
		logger:          opts.Logger,
	}

	// Check if file is new or existing
	info, err := file.Stat()
	if err != nil {
		file.Close()
		wal.Close()
		return nil, fmt.Errorf("error getting file info: %w", err)
	}

	if info.Size() == 0 {
		// New file - write header
		if err := skv.writeHeader(); err != nil {
			file.Close()
			wal.Close()
			return nil, fmt.Errorf("error writing header: %w", err)
		}
	} else {
		// Existing file - verify header
		if err := skv.verifyHeader(); err != nil {
			file.Close()
			wal.Close()
			return nil, fmt.Errorf("error verifying header: %w", err)
		}

		// Recover from WAL if there are uncommitted operations
		if err := skv.recoverFromWAL(); err != nil {
			file.Close()
			wal.Close()
			return nil, fmt.Errorf("error recovering from WAL: %w", err)
		}
	}

	// Build cache by scanning the file
	if err := skv.rebuildCache(); err != nil {
		file.Close()
		wal.Close()
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

// recoverFromWAL replays operations from the WAL if there are uncommitted changes
func (s *SKV) recoverFromWAL() error {
	// Check if WAL has any entries
	walSize, err := s.wal.Size()
	if err != nil {
		return fmt.Errorf("error getting WAL size: %w", err)
	}

	// If WAL only has header, nothing to recover
	if walSize <= WALHeaderSize {
		return nil
	}

	// Recover entries from WAL
	entries, err := s.wal.Recover()
	if err != nil {
		return fmt.Errorf("error recovering from WAL: %w", err)
	}

	// If no entries or only commit marker, nothing to do
	if len(entries) == 0 {
		return nil
	}

	// Disable WAL during recovery to avoid recursive logging
	s.wal.Disable()
	defer s.wal.Enable()

	// Track active transactions: txID -> list of operations
	activeTxs := make(map[uint64][]WALEntry)
	var currentTxID uint64 = 0

	// Replay operations
	for i, entry := range entries {
		switch entry.OpType {
		case WALOpBeginTx:
			// Start a new transaction
			if len(entry.Key) == 8 {
				txID := binary.BigEndian.Uint64(entry.Key)
				activeTxs[txID] = make([]WALEntry, 0)
				currentTxID = txID
				if s.logger != nil {
					s.logger.Debug("WAL recovery: transaction begin",
						Field{Key: "tx_id", Value: txID})
				}
			}

		case WALOpCommitTx:
			// Commit a transaction - apply all its operations
			if len(entry.Key) == 8 {
				txID := binary.BigEndian.Uint64(entry.Key)
				if ops, exists := activeTxs[txID]; exists {
					// Apply all operations in the transaction
					for _, op := range ops {
						if err := s.applyWALOperation(op); err != nil {
							if s.logger != nil {
								s.logger.Warn("WAL recovery: error applying transaction operation",
									Field{Key: "tx_id", Value: txID},
									Field{Key: "error", Value: err.Error()})
							}
						}
					}
					delete(activeTxs, txID)
					if s.logger != nil {
						s.logger.Debug("WAL recovery: transaction committed",
							Field{Key: "tx_id", Value: txID},
							Field{Key: "operations", Value: len(ops)})
					}
				}
				currentTxID = 0
			}

		case WALOpRollbackTx:
			// Rollback a transaction - discard all its operations
			if len(entry.Key) == 8 {
				txID := binary.BigEndian.Uint64(entry.Key)
				delete(activeTxs, txID)
				currentTxID = 0
				if s.logger != nil {
					s.logger.Debug("WAL recovery: transaction rolled back",
						Field{Key: "tx_id", Value: txID})
				}
			}

		case WALOpPut, WALOpDelete:
			if currentTxID != 0 {
				// Operation is part of a transaction - buffer it
				activeTxs[currentTxID] = append(activeTxs[currentTxID], *entry)
			} else {
				// Standalone operation - apply immediately
				if err := s.applyWALOperation(*entry); err != nil {
					if s.logger != nil {
						s.logger.Warn("WAL recovery: error applying standalone operation",
							Field{Key: "error", Value: err.Error()})
					}
				}
			}

		case WALOpCommit:
			// Old-style commit marker - stop processing here
			// (for backward compatibility with pre-transaction WAL)
			if s.logger != nil {
				s.logger.Debug("WAL recovery: found old-style commit marker, stopping",
					Field{Key: "entry_index", Value: i})
			}
			goto endRecovery
		}
	}

endRecovery:
	// Any transactions that were not committed or rolled back are discarded
	if len(activeTxs) > 0 {
		if s.logger != nil {
			s.logger.Warn("WAL recovery: discarding incomplete transactions",
				Field{Key: "count", Value: len(activeTxs)})
		}
	}

	// Clear WAL after successful recovery
	if err := s.wal.Truncate(); err != nil {
		return fmt.Errorf("error truncating WAL after recovery: %w", err)
	}

	return nil
}

// applyWALOperation applies a single WAL operation during recovery
func (s *SKV) applyWALOperation(entry WALEntry) error {
	switch entry.OpType {
	case WALOpPut:
		// Use putInternal - ignore ErrKeyExists since the operation may have
		// already been applied before the crash (idempotent recovery)
		if err := s.putInternal(entry.Key, entry.Data); err != nil && err != ErrKeyExists {
			return fmt.Errorf("error replaying put from WAL: %w", err)
		}
	case WALOpDelete:
		if err := s.deleteInternal(entry.Key); err != nil && err != ErrKeyNotFound {
			return fmt.Errorf("error replaying delete from WAL: %w", err)
		}
	}
	return nil
}

// Close closes the database file
func (s *SKV) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var closeErr error

	// Close WAL first
	if s.wal != nil {
		if err := s.wal.Close(); err != nil {
			closeErr = fmt.Errorf("error closing WAL: %w", err)
		}
	}

	// Close main file
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			if closeErr != nil {
				return fmt.Errorf("%w; also error closing file: %v", closeErr, err)
			}
			return err
		}
	}

	return closeErr
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
		// Even if compact fails, try to close files
		if s.wal != nil {
			s.wal.Close()
		}
		if s.file != nil {
			s.file.Close()
		}
		return fmt.Errorf("error compacting before close: %w", err)
	}

	// Close WAL
	var closeErr error
	if s.wal != nil {
		if err := s.wal.Close(); err != nil {
			closeErr = fmt.Errorf("error closing WAL: %w", err)
		}
	}

	// Close main file
	if err := s.file.Close(); err != nil {
		if closeErr != nil {
			return fmt.Errorf("%w; also error closing file: %v", closeErr, err)
		}
		return err
	}

	return closeErr
}

// writeRecordAtPosition writes a complete record (type, key, data) at the current file position
// Returns the position where the record was written
// Uses streaming CRC calculation to avoid loading large records into memory
func (s *SKV) writeRecordAtPosition(key []byte, data []byte, skipEncryption bool) (int64, uint64, error) {
	// Encrypt key and data BEFORE compression (if encryption is enabled and not skipped)
	var encryptedKey, encryptedData []byte
	var err error

	if skipEncryption {
		// Skip encryption - use data as-is (for backup/restore)
		encryptedKey = key
		encryptedData = data
	} else {
		// Normal path: encrypt if configured
		encryptedKey, err = s.encryptor.encrypt(key)
		if err != nil {
			return 0, 0, fmt.Errorf("key encryption error: %w", err)
		}

		encryptedData, err = s.encryptor.encrypt(data)
		if err != nil {
			return 0, 0, fmt.Errorf("data encryption error: %w", err)
		}
	}

	// Check encrypted key size doesn't exceed 255 bytes
	if len(encryptedKey) > 255 {
		return 0, 0, fmt.Errorf("encrypted key too long (max 255 bytes, got %d)", len(encryptedKey))
	}

	// Try to compress encrypted data if compression is enabled
	originalSize := uint64(len(encryptedData))
	compressedData, actualCompressionType, err := compress(encryptedData, s.compressionType)
	if err != nil {
		return 0, 0, fmt.Errorf("compression error: %w", err)
	}

	// Use compressed data if compression was applied
	dataToWrite := encryptedData
	if actualCompressionType != CompressionNone {
		dataToWrite = compressedData
	}

	// Determine the type based on BOTH compressed data size AND original size
	// We need to use the larger of the two to ensure both fit in the size fields
	dataSize := uint64(len(dataToWrite))
	maxSize := dataSize
	if actualCompressionType != CompressionNone && originalSize > maxSize {
		maxSize = originalSize
	}

	var recordType byte
	switch {
	case maxSize <= 0xFF: // 255 bytes
		recordType = Type1Byte
	case maxSize <= 0xFFFF: // 64KB
		recordType = Type2Bytes
	case maxSize <= 0xFFFFFFFF: // 4GB
		recordType = Type4Bytes
	default:
		recordType = Type8Bytes
	}

	// Set compression flag in record type
	recordType = setCompressionType(recordType, actualCompressionType)

	// Save position before writing
	recordPos, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting current position: %w", err)
	}

	// Initialize streaming CRC calculation based on record type
	var hasher16 *crc16Hash
	var hasher32 hash.Hash32
	baseType := getBaseType(recordType)
	if baseType == Type1Byte {
		hasher16 = newCRC16Hash()
	} else {
		hasher32 = crc32.NewIEEE()
	}

	// Helper function to write and update CRC
	writeAndHash := func(buf []byte) error {
		if _, err := s.file.Write(buf); err != nil {
			return err
		}
		if hasher16 != nil {
			hasher16.Write(buf)
		} else {
			hasher32.Write(buf)
		}
		return nil
	}

	// Write the type
	typeBuf := []byte{recordType}
	if err := writeAndHash(typeBuf); err != nil {
		return 0, 0, fmt.Errorf("error writing type: %w", err)
	}

	// Write the encrypted key size
	keySize := byte(len(encryptedKey))
	keySizeBuf := []byte{keySize}
	if err := writeAndHash(keySizeBuf); err != nil {
		return 0, 0, fmt.Errorf("error writing key size: %w", err)
	}

	// Write the encrypted key
	if err := writeAndHash(encryptedKey); err != nil {
		return 0, 0, fmt.Errorf("error writing key: %w", err)
	}

	// If compressed, write original size first
	if actualCompressionType != CompressionNone {
		var originalSizeBuf []byte
		switch baseType {
		case Type1Byte:
			originalSizeBuf = []byte{byte(originalSize)}
		case Type2Bytes:
			originalSizeBuf = make([]byte, 2)
			binary.LittleEndian.PutUint16(originalSizeBuf, uint16(originalSize))
		case Type4Bytes:
			originalSizeBuf = make([]byte, 4)
			binary.LittleEndian.PutUint32(originalSizeBuf, uint32(originalSize))
		case Type8Bytes:
			originalSizeBuf = make([]byte, 8)
			binary.LittleEndian.PutUint64(originalSizeBuf, originalSize)
		}
		if err := writeAndHash(originalSizeBuf); err != nil {
			return 0, 0, fmt.Errorf("error writing original size: %w", err)
		}
	}

	// Write the compressed data size according to the type
	var dataSizeBuf []byte
	switch baseType {
	case Type1Byte:
		dataSizeBuf = []byte{byte(dataSize)}
	case Type2Bytes:
		dataSizeBuf = make([]byte, 2)
		binary.LittleEndian.PutUint16(dataSizeBuf, uint16(dataSize))
	case Type4Bytes:
		dataSizeBuf = make([]byte, 4)
		binary.LittleEndian.PutUint32(dataSizeBuf, uint32(dataSize))
	case Type8Bytes:
		dataSizeBuf = make([]byte, 8)
		binary.LittleEndian.PutUint64(dataSizeBuf, dataSize)
	}
	if err := writeAndHash(dataSizeBuf); err != nil {
		return 0, 0, fmt.Errorf("error writing data size: %w", err)
	}

	// Write the data in chunks to avoid memory pressure for large values
	const bufferSize = 64 * 1024 // 64KB buffer
	for offset := 0; offset < len(dataToWrite); offset += bufferSize {
		end := offset + bufferSize
		if end > len(dataToWrite) {
			end = len(dataToWrite)
		}
		if err := writeAndHash(dataToWrite[offset:end]); err != nil {
			return 0, 0, fmt.Errorf("error writing data chunk: %w", err)
		}
	}

	// Calculate and write CRC
	if baseType == Type1Byte {
		// CRC-16
		crc := hasher16.Sum16()
		crcBuf := make([]byte, 2)
		binary.LittleEndian.PutUint16(crcBuf, crc)
		if _, err := s.file.Write(crcBuf); err != nil {
			return 0, 0, fmt.Errorf("error writing CRC: %w", err)
		}
	} else {
		// CRC-32
		crc := hasher32.Sum32()
		crcBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(crcBuf, crc)
		if _, err := s.file.Write(crcBuf); err != nil {
			return 0, 0, fmt.Errorf("error writing CRC: %w", err)
		}
	}

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		return 0, 0, fmt.Errorf("error syncing to disk: %w", err)
	}

	// Calculate actual record size written (using encrypted key size)
	actualRecordSize := calculateRecordSize(byte(len(encryptedKey)), dataSize, recordType)

	return recordPos, actualRecordSize, nil
}

// writeRecord writes a complete record (type, key, data)
// Returns the position where the record was written
// Tries to reuse free space if available, otherwise appends to end of file
func (s *SKV) writeRecord(key []byte, data []byte) (int64, error) {
	// Calculate estimated size for finding free space
	// This may be larger than actual if compression reduces size
	estimatedRecordType := getRecordType(uint64(len(data)))
	estimatedSize := calculateRecordSize(byte(len(key)), uint64(len(data)), estimatedRecordType)

	// Try to find suitable free space (using estimated size)
	freeIdx := s.findBestFreeSpace(estimatedSize)

	if freeIdx >= 0 {
		// Reuse free space
		freeSlot := s.freeSpace[freeIdx]
		recordPos := freeSlot.position

		// Seek to the free space position
		if _, err := s.file.Seek(recordPos, io.SeekStart); err != nil {
			return 0, fmt.Errorf("error seeking to free space: %w", err)
		}

		// Write the record and get actual size written
		_, actualSize, err := s.writeRecordAtPosition(key, data, false)
		if err != nil {
			return 0, err
		}

		// Calculate leftover space using actual size written
		leftover := freeSlot.size - actualSize
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

	pos, _, err := s.writeRecordAtPosition(key, data, false)
	return pos, err
}

// readRecord reads a complete record from the current file position
// If readData is false, the data portion is skipped for efficiency
// If skipEncryption is true, returns encrypted data as-is without decryption (for backup)
// Returns: recordType, key, data, recordSize, error
func (s *SKV) readRecord(readData bool, skipEncryption bool) (recordType byte, key []byte, data []byte, recordSize uint64, err error) {
	// Buffer to accumulate record bytes for CRC verification
	var recordBuf []byte

	// Read type
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, typeBuf); err != nil {
		if err == io.EOF {
			return 0, nil, nil, 0, io.EOF // Return EOF directly
		}
		return 0, nil, nil, 0, fmt.Errorf("error reading type: %w", err)
	}
	recordType = typeBuf[0]
	recordBuf = append(recordBuf, typeBuf...)

	// Read key size
	keySizeBuf := make([]byte, 1)
	if _, err := io.ReadFull(s.file, keySizeBuf); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("error reading key size: %w", err)
	}
	keySize := keySizeBuf[0]
	recordBuf = append(recordBuf, keySizeBuf...)

	// Read encrypted key
	encryptedKey := make([]byte, keySize)
	if _, err := io.ReadFull(s.file, encryptedKey); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("error reading key: %w", err)
	}
	recordBuf = append(recordBuf, encryptedKey...)

	// Decrypt key (unless skipEncryption is true)
	if skipEncryption {
		key = encryptedKey
	} else {
		key, err = s.encryptor.decrypt(encryptedKey)
		if err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error decrypting key: %w", err)
		}
	}

	// Get compression type from record type
	baseType := getBaseType(recordType)
	compressionType := getCompressionType(recordType)

	// Read original size if compressed
	var originalSize uint64
	var originalSizeBuf []byte
	if compressionType != CompressionNone {
		switch baseType {
		case Type1Byte:
			originalSizeBuf = make([]byte, 1)
			if _, err := io.ReadFull(s.file, originalSizeBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading original size: %w", err)
			}
			originalSize = uint64(originalSizeBuf[0])
		case Type2Bytes:
			originalSizeBuf = make([]byte, 2)
			if _, err := io.ReadFull(s.file, originalSizeBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading original size: %w", err)
			}
			originalSize = uint64(binary.LittleEndian.Uint16(originalSizeBuf))
		case Type4Bytes:
			originalSizeBuf = make([]byte, 4)
			if _, err := io.ReadFull(s.file, originalSizeBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading original size: %w", err)
			}
			originalSize = uint64(binary.LittleEndian.Uint32(originalSizeBuf))
		case Type8Bytes:
			originalSizeBuf = make([]byte, 8)
			if _, err := io.ReadFull(s.file, originalSizeBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading original size: %w", err)
			}
			originalSize = binary.LittleEndian.Uint64(originalSizeBuf)
		default:
			return 0, nil, nil, 0, fmt.Errorf("unknown record type: 0x%02X", recordType)
		}
		recordBuf = append(recordBuf, originalSizeBuf...)
	}

	// Read data size
	var dataSize uint64
	var dataSizeBuf []byte

	switch baseType {
	case Type1Byte:
		dataSizeBuf = make([]byte, 1)
		if _, err := io.ReadFull(s.file, dataSizeBuf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(dataSizeBuf[0])
	case Type2Bytes:
		dataSizeBuf = make([]byte, 2)
		if _, err := io.ReadFull(s.file, dataSizeBuf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint16(dataSizeBuf))
	case Type4Bytes:
		dataSizeBuf = make([]byte, 4)
		if _, err := io.ReadFull(s.file, dataSizeBuf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint32(dataSizeBuf))
	case Type8Bytes:
		dataSizeBuf = make([]byte, 8)
		if _, err := io.ReadFull(s.file, dataSizeBuf); err != nil {
			return 0, nil, nil, 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = binary.LittleEndian.Uint64(dataSizeBuf)
	default:
		return 0, nil, nil, 0, fmt.Errorf("unknown record type: 0x%02X", recordType)
	}
	recordBuf = append(recordBuf, dataSizeBuf...)

	// Calculate total record size
	recordSize = calculateRecordSize(keySize, dataSize, recordType)

	// Optimization: If we don't need the data and only need to verify CRC,
	// we can use incremental CRC to avoid loading all data into memory
	if !readData && dataSize > 0 {
		// Skip verification for deleted records
		if !isDeleted(recordType) {
			// Use incremental CRC verification without loading data into memory
			var hasher hash.Hash
			if baseType == Type1Byte {
				hasher = newCRC16Hash()
			} else {
				hasher = crc32.NewIEEE()
			}

			// Write header to hasher (type, keySize, key, dataSize)
			hasher.Write(recordBuf)

			// Stream data through hasher in chunks without keeping in memory
			const chunkSize = 64 * 1024
			remaining := dataSize
			chunk := make([]byte, chunkSize)

			for remaining > 0 {
				toRead := remaining
				if toRead > chunkSize {
					toRead = chunkSize
				}

				n, err := io.ReadFull(s.file, chunk[:toRead])
				if err != nil {
					return 0, nil, nil, 0, fmt.Errorf("error reading data chunk: %w", err)
				}
				hasher.Write(chunk[:n])
				remaining -= uint64(n)
			}

			// Read and verify CRC
			if baseType == Type1Byte {
				crcBuf := make([]byte, 2)
				if _, err := io.ReadFull(s.file, crcBuf); err != nil {
					return 0, nil, nil, 0, fmt.Errorf("error reading CRC: %w", err)
				}
				storedCRC := binary.LittleEndian.Uint16(crcBuf)
				h := hasher.(*crc16Hash)
				calculatedCRC := h.Sum16()
				if storedCRC != calculatedCRC {
					return 0, nil, nil, 0, fmt.Errorf("CRC mismatch: expected 0x%04X, got 0x%04X (record may be corrupted)", storedCRC, calculatedCRC)
				}
			} else {
				crcBuf := make([]byte, 4)
				if _, err := io.ReadFull(s.file, crcBuf); err != nil {
					return 0, nil, nil, 0, fmt.Errorf("error reading CRC: %w", err)
				}
				storedCRC := binary.LittleEndian.Uint32(crcBuf)
				h := hasher.(hash.Hash32)
				calculatedCRC := h.Sum32()
				if storedCRC != calculatedCRC {
					return 0, nil, nil, 0, fmt.Errorf("CRC mismatch: expected 0x%08X, got 0x%08X (record may be corrupted)", storedCRC, calculatedCRC)
				}
			}
		} else {
			// Deleted record - skip data and CRC without verification
			skipBytes := dataSize
			if baseType == Type1Byte {
				skipBytes += 2 // CRC-16
			} else {
				skipBytes += 4 // CRC-32
			}
			if _, err := s.file.Seek(int64(skipBytes), io.SeekCurrent); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error skipping data and CRC: %w", err)
			}
		}

		// Data not requested and not read
		data = nil

	} else {
		// readData=true or dataSize=0: read data normally
		tempData := make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := io.ReadFull(s.file, tempData); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading data: %w", err)
			}
		}
		recordBuf = append(recordBuf, tempData...)

		// Only return data if requested
		if readData {
			// Decompress if necessary
			if compressionType != CompressionNone && dataSize > 0 {
				decompressed, err := decompress(tempData, compressionType, int(originalSize))
				if err != nil {
					return 0, nil, nil, 0, fmt.Errorf("error decompressing data: %w", err)
				}
				data = decompressed
			} else {
				data = tempData
			}

			// Decrypt data after decompression (unless skipEncryption is true)
			if skipEncryption {
				// Keep data as-is (encrypted) for backup
			} else {
				decryptedData, err := s.encryptor.decrypt(data)
				if err != nil {
					return 0, nil, nil, 0, fmt.Errorf("error decrypting data: %w", err)
				}
				data = decryptedData
			}
		}

		// Read and verify CRC (skip verification for deleted records since type byte was modified)
		if baseType == Type1Byte {
			// CRC-16
			crcBuf := make([]byte, 2)
			if _, err := io.ReadFull(s.file, crcBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading CRC: %w", err)
			}
			// Only verify CRC for active records
			if !isDeleted(recordType) {
				storedCRC := binary.LittleEndian.Uint16(crcBuf)
				calculatedCRC := calculateCRC16(recordBuf)
				if storedCRC != calculatedCRC {
					return 0, nil, nil, 0, fmt.Errorf("CRC mismatch: expected 0x%04X, got 0x%04X (record may be corrupted)", storedCRC, calculatedCRC)
				}
			}
		} else {
			// CRC-32
			crcBuf := make([]byte, 4)
			if _, err := io.ReadFull(s.file, crcBuf); err != nil {
				return 0, nil, nil, 0, fmt.Errorf("error reading CRC: %w", err)
			}
			// Only verify CRC for active records
			if !isDeleted(recordType) {
				storedCRC := binary.LittleEndian.Uint32(crcBuf)
				calculatedCRC := calculateCRC32(recordBuf)
				if storedCRC != calculatedCRC {
					return 0, nil, nil, 0, fmt.Errorf("CRC mismatch: expected 0x%08X, got 0x%08X (record may be corrupted)", storedCRC, calculatedCRC)
				}
			}
		}
	}

	return recordType, key, data, recordSize, nil
}

// PutCtx stores a new key with its value with context support
// Returns ErrKeyExists if the key already exists
// Returns ctx.Err() if context is cancelled
func (s *SKV) PutCtx(ctx context.Context, key []byte, data []byte) error {
	startTime := time.Now()

	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context after acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

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

	// Log to WAL first
	if err := s.wal.LogPut(key, data); err != nil {
		return fmt.Errorf("error logging to WAL: %w", err)
	}

	// Write the record
	recordPos, err := s.writeRecord(key, data)
	if err != nil {
		s.logger.Error("Put failed",
			Field{Key: "key", Value: string(key)},
			Field{Key: "data_size", Value: len(data)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	// Update indexes
	s.updateIndexes(key, data)

	// Commit to WAL (operation successful)
	if err := s.wal.LogCommit(); err != nil {
		return fmt.Errorf("error committing to WAL: %w", err)
	}

	// Truncate WAL after successful commit
	if err := s.wal.Truncate(); err != nil {
		return fmt.Errorf("error truncating WAL: %w", err)
	}

	s.logger.Debug("Put successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "data_size", Value: len(data)},
		Field{Key: "compression", Value: s.compressionType.String()},
		Field{Key: "position", Value: recordPos},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return nil
}

// Put stores a new key with its value
// Returns ErrKeyExists if the key already exists
func (s *SKV) Put(key []byte, data []byte) error {
	return s.PutCtx(context.Background(), key, data)
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

	// Update indexes
	s.updateIndexes(key, data)

	return nil
}

// UpdateCtx modifies the value of an existing key with context support
// Returns ErrKeyNotFound if the key doesn't exist
// Returns ctx.Err() if context is cancelled
func (s *SKV) UpdateCtx(ctx context.Context, key []byte, data []byte) error {
	startTime := time.Now()

	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context after acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if the key exists in cache
	if _, exists := s.cache[string(key)]; !exists {
		return ErrKeyNotFound
	}

	// Log to WAL first
	if err := s.wal.LogPut(key, data); err != nil {
		return fmt.Errorf("error logging to WAL: %w", err)
	}

	// Key exists, delete it first (internal version without lock)
	if err := s.deleteInternal(key); err != nil {
		s.logger.Error("Update failed during delete",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return err
	}

	// Write the record
	recordPos, err := s.writeRecord(key, data)
	if err != nil {
		s.logger.Error("Update failed during write",
			Field{Key: "key", Value: string(key)},
			Field{Key: "data_size", Value: len(data)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return err
	}

	// Update cache with record start position
	s.cache[string(key)] = recordPos

	// Update indexes
	s.updateIndexes(key, data)

	// Commit to WAL (operation successful)
	if err := s.wal.LogCommit(); err != nil {
		return fmt.Errorf("error committing to WAL: %w", err)
	}

	// Truncate WAL after successful commit
	if err := s.wal.Truncate(); err != nil {
		return fmt.Errorf("error truncating WAL: %w", err)
	}

	s.logger.Debug("Update successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "data_size", Value: len(data)},
		Field{Key: "position", Value: recordPos},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return nil
}

// Update modifies the value of an existing key
// Returns ErrKeyNotFound if the key doesn't exist
func (s *SKV) Update(key []byte, data []byte) error {
	return s.UpdateCtx(context.Background(), key, data)
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
		recordType, key, _, recordSize, err := s.readRecord(false, false)
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
				position: currentPos,
				size:     totalFreeSize,
			})
		} else {
			// Add or update in cache (currentPos is already after padding)
			s.cache[keyStr] = currentPos
		}
	}

	return nil
} // ErrKeyNotFound is returned when the key is not found
var ErrKeyNotFound = errors.New("key not found")

// ErrKeyExists is returned when trying to insert a key that already exists
var ErrKeyExists = errors.New("key already exists")

// GetCtx retrieves the value associated with a key with context support
// Returns ErrKeyNotFound if the key doesn't exist or is deleted
// Returns ctx.Err() if context is cancelled
func (s *SKV) GetCtx(ctx context.Context, key []byte) ([]byte, error) {
	startTime := time.Now()

	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context after acquiring lock
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	// Check cache for position
	position, found := s.cache[string(key)]
	if !found {
		s.logger.Debug("Get failed: key not found",
			Field{Key: "key", Value: string(key)},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return nil, ErrKeyNotFound
	}

	// Read from file at cached position
	if _, err := s.file.Seek(position, io.SeekStart); err != nil {
		s.logger.Error("Get failed: seek error",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return nil, fmt.Errorf("error seeking to position: %w", err)
	}

	// Read the record
	_, _, data, _, err := s.readRecord(true, false)
	if err != nil {
		s.logger.Error("Get failed: read error",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return nil, err
	}

	s.logger.Debug("Get successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "data_size", Value: len(data)},
		Field{Key: "cache_hit", Value: true},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return data, nil
}

// Get retrieves the value associated with a key
// Returns ErrKeyNotFound if the key doesn't exist or is deleted
func (s *SKV) Get(key []byte) ([]byte, error) {
	return s.GetCtx(context.Background(), key)
}

// DeleteCtx deletes a key by setting the deleted bit in its record with context support
// Returns ctx.Err() if context is cancelled
func (s *SKV) DeleteCtx(ctx context.Context, key []byte) error {
	startTime := time.Now()

	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context after acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	// Log to WAL first
	if err := s.wal.LogDelete(key); err != nil {
		return fmt.Errorf("error logging to WAL: %w", err)
	}

	// Perform the delete
	if err := s.deleteInternal(key); err != nil {
		s.logger.Error("Delete failed",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return err
	}

	// Commit to WAL (operation successful)
	if err := s.wal.LogCommit(); err != nil {
		return fmt.Errorf("error committing to WAL: %w", err)
	}

	// Truncate WAL after successful commit
	if err := s.wal.Truncate(); err != nil {
		return fmt.Errorf("error truncating WAL: %w", err)
	}

	s.logger.Debug("Delete successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return nil
}

// Delete deletes a key by setting the deleted bit in its record
func (s *SKV) Delete(key []byte) error {
	return s.DeleteCtx(context.Background(), key)
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

	// Read the record to get its size (CRC verification will fail after we modify type byte, but that's ok)
	recordType, _, _, recordSize, err := s.readRecord(false, false)
	if err != nil {
		return fmt.Errorf("error reading record: %w", err)
	}

	// Set the deleted bit
	deletedType := recordType | DeletedFlag

	// Go back to overwrite just the type byte
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

	// Remove from indexes
	s.removeFromIndexes(keyStr)

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
	startTime := time.Now()

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
	var totalDataSize int64    // Uncompressed data size for averages
	var activeRecordSize int64 // On-disk size of active records (compressed)

	// Read all records in the file
	for {
		// Skip any padding bytes
		paddingCount, err := s.skipPaddingBytes()
		if err != nil {
			if err == io.EOF {
				// Add final padding before breaking
				stats.PaddingBytes += paddingCount
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
		recordType, key, data, recordSize, err := s.readRecord(true, false)
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			// Any other error indicates corruption (e.g., truncated record, CRC mismatch, invalid format)
			return nil, fmt.Errorf("database corruption detected at position 0x%x: %w", posAfterPadding, err)
		}

		// Count the record
		stats.TotalRecords++
		totalKeySize += int64(len(key))
		totalDataSize += int64(len(data)) // Uncompressed size

		if isDeleted(recordType) {
			stats.DeletedRecords++
			stats.WastedSpace += int64(recordSize)
		} else {
			stats.ActiveRecords++
			activeRecordSize += int64(recordSize) // On-disk size (compressed + metadata)
		}
	}

	// Calculate data size (all records, excluding header and padding)
	stats.DataSize = stats.FileSize - stats.HeaderSize - stats.PaddingBytes

	// Calculate wasted space percentage and efficiency
	usableSpace := stats.FileSize - stats.HeaderSize
	if usableSpace > 0 {
		totalWasted := stats.WastedSpace + stats.PaddingBytes
		stats.WastedPercent = (float64(totalWasted) / float64(usableSpace)) * 100.0
		// Efficiency = percentage of disk space used by active records (compressed)
		stats.Efficiency = (float64(activeRecordSize) / float64(usableSpace)) * 100.0
	}

	// Calculate averages
	if stats.TotalRecords > 0 {
		stats.AverageKeySize = float64(totalKeySize) / float64(stats.TotalRecords)
		stats.AverageDataSize = float64(totalDataSize) / float64(stats.TotalRecords)
	}

	s.logger.Info("Verify completed",
		Field{Key: "file_size", Value: stats.FileSize},
		Field{Key: "total_records", Value: stats.TotalRecords},
		Field{Key: "active_records", Value: stats.ActiveRecords},
		Field{Key: "deleted_records", Value: stats.DeletedRecords},
		Field{Key: "wasted_percent", Value: fmt.Sprintf("%.2f", stats.WastedPercent)},
		Field{Key: "efficiency", Value: fmt.Sprintf("%.2f", stats.Efficiency)},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return stats, nil
}

// CompactCtx removes deleted records by creating a new file with only active records with context support
// For keys that appear multiple times, only the last occurrence is kept
// Returns ctx.Err() if context is cancelled
func (s *SKV) CompactCtx(ctx context.Context) error {
	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context after acquiring lock
	if err := ctx.Err(); err != nil {
		return err
	}

	return s.compactInternalCtx(ctx)
}

// Compact removes deleted records by creating a new file with only active records
// For keys that appear multiple times, only the last occurrence is kept
func (s *SKV) Compact() error {
	return s.CompactCtx(context.Background())
}

// compactInternalCtx is the internal implementation of Compact with context support
// Used by CompactCtx and CloseWithCompact
//
// This function uses a safe atomic approach:
// 1. Write compacted data to a temporary file
// 2. Sync the temporary file to disk
// 3. Close the original file
// 4. Atomically rename temp file over original (OS atomic operation)
// 5. Reopen the file
//
// This ensures that if there's any failure during compaction, the original
// file remains intact and no data is lost.
func (s *SKV) compactInternalCtx(ctx context.Context) error {
	startTime := time.Now()

	// Get initial stats
	initialSize, _ := s.file.Stat()
	var beforeBytes int64 = 0
	if initialSize != nil {
		beforeBytes = initialSize.Size()
	}

	// Create temporary file in the same directory as the database
	// (important for atomic rename to work on all platforms)
	tmpFile, err := os.CreateTemp(filepath.Dir(s.filePath), ".skv-compact-*.tmp")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	tmpFilename := tmpFile.Name()

	// Ensure temp file is cleaned up on error
	defer func() {
		// Only remove if it still exists (won't exist after successful rename)
		if _, err := os.Stat(tmpFilename); err == nil {
			os.Remove(tmpFilename)
		}
	}()

	// Write header to temp file
	header := make([]byte, HeaderSize)
	copy(header[0:3], HeaderMagic)
	header[3] = byte(VersionMajor)
	header[4] = byte(VersionMinor)
	header[5] = byte(VersionPatch)
	if _, err := tmpFile.Write(header); err != nil {
		tmpFile.Close()
		s.logger.Error("Compact failed during header write",
			Field{Key: "error", Value: err.Error()},
			Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
		)
		return fmt.Errorf("error writing header to temp file: %w", err)
	}

	// Process records one by one using streaming approach
	newCache := make(map[string]int64)

	for keyStr, position := range s.cache {
		// Check context periodically
		if err := ctx.Err(); err != nil {
			tmpFile.Close()
			return err
		}

		// Seek to record position in original file
		if _, err := s.file.Seek(position, io.SeekStart); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error seeking to position: %w", err)
		}

		// Read record from original file
		_, key, data, _, err := s.readRecord(true, false)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("error reading record: %w", err)
		}

		// Get current position in temp file (this is the record position)
		pos, err := tmpFile.Seek(0, io.SeekCurrent)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("error getting position in temp file: %w", err)
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

		// Initialize streaming CRC calculation based on record type
		var hasher16 *crc16Hash
		var hasher32 hash.Hash32
		if recordType == Type1Byte {
			hasher16 = newCRC16Hash()
		} else {
			hasher32 = crc32.NewIEEE()
		}

		// Helper function to write and update CRC
		writeAndHash := func(buf []byte) error {
			if _, err := tmpFile.Write(buf); err != nil {
				return err
			}
			if hasher16 != nil {
				hasher16.Write(buf)
			} else {
				hasher32.Write(buf)
			}
			return nil
		}

		// Write the type
		typeBuf := []byte{recordType}
		if err := writeAndHash(typeBuf); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error writing type: %w", err)
		}

		// Write the key size
		keySize := byte(len(key))
		keySizeBuf := []byte{keySize}
		if err := writeAndHash(keySizeBuf); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error writing key size: %w", err)
		}

		// Write the key
		if err := writeAndHash(key); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error writing key: %w", err)
		}

		// Write the data size according to the type
		var dataSizeBuf []byte
		switch recordType {
		case Type1Byte:
			dataSizeBuf = []byte{byte(dataSize)}
		case Type2Bytes:
			dataSizeBuf = make([]byte, 2)
			binary.LittleEndian.PutUint16(dataSizeBuf, uint16(dataSize))
		case Type4Bytes:
			dataSizeBuf = make([]byte, 4)
			binary.LittleEndian.PutUint32(dataSizeBuf, uint32(dataSize))
		case Type8Bytes:
			dataSizeBuf = make([]byte, 8)
			binary.LittleEndian.PutUint64(dataSizeBuf, dataSize)
		}
		if err := writeAndHash(dataSizeBuf); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error writing data size: %w", err)
		}

		// Write the data in chunks to avoid memory pressure for large values
		const bufferSize = 64 * 1024 // 64KB buffer
		for offset := 0; offset < len(data); offset += bufferSize {
			end := offset + bufferSize
			if end > len(data) {
				end = len(data)
			}
			if err := writeAndHash(data[offset:end]); err != nil {
				tmpFile.Close()
				return fmt.Errorf("error writing data chunk: %w", err)
			}
		}

		// Calculate and write CRC
		if recordType == Type1Byte {
			// CRC-16
			crc := hasher16.Sum16()
			crcBuf := make([]byte, 2)
			binary.LittleEndian.PutUint16(crcBuf, crc)
			if _, err := tmpFile.Write(crcBuf); err != nil {
				tmpFile.Close()
				return fmt.Errorf("error writing CRC: %w", err)
			}
		} else {
			// CRC-32
			crc := hasher32.Sum32()
			crcBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(crcBuf, crc)
			if _, err := tmpFile.Write(crcBuf); err != nil {
				tmpFile.Close()
				return fmt.Errorf("error writing CRC: %w", err)
			}
		}

		newCache[keyStr] = pos
	}

	// Sync temp file to ensure all data is written to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error syncing temp file: %w", err)
	}

	// Close temp file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file: %w", err)
	}

	// Close original file
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("error closing original file: %w", err)
	}

	// Atomically rename temp file over original
	// This is an atomic operation on all major platforms
	if err := os.Rename(tmpFilename, s.filePath); err != nil {
		// Critical error: try to reopen original file
		if f, reopenErr := os.OpenFile(s.filePath, os.O_RDWR, 0644); reopenErr == nil {
			s.file = f
		}
		return fmt.Errorf("error renaming temp file (original file may still be intact): %w", err)
	}

	// Reopen the file
	f, err := os.OpenFile(s.filePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error reopening file after compact: %w", err)
	}
	s.file = f

	// Update cache with new positions
	s.cache = newCache

	// Clear free space list (compaction eliminates all deleted records)
	s.freeSpace = make([]FreeSpace, 0)

	// Get final stats
	finalSize, _ := s.file.Stat()
	var afterBytes int64 = 0
	if finalSize != nil {
		afterBytes = finalSize.Size()
	}

	savedBytes := beforeBytes - afterBytes
	savedPercent := float64(0)
	if beforeBytes > 0 {
		savedPercent = (float64(savedBytes) / float64(beforeBytes)) * 100.0
	}

	s.logger.Info("Compact completed",
		Field{Key: "before_bytes", Value: beforeBytes},
		Field{Key: "after_bytes", Value: afterBytes},
		Field{Key: "saved_bytes", Value: savedBytes},
		Field{Key: "saved_percent", Value: fmt.Sprintf("%.2f", savedPercent)},
		Field{Key: "active_records", Value: len(s.cache)},
		Field{Key: "duration_ms", Value: time.Since(startTime).Milliseconds()},
	)

	return nil
}

// compactInternal is the internal implementation of Compact without locking
// Used by CloseWithCompact to avoid deadlock
func (s *SKV) compactInternal() error {
	return s.compactInternalCtx(context.Background())
}

// Keys returns a list of all active keys in the database
func (s *SKV) Keys() ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
//
// Note: Iteration order is NOT guaranteed (iterates over internal cache map)
// For ordered iteration, use NewCursor() or AllCursor()
//
// Performance: Values are read on-demand from disk, so memory usage is constant
// regardless of database size (only keys are in memory via cache)
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
		_, key, data, _, err := s.readRecord(true, false)
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
		_, _, data, _, err := s.readRecord(true, false)
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
// IMPORTANT: Keys and values are stored as-is (encrypted if encryption is enabled)
// The backup preserves the encryption state - encrypted data stays encrypted
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

		// Read the record WITHOUT decryption (skipEncryption=true)
		// This preserves encrypted data as-is
		_, keyBytes, data, _, err := s.readRecord(true, true)
		if err != nil {
			return fmt.Errorf("error reading record for key %q: %w", key, err)
		}

		record := BackupRecord{
			Key: string(keyBytes),
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
// IMPORTANT: Restores data as-is (encrypted data stays encrypted)
// If the backup was created from an encrypted database, this database must use the same encryption
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

		// Write the record WITHOUT encryption (skipEncryption=true)
		// This preserves the data as-is from the backup
		key := []byte(record.Key)

		// Seek to end of file
		if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("error seeking to end of file: %w", err)
		}

		// Write record with skipEncryption=true to preserve data as-is
		position, _, err := s.writeRecordAtPosition(key, data, true)
		if err != nil {
			return fmt.Errorf("error restoring key %q: %w", record.Key, err)
		}

		// Decrypt key for cache (cache is indexed by decrypted keys)
		decryptedKey, err := s.encryptor.decrypt(key)
		if err != nil {
			return fmt.Errorf("error decrypting key for cache: %w", err)
		}

		// Update cache with decrypted key
		s.cache[string(decryptedKey)] = position
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

	// Save checkpoint for potential rollback
	checkpoint, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("error getting file position: %w", err)
	}

	// Write the record using streaming approach
	recordPos, err := s.writeRecordStream(key, reader, uint64(size))
	if err != nil {
		// Rollback: truncate file to checkpoint position
		if truncErr := s.file.Truncate(checkpoint); truncErr != nil {
			s.logger.Error("Failed to rollback after PutStream error",
				Field{Key: "key", Value: string(key)},
				Field{Key: "original_error", Value: err.Error()},
				Field{Key: "rollback_error", Value: truncErr.Error()},
			)
			return fmt.Errorf("write failed and rollback failed: %w (rollback: %v)", err, truncErr)
		}
		// Restore file position
		s.file.Seek(checkpoint, io.SeekStart)

		s.logger.Warn("PutStream failed, rolled back",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
		)
		return err
	}

	// Sync to ensure durability before updating cache
	if err := s.file.Sync(); err != nil {
		// Rollback on sync failure
		s.file.Truncate(checkpoint)
		s.file.Seek(checkpoint, io.SeekStart)
		s.logger.Error("PutStream sync failed, rolled back",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
		)
		return fmt.Errorf("error syncing file: %w", err)
	}

	// Update cache with record start position (commit point)
	s.cache[string(key)] = recordPos

	s.logger.Debug("PutStream successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "size", Value: size},
		Field{Key: "position", Value: recordPos},
	)

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

	// Save checkpoint BEFORE any modifications
	checkpoint, err := s.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("error getting file position: %w", err)
	}

	// Write the new record using streaming approach
	recordPos, err := s.writeRecordStream(key, reader, uint64(size))
	if err != nil {
		// Rollback: truncate file to checkpoint position
		if truncErr := s.file.Truncate(checkpoint); truncErr != nil {
			s.logger.Error("Failed to rollback after UpdateStream error",
				Field{Key: "key", Value: string(key)},
				Field{Key: "original_error", Value: err.Error()},
				Field{Key: "rollback_error", Value: truncErr.Error()},
			)
			return fmt.Errorf("write failed and rollback failed: %w (rollback: %v)", err, truncErr)
		}
		// Restore file position
		s.file.Seek(checkpoint, io.SeekStart)

		s.logger.Warn("UpdateStream failed, rolled back",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
		)
		return err
	}

	// Sync to ensure new record is durable before deleting old one
	if err := s.file.Sync(); err != nil {
		// Rollback on sync failure
		s.file.Truncate(checkpoint)
		s.file.Seek(checkpoint, io.SeekStart)
		s.logger.Error("UpdateStream sync failed, rolled back",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
		)
		return fmt.Errorf("error syncing file: %w", err)
	}

	// New record is now durable. Delete old record (mark as tombstone).
	// We do this after the write succeeds to ensure atomicity.
	if err := s.deleteInternal(key); err != nil {
		// This is a problem - new record is written but old record wasn't deleted
		// Log the error but continue since the update is partially successful
		s.logger.Error("UpdateStream: failed to delete old record, but new record was written",
			Field{Key: "key", Value: string(key)},
			Field{Key: "error", Value: err.Error()},
		)
	}

	// Update cache with new record start position (commit point)
	s.cache[string(key)] = recordPos

	s.logger.Debug("UpdateStream successful",
		Field{Key: "key", Value: string(key)},
		Field{Key: "size", Value: size},
		Field{Key: "position", Value: recordPos},
	)

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

	// Initialize streaming CRC calculation based on record type
	var hasher16 *crc16Hash
	var hasher32 hash.Hash32
	if recordType == Type1Byte {
		hasher16 = newCRC16Hash()
	} else {
		hasher32 = crc32.NewIEEE()
	}

	// Helper function to write and update CRC
	writeAndHash := func(data []byte) error {
		if _, err := s.file.Write(data); err != nil {
			return err
		}
		if hasher16 != nil {
			hasher16.Write(data)
		} else {
			hasher32.Write(data)
		}
		return nil
	}

	// Write the type
	typeBuf := []byte{recordType}
	if err := writeAndHash(typeBuf); err != nil {
		return 0, fmt.Errorf("error writing type: %w", err)
	}

	// Write the key size
	keySize := byte(len(key))
	keySizeBuf := []byte{keySize}
	if err := writeAndHash(keySizeBuf); err != nil {
		return 0, fmt.Errorf("error writing key size: %w", err)
	}

	// Write the key
	if err := writeAndHash(key); err != nil {
		return 0, fmt.Errorf("error writing key: %w", err)
	}

	// Write the data size according to the type
	var dataSizeBuf []byte
	switch recordType {
	case Type1Byte:
		dataSizeBuf = []byte{byte(dataSize)}
	case Type2Bytes:
		dataSizeBuf = make([]byte, 2)
		binary.LittleEndian.PutUint16(dataSizeBuf, uint16(dataSize))
	case Type4Bytes:
		dataSizeBuf = make([]byte, 4)
		binary.LittleEndian.PutUint32(dataSizeBuf, uint32(dataSize))
	case Type8Bytes:
		dataSizeBuf = make([]byte, 8)
		binary.LittleEndian.PutUint64(dataSizeBuf, dataSize)
	}
	if err := writeAndHash(dataSizeBuf); err != nil {
		return 0, fmt.Errorf("error writing data size: %w", err)
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

		// Write chunk and update CRC
		if err := writeAndHash(chunk[:n]); err != nil {
			return 0, fmt.Errorf("error writing data chunk: %w", err)
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

	// Calculate and write CRC
	if recordType == Type1Byte {
		// CRC-16
		crc := hasher16.Sum16()
		crcBuf := make([]byte, 2)
		binary.LittleEndian.PutUint16(crcBuf, crc)
		if _, err := s.file.Write(crcBuf); err != nil {
			return 0, fmt.Errorf("error writing CRC: %w", err)
		}
	} else {
		// CRC-32
		crc := hasher32.Sum32()
		crcBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(crcBuf, crc)
		if _, err := s.file.Write(crcBuf); err != nil {
			return 0, fmt.Errorf("error writing CRC: %w", err)
		}
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

	// Initialize streaming CRC calculation based on record type
	baseType := getBaseType(recordType)
	var hasher16 *crc16Hash
	var hasher32 hash.Hash32
	if baseType == Type1Byte {
		hasher16 = newCRC16Hash()
		hasher16.Write(typeBuf)
	} else {
		hasher32 = crc32.NewIEEE()
		hasher32.Write(typeBuf)
	}

	// Helper function to read and update CRC
	readAndHash := func(buf []byte) error {
		if _, err := io.ReadFull(s.file, buf); err != nil {
			return err
		}
		if hasher16 != nil {
			hasher16.Write(buf)
		} else {
			hasher32.Write(buf)
		}
		return nil
	}

	// Read key size
	keySizeBuf := make([]byte, 1)
	if err := readAndHash(keySizeBuf); err != nil {
		return 0, fmt.Errorf("error reading key size: %w", err)
	}
	keySize := keySizeBuf[0]

	// Read the key (need it for CRC)
	keyData := make([]byte, keySize)
	if err := readAndHash(keyData); err != nil {
		return 0, fmt.Errorf("error reading key: %w", err)
	}

	// Read data size
	var dataSize uint64
	var dataSizeBuf []byte

	switch baseType {
	case Type1Byte:
		dataSizeBuf = make([]byte, 1)
		if err := readAndHash(dataSizeBuf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(dataSizeBuf[0])
	case Type2Bytes:
		dataSizeBuf = make([]byte, 2)
		if err := readAndHash(dataSizeBuf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint16(dataSizeBuf))
	case Type4Bytes:
		dataSizeBuf = make([]byte, 4)
		if err := readAndHash(dataSizeBuf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = uint64(binary.LittleEndian.Uint32(dataSizeBuf))
	case Type8Bytes:
		dataSizeBuf = make([]byte, 8)
		if err := readAndHash(dataSizeBuf); err != nil {
			return 0, fmt.Errorf("error reading data size: %w", err)
		}
		dataSize = binary.LittleEndian.Uint64(dataSizeBuf)
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
		if err := readAndHash(chunk); err != nil {
			return totalWritten, fmt.Errorf("error reading data chunk: %w", err)
		}

		written, err := writer.Write(chunk)
		if err != nil {
			return totalWritten, fmt.Errorf("error writing to stream: %w", err)
		}

		totalWritten += int64(written)
		remaining -= uint64(chunkSize)
	}

	// Read and verify CRC
	if baseType == Type1Byte {
		// CRC-16
		crcBuf := make([]byte, 2)
		if _, err := io.ReadFull(s.file, crcBuf); err != nil {
			return totalWritten, fmt.Errorf("error reading CRC: %w", err)
		}
		storedCRC := binary.LittleEndian.Uint16(crcBuf)
		calculatedCRC := hasher16.Sum16()
		if storedCRC != calculatedCRC {
			return totalWritten, fmt.Errorf("CRC mismatch: expected 0x%04X, got 0x%04X (record may be corrupted)", storedCRC, calculatedCRC)
		}
	} else {
		// CRC-32
		crcBuf := make([]byte, 4)
		if _, err := io.ReadFull(s.file, crcBuf); err != nil {
			return totalWritten, fmt.Errorf("error reading CRC: %w", err)
		}
		storedCRC := binary.LittleEndian.Uint32(crcBuf)
		calculatedCRC := hasher32.Sum32()
		if storedCRC != calculatedCRC {
			return totalWritten, fmt.Errorf("CRC mismatch: expected 0x%08X, got 0x%08X (record may be corrupted)", storedCRC, calculatedCRC)
		}
	}

	return totalWritten, nil
}

// GetStreamString is a convenience wrapper for GetStream using string keys
func (s *SKV) GetStreamString(key string, writer io.Writer) (int64, error) {
	return s.GetStream([]byte(key), writer)
}
