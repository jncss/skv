package skv

import (
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/pierrec/lz4/v4"
)

// CompressionType defines the compression algorithm to use
type CompressionType byte

const (
	// CompressionNone means no compression is applied
	CompressionNone CompressionType = 0x00

	// CompressionSnappy uses Google's Snappy compression
	// - Fast compression and decompression
	// - Lower compression ratio than LZ4
	// - Good for small to medium data
	CompressionSnappy CompressionType = 0x01

	// CompressionLZ4 uses LZ4 compression
	// - Very fast compression and decompression
	// - Better compression ratio than Snappy
	// - Good for all data sizes
	CompressionLZ4 CompressionType = 0x02
)

// String returns the name of the compression type
func (c CompressionType) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionSnappy:
		return "snappy"
	case CompressionLZ4:
		return "lz4"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

// CompressionThreshold is the minimum size (in bytes) for data to be compressed
// Data smaller than this will not be compressed to avoid overhead
const CompressionThreshold = 128 // 128 bytes

// compress compresses data using the specified algorithm
// Returns the compressed data and the actual compression type used
// If data is too small (< CompressionThreshold), returns original data with CompressionNone
func compress(data []byte, compressionType CompressionType) ([]byte, CompressionType, error) {
	// Don't compress small data
	if len(data) < CompressionThreshold {
		return data, CompressionNone, nil
	}

	// Don't compress if compression is disabled
	if compressionType == CompressionNone {
		return data, CompressionNone, nil
	}

	switch compressionType {
	case CompressionSnappy:
		compressed := snappy.Encode(nil, data)
		// Only use compression if it actually reduces size
		if len(compressed) < len(data) {
			return compressed, CompressionSnappy, nil
		}
		return data, CompressionNone, nil

	case CompressionLZ4:
		compressed := make([]byte, lz4.CompressBlockBound(len(data)))
		n, err := lz4.CompressBlock(data, compressed, nil)
		if err != nil {
			return nil, CompressionNone, fmt.Errorf("lz4 compression error: %w", err)
		}
		compressed = compressed[:n]
		// Only use compression if it actually reduces size
		if len(compressed) < len(data) {
			return compressed, CompressionLZ4, nil
		}
		return data, CompressionNone, nil

	default:
		return nil, CompressionNone, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

// decompress decompresses data using the specified algorithm
func decompress(data []byte, compressionType CompressionType, originalSize int) ([]byte, error) {
	if compressionType == CompressionNone {
		return data, nil
	}

	switch compressionType {
	case CompressionSnappy:
		decompressed, err := snappy.Decode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("snappy decompression error: %w", err)
		}
		return decompressed, nil

	case CompressionLZ4:
		decompressed := make([]byte, originalSize)
		n, err := lz4.UncompressBlock(data, decompressed)
		if err != nil {
			return nil, fmt.Errorf("lz4 decompression error: %w", err)
		}
		if n != originalSize {
			return nil, fmt.Errorf("lz4 decompression size mismatch: expected %d, got %d", originalSize, n)
		}
		return decompressed, nil

	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

// compressWriter wraps an io.Writer and compresses data before writing
type compressWriter struct {
	w               io.Writer
	compressionType CompressionType
}

// newCompressWriter creates a new compressing writer
func newCompressWriter(w io.Writer, compressionType CompressionType) *compressWriter {
	return &compressWriter{
		w:               w,
		compressionType: compressionType,
	}
}

// Write compresses and writes data
func (cw *compressWriter) Write(p []byte) (n int, err error) {
	compressed, actualType, err := compress(p, cw.compressionType)
	if err != nil {
		return 0, err
	}

	// Write compression type as first byte
	if _, err := cw.w.Write([]byte{byte(actualType)}); err != nil {
		return 0, err
	}

	// Write compressed (or original) data
	if _, err := cw.w.Write(compressed); err != nil {
		return 0, err
	}

	return len(p), nil // Return original size
}
