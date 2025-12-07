package skv

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Cursor represents an iterator for traversing records in order
type Cursor struct {
	skv       *SKV
	keys      []string // Sorted keys
	current   int      // Current position
	indexName string   // Index name (empty for primary key cursor)
	fromKey   string   // Start key (inclusive, empty for beginning)
	toKey     string   // End key (inclusive, empty for end)
	reverse   bool     // Traverse in reverse order
	closed    bool     // Whether cursor is closed
}

// CursorOptions configures cursor behavior
type CursorOptions struct {
	From    []byte // Start key (inclusive), nil for beginning
	To      []byte // End key (inclusive), nil for end
	Reverse bool   // Traverse in reverse order
}

// NewCursor creates a cursor for traversing primary keys in order
func (s *SKV) NewCursor(opts *CursorOptions) *Cursor {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if opts == nil {
		opts = &CursorOptions{}
	}

	// Collect and sort primary keys
	keys := make([]string, 0, len(s.cache))
	for key := range s.cache {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	cursor := &Cursor{
		skv:     s,
		keys:    keys,
		current: -1, // Start before first element
		reverse: opts.Reverse,
	}

	if opts.From != nil {
		cursor.fromKey = string(opts.From)
	}
	if opts.To != nil {
		cursor.toKey = string(opts.To)
	}

	// Apply range filtering
	cursor.applyRange()

	return cursor
}

// NewIndexCursor creates a cursor for traversing an index in order
func (s *SKV) NewIndexCursor(indexName string, opts *CursorOptions) (*Cursor, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s not found", indexName)
	}

	if opts == nil {
		opts = &CursorOptions{}
	}

	// Collect and sort secondary keys from index, then expand to primary keys
	idx.mu.RLock()
	type keyPair struct {
		secondary string
		primary   string
	}
	var pairs []keyPair

	for secKey, primKeys := range idx.index {
		// Apply range filtering on secondary keys
		if opts.From != nil && secKey < string(opts.From) {
			continue
		}
		if opts.To != nil && secKey > string(opts.To) {
			continue
		}

		for _, pk := range primKeys {
			pairs = append(pairs, keyPair{secondary: secKey, primary: pk})
		}
	}
	idx.mu.RUnlock()

	// Sort by secondary key first, then by primary key for stable ordering
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].secondary != pairs[j].secondary {
			return pairs[i].secondary < pairs[j].secondary
		}
		return pairs[i].primary < pairs[j].primary
	})

	// Extract primary keys in sorted order
	keys := make([]string, len(pairs))
	for i, pair := range pairs {
		keys[i] = pair.primary
	}

	cursor := &Cursor{
		skv:       s,
		keys:      keys,
		current:   -1,
		indexName: indexName,
		reverse:   opts.Reverse,
	}

	// Range filtering already applied above for index cursors
	// No need to call applyRange() here

	return cursor, nil
}

// applyRange filters keys based on from/to range
func (c *Cursor) applyRange() {
	if c.fromKey == "" && c.toKey == "" {
		return // No filtering needed
	}

	filtered := make([]string, 0, len(c.keys))
	for _, key := range c.keys {
		// Check from boundary
		if c.fromKey != "" && key < c.fromKey {
			continue
		}
		// Check to boundary
		if c.toKey != "" && key > c.toKey {
			continue
		}
		filtered = append(filtered, key)
	}

	c.keys = filtered
}

// Next advances the cursor and returns the next key-value pair
// Returns io.EOF when there are no more records
func (c *Cursor) Next() (key []byte, value []byte, err error) {
	if c.closed {
		return nil, nil, fmt.Errorf("cursor is closed")
	}

	// Move to next position
	if c.reverse {
		if c.current == -1 {
			c.current = len(c.keys) // Start at end
		}
		c.current--
		if c.current < 0 {
			return nil, nil, io.EOF
		}
	} else {
		c.current++
		if c.current >= len(c.keys) {
			return nil, nil, io.EOF
		}
	}

	currentKey := c.keys[c.current]

	// Get value
	var data []byte
	if c.indexName == "" {
		// Primary key cursor
		data, err = c.skv.Get([]byte(currentKey))
	} else {
		// Index cursor - currentKey is already a primary key
		data, err = c.skv.Get([]byte(currentKey))
	}

	if err != nil {
		return nil, nil, err
	}

	return []byte(currentKey), data, nil
}

// Seek positions the cursor at the first key >= the given key
// For reverse cursors, positions at first key <= the given key
func (c *Cursor) Seek(key []byte) error {
	if c.closed {
		return fmt.Errorf("cursor is closed")
	}

	searchKey := string(key)

	if c.reverse {
		// Find last key <= searchKey
		for i := len(c.keys) - 1; i >= 0; i-- {
			if c.keys[i] <= searchKey {
				c.current = i + 1 // Will be decremented by Next()
				return nil
			}
		}
		// All keys are > searchKey
		c.current = 0 // Will immediately return EOF
	} else {
		// Find first key >= searchKey
		for i, k := range c.keys {
			if k >= searchKey {
				c.current = i - 1 // Will be incremented by Next()
				return nil
			}
		}
		// All keys are < searchKey
		c.current = len(c.keys) // Will immediately return EOF
	}

	return nil
}

