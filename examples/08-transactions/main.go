package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jncss/skv"
)

func main() {
	fmt.Println("=== SKV Atomic Transactions Demo ===\n")

	// Clean up any existing demo file
	os.Remove("demo_transactions.skv")
	os.Remove("demo_transactions.skv.wal")

	// Open database
	db, err := skv.Open("demo_transactions.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Demo 1: Basic Transaction
	fmt.Println("1. Basic Transaction - Creating Multiple Users")
	fmt.Println("   Creating 3 users atomically...")

	tx := db.Begin()
	tx.PutString("user:alice", `{"name":"Alice","age":30,"role":"admin"}`)
	tx.PutString("user:bob", `{"name":"Bob","age":25,"role":"user"}`)
	tx.PutString("user:charlie", `{"name":"Charlie","age":35,"role":"user"}`)

	fmt.Printf("   Transaction has %d operations\n", tx.Len())
	fmt.Printf("   Transaction ID: %d\n", tx.ID())

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ All users created atomically")
	fmt.Printf("   Database now has %d keys\n\n", db.Count())

	// Demo 2: Bank Transfer
	fmt.Println("2. Atomic Bank Transfer")

	// Setup accounts
	db.PutString("balance:alice", "1000.00")
	db.PutString("balance:bob", "500.00")

	fmt.Println("   Initial balances:")
	aliceBalance, _ := db.GetString("balance:alice")
	bobBalance, _ := db.GetString("balance:bob")
	fmt.Printf("   - Alice: $%s\n", aliceBalance)
	fmt.Printf("   - Bob:   $%s\n", bobBalance)

	// Transfer $100 from Alice to Bob
	fmt.Println("\n   Transferring $100 from Alice to Bob...")

	tx = db.Begin()

	// Deduct from Alice
	alice, _ := strconv.ParseFloat(aliceBalance, 64)
	tx.UpdateString("balance:alice", fmt.Sprintf("%.2f", alice-100))

	// Add to Bob
	bob, _ := strconv.ParseFloat(bobBalance, 64)
	tx.UpdateString("balance:bob", fmt.Sprintf("%.2f", bob+100))

	// Log the transfer
	transferID := fmt.Sprintf("transfer:%d", time.Now().Unix())
	tx.PutString(transferID, "alice->bob:100.00")

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Transfer completed atomically")

	aliceBalance, _ = db.GetString("balance:alice")
	bobBalance, _ = db.GetString("balance:bob")
	fmt.Println("\n   Final balances:")
	fmt.Printf("   - Alice: $%s\n", aliceBalance)
	fmt.Printf("   - Bob:   $%s\n\n", bobBalance)

	// Demo 3: Transaction Rollback
	fmt.Println("3. Transaction Rollback")
	fmt.Println("   Starting transaction with Put operations...")

	tx = db.Begin()
	tx.PutString("temp:key1", "value1")
	tx.PutString("temp:key2", "value2")
	tx.PutString("temp:key3", "value3")

	fmt.Printf("   Transaction has %d operations\n", tx.Len())
	fmt.Println("   Rolling back...")

	if err := tx.Rollback(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Transaction rolled back")

	// Verify keys don't exist
	if !db.HasString("temp:key1") && !db.HasString("temp:key2") && !db.HasString("temp:key3") {
		fmt.Println("   ✓ No keys were created (rollback successful)\n")
	}

	// Demo 4: Validation Error
	fmt.Println("4. Validation Error - Attempting to Put Existing Key")

	// This key already exists
	existingKey := "user:alice"
	fmt.Printf("   Key '%s' already exists\n", existingKey)

	tx = db.Begin()
	tx.PutString("user:dave", `{"name":"Dave","age":40}`)
	tx.PutString(existingKey, "new value") // This will fail!

	fmt.Println("   Attempting to commit...")
	if err := tx.Commit(); err != nil {
		fmt.Printf("   ✗ Commit failed (as expected): %v\n", err)
		fmt.Println("   ✓ All operations rolled back automatically")

		// Verify dave wasn't created
		if !db.HasString("user:dave") {
			fmt.Println("   ✓ user:dave was not created (transaction atomicity preserved)\n")
		}
	}

	// Demo 5: Mixed Operations
	fmt.Println("5. Mixed Operations - Put, Update, Delete")

	db.PutString("status:active", "true")
	db.PutString("status:pending", "true")

	tx = db.Begin()
	tx.PutString("status:new", "true")         // New key
	tx.UpdateString("status:active", "false")  // Update existing
	tx.DeleteString("status:pending")          // Delete existing

	fmt.Println("   Operations:")
	fmt.Println("   - Put:    status:new")
	fmt.Println("   - Update: status:active")
	fmt.Println("   - Delete: status:pending")

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ All operations committed atomically")

	// Verify results
	newVal, _ := db.GetString("status:new")
	activeVal, _ := db.GetString("status:active")
	fmt.Println("\n   Results:")
	fmt.Printf("   - status:new: %s\n", newVal)
	fmt.Printf("   - status:active: %s\n", activeVal)
	if !db.HasString("status:pending") {
		fmt.Println("   - status:pending: (deleted)\n")
	}

	// Demo 6: Large Transaction
	fmt.Println("6. Large Transaction - Batch Insert")

	tx = db.Begin()
	for i := 1; i <= 50; i++ {
		key := fmt.Sprintf("product:%03d", i)
		value := fmt.Sprintf(`{"id":%d,"name":"Product %d","price":%.2f}`, i, i, float64(i)*9.99)
		tx.PutString(key, value)
	}

	fmt.Printf("   Creating 50 products in single transaction...\n")
	fmt.Printf("   Transaction size: %d operations\n", tx.Len())

	startTime := time.Now()
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	duration := time.Since(startTime)

	fmt.Printf("   ✓ All 50 products created atomically\n")
	fmt.Printf("   Commit time: %v\n", duration)
	fmt.Printf("   Total keys in database: %d\n\n", db.Count())

	// Demo 7: Transaction Recovery Simulation
	fmt.Println("7. Transaction Recovery (Simulated)")
	fmt.Println("   This demonstrates that committed transactions survive crashes")
	fmt.Println("   while incomplete transactions are discarded.\n")

	fmt.Println("   Scenario 1: Committed transaction")
	db.Close()
	db, _ = skv.Open("demo_transactions.skv")
	defer db.Close()

	tx = db.Begin()
	tx.PutString("persistent:key1", "value1")
	tx.PutString("persistent:key2", "value2")
	tx.Commit()

	db.Close()

	// Reopen - committed transaction should be there
	db, _ = skv.Open("demo_transactions.skv")
	if db.HasString("persistent:key1") && db.HasString("persistent:key2") {
		fmt.Println("   ✓ Committed transaction survived database reopen")
	}

	fmt.Println("\n   Scenario 2: Incomplete transaction (simulated)")
	fmt.Println("   In a real crash, uncommitted transactions are discarded")
	fmt.Println("   The WAL ensures only committed transactions are applied\n")

	// Demo 8: Performance Stats
	fmt.Println("8. Transaction Performance")
	fmt.Println("   Testing sequential transaction throughput...")

	startTime = time.Now()
	txCount := 100
	for i := 0; i < txCount; i++ {
		tx = db.Begin()
		tx.PutString(fmt.Sprintf("perf:%d", i), fmt.Sprintf("value_%d", i))
		tx.Commit()
	}
	duration = time.Since(startTime)

	fmt.Printf("   Completed %d transactions\n", txCount)
	fmt.Printf("   Total time: %v\n", duration)
	fmt.Printf("   Average: %v per transaction\n", duration/time.Duration(txCount))
	fmt.Printf("   Throughput: %.0f tx/sec\n\n", float64(txCount)/duration.Seconds())

	// Summary
	fmt.Println("=== Summary ===")
	stats, _ := db.Verify()
	fmt.Printf("Total records: %d\n", stats.TotalRecords)
	fmt.Printf("Active records: %d\n", stats.ActiveRecords)
	fmt.Printf("Deleted records: %d\n", stats.DeletedRecords)
	fmt.Printf("Database size: %d bytes\n", stats.FileSize)

	fmt.Println("\n✓ All transaction demos completed successfully!")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("  - Transactions provide all-or-nothing atomicity")
	fmt.Println("  - Validation ensures database consistency")
	fmt.Println("  - Committed transactions are durable (survive crashes)")
	fmt.Println("  - Rollback is always available before commit")
	fmt.Println("  - Mix Put, Update, Delete in single transaction")
}
