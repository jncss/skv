package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jncss/skv"
)

func main() {
	fmt.Println("=== SKV Encryption Demo ===\n")

	// Example 1: Database without encryption
	fmt.Println("1. Database WITHOUT encryption:")
	db1, err := skv.Open("plaintext_demo.skv")
	if err != nil {
		log.Fatal(err)
	}

	err = db1.Put([]byte("username"), []byte("alice"))
	if err != nil {
		log.Fatal(err)
	}

	err = db1.Put([]byte("password"), []byte("secret123"))
	if err != nil {
		log.Fatal(err)
	}

	db1.Close()
	fmt.Println("   ✓ Data stored in plaintext")
	fmt.Println()

	// Example 2: Database with EasyAES encryption
	fmt.Println("2. Database with EasyAES encryption:")
	opts := &skv.Options{
		Encryption:         skv.EncryptionAES,
		EncryptionPassword: "my-super-secret-password",
		Compression:        skv.CompressionNone,
		Logger:             skv.NullLogger(),
	}

	db2, err := skv.OpenWithOptions("encrypted_easyaes.skv", opts)
	if err != nil {
		log.Fatal(err)
	}

	err = db2.Put([]byte("username"), []byte("alice"))
	if err != nil {
		log.Fatal(err)
	}

	err = db2.Put([]byte("password"), []byte("secret123"))
	if err != nil {
		log.Fatal(err)
	}

	err = db2.Put([]byte("credit_card"), []byte("1234-5678-9012-3456"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Data stored with EasyAES encryption")

	username, err := db2.Get([]byte("username"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Read back: username = %s\n", string(username))

	password, err := db2.Get([]byte("password"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Read back: password = %s\n", string(password))

	db2.Close()
	fmt.Println()

	// Example 3: Database with SimpleCipher encryption
	fmt.Println("3. Database with SimpleCipher encryption:")
	opts3 := &skv.Options{
		Encryption:         skv.EncryptionSimpleCipher,
		EncryptionPassword: "another-secret-key",
		Compression:        skv.CompressionNone,
		Logger:             skv.NullLogger(),
	}

	db3, err := skv.OpenWithOptions("encrypted_simplecipher.skv", opts3)
	if err != nil {
		log.Fatal(err)
	}

	err = db3.Put([]byte("api_key"), []byte("sk-1234567890abcdef"))
	if err != nil {
		log.Fatal(err)
	}

	err = db3.Put([]byte("token"), []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Data stored with SimpleCipher encryption")

	apiKey, err := db3.Get([]byte("api_key"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Read back: api_key = %s\n", string(apiKey))

	db3.Close()
	fmt.Println()

	// Example 4: Encryption with Compression
	fmt.Println("4. Database with BOTH encryption AND compression:")
	opts4 := &skv.Options{
		Encryption:         skv.EncryptionAES,
		EncryptionPassword: "compress-and-encrypt",
		Compression:        skv.CompressionSnappy,
		Logger:             skv.NullLogger(),
	}

	db4, err := skv.OpenWithOptions("encrypted_compressed.skv", opts4)
	if err != nil {
		log.Fatal(err)
	}

	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}

	err = db4.Put([]byte("large_file"), largeData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Large data encrypted and compressed")

	retrieved, err := db4.Get([]byte("large_file"))
	if err != nil {
		log.Fatal(err)
	}

	if len(retrieved) == len(largeData) {
		fmt.Printf("   ✓ Retrieved %d bytes correctly\n", len(retrieved))
	}

	db4.Close()
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println("✓ Encryption is applied to BOTH keys and values")
	fmt.Println("✓ Encryption happens BEFORE compression")
	fmt.Println("✓ Supports EasyAES and SimpleCipher")
	fmt.Println("✓ Base64 encoding is used for encrypted data")
	fmt.Println()

	// Cleanup
	os.Remove("plaintext_demo.skv")
	os.Remove("plaintext_demo.skv.wal")
	os.Remove("encrypted_easyaes.skv")
	os.Remove("encrypted_easyaes.skv.wal")
	os.Remove("encrypted_simplecipher.skv")
	os.Remove("encrypted_simplecipher.skv.wal")
	os.Remove("encrypted_compressed.skv")
	os.Remove("encrypted_compressed.skv.wal")
}
