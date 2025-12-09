package skv

import (
	"testing"
)

func TestForEachAbsolute(t *testing.T) {
	dbFile := "tools/eskv/data/absolute.skv"

	db, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	err = db.ForEach(func(key []byte, value []byte) error {
		t.Logf("Key: %s, Value: %s", string(key), string(value))
		return nil
	})

	if err != nil {
		t.Fatalf("Error in ForEach: %v", err)
	}
}
