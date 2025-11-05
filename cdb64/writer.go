package cdb64

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// DuplicateMode defines how to handle duplicate keys
type DuplicateMode int

const (
	// DuplicateModeAllow allows duplicate keys (default)
	DuplicateModeAllow DuplicateMode = iota
	// DuplicateModeWarn allows duplicates but warns
	DuplicateModeWarn
	// DuplicateModeError rejects duplicates with an error
	DuplicateModeError
	// DuplicateModeReplace replaces existing key with new value
	DuplicateModeReplace
	// DuplicateModeUnique ignores duplicate keys (keeps first)
	DuplicateModeUnique
	// DuplicateModeZeroFill zero-fills duplicate records
	DuplicateModeZeroFill
)

// Writer is used to create CDB64 databases
type Writer struct {
	file          *os.File
	tempFile      string
	finalFile     string
	pos           uint64
	entries       []entry
	duplicates    map[string][]int // map key to entry indices
	duplicateMode DuplicateMode
	warned        map[string]bool
}

type entry struct {
	hash  uint64
	key   []byte
	value []byte
	pos   uint64
}

// Create creates a new CDB64 writer
// If tempFile is empty, finalFile + ".tmp" will be used
// If tempFile is "-", the file will be created in-place (no temp file)
func Create(finalFile, tempFile string) (*Writer, error) {
	if tempFile == "" {
		tempFile = finalFile + ".tmp"
	}

	var file *os.File
	var err error

	if tempFile == "-" {
		// Create in-place
		file, err = os.Create(finalFile)
		tempFile = ""
	} else {
		file, err = os.Create(tempFile)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	w := &Writer{
		file:          file,
		tempFile:      tempFile,
		finalFile:     finalFile,
		pos:           HeaderSize,
		entries:       make([]entry, 0),
		duplicates:    make(map[string][]int),
		duplicateMode: DuplicateModeAllow,
		warned:        make(map[string]bool),
	}

	// Write placeholder header
	header := make([]byte, HeaderSize)
	if _, err := file.Write(header); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close file after write error", "error", closeErr)
		}
		if tempFile != "" {
			if removeErr := os.Remove(tempFile); removeErr != nil {
				slog.Warn("failed to remove temp file", "file", tempFile, "error", removeErr)
			}
		}
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	return w, nil
}

// SetDuplicateMode sets how duplicate keys should be handled
func (w *Writer) SetDuplicateMode(mode DuplicateMode) {
	w.duplicateMode = mode
}

// Put adds a key-value pair to the database
func (w *Writer) Put(key, value []byte) error {
	keyStr := string(key)
	h := hash(key)

	// Check for duplicates
	if indices, exists := w.duplicates[keyStr]; exists {
		switch w.duplicateMode {
		case DuplicateModeWarn:
			if !w.warned[keyStr] {
				fmt.Fprintf(os.Stderr, "warning: duplicate key: %s\n", keyStr)
				w.warned[keyStr] = true
			}
		case DuplicateModeError:
			return fmt.Errorf("duplicate key: %s", keyStr)
		case DuplicateModeUnique:
			return nil // Ignore duplicate
		case DuplicateModeReplace:
			// Mark old entries for zero-fill
			for _, idx := range indices {
				w.entries[idx].value = make([]byte, len(w.entries[idx].value))
			}
		case DuplicateModeZeroFill:
			// Mark old entries for zero-fill
			for _, idx := range indices {
				w.entries[idx].value = make([]byte, len(w.entries[idx].value))
			}
		}
	}

	// Create entry
	e := entry{
		hash:  h,
		key:   make([]byte, len(key)),
		value: make([]byte, len(value)),
		pos:   w.pos,
	}
	copy(e.key, key)
	copy(e.value, value)

	// Track duplicate
	w.duplicates[keyStr] = append(w.duplicates[keyStr], len(w.entries))
	w.entries = append(w.entries, e)

	// Write record: klen(8) + vlen(8) + key + value
	recordSize := 16 + len(key) + len(value)
	record := make([]byte, recordSize)

	binary.LittleEndian.PutUint64(record[0:8], uint64(len(key)))
	binary.LittleEndian.PutUint64(record[8:16], uint64(len(value)))
	copy(record[16:], key)
	copy(record[16+len(key):], value)

	if _, err := w.file.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	w.pos += uint64(recordSize)
	return nil
}

// Finalize completes the database creation by writing hash tables and atomically renaming.
// The atomic rename ensures that readers with the old file open continue to work,
// while new opens immediately see the new version. This allows zero-downtime updates.
func (w *Writer) Finalize() error {
	// Group entries by hash table
	tables := make([][]entry, NumTables)
	for i := range tables {
		tables[i] = make([]entry, 0)
	}

	for _, e := range w.entries {
		tableIdx := e.hash % NumTables
		tables[tableIdx] = append(tables[tableIdx], e)
	}

	// Write hash tables and build header
	header := make([]byte, HeaderSize)
	for i := 0; i < NumTables; i++ {
		entries := tables[i]
		nslots := uint64(len(entries) * 2)
		if nslots == 0 {
			// Empty table
			binary.LittleEndian.PutUint64(header[i*16:i*16+8], 0)
			binary.LittleEndian.PutUint64(header[i*16+8:i*16+16], 0)
			continue
		}

		// Create hash table
		slots := make([]byte, nslots*16)

		for _, e := range entries {
			// Find empty slot using linear probing
			slot := (e.hash / NumTables) % nslots
			for {
				offset := slot * 16
				// Check if slot is empty (pos == 0)
				if binary.LittleEndian.Uint64(slots[offset+8:offset+16]) == 0 {
					// Write hash and position
					binary.LittleEndian.PutUint64(slots[offset:offset+8], e.hash)
					binary.LittleEndian.PutUint64(slots[offset+8:offset+16], e.pos)
					break
				}
				slot = (slot + 1) % nslots
			}
		}

		// Write hash table to file
		tablePos := w.pos
		if _, err := w.file.Write(slots); err != nil {
			return fmt.Errorf("failed to write hash table: %w", err)
		}

		// Update header
		binary.LittleEndian.PutUint64(header[i*16:i*16+8], tablePos)
		binary.LittleEndian.PutUint64(header[i*16+8:i*16+16], nslots)

		w.pos += uint64(len(slots))
	}

	// Write header at the beginning of the file
	if _, err := w.file.WriteAt(header, 0); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Sync and close
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Rename temp file to final file
	if w.tempFile != "" && w.tempFile != w.finalFile {
		if err := os.Rename(w.tempFile, w.finalFile); err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}

	return nil
}

