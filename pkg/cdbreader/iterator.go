package cdbreader

import (
	"encoding/binary"
	"io"
)

// Iterator provides sequential access to key-value pairs in a CDB file.
//
// It maintains its position in the file and reads entries one by one,
// starting after the hash table (position 2048).
type Iterator struct {
	cdb      *CDBReader
	position uint32 // Current position in the file
}

// Iterator creates and returns a new Iterator instance for the CDB file.
//
// The iterator starts at position 2048, skipping the hash table section.
func (cdb *CDBReader) Iterator() *Iterator {
	return &Iterator{
		cdb:      cdb,
		position: 2048, // Skip the header
	}
}

// Next reads and returns the next key-value pair from the CDB file.
//
// Returns:
//   - key ([]byte): The key data, or nil if end of file is reached
//   - value ([]byte): The value data, or nil if end of file is reached
//   - error: Any error encountered during reading, or nil on success
//
// The method is thread-safe as it uses mutex locking.
// When the end of the file is reached, it returns (nil, nil, nil).
func (iterator *Iterator) Next() ([]byte, []byte, error) {
	iterator.cdb.mutex.Lock()
	defer iterator.cdb.mutex.Unlock()

	if iterator.position > iterator.cdb.endPosition {
		return nil, nil, nil
	}

	// Use a single buffer for reading lengths
	buf := make([]byte, 8)
	_, err := iterator.cdb.file.ReadAt(buf, int64(iterator.position))
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// Read lengths from buffer
	klen := binary.LittleEndian.Uint32(buf[:4])
	vlen := binary.LittleEndian.Uint32(buf[4:])

	// Read key and value in one operation
	combined := make([]byte, klen+vlen)
	_, err = iterator.cdb.file.ReadAt(combined, int64(iterator.position+8))
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	iterator.position += 8 + klen + vlen
	return combined[:klen], combined[klen:], nil
}
