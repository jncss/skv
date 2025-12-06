package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	os.MkdirAll("data", 0755)
	os.MkdirAll("data/files", 0755)

	fmt.Println("=== File Operations Example ===")
	fmt.Println()

	// Open database
	db, err := skv.Open("data/fileops")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// --- Example 1: Store a text file ---
	fmt.Println("1. Storing text files:")
	fmt.Println()

	// Create a sample text file
	textContent := `This is a sample configuration file.

[settings]
debug = true
port = 8080
host = localhost

[database]
driver = postgres
connection = localhost:5432
`

	configFile := "data/files/config.ini"
	err = os.WriteFile(configFile, []byte(textContent), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Store the file in the database
	err = db.PutFile("config:app", configFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Stored file 'config.ini' under key 'config:app'\n")

	// Retrieve the file
	retrievedFile := "data/files/config_retrieved.ini"
	err = db.GetFile("config:app", retrievedFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Retrieved file to 'config_retrieved.ini'\n")

	// Verify contents
	retrieved, _ := os.ReadFile(retrievedFile)
	if string(retrieved) == textContent {
		fmt.Println("✓ File contents match!")
	}

	// --- Example 2: Store binary files (images, etc.) ---
	fmt.Println("\n2. Storing binary files:")
	fmt.Println()

	// Create a sample binary file (simulating an image)
	binaryData := make([]byte, 5000)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	imageFile := "data/files/image.bin"
	err = os.WriteFile(imageFile, binaryData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Store binary file
	err = db.PutFile("assets:logo", imageFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Stored binary file (%d bytes)\n", len(binaryData))

	// Retrieve binary file
	retrievedImage := "data/files/image_retrieved.bin"
	err = db.GetFile("assets:logo", retrievedImage)
	if err != nil {
		log.Fatal(err)
	}

	// Verify size
	info, _ := os.Stat(retrievedImage)
	fmt.Printf("✓ Retrieved binary file (%d bytes)\n", info.Size())

	// --- Example 3: Update a file ---
	fmt.Println("\n3. Updating files:")
	fmt.Println()

	// Modify the config file
	updatedConfig := textContent + "\n[cache]\nenabled = true\nttl = 3600\n"
	err = os.WriteFile(configFile, []byte(updatedConfig), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Update in database
	err = db.UpdateFile("config:app", configFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Updated config file in database")

	// Retrieve updated version
	updatedRetrievedFile := "data/files/config_updated.ini"
	err = db.GetFile("config:app", updatedRetrievedFile)
	if err != nil {
		log.Fatal(err)
	}

	// Verify update
	updatedRetrieved, _ := os.ReadFile(updatedRetrievedFile)
	if string(updatedRetrieved) == updatedConfig {
		fmt.Println("✓ Updated file retrieved successfully")
	}

	// --- Example 4: Store multiple files ---
	fmt.Println("\n4. Storing multiple files:")
	fmt.Println()

	files := map[string]string{
		"templates:header.html": `<header><h1>My Website</h1></header>`,
		"templates:footer.html": `<footer>&copy; 2024 My Company</footer>`,
		"scripts:main.js":       `console.log("Hello, World!");`,
		"styles:main.css":       `body { font-family: Arial; }`,
	}

	for key, content := range files {
		filename := "data/files/" + key[len("templates:"):] // Extract filename from key
		if len(key) > len("templates:") && key[:len("templates:")] != "templates:" {
			filename = "data/files/" + key[len("scripts:"):]
			if len(key) > len("scripts:") && key[:len("scripts:")] != "scripts:" {
				filename = "data/files/" + key[len("styles:"):]
			}
		}

		err = os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			continue
		}

		err = db.PutFile(key, filename)
		if err != nil {
			log.Printf("Error storing %s: %v", key, err)
		} else {
			fmt.Printf("✓ Stored: %s\n", key)
		}
	}

	// --- Example 5: List all stored files ---
	fmt.Println("\n5. Files stored in database:")
	fmt.Println()

	count := 0
	db.ForEachString(func(key string, value string) error {
		count++
		size := len(value)
		fmt.Printf("  [%d] %s: %d bytes\n", count, key, size)
		return nil
	})

	fmt.Printf("\n✓ Total files stored: %d\n", count)

	// --- Example 6: Extract all files from database ---
	fmt.Println("\n6. Extracting all files:")
	fmt.Println()

	extractDir := "data/files/extracted"
	os.MkdirAll(extractDir, 0755)

	extracted := 0
	db.ForEachString(func(key string, value string) error {
		// Sanitize key to create filename
		filename := extractDir + "/" + key

		// Create parent directories if needed
		err := os.MkdirAll(extractDir, 0755)
		if err != nil {
			return err
		}

		// Write file
		err = os.WriteFile(filename, []byte(value), 0644)
		if err != nil {
			log.Printf("Error extracting %s: %v", key, err)
			return nil
		}

		extracted++
		fmt.Printf("✓ Extracted: %s\n", key)
		return nil
	})

	fmt.Printf("\n✓ Extracted %d files\n", extracted)

	// --- Example 7: Streaming large values (reading) ---
	fmt.Println("\n7. Streaming large values - reading (memory-efficient):")
	fmt.Println()

	// Create a large data file (1 MB)
	largeData := make([]byte, 1*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	largeFile := "data/files/large.bin"
	err = os.WriteFile(largeFile, largeData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Store large file
	err = db.PutFile("large:data", largeFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Stored large file: %s (%.2f MB)\n", "large.bin", float64(len(largeData))/1024/1024)

	// Stream to a new file without loading entire value into memory
	streamOutput := "data/files/streamed_output.bin"
	outputFile, err := os.Create(streamOutput)
	if err != nil {
		log.Fatal(err)
	}

	bytesWritten, err := db.GetStreamString("large:data", outputFile)
	outputFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Streamed %d bytes to file (memory-efficient read)\n", bytesWritten)

	// Verify file size
	info, err = os.Stat(streamOutput)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Output file size: %.2f MB\n", float64(info.Size())/1024/1024)

	// --- Example 8: Streaming large values (writing) ---
	fmt.Println("\n8. Streaming large values - writing (memory-efficient):")
	fmt.Println()

	// Create another large file (2 MB)
	largeFile2 := "data/files/large2.bin"
	largeData2 := make([]byte, 2*1024*1024)
	for i := range largeData2 {
		largeData2[i] = byte((i * 7) % 256)
	}
	err = os.WriteFile(largeFile2, largeData2, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Open file for streaming
	inputFile, err := os.Open(largeFile2)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, err := inputFile.Stat()
	if err != nil {
		inputFile.Close()
		log.Fatal(err)
	}

	// Use PutStream to store without loading to memory
	err = db.PutStreamString("stream:large", inputFile, fileInfo.Size())
	inputFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Stored via PutStream: %.2f MB (no memory load)\n", float64(fileInfo.Size())/1024/1024)

	// Update using UpdateStream
	largeFile3 := "data/files/large3.bin"
	largeData3 := make([]byte, 3*1024*1024)
	for i := range largeData3 {
		largeData3[i] = byte((i * 13) % 256)
	}
	err = os.WriteFile(largeFile3, largeData3, 0644)
	if err != nil {
		log.Fatal(err)
	}

	updateFile, err := os.Open(largeFile3)
	if err != nil {
		log.Fatal(err)
	}

	updateInfo, err := updateFile.Stat()
	if err != nil {
		updateFile.Close()
		log.Fatal(err)
	}

	err = db.UpdateStreamString("stream:large", updateFile, updateInfo.Size())
	updateFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Updated via UpdateStream: %.2f MB (no memory load)\n", float64(updateInfo.Size())/1024/1024)

	// Verify by streaming out
	verifyOutput := "data/files/verify_stream.bin"
	verifyFile, err := os.Create(verifyOutput)
	if err != nil {
		log.Fatal(err)
	}

	bytesRead, err := db.GetStreamString("stream:large", verifyFile)
	verifyFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Verified via GetStream: %.2f MB read\n", float64(bytesRead)/1024/1024)

	// --- Example 9: Error handling ---
	fmt.Println("\n9. Error handling:")
	fmt.Println()

	// Try to store non-existent file
	err = db.PutFile("test", "nonexistent.txt")
	if err != nil {
		fmt.Printf("✗ Cannot store non-existent file: %v\n", err)
	}

	// Try to retrieve non-existent key
	err = db.GetFile("nonexistent", "output.txt")
	if err == skv.ErrKeyNotFound {
		fmt.Println("✗ Cannot retrieve non-existent key")
	}

	// Try to update with duplicate key (should fail with Put)
	err = db.PutFile("config:app", configFile)
	if err == skv.ErrKeyExists {
		fmt.Println("✗ Cannot Put - key already exists (use UpdateFile instead)")
	}

	fmt.Println("\n=== Use Cases ===")
	fmt.Println()
	fmt.Println("File operations are useful for:")
	fmt.Println("• Configuration file storage")
	fmt.Println("• Template management")
	fmt.Println("• Asset storage (images, CSS, JS)")
	fmt.Println("• Document archiving")
	fmt.Println("• Log file storage")
	fmt.Println("• Binary data storage")
	fmt.Println("• Backup of small files")
	fmt.Println("• Streaming large files (videos, backups, logs)")

	fmt.Println("\n✅ File operations example completed!")
}
