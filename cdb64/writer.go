package cdb64

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
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
	// DuplicateModeZeroFill zero-fills old duplicate records and keeps the new value
	DuplicateModeZeroFill
)

// Writer is used to create CDB64 databases
type Writer struct {
	file          *os.File
	tempFile      string
	finalFile     string
	pos           uint64
	entries       []entry
	duplicates    map[string][]int
	duplicateMode DuplicateMode
	warned        map[string]bool
	closed        bool
}

type entry struct {
	hash uint64
	pos  uint64
	klen uint64
	vlen uint64
	skip bool
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

func (w *Writer) handleDuplicate(key []byte) error {
	if w.duplicateMode == DuplicateModeAllow {
		return nil
	}

	keyStr := string(key)
	indices, exists := w.duplicates[keyStr]
	if !exists {
		return nil
	}

	switch w.duplicateMode {
	case DuplicateModeWarn:
		if !w.warned[keyStr] {
			fmt.Fprintf(os.Stderr, "warning: duplicate key: %s\n", keyStr)
			w.warned[keyStr] = true
		}
	case DuplicateModeError:
		return fmt.Errorf("duplicate key: %s", keyStr)
	case DuplicateModeUnique:
		return errSkipRecord
	case DuplicateModeReplace:
		for _, idx := range indices {
			w.entries[idx].skip = true
		}
	case DuplicateModeZeroFill:
		for _, idx := range indices {
			if err := w.zeroFillEntry(&w.entries[idx]); err != nil {
				return err
			}
			w.entries[idx].skip = true
		}
	}
	return nil
}

var errSkipRecord = fmt.Errorf("skip record")

func (w *Writer) zeroFillEntry(e *entry) error {
	if e.vlen == 0 {
		return nil
	}
	buf, err := allocBytes(e.vlen)
	if err != nil {
		return err
	}
	off := e.pos + recordHeaderSize + e.klen
	if off < e.pos || off > uint64(math.MaxInt64) {
		return ErrInvalidFormat
	}
	if _, err := w.file.WriteAt(buf, int64(off)); err != nil {
		return fmt.Errorf("failed to zero-fill record: %w", err)
	}
	return nil
}

func (w *Writer) checkRecordSize(klen, vlen int) (uint64, error) {
	recordSize := uint64(recordHeaderSize) + uint64(klen) + uint64(vlen)
	if recordSize < uint64(recordHeaderSize) {
		return 0, ErrTooLarge
	}
	if w.pos > math.MaxUint64-recordSize {
		return 0, ErrTooLarge
	}
	newPos := w.pos + recordSize
	if newPos > uint64(math.MaxInt64) {
		return 0, ErrTooLarge
	}
	return recordSize, nil
}

// Put adds a key-value pair to the database
func (w *Writer) Put(key, value []byte) error {
	if err := w.handleDuplicate(key); err != nil {
		if errors.Is(err, errSkipRecord) {
			return nil
		}
		return err
	}

	recordSize, err := w.checkRecordSize(len(key), len(value))
	if err != nil {
		return err
	}

	e := entry{
		hash: hash(key),
		pos:  w.pos,
		klen: uint64(len(key)),
		vlen: uint64(len(value)),
	}

	if w.duplicateMode != DuplicateModeAllow {
		keyStr := string(key)
		w.duplicates[keyStr] = append(w.duplicates[keyStr], len(w.entries))
	}
	w.entries = append(w.entries, e)

	record := make([]byte, recordSize)
	binary.LittleEndian.PutUint64(record[0:8], uint64(len(key)))
	binary.LittleEndian.PutUint64(record[8:16], uint64(len(value)))
	copy(record[recordHeaderSize:], key)
	copy(record[recordHeaderSize+len(key):], value)

	if _, err := w.file.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	w.pos += recordSize
	return nil
}

// Finalize completes the database creation by writing hash tables and atomically renaming.
// The atomic rename ensures that readers with the old file open continue to work,
// while new opens immediately see the new version. This allows zero-downtime updates.
func (w *Writer) Finalize() error {
	if err := w.finalize(); err != nil {
		_ = w.Abort()
		return err
	}
	return nil
}

func (w *Writer) finalize() error {
	tables := make([][]entry, NumTables)
	for _, e := range w.entries {
		if e.skip {
			continue
		}
		tableIdx := e.hash % NumTables
		tables[tableIdx] = append(tables[tableIdx], e)
	}

	header := make([]byte, HeaderSize)
	for i := 0; i < NumTables; i++ {
		ents := tables[i]
		nslots := uint64(len(ents) * 2)
		if nslots == 0 {
			binary.LittleEndian.PutUint64(header[i*slotSize:i*slotSize+8], 0)
			binary.LittleEndian.PutUint64(header[i*slotSize+8:i*slotSize+16], 0)
			continue
		}

		if nslots > math.MaxUint64/slotSize {
			return ErrTooLarge
		}
		tableBytes := nslots * slotSize
		if w.pos > math.MaxUint64-tableBytes || w.pos+tableBytes > uint64(math.MaxInt64) {
			return ErrTooLarge
		}

		slots := make([]byte, tableBytes)
		for _, e := range ents {
			slot := (e.hash / NumTables) % nslots
			for {
				offset := slot * slotSize
				if binary.LittleEndian.Uint64(slots[offset+8:offset+16]) == 0 {
					binary.LittleEndian.PutUint64(slots[offset:offset+8], e.hash)
					binary.LittleEndian.PutUint64(slots[offset+8:offset+16], e.pos)
					break
				}
				slot = (slot + 1) % nslots
			}
		}

		tablePos := w.pos
		if _, err := w.file.Write(slots); err != nil {
			return fmt.Errorf("failed to write hash table: %w", err)
		}

		binary.LittleEndian.PutUint64(header[i*slotSize:i*slotSize+8], tablePos)
		binary.LittleEndian.PutUint64(header[i*slotSize+8:i*slotSize+16], nslots)
		w.pos += tableBytes
	}

	if _, err := w.file.WriteAt(header, 0); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := w.file.Close(); err != nil {
		w.file = nil
		w.closed = true
		return fmt.Errorf("failed to close file: %w", err)
	}
	w.file = nil
	w.closed = true

	if w.tempFile != "" && w.tempFile != w.finalFile {
		if err := replaceFile(w.tempFile, w.finalFile); err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}
	w.tempFile = ""
	return nil
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if runtime.GOOS != "windows" {
		return err
	}

	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}

// Abort cancels the database creation and removes the temp file
func (w *Writer) Abort() error {
	var firstErr error

	if w.file != nil && !w.closed {
		if err := w.file.Close(); err != nil {
			firstErr = fmt.Errorf("failed to close file: %w", err)
		}
		w.file = nil
		w.closed = true
	}

	if w.tempFile != "" {
		if err := os.Remove(w.tempFile); err != nil && !os.IsNotExist(err) {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to remove temp file: %w", err)
			} else {
				slog.Warn("failed to remove temp file after close error", "error", err)
			}
		}
		w.tempFile = ""
	}

	return firstErr
}

