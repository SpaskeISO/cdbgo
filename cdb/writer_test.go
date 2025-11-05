package cdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBasicPut(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.PutString("key", "value"); err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify the database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	value, err := db.Get([]byte("key"))
	if err != nil {
		t.Errorf("Failed to get: %v", err)
	}
	if string(value) != "value" {
		t.Errorf("Expected 'value', got %s", string(value))
	}
}

func TestDuplicateModeError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	writer.SetDuplicateMode(DuplicateModeError)

	if err := writer.PutString("key", "value1"); err != nil {
		t.Fatalf("Failed to put first value: %v", err)
	}

	err = writer.PutString("key", "value2")
	if err == nil {
		t.Error("Expected error on duplicate key")
	}
}

func TestDuplicateModeUnique(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	writer.SetDuplicateMode(DuplicateModeUnique)

	if err := writer.PutString("key", "value1"); err != nil {
		t.Fatalf("Failed to put first value: %v", err)
	}

	if err := writer.PutString("key", "value2"); err != nil {
		t.Fatalf("Failed to put second value: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify only first value is stored
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	values, err := db.GetAll([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get all: %v", err)
	}

	if len(values) != 1 {
		t.Errorf("Expected 1 value, got %d", len(values))
	}

	if string(values[0]) != "value1" {
		t.Errorf("Expected 'value1', got %s", string(values[0]))
	}
}

func TestDuplicateModeAllow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	writer.SetDuplicateMode(DuplicateModeAllow)

	if err := writer.PutString("key", "value1"); err != nil {
		t.Fatalf("Failed to put first value: %v", err)
	}

	if err := writer.PutString("key", "value2"); err != nil {
		t.Fatalf("Failed to put second value: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify both values are stored
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	values, err := db.GetAll([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get all: %v", err)
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(values))
	}
}

func TestEmptyDatabaseCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify the database exists and can be opened
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	// Verify empty database returns not found
	_, err = db.Get([]byte("anykey"))
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestInPlaceCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	// Create in-place (no temp file)
	writer, err := Create(dbPath, "-")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.PutString("key", "value"); err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify the database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
	}()

	value, err := db.Get([]byte("key"))
	if err != nil {
		t.Errorf("Failed to get: %v", err)
	}
	if string(value) != "value" {
		t.Errorf("Expected 'value', got %s", string(value))
	}
}

func TestAbort(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")
	tmpPath := dbPath + ".tmp"

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.PutString("key", "value"); err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	if err := writer.Abort(); err != nil {
		t.Fatalf("Failed to abort: %v", err)
	}

	// Verify temp file is removed
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should be removed after abort")
	}

	// Verify final file doesn't exist
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("Final file should not exist after abort")
	}
}

func TestSetPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := writer.PutString("key", "value"); err != nil {
		t.Fatalf("Failed to put: %v", err)
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Set permissions
	if err := SetPermissions(dbPath, 0600); err != nil {
		t.Fatalf("Failed to set permissions: %v", err)
	}

	// Verify permissions
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}
}
