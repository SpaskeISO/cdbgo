package cdb

import (
	"os"
	"path/filepath"
	"strings"
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
	if abortErr := writer.Abort(); abortErr != nil {
		t.Errorf("Failed to abort: %v", abortErr)
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

func TestDuplicateModeReplace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	writer.SetDuplicateMode(DuplicateModeReplace)

	if err := writer.PutString("key", "old"); err != nil {
		t.Fatalf("Failed to put first value: %v", err)
	}
	if err := writer.PutString("key", "new"); err != nil {
		t.Fatalf("Failed to put replacement: %v", err)
	}
	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer db.Close()

	value, err := db.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if string(value) != "new" {
		t.Errorf("Expected 'new', got %q", value)
	}

	values, err := db.GetAll([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get all: %v", err)
	}
	if len(values) != 1 || string(values[0]) != "new" {
		t.Errorf("GetAll should return only the replacement, got %v", values)
	}
}

func TestDuplicateModeZeroFill(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	writer.SetDuplicateMode(DuplicateModeZeroFill)

	if err := writer.PutString("key", "secret"); err != nil {
		t.Fatalf("Failed to put first value: %v", err)
	}
	oldPos := writer.entries[0].pos
	oldVlen := writer.entries[0].vlen

	if err := writer.PutString("key", "public"); err != nil {
		t.Fatalf("Failed to put second value: %v", err)
	}
	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer db.Close()

	value, err := db.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if string(value) != "public" {
		t.Errorf("Expected 'public', got %q", value)
	}

	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	start := int(oldPos) + recordHeaderSize + len("key")
	end := start + int(oldVlen)
	for i := start; i < end; i++ {
		if raw[i] != 0 {
			t.Fatalf("expected zero-filled old value, got byte %d at %d", raw[i], i)
		}
	}
}

func TestPutOverflow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Abort()

	writer.pos = ^uint32(0) - 4
	err = writer.Put([]byte("k"), []byte("v"))
	if err != ErrTooLarge {
		t.Errorf("Expected ErrTooLarge, got %v", err)
	}
}

func TestWriteFromNativeFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	input := "+3,5:foo->hello\n+3,5:bar->world\n\n"
	if err := writer.WriteFrom(strings.NewReader(input), false); err != nil {
		t.Fatalf("WriteFrom failed: %v", err)
	}
	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer db.Close()

	value, err := db.Get([]byte("foo"))
	if err != nil {
		t.Fatalf("Failed to get foo: %v", err)
	}
	if string(value) != "hello" {
		t.Errorf("Expected hello, got %q", value)
	}
}

func TestWriteFromMapFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.cdb")

	writer, err := Create(dbPath, "")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	input := "# comment\nfoo hello world\nbar\n"
	if err := writer.WriteFrom(strings.NewReader(input), true); err != nil {
		t.Fatalf("WriteFrom failed: %v", err)
	}
	if err := writer.Finalize(); err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	defer db.Close()

	value, err := db.Get([]byte("foo"))
	if err != nil {
		t.Fatalf("Failed to get foo: %v", err)
	}
	if string(value) != "hello world" {
		t.Errorf("Expected 'hello world', got %q", value)
	}
}
