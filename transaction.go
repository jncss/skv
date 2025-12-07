package skv

import (
	"fmt"
	"sync"
	"time"
)

// Transaction represents an atomic batch of operations.
// All operations within a transaction are either committed together (all-or-nothing)
// or rolled back if any error occurs.
//
// Example usage:
//
//	tx := skv.Begin()
//	tx.Put([]byte("key1"), []byte("value1"))
//	tx.Put([]byte("key2"), []byte("value2"))
//	tx.Delete([]byte("key3"))
//	if err := tx.Commit(); err != nil {
//	    // All operations are rolled back
//	    return err
//	}
//	// All operations are now persisted atomically
type Transaction struct {
	skv        *SKV
	id         uint64
	operations []txOperation
	committed  bool
	rolledBack bool
	mu         sync.Mutex
	startTime  time.Time
}

// txOperation represents a single operation within a transaction
type txOperation struct {
	opType txOpType
	key    []byte
	data   []byte
}

// txOpType represents the type of transaction operation
type txOpType byte

const (
	txOpPut    txOpType = 0x01
	txOpUpdate txOpType = 0x02
	txOpDelete txOpType = 0x03
)

// String returns a human-readable representation of the operation type
func (t txOpType) String() string {
	switch t {
	case txOpPut:
		return "PUT"
	case txOpUpdate:
		return "UPDATE"
	case txOpDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// Begin starts a new transaction.
// The transaction must be committed with Commit() or rolled back with Rollback().
// Operations within the transaction are isolated and not visible to other
// operations until the transaction is committed.
func (s *SKV) Begin() *Transaction {
	s.mu.Lock()
	s.txCounter++
	txID := s.txCounter
	s.mu.Unlock()

	tx := &Transaction{
		skv:        s,
		id:         txID,
		operations: make([]txOperation, 0),
		startTime:  time.Now(),
	}

	if s.logger != nil {
		s.logger.Debug("Transaction started",
			Field{Key: "tx_id", Value: txID})
	}

	return tx
}

// Put adds a Put operation to the transaction.
// The key must not already exist when the transaction is committed.
// Returns an error if the transaction has already been committed or rolled back.
func (tx *Transaction) Put(key, data []byte) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long (max 255 bytes)")
	}

	// Make copies to avoid external modifications
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	tx.operations = append(tx.operations, txOperation{
		opType: txOpPut,
		key:    keyCopy,
		data:   dataCopy,
	})

	return nil
}

// PutString adds a Put operation using strings.
func (tx *Transaction) PutString(key, value string) error {
	return tx.Put([]byte(key), []byte(value))
}

// Update adds an Update operation to the transaction.
// The key must exist when the transaction is committed.
// Returns an error if the transaction has already been committed or rolled back.
func (tx *Transaction) Update(key, data []byte) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long (max 255 bytes)")
	}

	// Make copies to avoid external modifications
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	tx.operations = append(tx.operations, txOperation{
		opType: txOpUpdate,
		key:    keyCopy,
		data:   dataCopy,
	})

	return nil
}

// UpdateString adds an Update operation using strings.
func (tx *Transaction) UpdateString(key, value string) error {
	return tx.Update([]byte(key), []byte(value))
}

// Delete adds a Delete operation to the transaction.
// The key must exist when the transaction is committed.
// Returns an error if the transaction has already been committed or rolled back.
func (tx *Transaction) Delete(key []byte) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	// Make copy to avoid external modifications
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	tx.operations = append(tx.operations, txOperation{
		opType: txOpDelete,
		key:    keyCopy,
		data:   nil,
	})

	return nil
}

// DeleteString adds a Delete operation using a string key.
func (tx *Transaction) DeleteString(key string) error {
	return tx.Delete([]byte(key))
}

