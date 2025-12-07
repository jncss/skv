package skv

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestPutCtx tests the PutCtx function with context
func TestPutCtx(t *testing.T) {
	dbPath := "test_put_ctx.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test normal put
	err = db.PutCtx(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("PutCtx failed: %v", err)
	}

	// Verify value
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", string(value))
	}
}

// TestPutCtxCancelled tests PutCtx with a cancelled context
func TestPutCtxCancelled(t *testing.T) {
	dbPath := "test_put_ctx_cancelled.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to put with cancelled context
	err = db.PutCtx(ctx, []byte("key1"), []byte("value1"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestGetCtx tests the GetCtx function with context
func TestGetCtx(t *testing.T) {
	dbPath := "test_get_ctx.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put a value first
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx := context.Background()

	// Test normal get
	value, err := db.GetCtx(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("GetCtx failed: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", string(value))
	}
}

// TestGetCtxCancelled tests GetCtx with a cancelled context
func TestGetCtxCancelled(t *testing.T) {
	dbPath := "test_get_ctx_cancelled.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put a value first
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to get with cancelled context
	_, err = db.GetCtx(ctx, []byte("key1"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestUpdateCtx tests the UpdateCtx function with context
func TestUpdateCtx(t *testing.T) {
	dbPath := "test_update_ctx.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx := context.Background()

	// Update with context
	err = db.UpdateCtx(ctx, []byte("key1"), []byte("value2"))
	if err != nil {
		t.Fatalf("UpdateCtx failed: %v", err)
	}

	// Verify updated value
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(value) != "value2" {
		t.Errorf("Expected value2, got %s", string(value))
	}
}

// TestUpdateCtxCancelled tests UpdateCtx with a cancelled context
func TestUpdateCtxCancelled(t *testing.T) {
	dbPath := "test_update_ctx_cancelled.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put initial value
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to update with cancelled context
	err = db.UpdateCtx(ctx, []byte("key1"), []byte("value2"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Verify value wasn't updated
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1 (unchanged), got %s", string(value))
	}
}

// TestDeleteCtx tests the DeleteCtx function with context
func TestDeleteCtx(t *testing.T) {
	dbPath := "test_delete_ctx.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put a value first
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx := context.Background()

	// Delete with context
	err = db.DeleteCtx(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("DeleteCtx failed: %v", err)
	}

	// Verify deletion
	_, err = db.Get([]byte("key1"))
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

// TestDeleteCtxCancelled tests DeleteCtx with a cancelled context
func TestDeleteCtxCancelled(t *testing.T) {
	dbPath := "test_delete_ctx_cancelled.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Put a value first
	err = db.Put([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to delete with cancelled context
	err = db.DeleteCtx(ctx, []byte("key1"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Verify key still exists
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1 (not deleted), got %s", string(value))
	}
}

// TestCompactCtx tests the CompactCtx function with context
func TestCompactCtx(t *testing.T) {
	dbPath := "test_compact_ctx.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add and delete some records
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i)}
		err = db.Put(key, value)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Delete half of them
	for i := 0; i < 50; i++ {
		key := []byte{byte(i)}
		err = db.Delete(key)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

	ctx := context.Background()

	// Compact with context
	err = db.CompactCtx(ctx)
	if err != nil {
		t.Fatalf("CompactCtx failed: %v", err)
	}

	// Verify stats after compaction
	stats, err := db.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if stats.ActiveRecords != 50 {
		t.Errorf("Expected 50 active records, got %d", stats.ActiveRecords)
	}
	if stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records after compact, got %d", stats.DeletedRecords)
	}
}

// TestCompactCtxCancelled tests CompactCtx with a cancelled context
func TestCompactCtxCancelled(t *testing.T) {
	dbPath := "test_compact_ctx_cancelled.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add some records
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i)}
		err = db.Put(key, value)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to compact with cancelled context
	err = db.CompactCtx(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestContextTimeout tests operations with timeout context
func TestContextTimeout(t *testing.T) {
	dbPath := "test_context_timeout.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Try to put with timed out context
	err = db.PutCtx(ctx, []byte("key1"), []byte("value1"))
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

// TestContextPropagation tests that context is properly checked throughout operations
func TestContextPropagation(t *testing.T) {
	dbPath := "test_context_propagation.skv"
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Add many records to make compact take some time
	for i := 0; i < 1000; i++ {
		key := []byte{byte(i / 256), byte(i % 256)}
		value := make([]byte, 1000)
		err = db.Put(key, value)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Delete half
	for i := 0; i < 500; i++ {
		key := []byte{byte(i / 256), byte(i % 256)}
		err = db.Delete(key)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

	// Create context that will be cancelled during compact
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Try to compact - should be cancelled mid-operation
	err = db.CompactCtx(ctx)
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Logf("Expected context error, got %v (may complete before cancellation)", err)
	}
}
