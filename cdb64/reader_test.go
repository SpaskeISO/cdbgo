package cdb64

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestDB(t *testing.T, records map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb64")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	for k, v := range records {
		if err := writer.PutString(k, v); err != nil {
			t.Fatalf("Failed to put record: %v", err)
		}
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	return dbPath
}

func TestBasicGet(t *testing.T) {
	dbPath := createTestDB(t, map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	// Test existing keys
	value, err := db.Get([]byte("key1"))
	if err != nil {
		t.Errorf("Failed to get key1: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", string(value))
	}

	value, err = db.Get([]byte("key2"))
	if err != nil {
		t.Errorf("Failed to get key2: %v", err)
	}
	if string(value) != "value2" {
		t.Errorf("Expected value2, got %s", string(value))
	}

	// Test non-existent key
	_, err = db.Get([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestGetAll(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb64")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Add multiple values for same key
	if err := writer.PutString("key", "value1"); err != nil {
		t.Fatalf("Failed to put value1: %v", err)
	}
	if err := writer.PutString("key", "value2"); err != nil {
		t.Fatalf("Failed to put value2: %v", err)
	}
	if err := writer.PutString("key", "value3"); err != nil {
		t.Fatalf("Failed to put value3: %v", err)
	}
	if err := writer.PutString("other", "othervalue"); err != nil {
		t.Fatalf("Failed to put othervalue: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	values, err := db.GetAll([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get all values: %v", err)
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	// Check that all values are present
	found := make(map[string]bool)
	for _, v := range values {
		found[string(v)] = true
	}

	if !found["value1"] || !found["value2"] || !found["value3"] {
		t.Errorf("Not all expected values found: %v", found)
	}
}

func TestGetN(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb64")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Add multiple values for same key
	if err := writer.PutString("key", "first"); err != nil {
		t.Fatalf("Failed to put first: %v", err)
	}
	if err := writer.PutString("key", "second"); err != nil {
		t.Fatalf("Failed to put second: %v", err)
	}
	if err := writer.PutString("key", "third"); err != nil {
		t.Fatalf("Failed to put third: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	// Get specific records
	value, err := db.GetN([]byte("key"), 1)
	if err != nil {
		t.Errorf("Failed to get record 1: %v", err)
	}
	if string(value) != "first" {
		t.Errorf("Expected 'first', got %s", string(value))
	}

	value, err = db.GetN([]byte("key"), 2)
	if err != nil {
		t.Errorf("Failed to get record 2: %v", err)
	}
	if string(value) != "second" {
		t.Errorf("Expected 'second', got %s", string(value))
	}

	// Get non-existent record number
	_, err = db.GetN([]byte("key"), 10)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for record 10, got %v", err)
	}
}

func TestEmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb64")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	_, err = db.Get([]byte("anykey"))
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestIterator(t *testing.T) {
	dbPath := createTestDB(t, map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	it := NewIterator(db)
	count := 0
	records := make(map[string]string)

	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}
		count++
		records[string(key)] = string(value)
	}

	if err := it.Err(); err != nil {
		t.Errorf("Iterator error: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 records, got %d", count)
	}

	if records["key1"] != "value1" || records["key2"] != "value2" || records["key3"] != "value3" {
		t.Errorf("Unexpected records: %v", records)
	}
}

func TestInvalidFile(t *testing.T) {
	_, err := Open("/nonexistent/path/to/database.cdb64")
	if err == nil {
		t.Error("Expected error opening nonexistent file")
	}
}

func TestOpenInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.cdb64")

	// Create a file that's too small
	if err := os.WriteFile(invalidPath, []byte("not a valid cdb64"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	_, err := Open(invalidPath)
	if err == nil {
		t.Error("Expected error opening invalid file")
	}
}

func TestLargeKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb64")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Test with large key/value that would exceed 32-bit limits
	largeKey := make([]byte, 10000)
	largeValue := make([]byte, 100000)
	for i := range largeKey {
		largeKey[i] = byte(i % 256)
	}
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	if err := writer.Put(largeKey, largeValue); err != nil {
		t.Fatalf("Failed to put large record: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	value, err := db.Get(largeKey)
	if err != nil {
		t.Errorf("Failed to get large key: %v", err)
	}
	if len(value) != len(largeValue) {
		t.Errorf("Expected value length %d, got %d", len(largeValue), len(value))
	}
}
