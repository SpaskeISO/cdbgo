package cdb64

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
)

const (
	// HeaderSize is the size of the CDB64 header (256 hash table pointers, 16 bytes each)
	HeaderSize = 4096
	// NumTables is the number of hash tables in a CDB64 file
	NumTables = 256

	recordHeaderSize = 16
	slotSize         = 16
)

var (
	// ErrNotFound is returned when a key is not found in the database
	ErrNotFound = errors.New("key not found")
	// ErrInvalidFormat is returned when the CDB64 file is malformed
	ErrInvalidFormat = errors.New("invalid CDB64 format")
	// ErrTooLarge is returned when a write would exceed the maximum seekable size
	ErrTooLarge = errors.New("CDB64 file would exceed the maximum seekable size")
)

// hashTable represents a single hash table in the CDB64
type hashTable struct {
	pos    uint64
	nslots uint64
}

// CDB represents an open CDB64 database
type CDB struct {
	file    *os.File
	tables  [NumTables]hashTable
	size    int64
	dataEnd int64 // first hash-table byte; records occupy [HeaderSize, dataEnd)
}

// Open opens a CDB64 database file for reading
func Open(filename string) (*CDB, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close file after stat error", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() < HeaderSize {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close file", "error", closeErr)
		}
		return nil, ErrInvalidFormat
	}

	cdb := &CDB{
		file: file,
		size: info.Size(),
	}

	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close file after read error", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	for i := 0; i < NumTables; i++ {
		offset := i * slotSize
		cdb.tables[i].pos = binary.LittleEndian.Uint64(header[offset : offset+8])
		cdb.tables[i].nslots = binary.LittleEndian.Uint64(header[offset+8 : offset+16])
	}
	cdb.dataEnd = cdb.computeDataEnd()

	return cdb, nil
}

func (c *CDB) computeDataEnd() int64 {
	end := c.size
	for i := range c.tables {
		t := &c.tables[i]
		if t.nslots == 0 || t.pos == 0 {
			continue
		}
		if t.pos > uint64(math.MaxInt64) {
			continue
		}
		if pos := int64(t.pos); pos < end {
			end = pos
		}
	}
	return end
}

// Close closes the CDB64 database
func (c *CDB) Close() error {
	if c.file != nil {
		return c.file.Close()
	}
	return nil
}

// GetFile returns the underlying file (for internal use)
func (c *CDB) GetFile() *os.File {
	return c.file
}

// Size returns the size of the database file in bytes
func (c *CDB) Size() int64 {
	return c.size
}

func hash(key []byte) uint64 {
	h := uint64(5381)
	for _, b := range key {
		h = ((h << 5) + h) ^ uint64(b)
	}
	return h
}

func allocBytes(n uint64) ([]byte, error) {
	if n > uint64(math.MaxInt) {
		return nil, ErrInvalidFormat
	}
	return make([]byte, n), nil
}

func (c *CDB) readAt(buf []byte, off uint64) error {
	if len(buf) == 0 {
		return nil
	}
	if off > uint64(math.MaxInt64) {
		return ErrInvalidFormat
	}
	end := off + uint64(len(buf))
	if end < off || end > uint64(c.size) {
		return ErrInvalidFormat
	}
	if _, err := c.file.ReadAt(buf, int64(off)); err != nil {
		return err
	}
	return nil
}

func (c *CDB) readSlot(tablePos uint64, slot uint64) (slotHash, recPos uint64, err error) {
	if slot > (math.MaxUint64-tablePos)/slotSize {
		return 0, 0, ErrInvalidFormat
	}
	off := tablePos + slot*slotSize
	var buf [slotSize]byte
	if err := c.readAt(buf[:], off); err != nil {
		return 0, 0, fmt.Errorf("failed to read slot: %w", err)
	}
	return binary.LittleEndian.Uint64(buf[0:8]), binary.LittleEndian.Uint64(buf[8:16]), nil
}

func (c *CDB) matchRecord(pos uint64, key []byte) ([]byte, bool, error) {
	if pos < HeaderSize {
		return nil, false, ErrInvalidFormat
	}
	var hdr [recordHeaderSize]byte
	if err := c.readAt(hdr[:], pos); err != nil {
		return nil, false, fmt.Errorf("failed to read record: %w", err)
	}

	klen := binary.LittleEndian.Uint64(hdr[0:8])
	vlen := binary.LittleEndian.Uint64(hdr[8:16])
	if klen != uint64(len(key)) {
		return nil, false, nil
	}

	payloadLen := klen + vlen
	if payloadLen < klen {
		return nil, false, ErrInvalidFormat
	}
	recEnd := pos + recordHeaderSize + payloadLen
	if recEnd < pos || recEnd > uint64(c.size) {
		return nil, false, ErrInvalidFormat
	}

	buf, err := allocBytes(payloadLen)
	if err != nil {
		return nil, false, err
	}
	if err := c.readAt(buf, pos+recordHeaderSize); err != nil {
		return nil, false, fmt.Errorf("failed to read record payload: %w", err)
	}
	if !bytes.Equal(buf[:klen], key) {
		return nil, false, nil
	}
	return buf[klen:], true, nil
}

// Get returns the first value associated with the given key
func (c *CDB) Get(key []byte) ([]byte, error) {
	return c.GetN(key, 1)
}

// GetN returns the nth value associated with the given key (1-based)
func (c *CDB) GetN(key []byte, n int) ([]byte, error) {
	if n < 1 {
		return nil, fmt.Errorf("record number must be >= 1")
	}

	h := hash(key)
	table := &c.tables[h%NumTables]
	if table.nslots == 0 {
		return nil, ErrNotFound
	}

	slot := (h / NumTables) % table.nslots
	count := 0

	for i := uint64(0); i < table.nslots; i++ {
		slotHash, recPos, err := c.readSlot(table.pos, slot)
		if err != nil {
			return nil, err
		}
		if recPos == 0 {
			return nil, ErrNotFound
		}
		if slotHash == h {
			recVal, ok, err := c.matchRecord(recPos, key)
			if err != nil {
				return nil, err
			}
			if ok {
				count++
				if count == n {
					return recVal, nil
				}
			}
		}
		slot = (slot + 1) % table.nslots
	}

	return nil, ErrNotFound
}

// GetAll returns all values associated with the given key
func (c *CDB) GetAll(key []byte) ([][]byte, error) {
	var results [][]byte

	h := hash(key)
	table := &c.tables[h%NumTables]
	if table.nslots == 0 {
		return results, nil
	}

	slot := (h / NumTables) % table.nslots

	for i := uint64(0); i < table.nslots; i++ {
		slotHash, recPos, err := c.readSlot(table.pos, slot)
		if err != nil {
			return nil, err
		}
		if recPos == 0 {
			break
		}
		if slotHash == h {
			recVal, ok, err := c.matchRecord(recPos, key)
			if err != nil {
				return nil, err
			}
			if ok {
				results = append(results, recVal)
			}
		}
		slot = (slot + 1) % table.nslots
	}

	return results, nil
}
