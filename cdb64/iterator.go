package cdb64

import (
	"encoding/binary"
	"math"
)

// Iterator provides sequential access to all records in a CDB64 database
type Iterator struct {
	cdb *CDB
	pos int64
	err error
}

// NewIterator creates a new iterator for sequential scanning of the database
func NewIterator(cdb *CDB) *Iterator {
	return &Iterator{
		cdb: cdb,
		pos: HeaderSize,
	}
}

// Next advances the iterator to the next record and returns the key and value.
// It returns false when there are no more records or an error occurred.
// Use Err() to check for errors after Next returns false.
func (it *Iterator) Next() (key, value []byte, ok bool) {
	if it.err != nil {
		return nil, nil, false
	}

	if err := it.skipHashTables(); err != nil {
		it.err = err
		return nil, nil, false
	}

	if it.pos >= it.cdb.size {
		return nil, nil, false
	}

	if uint64(it.pos)+recordHeaderSize > uint64(it.cdb.size) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}

	var hdr [recordHeaderSize]byte
	if err := it.cdb.readAt(hdr[:], uint64(it.pos)); err != nil {
		it.err = err
		return nil, nil, false
	}

	klen := binary.LittleEndian.Uint64(hdr[0:8])
	vlen := binary.LittleEndian.Uint64(hdr[8:16])
	recEnd := uint64(it.pos) + recordHeaderSize + klen + vlen
	if recEnd < uint64(it.pos) || recEnd > uint64(it.cdb.size) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}

	key, err := allocBytes(klen)
	if err != nil {
		it.err = err
		return nil, nil, false
	}
	if err := it.cdb.readAt(key, uint64(it.pos)+recordHeaderSize); err != nil {
		it.err = err
		return nil, nil, false
	}

	value, err = allocBytes(vlen)
	if err != nil {
		it.err = err
		return nil, nil, false
	}
	if err := it.cdb.readAt(value, uint64(it.pos)+recordHeaderSize+klen); err != nil {
		it.err = err
		return nil, nil, false
	}

	if recEnd > uint64(math.MaxInt64) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}
	it.pos = int64(recEnd)
	return key, value, true
}

func (it *Iterator) skipHashTables() error {
	for i := 0; i < NumTables; i++ {
		table := &it.cdb.tables[i]
		if table.pos == 0 || table.nslots == 0 {
			continue
		}

		if table.nslots > math.MaxUint64/slotSize {
			return ErrInvalidFormat
		}
		tableBytes := table.nslots * slotSize
		start := table.pos
		end := start + tableBytes
		if end < start {
			return ErrInvalidFormat
		}

		pos := uint64(it.pos)
		if pos >= start && pos < end {
			if end > uint64(math.MaxInt64) {
				return ErrInvalidFormat
			}
			it.pos = int64(end)
		}
	}
	return nil
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