// Abort cancels the database creation and removes the temp file
func (w *Writer) Abort() error {
	var firstErr error

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			firstErr = fmt.Errorf("failed to close file: %w", err)
		}
	}

	if w.tempFile != "" {
		if err := os.Remove(w.tempFile); err != nil {
			if firstErr == nil {
				return fmt.Errorf("failed to remove temp file: %w", err)
			}
			slog.Warn("failed to remove temp file after close error", "error", err)
		}
	}

	return firstErr
}

// SetPermissions sets the file permissions (must be called before Finalize)
func SetPermissions(filename string, perms os.FileMode) error {
	return os.Chmod(filename, perms)
}

// PutString is a convenience method for adding string key-value pairs
func (w *Writer) PutString(key, value string) error {
	return w.Put([]byte(key), []byte(value))
}

// Copy copies all records from a reader to this writer
func (w *Writer) Copy(r *CDB) error {
	it := NewIterator(r)
	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}
		if err := w.Put(key, value); err != nil {
			return err
		}
	}
	return it.Err()
}

// WriteFrom reads records from a reader and writes them to the CDB64
func (w *Writer) WriteFrom(r io.Reader, mapFormat bool) error {
	if mapFormat {
		return w.writeMapFormat(r)
	}
	return w.writeNativeFormat(r)
}

func (w *Writer) writeNativeFormat(r io.Reader) error {
	// Native format: +klen,vlen:key->val\n
	buf := make([]byte, 1)
	for {
		// Read '+'
		if _, err := io.ReadFull(r, buf); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if buf[0] == '\n' {
			// Empty line, end of data
			return nil
		}

		if buf[0] != '+' {
			return fmt.Errorf("invalid format: expected '+'")
		}

		// Read klen
		var klen uint64
		if _, err := fmt.Fscanf(r, "%d", &klen); err != nil {
			return fmt.Errorf("invalid format: failed to read klen: %w", err)
		}

		// Read ','
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != ',' {
			return fmt.Errorf("invalid format: expected ','")
		}

		// Read vlen
		var vlen uint64
		if _, err := fmt.Fscanf(r, "%d", &vlen); err != nil {
			return fmt.Errorf("invalid format: failed to read vlen: %w", err)
		}

		// Read ':'
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != ':' {
			return fmt.Errorf("invalid format: expected ':'")
		}

		// Read key
		key := make([]byte, klen)
		if _, err := io.ReadFull(r, key); err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}

		// Read '->'
		arrow := make([]byte, 2)
		if _, err := io.ReadFull(r, arrow); err != nil {
			return err
		}
		if arrow[0] != '-' || arrow[1] != '>' {
			return fmt.Errorf("invalid format: expected '->'")
		}

		// Read value
		value := make([]byte, vlen)
		if _, err := io.ReadFull(r, value); err != nil {
			return fmt.Errorf("failed to read value: %w", err)
		}

		// Read newline
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != '\n' {
			return fmt.Errorf("invalid format: expected newline")
		}

		// Add to database
		if err := w.Put(key, value); err != nil {
			return err
		}
	}
}

func (w *Writer) writeMapFormat(r io.Reader) error {
	// Map format: key<whitespace>value\n
	// Lines starting with # and empty lines are ignored
	line := make([]byte, 0, 4096)
	buf := make([]byte, 1)

	for {
		line = line[:0]

		// Read line
		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				if err == io.EOF {
					if len(line) > 0 {
						// Process last line
						if err := w.processMapLine(line); err != nil {
							return err
						}
					}
					return nil
				}
				return err
			}

			if buf[0] == '\n' {
				break
			}

			line = append(line, buf[0])
		}

		if err := w.processMapLine(line); err != nil {
			return err
		}
	}
}

func (w *Writer) processMapLine(line []byte) error {
	// Skip empty lines and comments
	if len(line) == 0 {
		return nil
	}

	// Trim leading whitespace
	start := 0
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}

	if start >= len(line) || line[start] == '#' {
		return nil
	}

	// Find key (non-whitespace)
	keyStart := start
	keyEnd := keyStart
	for keyEnd < len(line) && line[keyEnd] != ' ' && line[keyEnd] != '\t' {
		keyEnd++
	}

	if keyEnd >= len(line) {
		// No value, just key
		return w.Put(line[keyStart:keyEnd], []byte{})
	}

	key := line[keyStart:keyEnd]

	// Skip whitespace to find value
	valueStart := keyEnd
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(line) {
		// No value after whitespace
		return w.Put(key, []byte{})
	}

	value := line[valueStart:]
	return w.Put(key, value)
}
