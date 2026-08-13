package cdb

import (
	"encoding/binary"
	"math"
)

// Iterator provides sequential access to all records in a CDB database
type Iterator struct {
	cdb *CDB
	pos int64
	end int64
	err error
}

// NewIterator creates a new iterator for sequential scanning of the database
func NewIterator(cdb *CDB) *Iterator {
	return &Iterator{
		cdb: cdb,
		pos: HeaderSize,
		end: cdb.dataEnd,
	}
}

// Next advances the iterator to the next record and returns the key and value.
// It returns false when there are no more records or an error occurred.
// Use Err() to check for errors after Next returns false.
func (it *Iterator) Next() (key, value []byte, ok bool) {
	if it.err != nil {
		return nil, nil, false
	}

	if it.pos >= it.end {
		return nil, nil, false
	}

	if uint64(it.pos)+recordHeaderSize > uint64(it.end) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}

	var hdr [recordHeaderSize]byte
	if err := it.cdb.readAt(hdr[:], uint64(it.pos)); err != nil {
		it.err = err
		return nil, nil, false
	}

	klen := binary.LittleEndian.Uint32(hdr[0:4])
	vlen := binary.LittleEndian.Uint32(hdr[4:8])
	payloadLen := uint64(klen) + uint64(vlen)
	recEnd := uint64(it.pos) + recordHeaderSize + payloadLen
	if recEnd < uint64(it.pos) || recEnd > uint64(it.end) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}

	buf, err := allocBytes(payloadLen)
	if err != nil {
		it.err = err
		return nil, nil, false
	}
	if err := it.cdb.readAt(buf, uint64(it.pos)+recordHeaderSize); err != nil {
		it.err = err
		return nil, nil, false
	}

	if recEnd > uint64(math.MaxInt64) {
		it.err = ErrInvalidFormat
		return nil, nil, false
	}
	it.pos = int64(recEnd)
	return buf[:klen], buf[klen:], true
}

// Err returns any error that occurred during iteration
func (it *Iterator) Err() error {
	return it.err
}

// Reset resets the iterator to the beginning of the database
func (it *Iterator) Reset() {
	it.pos = HeaderSize
	it.end = it.cdb.dataEnd
	it.err = nil
}