// SetPermissions sets the file permissions (must be called after Finalize)
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
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if buf[0] == '\n' {
			return nil
		}

		if buf[0] != '+' {
			return fmt.Errorf("invalid format: expected '+'")
		}

		var klen, vlen uint64
		if _, err := fmt.Fscanf(r, "%d", &klen); err != nil {
			return fmt.Errorf("invalid format: failed to read klen: %w", err)
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != ',' {
			return fmt.Errorf("invalid format: expected ','")
		}

		if _, err := fmt.Fscanf(r, "%d", &vlen); err != nil {
			return fmt.Errorf("invalid format: failed to read vlen: %w", err)
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != ':' {
			return fmt.Errorf("invalid format: expected ':'")
		}

		if klen > uint64(math.MaxInt) || vlen > uint64(math.MaxInt) {
			return fmt.Errorf("invalid format: length too large")
		}

		key := make([]byte, klen)
		if _, err := io.ReadFull(r, key); err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}

		arrow := make([]byte, 2)
		if _, err := io.ReadFull(r, arrow); err != nil {
			return err
		}
		if arrow[0] != '-' || arrow[1] != '>' {
			return fmt.Errorf("invalid format: expected '->'")
		}

		value := make([]byte, vlen)
		if _, err := io.ReadFull(r, value); err != nil {
			return fmt.Errorf("failed to read value: %w", err)
		}

		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if buf[0] != '\n' {
			return fmt.Errorf("invalid format: expected newline")
		}

		if err := w.Put(key, value); err != nil {
			return err
		}
	}
}

func (w *Writer) writeMapFormat(r io.Reader) error {
	line := make([]byte, 0, 4096)
	buf := make([]byte, 1)

	for {
		line = line[:0]

		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				if err == io.EOF {
					if len(line) > 0 {
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
	if len(line) == 0 {
		return nil
	}

	start := 0
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}

	if start >= len(line) || line[start] == '#' {
		return nil
	}

	keyStart := start
	keyEnd := keyStart
	for keyEnd < len(line) && line[keyEnd] != ' ' && line[keyEnd] != '\t' {
		keyEnd++
	}

	if keyEnd >= len(line) {
		return w.Put(line[keyStart:keyEnd], []byte{})
	}

	key := line[keyStart:keyEnd]

	valueStart := keyEnd
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(line) {
		return w.Put(key, []byte{})
	}

	return w.Put(key, line[valueStart:])
}
