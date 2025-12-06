package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	fmt.Println("=== SKV File Format Inspector ===\n")

	os.MkdirAll("data", 0755)

	// Create a sample database
	dbFile := "data/format_demo.skv"
	db, err := skv.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	// Add sample data with different sizes
	fmt.Println("Creating sample data with different sizes:\n")

	// Small data (1-byte size field)
	db.PutString("small", "Hi")
	fmt.Println("✓ Added 'small' (2 bytes) → Uses Type 0x01 (1-byte size)")

	// Medium data (2-byte size field)
	mediumData := make([]byte, 300)
	for i := range mediumData {
		mediumData[i] = byte(i % 256)
	}
	db.Put([]byte("medium"), mediumData)
	fmt.Println("✓ Added 'medium' (300 bytes) → Uses Type 0x02 (2-byte size)")

	// Large data (4-byte size field)
	largeData := make([]byte, 70000)
	for i := range largeData {
		largeData[i] = 'A'
	}
	db.Put([]byte("large"), largeData)
	fmt.Println("✓ Added 'large' (70000 bytes) → Uses Type 0x04 (4-byte size)")

	// Add some data that will be deleted
	db.PutString("temp1", "will be deleted")
	db.PutString("temp2", "also deleted")
	fmt.Println("✓ Added temporary keys")

	// Delete to show deleted flag
	db.DeleteString("temp1")
	fmt.Println("✓ Deleted 'temp1' → Record marked with 0x80 flag")

	db.Close()

	// --- Inspect the file format ---
	fmt.Println("\n=== File Format Inspection ===\n")

	file, err := os.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read header
	fmt.Println("1. FILE HEADER (6 bytes):")
	header := make([]byte, 6)
	file.Read(header)

	magic := string(header[0:3])
	major := header[3]
	minor := header[4]
	patch := header[5]

	fmt.Printf("   Magic: %q (0x%02X 0x%02X 0x%02X)\n", magic, header[0], header[1], header[2])
	fmt.Printf("   Version: %d.%d.%d\n", major, minor, patch)

	// Read records
	fmt.Println("\n2. RECORDS:\n")

	recordNum := 0
	for {
		recordNum++

		// Read type byte
		typeBuf := make([]byte, 1)
		n, err := file.Read(typeBuf)
		if err != nil || n == 0 {
			break
		}

		recordType := typeBuf[0]

		// Check if padding byte
		if recordType == 0x80 {
			fmt.Printf("   [Padding] 0x80 (skipped)\n")
			continue
		}

		// Check if deleted
		isDeleted := (recordType & 0x80) != 0
		baseType := recordType & 0x7F

		fmt.Printf("   Record #%d:\n", recordNum)
		fmt.Printf("   ├─ Type: 0x%02X", recordType)

		if isDeleted {
			fmt.Printf(" (DELETED, base type: 0x%02X)\n", baseType)
		} else {
			fmt.Printf("\n")
		}

		// Determine size field length
		var sizeFieldLen int
		switch baseType {
		case 0x01:
			sizeFieldLen = 1
			fmt.Printf("   ├─ Size field: 1 byte (max 255 bytes)\n")
		case 0x02:
			sizeFieldLen = 2
			fmt.Printf("   ├─ Size field: 2 bytes (max 64 KB)\n")
		case 0x04:
			sizeFieldLen = 4
			fmt.Printf("   ├─ Size field: 4 bytes (max 4 GB)\n")
		case 0x08:
			sizeFieldLen = 8
			fmt.Printf("   ├─ Size field: 8 bytes (max 18 EB)\n")
		default:
			fmt.Printf("   └─ Unknown type!\n\n")
			continue
		}

		// Read key size
		keySizeBuf := make([]byte, 1)
		file.Read(keySizeBuf)
		keySize := int(keySizeBuf[0])
		fmt.Printf("   ├─ Key size: %d bytes\n", keySize)

		// Read key
		keyBuf := make([]byte, keySize)
		file.Read(keyBuf)
		fmt.Printf("   ├─ Key: %q\n", string(keyBuf))

		// Read data size
		dataSizeBuf := make([]byte, sizeFieldLen)
		file.Read(dataSizeBuf)

		var dataSize uint64
		switch sizeFieldLen {
		case 1:
			dataSize = uint64(dataSizeBuf[0])
		case 2:
			dataSize = uint64(binary.LittleEndian.Uint16(dataSizeBuf))
		case 4:
			dataSize = uint64(binary.LittleEndian.Uint32(dataSizeBuf))
		case 8:
			dataSize = binary.LittleEndian.Uint64(dataSizeBuf)
		}

		fmt.Printf("   ├─ Data size: %d bytes\n", dataSize)

		// Read data (or skip if too large)
		if dataSize > 0 && dataSize <= 100 {
			dataBuf := make([]byte, dataSize)
			file.Read(dataBuf)
			fmt.Printf("   └─ Data: %q\n", string(dataBuf))
		} else if dataSize > 0 {
			// Skip large data
			file.Seek(int64(dataSize), 1)
			fmt.Printf("   └─ Data: <%d bytes, not shown>\n", dataSize)
		} else {
			fmt.Printf("   └─ Data: (empty)\n")
		}

		// Calculate total record size
		totalSize := 1 + 1 + keySize + sizeFieldLen + int(dataSize)
		fmt.Printf("   Total record size: %d bytes\n\n", totalSize)
	}

	// --- Format explanation ---
	fmt.Println("=== Format Explanation ===\n")

	fmt.Println("Record Structure:")
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Println("  │ Type (1 byte)                       │")
	fmt.Println("  ├─────────────────────────────────────┤")
	fmt.Println("  │ Key Size (1 byte)                   │")
	fmt.Println("  ├─────────────────────────────────────┤")
	fmt.Println("  │ Key (variable, max 255 bytes)       │")
	fmt.Println("  ├─────────────────────────────────────┤")
	fmt.Println("  │ Data Size (1/2/4/8 bytes)           │")
	fmt.Println("  ├─────────────────────────────────────┤")
	fmt.Println("  │ Data (variable)                     │")
	fmt.Println("  └─────────────────────────────────────┘")

	fmt.Println("\nType Byte Format:")
	fmt.Println("  Bit 7: Deleted flag (1 = deleted, 0 = active)")
	fmt.Println("  Bits 0-6: Size field type")
	fmt.Println("    0x01 = 1-byte size (max 255 bytes)")
	fmt.Println("    0x02 = 2-byte size (max 64 KB)")
	fmt.Println("    0x04 = 4-byte size (max 4 GB)")
	fmt.Println("    0x08 = 8-byte size (max 18 EB)")

	fmt.Println("\nDeleted Records:")
	fmt.Println("  • Type 0x81 = Deleted, was 1-byte size")
	fmt.Println("  • Type 0x82 = Deleted, was 2-byte size")
	fmt.Println("  • Type 0x84 = Deleted, was 4-byte size")
	fmt.Println("  • Type 0x88 = Deleted, was 8-byte size")

	fmt.Println("\nSpace Reuse:")
	fmt.Println("  • Deleted records are tracked in free space list")
	fmt.Println("  • New records try to reuse deleted space")
	fmt.Println("  • Padding byte (0x80) fills gaps too small for records")
	fmt.Println("  • Compact() removes all deleted records and duplicates")

	fmt.Println("\nKey Features:")
	fmt.Println("  • Sequential file format (append-only writes)")
	fmt.Println("  • Variable-length encoding (efficient storage)")
	fmt.Println("  • Soft deletes (mark as deleted, don't remove)")
	fmt.Println("  • Last-write-wins (updates append new version)")
	fmt.Println("  • In-memory cache for O(1) lookups")

	// Show file statistics
	info, _ := os.Stat(dbFile)
	fmt.Printf("\nFile size: %d bytes\n", info.Size())

	fmt.Println("\n✅ File format inspection completed!")
}
