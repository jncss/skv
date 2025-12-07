package skv

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Index represents a secondary index for efficient lookups by alternative keys
type Index struct {
	name    string
	keyFunc func([]byte) []byte // Function to extract secondary key from value
	index   map[string][]string // secondary_key -> []primary_key (supports duplicates)
	mu      sync.RWMutex        // Mutex for thread-safe index access
}

// CreateIndex creates a new secondary index with the given name and key extraction function.
// The keyFunc receives the record value and should return the secondary key to index,
// or nil if the value should not be indexed.
//
// Example:
//
//	db.CreateIndex("by_email", func(data []byte) []byte {
//	    var user struct { Email string `json:"email"` }
//	    json.Unmarshal(data, &user)
//	    return []byte(user.Email)
//	})
func (s *SKV) CreateIndex(name string, keyFunc func([]byte) []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if index already exists
	if _, exists := s.indexes[name]; exists {
		return fmt.Errorf("index %s already exists", name)
	}

	idx := &Index{
		name:    name,
		keyFunc: keyFunc,
		index:   make(map[string][]string),
	}

	// Build index from existing data
	for primaryKey, pos := range s.cache {
		// Seek to record position
		if _, err := s.file.Seek(pos, io.SeekStart); err != nil {
			continue
		}

		// Read record data
		_, _, data, _, err := s.readRecord(true)
		if err != nil {
			continue
		}

		// Extract secondary key
		secondaryKey := keyFunc(data)
		if secondaryKey != nil {
			secKey := string(secondaryKey)
			idx.mu.Lock()
			if !contains(idx.index[secKey], primaryKey) {
				idx.index[secKey] = append(idx.index[secKey], primaryKey)
			}
			idx.mu.Unlock()
		}
	}

	s.indexes[name] = idx
	return nil
}

// DropIndex removes a secondary index
func (s *SKV) DropIndex(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.indexes[name]; !exists {
		return fmt.Errorf("index %s not found", name)
	}

	delete(s.indexes, name)
	return nil
}

// GetByIndex retrieves a value using a secondary index.
// Returns ErrKeyNotFound if the secondary key is not found in the index.
//
// Example:
//
//	data, err := db.GetByIndex("by_email", []byte("user@example.com"))
func (s *SKV) GetByIndex(indexName string, secondaryKey []byte) ([]byte, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	idx.mu.RLock()
	primaryKeys, exists := idx.index[string(secondaryKey)]
	idx.mu.RUnlock()

	if !exists || len(primaryKeys) == 0 {
		return nil, ErrKeyNotFound
	}

	// Return first matching primary key
	return s.Get([]byte(primaryKeys[0]))
}

// GetAllByIndex retrieves all values for a secondary key that may have multiple matches.
// Returns a slice of values corresponding to all primary keys with the given secondary key.
// Returns ErrKeyNotFound if the secondary key is not found in the index.
//
// Example:
//
//	values, err := db.GetAllByIndex("by_category", []byte("electronics"))
func (s *SKV) GetAllByIndex(indexName string, secondaryKey []byte) ([][]byte, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	idx.mu.RLock()
	primaryKeys, exists := idx.index[string(secondaryKey)]
	idx.mu.RUnlock()

	if !exists || len(primaryKeys) == 0 {
		return nil, ErrKeyNotFound
	}

	// Retrieve all values
	values := make([][]byte, 0, len(primaryKeys))
	for _, pk := range primaryKeys {
		value, err := s.Get([]byte(pk))
		if err == nil {
			values = append(values, value)
		}
	}

	if len(values) == 0 {
		return nil, ErrKeyNotFound
	}

	return values, nil
}

// GetByIndexString is a convenience wrapper for GetByIndex using string keys
func (s *SKV) GetByIndexString(indexName string, secondaryKey string) ([]byte, error) {
	return s.GetByIndex(indexName, []byte(secondaryKey))
}

// HasIndex checks if a secondary key exists in the given index
func (s *SKV) HasIndex(indexName string, secondaryKey []byte) bool {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()
	primaryKeys, exists := idx.index[string(secondaryKey)]
	return exists && len(primaryKeys) > 0
}

