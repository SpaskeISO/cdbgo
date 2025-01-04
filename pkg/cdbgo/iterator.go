package cdbgo

import (
	"encoding/binary"
	"io"
)

type Iterator struct {
	cdb      *CDB
	position uint32 // Current position in the file
}

func (cdb *CDB) Iterator() *Iterator {
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

	_, err := iterator.cdb.file.Seek(int64(iterator.position), 0)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var klen uint32
	var vlen uint32
	err = binary.Read(iterator.cdb.file, binary.LittleEndian, &klen)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	err = binary.Read(iterator.cdb.file, binary.LittleEndian, &vlen)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	key := make([]byte, klen)
	_, err = io.ReadFull(iterator.cdb.file, key)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	value := make([]byte, vlen)
	_, err = io.ReadFull(iterator.cdb.file, value)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	iterator.position += 8 + klen + vlen
	return key, value, nil
}
