package cdbreader

import (
	"encoding/binary"
	"io"
)

type Iterator struct {
	cdb      *CDBReader
	position uint32 // Current position in the file
}

func (cdb *CDBReader) Iterator() *Iterator {
	return &Iterator{
		cdb:      cdb,
		position: 2048, // Skip the header
	}
}

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