// HasIndexString is a convenience wrapper for HasIndex using string keys
func (s *SKV) HasIndexString(indexName string, secondaryKey string) bool {
	return s.HasIndex(indexName, []byte(secondaryKey))
}

// GetAllByIndexString is a convenience wrapper for GetAllByIndex using string keys
func (s *SKV) GetAllByIndexString(indexName string, secondaryKey string) ([][]byte, error) {
	return s.GetAllByIndex(indexName, []byte(secondaryKey))
}

// ListIndexes returns the names of all created indexes
func (s *SKV) ListIndexes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.indexes))
	for name := range s.indexes {
		names = append(names, name)
	}
	return names
}

// IndexSize returns the number of entries in the given index
func (s *SKV) IndexSize(name string) int {
	s.mu.RLock()
	idx, exists := s.indexes[name]
	s.mu.RUnlock()

	if !exists {
		return 0
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.index)
}

// SaveIndex saves an index to a JSON file for persistence
func (s *SKV) SaveIndex(indexName, filename string) error {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("index %s not found", indexName)
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	data, err := json.MarshalIndent(idx.index, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling index: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing index file: %w", err)
	}

	return nil
}

// LoadIndex loads an index from a JSON file
func (s *SKV) LoadIndex(indexName, filename string, keyFunc func([]byte) []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if index already exists
	if _, exists := s.indexes[indexName]; exists {
		return fmt.Errorf("index %s already exists", indexName)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading index file: %w", err)
	}

	indexMap := make(map[string][]string)
	if err := json.Unmarshal(data, &indexMap); err != nil {
		return fmt.Errorf("error unmarshaling index: %w", err)
	}

	s.indexes[indexName] = &Index{
		name:    indexName,
		keyFunc: keyFunc,
		index:   indexMap,
	}

	return nil
}

// RebuildIndex rebuilds an existing index from scratch
func (s *SKV) RebuildIndex(name string) error {
	s.mu.Lock()
	idx, exists := s.indexes[name]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("index %s not found", name)
	}

	// Clear existing index
	idx.mu.Lock()
	idx.index = make(map[string][]string)
	idx.mu.Unlock()
	s.mu.Unlock()

	// Rebuild from data
	s.mu.RLock()
	defer s.mu.RUnlock()

	for primaryKey, pos := range s.cache {
		// Seek to record position
		if _, err := s.file.Seek(pos, io.SeekStart); err != nil {
			continue
		}

		// Read record data
		_, _, data, _, err := s.readRecord(true)
		if err != nil {
			continue
		}

		// Extract secondary key
		secondaryKey := idx.keyFunc(data)
		if secondaryKey != nil {
			secKey := string(secondaryKey)
			idx.mu.Lock()
			if !contains(idx.index[secKey], primaryKey) {
				idx.index[secKey] = append(idx.index[secKey], primaryKey)
			}
			idx.mu.Unlock()
		}
	}

	return nil
}

// updateIndexes updates all indexes when a record is added or modified
// Must be called with s.mu locked
func (s *SKV) updateIndexes(primaryKey []byte, data []byte) {
	pkStr := string(primaryKey)
	for _, idx := range s.indexes {
		secondaryKey := idx.keyFunc(data)
		if secondaryKey != nil {
			secKey := string(secondaryKey)
			idx.mu.Lock()
			// Check if primary key already exists in this secondary key's slice
			if !contains(idx.index[secKey], pkStr) {
				idx.index[secKey] = append(idx.index[secKey], pkStr)
			}
			idx.mu.Unlock()
		}
	}
}

// removeFromIndexes removes a primary key from all indexes
// Must be called with s.mu locked
func (s *SKV) removeFromIndexes(primaryKey string) {
	for _, idx := range s.indexes {
		idx.mu.Lock()
		// Find and remove entries pointing to this primary key
		for secKey, primKeys := range idx.index {
			idx.index[secKey] = removeString(primKeys, primaryKey)
			// Clean up empty slices
			if len(idx.index[secKey]) == 0 {
				delete(idx.index, secKey)
			}
		}
		idx.mu.Unlock()
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// Helper function to remove a string from a slice
func removeString(slice []string, str string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != str {
			result = append(result, s)
		}
	}
	return result
}