// Commit applies all operations in the transaction atomically.
// If any operation fails, all operations are rolled back and an error is returned.
// The transaction cannot be used after calling Commit.
func (tx *Transaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	if len(tx.operations) == 0 {
		tx.committed = true
		if tx.skv.logger != nil {
			tx.skv.logger.Debug("Empty transaction committed",
				Field{Key: "tx_id", Value: tx.id})
		}
		return nil
	}

	// Lock the SKV for the entire transaction
	tx.skv.mu.Lock()
	defer tx.skv.mu.Unlock()

	startTime := time.Now()

	// Log transaction begin to WAL
	if tx.skv.wal != nil {
		if err := tx.skv.wal.LogBeginTx(tx.id); err != nil {
			if tx.skv.logger != nil {
				tx.skv.logger.Error("Failed to log transaction begin",
					Field{Key: "tx_id", Value: tx.id},
					Field{Key: "error", Value: err.Error()})
			}
			return fmt.Errorf("failed to log transaction begin: %w", err)
		}
	}

	// Validate all operations first (before any writes)
	for i, op := range tx.operations {
		key := string(op.key)

		switch op.opType {
		case txOpPut:
			// Key must NOT exist
			if _, exists := tx.skv.cache[key]; exists {
				// Rollback transaction in WAL
				if tx.skv.wal != nil {
					_ = tx.skv.wal.LogRollbackTx(tx.id)
				}
				if tx.skv.logger != nil {
					tx.skv.logger.Warn("Transaction validation failed: key exists",
						Field{Key: "tx_id", Value: tx.id},
						Field{Key: "operation", Value: i},
						Field{Key: "key", Value: key})
				}
				return fmt.Errorf("operation %d: key %q already exists: %w", i, key, ErrKeyExists)
			}

		case txOpUpdate, txOpDelete:
			// Key MUST exist
			if _, exists := tx.skv.cache[key]; !exists {
				// Rollback transaction in WAL
				if tx.skv.wal != nil {
					_ = tx.skv.wal.LogRollbackTx(tx.id)
				}
				if tx.skv.logger != nil {
					tx.skv.logger.Warn("Transaction validation failed: key not found",
						Field{Key: "tx_id", Value: tx.id},
						Field{Key: "operation", Value: i},
						Field{Key: "key", Value: key})
				}
				return fmt.Errorf("operation %d: key %q not found: %w", i, key, ErrKeyNotFound)
			}
		}
	}

	// All validations passed - now execute operations
	// Save original state for rollback
	originalCache := make(map[string]int64)
	for key, pos := range tx.skv.cache {
		originalCache[key] = pos
	}

	// Execute all operations
	for i, op := range tx.operations {
		key := string(op.key)

		// Log to WAL
		if tx.skv.wal != nil {
			var err error
			switch op.opType {
			case txOpPut, txOpUpdate:
				err = tx.skv.wal.LogPut(op.key, op.data)
			case txOpDelete:
				err = tx.skv.wal.LogDelete(op.key)
			}
			if err != nil {
				// Restore original state
				tx.skv.cache = originalCache
				if tx.skv.wal != nil {
					_ = tx.skv.wal.LogRollbackTx(tx.id)
				}
				if tx.skv.logger != nil {
					tx.skv.logger.Error("Failed to log operation to WAL",
						Field{Key: "tx_id", Value: tx.id},
						Field{Key: "operation", Value: i},
						Field{Key: "error", Value: err.Error()})
				}
				return fmt.Errorf("failed to log operation %d: %w", i, err)
			}
		}

		// Write to data file
		switch op.opType {
		case txOpPut, txOpUpdate:
			recordPos, err := tx.skv.writeRecord(op.key, op.data)
			if err != nil {
				// Restore original state
				tx.skv.cache = originalCache
				if tx.skv.wal != nil {
					_ = tx.skv.wal.LogRollbackTx(tx.id)
				}
				if tx.skv.logger != nil {
					tx.skv.logger.Error("Failed to write record",
						Field{Key: "tx_id", Value: tx.id},
						Field{Key: "operation", Value: i},
						Field{Key: "error", Value: err.Error()})
				}
				return fmt.Errorf("failed to write record for operation %d: %w", i, err)
			}
			tx.skv.cache[key] = recordPos

		case txOpDelete:
			if err := tx.skv.deleteInternal(op.key); err != nil {
				// Restore original state
				tx.skv.cache = originalCache
				if tx.skv.wal != nil {
					_ = tx.skv.wal.LogRollbackTx(tx.id)
				}
				if tx.skv.logger != nil {
					tx.skv.logger.Error("Failed to delete record",
						Field{Key: "tx_id", Value: tx.id},
						Field{Key: "operation", Value: i},
						Field{Key: "error", Value: err.Error()})
				}
				return fmt.Errorf("failed to delete record for operation %d: %w", i, err)
			}
			delete(tx.skv.cache, key)
		}
	}

	// Log commit to WAL
	if tx.skv.wal != nil {
		if err := tx.skv.wal.LogCommitTx(tx.id); err != nil {
			// This is serious - we've applied changes but can't log commit
			// We can't easily rollback at this point
			if tx.skv.logger != nil {
				tx.skv.logger.Error("Failed to log transaction commit (changes applied)",
					Field{Key: "tx_id", Value: tx.id},
					Field{Key: "error", Value: err.Error()})
			}
			return fmt.Errorf("failed to log transaction commit: %w", err)
		}
	}

	// Sync to disk
	if err := tx.skv.file.Sync(); err != nil {
		if tx.skv.logger != nil {
			tx.skv.logger.Warn("Failed to sync file after transaction",
				Field{Key: "tx_id", Value: tx.id},
				Field{Key: "error", Value: err.Error()})
		}
	}

	tx.committed = true
	duration := time.Since(startTime)

	if tx.skv.logger != nil {
		tx.skv.logger.Info("Transaction committed",
			Field{Key: "tx_id", Value: tx.id},
			Field{Key: "operations", Value: len(tx.operations)},
			Field{Key: "duration_ms", Value: duration.Milliseconds()})
	}

	return nil
}

// Rollback discards all operations in the transaction.
// The transaction cannot be used after calling Rollback.
func (tx *Transaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	// Log rollback to WAL if we have operations
	if len(tx.operations) > 0 && tx.skv.wal != nil {
		if err := tx.skv.wal.LogRollbackTx(tx.id); err != nil {
			if tx.skv.logger != nil {
				tx.skv.logger.Warn("Failed to log transaction rollback",
					Field{Key: "tx_id", Value: tx.id},
					Field{Key: "error", Value: err.Error()})
			}
		}
	}

	tx.rolledBack = true
	tx.operations = nil

	if tx.skv.logger != nil {
		tx.skv.logger.Debug("Transaction rolled back",
			Field{Key: "tx_id", Value: tx.id})
	}

	return nil
}

// Len returns the number of operations in the transaction.
func (tx *Transaction) Len() int {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return len(tx.operations)
}

// ID returns the unique identifier for this transaction.
func (tx *Transaction) ID() uint64 {
	return tx.id
}

// IsCommitted returns true if the transaction has been committed.
func (tx *Transaction) IsCommitted() bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.committed
}

// IsRolledBack returns true if the transaction has been rolled back.
func (tx *Transaction) IsRolledBack() bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.rolledBack
}
