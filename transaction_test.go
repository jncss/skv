package skv

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestTransactionBasic tests basic transaction functionality
func TestTransactionBasic(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_basic.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Begin transaction
	tx := skv.Begin()
	if tx == nil {
		t.Fatal("Begin() returned nil transaction")
	}

	// Add operations
	if err := tx.PutString("key1", "value1"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := tx.PutString("key2", "value2"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := tx.PutString("key3", "value3"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Check operation count
	if tx.Len() != 3 {
		t.Errorf("expected 3 operations, got %d", tx.Len())
	}

	// Keys should not be visible yet
	if _, err := skv.GetString("key1"); err != ErrKeyNotFound {
		t.Error("key should not be visible before commit")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Now keys should be visible
	val, err := skv.GetString("key1")
	if err != nil {
		t.Fatalf("failed to get key1: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	val, err = skv.GetString("key2")
	if err != nil {
		t.Fatalf("failed to get key2: %v", err)
	}
	if val != "value2" {
		t.Errorf("expected value2, got %s", val)
	}

	val, err = skv.GetString("key3")
	if err != nil {
		t.Fatalf("failed to get key3: %v", err)
	}
	if val != "value3" {
		t.Errorf("expected value3, got %s", val)
	}

	// Verify transaction state
	if !tx.IsCommitted() {
		t.Error("transaction should be committed")
	}
	if tx.IsRolledBack() {
		t.Error("transaction should not be rolled back")
	}
}

// TestTransactionRollback tests transaction rollback
func TestTransactionRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_rollback.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Add some initial data
	if err := skv.PutString("existing", "data"); err != nil {
		t.Fatalf("failed to put initial data: %v", err)
	}

	// Begin transaction
	tx := skv.Begin()

	// Add operations
	if err := tx.PutString("key1", "value1"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := tx.PutString("key2", "value2"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Rollback transaction
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Keys should not exist
	if _, err := skv.GetString("key1"); err != ErrKeyNotFound {
		t.Error("key1 should not exist after rollback")
	}
	if _, err := skv.GetString("key2"); err != ErrKeyNotFound {
		t.Error("key2 should not exist after rollback")
	}

	// Existing data should still be there
	val, err := skv.GetString("existing")
	if err != nil {
		t.Fatalf("failed to get existing key: %v", err)
	}
	if val != "data" {
		t.Errorf("expected 'data', got %s", val)
	}

	// Verify transaction state
	if tx.IsCommitted() {
		t.Error("transaction should not be committed")
	}
	if !tx.IsRolledBack() {
		t.Error("transaction should be rolled back")
	}
}

// TestTransactionPutExistingKey tests that Put fails if key exists
func TestTransactionPutExistingKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_put_existing.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Add initial key
	if err := skv.PutString("existing", "value"); err != nil {
		t.Fatalf("failed to put initial key: %v", err)
	}

	// Begin transaction
	tx := skv.Begin()
	if err := tx.PutString("key1", "value1"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := tx.PutString("existing", "new_value"); err != nil {
		t.Fatalf("Put should succeed in transaction: %v", err)
	}

	// Commit should fail because "existing" already exists
	err = tx.Commit()
	if err == nil {
		t.Fatal("Commit should have failed for existing key")
	}
	if err != ErrKeyExists && fmt.Sprintf("%v", err) == "" {
		t.Errorf("expected ErrKeyExists, got: %v", err)
	}

	// No keys from transaction should exist
	if _, err := skv.GetString("key1"); err != ErrKeyNotFound {
		t.Error("key1 should not exist after failed commit")
	}

	// Original value should still be there
	val, err := skv.GetString("existing")
	if err != nil {
		t.Fatalf("failed to get existing key: %v", err)
	}
	if val != "value" {
		t.Errorf("expected 'value', got %s", val)
	}
}

// TestTransactionUpdateNonExistingKey tests that Update fails if key doesn't exist
func TestTransactionUpdateNonExistingKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_update_nonexisting.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Add a key
	if err := skv.PutString("key1", "value1"); err != nil {
		t.Fatalf("failed to put key: %v", err)
	}

	// Begin transaction
	tx := skv.Begin()
	if err := tx.UpdateString("key1", "updated1"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if err := tx.UpdateString("nonexisting", "value"); err != nil {
		t.Fatalf("Update should succeed in transaction: %v", err)
	}

	// Commit should fail because "nonexisting" doesn't exist
	err = tx.Commit()
	if err == nil {
		t.Fatal("Commit should have failed for non-existing key")
	}
	if err != ErrKeyNotFound && fmt.Sprintf("%v", err) == "" {
		t.Errorf("expected ErrKeyNotFound, got: %v", err)
	}

	// Original value should still be there
	val, err := skv.GetString("key1")
	if err != nil {
		t.Fatalf("failed to get key1: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %s", val)
	}
}

// TestTransactionDeleteNonExistingKey tests that Delete fails if key doesn't exist
func TestTransactionDeleteNonExistingKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_delete_nonexisting.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Add a key
	if err := skv.PutString("key1", "value1"); err != nil {
		t.Fatalf("failed to put key: %v", err)
	}

	// Begin transaction
	tx := skv.Begin()
	if err := tx.DeleteString("key1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if err := tx.DeleteString("nonexisting"); err != nil {
		t.Fatalf("Delete should succeed in transaction: %v", err)
	}

	// Commit should fail because "nonexisting" doesn't exist
	err = tx.Commit()
	if err == nil {
		t.Fatal("Commit should have failed for non-existing key")
	}

	// Original key should still be there
	val, err := skv.GetString("key1")
	if err != nil {
		t.Fatalf("failed to get key1: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %s", val)
	}
}

// TestTransactionMixedOperations tests transactions with Put, Update, and Delete
func TestTransactionMixedOperations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_mixed.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Add initial data
	if err := skv.PutString("update_me", "old_value"); err != nil {
		t.Fatalf("failed to put initial data: %v", err)
	}
	if err := skv.PutString("delete_me", "will_be_deleted"); err != nil {
		t.Fatalf("failed to put initial data: %v", err)
	}

	// Begin transaction
	tx := skv.Begin()

	// Mix of operations
	if err := tx.PutString("new_key", "new_value"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := tx.UpdateString("update_me", "new_value"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if err := tx.DeleteString("delete_me"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify new key
	val, err := skv.GetString("new_key")
	if err != nil {
		t.Fatalf("failed to get new_key: %v", err)
	}
	if val != "new_value" {
		t.Errorf("expected 'new_value', got %s", val)
	}

	// Verify updated key
	val, err = skv.GetString("update_me")
	if err != nil {
		t.Fatalf("failed to get update_me: %v", err)
	}
	if val != "new_value" {
		t.Errorf("expected 'new_value', got %s", val)
	}

	// Verify deleted key
	if _, err := skv.GetString("delete_me"); err != ErrKeyNotFound {
		t.Error("delete_me should not exist")
	}
}

// TestTransactionEmpty tests empty transaction
func TestTransactionEmpty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_empty.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Begin and commit empty transaction
	tx := skv.Begin()
	if tx.Len() != 0 {
		t.Errorf("expected 0 operations, got %d", tx.Len())
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed for empty transaction: %v", err)
	}

	if !tx.IsCommitted() {
		t.Error("transaction should be committed")
	}
}

// TestTransactionDoubleCommit tests that double commit fails
func TestTransactionDoubleCommit(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_double_commit.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	tx := skv.Begin()
	if err := tx.PutString("key", "value"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// First commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// Second commit should fail
	err = tx.Commit()
	if err == nil {
		t.Fatal("Second commit should have failed")
	}
}

// TestTransactionDoubleRollback tests that double rollback fails
func TestTransactionDoubleRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_double_rollback.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	tx := skv.Begin()
	if err := tx.PutString("key", "value"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// First rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("First rollback failed: %v", err)
	}

	// Second rollback should fail
	err = tx.Rollback()
	if err == nil {
		t.Fatal("Second rollback should have failed")
	}
}

// TestTransactionCommitAfterRollback tests that commit after rollback fails
func TestTransactionCommitAfterRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_commit_after_rollback.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	tx := skv.Begin()
	if err := tx.PutString("key", "value"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Commit should fail
	err = tx.Commit()
	if err == nil {
		t.Fatal("Commit after rollback should have failed")
	}
}

// TestTransactionRecovery tests transaction recovery after crash
func TestTransactionRecovery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_recovery.skv")

	// Create database and commit a transaction
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		tx := skv.Begin()
		if err := tx.PutString("key1", "value1"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := tx.PutString("key2", "value2"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		skv.Close()
	}

	// Reopen database - should recover committed transaction
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to reopen database: %v", err)
		}
		defer skv.Close()

		// Keys should exist
		val, err := skv.GetString("key1")
		if err != nil {
			t.Fatalf("failed to get key1 after recovery: %v", err)
		}
		if val != "value1" {
			t.Errorf("expected 'value1', got %s", val)
		}

		val, err = skv.GetString("key2")
		if err != nil {
			t.Fatalf("failed to get key2 after recovery: %v", err)
		}
		if val != "value2" {
			t.Errorf("expected 'value2', got %s", val)
		}
	}
}

// TestTransactionRecoveryIncomplete tests recovery of incomplete transaction
func TestTransactionRecoveryIncomplete(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_recovery_incomplete.skv")

	// Create database and start a transaction without committing
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		// Add initial data
		if err := skv.PutString("existing", "data"); err != nil {
			t.Fatalf("failed to put existing key: %v", err)
		}

		// Start transaction but don't commit
		tx := skv.Begin()
		if err := tx.PutString("incomplete1", "value1"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := tx.PutString("incomplete2", "value2"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Manually log to WAL to simulate crash before commit
		if skv.wal != nil {
			skv.wal.LogBeginTx(tx.ID())
			skv.wal.LogPut([]byte("incomplete1"), []byte("value1"))
			skv.wal.LogPut([]byte("incomplete2"), []byte("value2"))
			// Don't log commit - simulates crash
		}

		skv.Close()
	}

	// Reopen database - should discard incomplete transaction
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to reopen database: %v", err)
		}
		defer skv.Close()

		// Incomplete keys should not exist
		if _, err := skv.GetString("incomplete1"); err != ErrKeyNotFound {
			t.Error("incomplete1 should not exist after recovery")
		}
		if _, err := skv.GetString("incomplete2"); err != ErrKeyNotFound {
			t.Error("incomplete2 should not exist after recovery")
		}

		// Existing data should still be there
		val, err := skv.GetString("existing")
		if err != nil {
			t.Fatalf("failed to get existing key: %v", err)
		}
		if val != "data" {
			t.Errorf("expected 'data', got %s", val)
		}
	}
}

// TestTransactionRecoveryRolledBack tests recovery of rolled back transaction
func TestTransactionRecoveryRolledBack(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_recovery_rollback.skv")

	// Create database and rollback a transaction
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		tx := skv.Begin()
		if err := tx.PutString("rollback1", "value1"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := tx.PutString("rollback2", "value2"); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		skv.Close()
	}

	// Reopen database - rolled back keys should not exist
	{
		skv, err := Open(dbPath)
		if err != nil {
			t.Fatalf("failed to reopen database: %v", err)
		}
		defer skv.Close()

		if _, err := skv.GetString("rollback1"); err != ErrKeyNotFound {
			t.Error("rollback1 should not exist after recovery")
		}
		if _, err := skv.GetString("rollback2"); err != ErrKeyNotFound {
			t.Error("rollback2 should not exist after recovery")
		}
	}
}

// TestTransactionLargeData tests transactions with large data
func TestTransactionLargeData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_large.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Create large data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	tx := skv.Begin()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("large_%d", i)
		if err := tx.Put([]byte(key), largeData); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify data
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("large_%d", i)
		val, err := skv.Get([]byte(key))
		if err != nil {
			t.Fatalf("failed to get %s: %v", key, err)
		}
		if len(val) != len(largeData) {
			t.Errorf("expected %d bytes, got %d", len(largeData), len(val))
		}
	}
}

// TestTransactionConcurrent tests multiple sequential transactions
func TestTransactionSequential(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx_sequential.skv")

	skv, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer skv.Close()

	// Execute multiple transactions
	for i := 0; i < 100; i++ {
		tx := skv.Begin()

		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)

		if err := tx.PutString(key, value); err != nil {
			t.Fatalf("Put failed in transaction %d: %v", i, err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed for transaction %d: %v", i, err)
		}
	}

	// Verify all keys
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		expectedValue := fmt.Sprintf("value_%d", i)

		val, err := skv.GetString(key)
		if err != nil {
			t.Fatalf("failed to get %s: %v", key, err)
		}
		if val != expectedValue {
			t.Errorf("expected '%s', got '%s'", expectedValue, val)
		}
	}
}
