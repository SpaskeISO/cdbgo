package cdb

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
	// HeaderSize is the size of the CDB header (256 hash table pointers)
	HeaderSize = 2048
	// NumTables is the number of hash tables in a CDB file
	NumTables = 256

	recordHeaderSize = 8
	slotSize         = 8
)

var (
	// ErrNotFound is returned when a key is not found in the database
	ErrNotFound = errors.New("key not found")
	// ErrInvalidFormat is returned when the CDB file is malformed
	ErrInvalidFormat = errors.New("invalid CDB format")
	// ErrTooLarge is returned when a write would exceed the 4GB CDB size limit
	ErrTooLarge = errors.New("CDB file would exceed the 4GB size limit")
)

// hashTable represents a single hash table in the CDB
type hashTable struct {
	pos    uint32 // position in file
	nslots uint32 // number of slots
}

// CDB represents an open CDB database
type CDB struct {
	file   *os.File
	tables [NumTables]hashTable
	size   int64
}

// Open opens a CDB database file for reading
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
		cdb.tables[i].pos = binary.LittleEndian.Uint32(header[offset : offset+4])
		cdb.tables[i].nslots = binary.LittleEndian.Uint32(header[offset+4 : offset+8])
	}

	return cdb, nil
}

// Close closes the CDB database
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

func hash(key []byte) uint32 {
	h := uint32(5381)
	for _, b := range key {
		h = ((h << 5) + h) ^ uint32(b)
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

func (c *CDB) readSlot(tablePos uint32, slot uint32) (slotHash, recPos uint32, err error) {
	off := uint64(tablePos) + uint64(slot)*slotSize
	var buf [slotSize]byte
	if err := c.readAt(buf[:], off); err != nil {
		return 0, 0, fmt.Errorf("failed to read slot: %w", err)
	}
	return binary.LittleEndian.Uint32(buf[0:4]), binary.LittleEndian.Uint32(buf[4:8]), nil
}

func (c *CDB) readRecord(pos uint32) (key, value []byte, err error) {
	if pos < HeaderSize {
		return nil, nil, ErrInvalidFormat
	}
	var hdr [recordHeaderSize]byte
	if err := c.readAt(hdr[:], uint64(pos)); err != nil {
		return nil, nil, fmt.Errorf("failed to read record: %w", err)
	}

	klen := binary.LittleEndian.Uint32(hdr[0:4])
	vlen := binary.LittleEndian.Uint32(hdr[4:8])
	recEnd := uint64(pos) + recordHeaderSize + uint64(klen) + uint64(vlen)
	if recEnd < uint64(pos) || recEnd > uint64(c.size) {
		return nil, nil, ErrInvalidFormat
	}

	key, err = allocBytes(uint64(klen))
	if err != nil {
		return nil, nil, err
	}
	if err := c.readAt(key, uint64(pos)+recordHeaderSize); err != nil {
		return nil, nil, fmt.Errorf("failed to read key: %w", err)
	}

	value, err = allocBytes(uint64(vlen))
	if err != nil {
		return nil, nil, err
	}
	if err := c.readAt(value, uint64(pos)+recordHeaderSize+uint64(klen)); err != nil {
		return nil, nil, fmt.Errorf("failed to read value: %w", err)
	}

	return key, value, nil
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

	for i := uint32(0); i < table.nslots; i++ {
		slotHash, recPos, err := c.readSlot(table.pos, slot)
		if err != nil {
			return nil, err
		}
		if recPos == 0 {
			return nil, ErrNotFound
		}
		if slotHash == h {
			recKey, recVal, err := c.readRecord(recPos)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(recKey, key) {
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

	for i := uint32(0); i < table.nslots; i++ {
		slotHash, recPos, err := c.readSlot(table.pos, slot)
		if err != nil {
			return nil, err
		}
		if recPos == 0 {
			break
		}
		if slotHash == h {
			recKey, recVal, err := c.readRecord(recPos)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(recKey, key) {
				results = append(results, recVal)
			}
		}
		slot = (slot + 1) % table.nslots
	}

	return results, nil
}
