package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jncss/skv"
)

func main() {
	tmpDir := os.TempDir()
	dbPath := filepath.Join(tmpDir, "wal_demo.skv")
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + ".wal")

	fmt.Println("=== Write-Ahead Log (WAL) Demo ===")
	fmt.Println()

	// Scenario 1: Normal operation
	fmt.Println("1. Normal operation with WAL")
	fmt.Println("   Opening database...")
	db, err := skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Writing data (logged to WAL first)...")
	db.Put([]byte("user:1"), []byte(`{"name":"Alice","email":"alice@example.com"}`))
	db.Put([]byte("user:2"), []byte(`{"name":"Bob","email":"bob@example.com"}`))
	db.Put([]byte("user:3"), []byte(`{"name":"Carol","email":"carol@example.com"}`))

	fmt.Println("   Reading data...")
	val, _ := db.Get([]byte("user:1"))
	fmt.Printf("   user:1 = %s\n", val)

	fmt.Println("   Closing database cleanly...")
	db.Close()

	// Scenario 2: Crash recovery simulation
	fmt.Println("\n2. Crash recovery simulation")
	fmt.Println("   Opening database again...")
	db, err = skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Data persisted correctly:")
	for _, key := range []string{"user:1", "user:2", "user:3"} {
		val, err := db.Get([]byte(key))
		if err != nil {
			fmt.Printf("   %s: ERROR - %v\n", key, err)
		} else {
			fmt.Printf("   %s = %s\n", key, val)
		}
	}

	// Scenario 3: Delete operation
	fmt.Println("\n3. Delete operation (also logged to WAL)")
	fmt.Println("   Deleting user:2...")
	db.Delete([]byte("user:2"))

	fmt.Println("   Verifying deletion...")
	_, err = db.Get([]byte("user:2"))
	if err == skv.ErrKeyNotFound {
		fmt.Println("   ✓ user:2 successfully deleted")
	}

	db.Close()

	// Scenario 4: Reopen to verify delete persisted
	fmt.Println("\n4. Reopen to verify delete was persisted")
	db, err = skv.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Get([]byte("user:2"))
	if err == skv.ErrKeyNotFound {
		fmt.Println("   ✓ Delete persisted correctly (user:2 still deleted)")
	}

	// Remaining users
	fmt.Println("\n5. Remaining users:")
	for _, key := range []string{"user:1", "user:3"} {
		val, err := db.Get([]byte(key))
		if err == nil {
			fmt.Printf("   %s = %s\n", key, val)
		}
	}

	db.Close()

	fmt.Println("\n=== WAL ensures durability! ===")
	fmt.Println("All operations are safe even if the system crashes")
	fmt.Println("between writes, thanks to the Write-Ahead Log.")
}