// Key returns the current key without advancing the cursor
func (c *Cursor) Key() ([]byte, error) {
	if c.closed {
		return nil, fmt.Errorf("cursor is closed")
	}

	if c.current < 0 || c.current >= len(c.keys) {
		return nil, fmt.Errorf("cursor not positioned")
	}

	return []byte(c.keys[c.current]), nil
}

// Value returns the current value without advancing the cursor
func (c *Cursor) Value() ([]byte, error) {
	if c.closed {
		return nil, fmt.Errorf("cursor is closed")
	}

	if c.current < 0 || c.current >= len(c.keys) {
		return nil, fmt.Errorf("cursor not positioned")
	}

	currentKey := c.keys[c.current]

	var data []byte
	var err error
	if c.indexName == "" {
		// Primary key cursor
		data, err = c.skv.Get([]byte(currentKey))
	} else {
		// Index cursor - get by secondary key
		data, err = c.skv.GetByIndex(c.indexName, []byte(currentKey))
	}

	return data, err
}

// Count returns the total number of keys in the cursor range
func (c *Cursor) Count() int {
	return len(c.keys)
}

// Reset resets the cursor to the beginning
func (c *Cursor) Reset() {
	c.current = -1
}

// Close closes the cursor and releases resources
func (c *Cursor) Close() {
	c.closed = true
	c.keys = nil
}

// ForEachCursor iterates over all key-value pairs using the cursor
// Callback returns false to stop iteration
func (c *Cursor) ForEach(fn func(key, value []byte) bool) error {
	c.Reset()
	for {
		key, value, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !fn(key, value) {
			break
		}
	}
	return nil
}

// PrefixCursor creates a cursor that only returns keys with the given prefix
func (s *SKV) PrefixCursor(prefix []byte, reverse bool) *Cursor {
	prefixStr := string(prefix)

	// Calculate the end of the prefix range
	// e.g., "user:" -> "user;" (next character after ':')
	toKey := prefixStr
	if len(toKey) > 0 {
		lastChar := toKey[len(toKey)-1]
		toKey = toKey[:len(toKey)-1] + string(lastChar+1)
	}

	return s.NewCursor(&CursorOptions{
		From:    []byte(prefixStr),
		To:      []byte(toKey),
		Reverse: reverse,
	})
}

// PrefixIndexCursor creates a cursor for an index with the given prefix
func (s *SKV) PrefixIndexCursor(indexName string, prefix []byte, reverse bool) (*Cursor, error) {
	prefixStr := string(prefix)

	// Calculate the end of the prefix range
	toKey := prefixStr
	if len(toKey) > 0 {
		lastChar := toKey[len(toKey)-1]
		toKey = toKey[:len(toKey)-1] + string(lastChar+1)
	}

	return s.NewIndexCursor(indexName, &CursorOptions{
		From:    []byte(prefixStr),
		To:      []byte(toKey),
		Reverse: reverse,
	})
}

// AllCursor creates a cursor that iterates over all records
func (s *SKV) AllCursor(reverse bool) *Cursor {
	return s.NewCursor(&CursorOptions{Reverse: reverse})
}

// AllIndexCursor creates a cursor that iterates over all entries in an index
func (s *SKV) AllIndexCursor(indexName string, reverse bool) (*Cursor, error) {
	return s.NewIndexCursor(indexName, &CursorOptions{Reverse: reverse})
}

// Collect collects all key-value pairs from the cursor into slices
func (c *Cursor) Collect() (keys [][]byte, values [][]byte, err error) {
	keys = make([][]byte, 0)
	values = make([][]byte, 0)

	c.Reset()
	for {
		key, value, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		keys = append(keys, key)
		values = append(values, value)
	}

	return keys, values, nil
}

// Keys returns all keys in the cursor range
func (c *Cursor) Keys() [][]byte {
	result := make([][]byte, len(c.keys))
	for i, k := range c.keys {
		result[i] = []byte(k)
	}
	return result
}

// HasPrefix checks if the current key has the given prefix
func (c *Cursor) HasPrefix(prefix []byte) bool {
	if c.current < 0 || c.current >= len(c.keys) {
		return false
	}
	return strings.HasPrefix(c.keys[c.current], string(prefix))
}

// IsFirst returns true if the cursor is at the first position
func (c *Cursor) IsFirst() bool {
	if c.reverse {
		return c.current == len(c.keys)-1
	}
	return c.current == 0
}

// IsLast returns true if the cursor is at the last position
func (c *Cursor) IsLast() bool {
	if c.reverse {
		return c.current == 0
	}
	return c.current == len(c.keys)-1
}

// IsValid returns true if the cursor is positioned on a valid entry
func (c *Cursor) IsValid() bool {
	return !c.closed && c.current >= 0 && c.current < len(c.keys)
}
