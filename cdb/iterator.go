package cdb

import (
	"encoding/binary"
	"io"
)

// Iterator provides sequential access to all records in a CDB database
type Iterator struct {
	cdb *CDB
	pos int64
	err error
}

// NewIterator creates a new iterator for sequential scanning of the database
func NewIterator(cdb *CDB) *Iterator {
	return &Iterator{
		cdb: cdb,
		pos: HeaderSize, // Start after header
	}
}

// Next advances the iterator to the next record and returns the key and value.
// It returns false when there are no more records or an error occurred.
// Use Err() to check for errors after Next returns false.
func (it *Iterator) Next() (key, value []byte, ok bool) {
	// Check if we've reached the end of the file
	if it.pos >= it.cdb.size {
		return nil, nil, false
	}

	// Read record header (klen + vlen)
	header := make([]byte, 8)
	n, err := it.cdb.file.ReadAt(header, it.pos)
	if err != nil {
		if err != io.EOF {
			it.err = err
		}
		return nil, nil, false
	}
	if n != 8 {
		return nil, nil, false
	}

	klen := binary.LittleEndian.Uint32(header[0:4])
	vlen := binary.LittleEndian.Uint32(header[4:8])

	// Validate lengths
	if it.pos+8+int64(klen)+int64(vlen) > it.cdb.size {
		return nil, nil, false
	}

	// Read key
	key = make([]byte, klen)
	if klen > 0 {
		if _, err := it.cdb.file.ReadAt(key, it.pos+8); err != nil {
			it.err = err
			return nil, nil, false
		}
	}

	// Read value
	value = make([]byte, vlen)
	if vlen > 0 {
		if _, err := it.cdb.file.ReadAt(value, it.pos+8+int64(klen)); err != nil {
			it.err = err
			return nil, nil, false
		}
	}

	// Move position to next record
	it.pos += 8 + int64(klen) + int64(vlen)

	// Check if this position is within a hash table
	// Hash tables can appear at any position after header, so we need to skip them
	for i := 0; i < NumTables; i++ {
		table := &it.cdb.tables[i]
		if table.pos > 0 && it.pos >= int64(table.pos) && it.pos < int64(table.pos)+int64(table.nslots*8) {
			// We're inside a hash table, skip it
			it.pos = int64(table.pos) + int64(table.nslots*8)
		}
	}

	return key, value, true
}

// Err returns any error that occurred during iteration
func (it *Iterator) Err() error {
	return it.err
}

// Reset resets the iterator to the beginning of the database
func (it *Iterator) Reset() {
	it.pos = HeaderSize
	it.err = nil
}
