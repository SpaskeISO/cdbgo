package cdb64

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const (
	// HeaderSize is the size of the CDB64 header (1024 hash table pointers, 16 bytes each)
	HeaderSize = 16384
	// NumTables is the number of hash tables in a CDB64 file
	NumTables = 1024
)

var (
	// ErrNotFound is returned when a key is not found in the database
	ErrNotFound = errors.New("key not found")
	// ErrInvalidFormat is returned when the CDB64 file is malformed
	ErrInvalidFormat = errors.New("invalid CDB64 format")
)

// hashTable represents a single hash table in the CDB64
type hashTable struct {
	pos    uint64 // position in file
	nslots uint64 // number of slots
}

// CDB represents an open CDB64 database
type CDB struct {
	file   *os.File
	tables [NumTables]hashTable
	size   int64
}

// Open opens a CDB64 database file for reading
func Open(filename string) (*CDB, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file size
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

	// Read hash table pointers from header
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close file after read error", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	for i := 0; i < NumTables; i++ {
		offset := i * 16
		cdb.tables[i].pos = binary.LittleEndian.Uint64(header[offset : offset+8])
		cdb.tables[i].nslots = binary.LittleEndian.Uint64(header[offset+8 : offset+16])
	}

	return cdb, nil
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

// hash computes the 64-bit DJB hash of a key
func hash(key []byte) uint64 {
	h := uint64(5381)
	for _, b := range key {
		h = ((h << 5) + h) ^ uint64(b)
	}
	return h
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

	// Start searching in the hash table
	slot := (h / NumTables) % table.nslots
	count := 0

	for i := uint64(0); i < table.nslots; i++ {
		slotPos := table.pos + (slot * 16)

		// Read slot entry (hash + pos)
		slotData := make([]byte, 16)
		if _, err := c.file.ReadAt(slotData, int64(slotPos)); err != nil {
			return nil, fmt.Errorf("failed to read slot: %w", err)
		}

		slotHash := binary.LittleEndian.Uint64(slotData[0:8])
		slotRecPos := binary.LittleEndian.Uint64(slotData[8:16])

		// Empty slot means key not found
		if slotRecPos == 0 {
			return nil, ErrNotFound
		}

		// Check if hash matches
		if slotHash == h {
			// Read the record at this position
			recData := make([]byte, 16)
			if _, err := c.file.ReadAt(recData, int64(slotRecPos)); err != nil {
				return nil, fmt.Errorf("failed to read record: %w", err)
			}

			klen := binary.LittleEndian.Uint64(recData[0:8])
			vlen := binary.LittleEndian.Uint64(recData[8:16])

			// Read key
			recKey := make([]byte, klen)
			if _, err := c.file.ReadAt(recKey, int64(slotRecPos)+16); err != nil {
				return nil, fmt.Errorf("failed to read key: %w", err)
			}

			// Check if key matches
			if string(recKey) == string(key) {
				count++
				if count == n {
					// Read value
					value := make([]byte, vlen)
					if _, err := c.file.ReadAt(value, int64(slotRecPos)+16+int64(klen)); err != nil {
						return nil, fmt.Errorf("failed to read value: %w", err)
					}
					return value, nil
				}
			}
		}

		// Move to next slot (linear probing)
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

	// Start searching in the hash table
	slot := (h / NumTables) % table.nslots

	for i := uint64(0); i < table.nslots; i++ {
		slotPos := table.pos + (slot * 16)

		// Read slot entry (hash + pos)
		slotData := make([]byte, 16)
		if _, err := c.file.ReadAt(slotData, int64(slotPos)); err != nil {
			return nil, fmt.Errorf("failed to read slot: %w", err)
		}

		slotHash := binary.LittleEndian.Uint64(slotData[0:8])
		slotRecPos := binary.LittleEndian.Uint64(slotData[8:16])

		// Empty slot means no more entries
		if slotRecPos == 0 {
			break
		}

		// Check if hash matches
		if slotHash == h {
			// Read the record at this position
			recData := make([]byte, 16)
			if _, err := c.file.ReadAt(recData, int64(slotRecPos)); err != nil {
				return nil, fmt.Errorf("failed to read record: %w", err)
			}

			klen := binary.LittleEndian.Uint64(recData[0:8])
			vlen := binary.LittleEndian.Uint64(recData[8:16])

			// Read key
			recKey := make([]byte, klen)
			if _, err := c.file.ReadAt(recKey, int64(slotRecPos)+16); err != nil {
				return nil, fmt.Errorf("failed to read key: %w", err)
			}

			// Check if key matches
			if string(recKey) == string(key) {
				// Read value
				value := make([]byte, vlen)
				if _, err := c.file.ReadAt(value, int64(slotRecPos)+16+int64(klen)); err != nil {
					return nil, fmt.Errorf("failed to read value: %w", err)
				}
				results = append(results, value)
			}
		}

		// Move to next slot (linear probing)
		slot = (slot + 1) % table.nslots
	}

	return results, nil
}
