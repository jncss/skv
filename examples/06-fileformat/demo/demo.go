package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/jncss/skv"
)

func main() {
	// Create a sample database
	testFile := "sample.skv"
	os.Remove(testFile)

	db, err := skv.Open(testFile)
	if err != nil {
		panic(err)
	}

	// Add some sample data
	db.PutString("name", "Alice")
	db.PutString("city", "Barcelona")
	db.Put([]byte("data"), make([]byte, 300)) // Large data (Type2Bytes)

	db.Close()

	// Now inspect the file format
	fmt.Println("=== SKV File Format Inspector ===")

	file, err := os.Open(testFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read and display header
	header := make([]byte, 6)
	if _, err := io.ReadFull(file, header); err != nil {
		panic(err)
	}

	fmt.Printf("Header (6 bytes):\n")
	fmt.Printf("  Magic: %s (0x%02X 0x%02X 0x%02X)\n", header[:3], header[0], header[1], header[2])
	fmt.Printf("  Version: %d.%d.%d\n\n", header[3], header[4], header[5])

	// Read and display records
	recordNum := 0
	for {
		pos, _ := file.Seek(0, io.SeekCurrent)

		// Read type
		typeBuf := make([]byte, 1)
		if _, err := file.Read(typeBuf); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		recordType := typeBuf[0]

		// Read key size
		keySizeBuf := make([]byte, 1)
		if _, err := io.ReadFull(file, keySizeBuf); err != nil {
			panic(err)
		}
		keySize := keySizeBuf[0]

		// Read key
		key := make([]byte, keySize)
		if _, err := io.ReadFull(file, key); err != nil {
			panic(err)
		}

		// Read data size based on type
		baseType := recordType & 0x7F // Remove deleted flag
		var dataSize uint64

		switch baseType {
		case 0x01:
			buf := make([]byte, 1)
			io.ReadFull(file, buf)
			dataSize = uint64(buf[0])
		case 0x02:
			buf := make([]byte, 2)
			io.ReadFull(file, buf)
			dataSize = uint64(binary.LittleEndian.Uint16(buf))
		case 0x04:
			buf := make([]byte, 4)
			io.ReadFull(file, buf)
			dataSize = uint64(binary.LittleEndian.Uint32(buf))
		case 0x08:
			buf := make([]byte, 8)
			io.ReadFull(file, buf)
			dataSize = binary.LittleEndian.Uint64(buf)
		}

		// Skip data
		file.Seek(int64(dataSize), io.SeekCurrent)

		// Display record info
		recordNum++
		deleted := ""
		if recordType&0x80 != 0 {
			deleted = " [DELETED]"
		}
		typeStr := ""
		switch baseType {
		case 0x01:
			typeStr = "Type1Byte"
		case 0x02:
			typeStr = "Type2Bytes"
		case 0x04:
			typeStr = "Type4Bytes"
		case 0x08:
			typeStr = "Type8Bytes"
		}

		fmt.Printf("Record #%d at offset %d:%s\n", recordNum, pos, deleted)
		fmt.Printf("  Type: 0x%02X (%s)\n", recordType, typeStr)
		fmt.Printf("  Key: %q (size: %d)\n", key, keySize)
		fmt.Printf("  Data Size: %d bytes\n\n", dataSize)
	}

	fmt.Printf("Total records: %d\n", recordNum)

	// Clean up
	os.Remove(testFile)
}
