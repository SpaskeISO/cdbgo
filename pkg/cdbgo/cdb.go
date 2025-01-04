package cdbgo

import (
	"encoding/binary"
	"io"
	"os"
)

type CDB struct {
	file        *os.File
	tables      [256]table
	endPosition uint32
}

type table struct {
	position uint32
	slots    uint32
}

func Open(filepath string) (*CDB, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	cdb := &CDB{file: file}

	for i := 0; i < 256; i++ {
		var position uint32
		var slots uint32
		err = binary.Read(file, binary.LittleEndian, &position)
		if err != nil {
			return nil, err
		}
		err = binary.Read(file, binary.LittleEndian, &slots)
		if err != nil {
			return nil, err
		}

		cdb.tables[i] = table{position: position, slots: slots}
	}

	return cdb, nil
}

func (cdb *CDB) Close() error {
	return cdb.file.Close()
}

func (cdb *CDB) Get(key []byte) ([]byte, error) {
	hashedKey := hash(key)

	tableNumber := hashedKey & 0xff

	table := cdb.tables[tableNumber]

	if table.slots == 0 {
		return nil, nil
	}

	// Calculate slot number
	slotNumber := ((hashedKey >> 8) % table.slots)

	// Seek to hash table position
	_, err := cdb.file.Seek(int64(table.position+slotNumber*8), 0)
	if err != nil {
		return nil, err
	}

	var hash uint32
	var position uint32
	err = binary.Read(cdb.file, binary.LittleEndian, &hash)
	if err != nil {
		return nil, err
	}
	err = binary.Read(cdb.file, binary.LittleEndian, &position)
	if err != nil {
		return nil, err
	}

	// If slot is empty, key is not found
	if position == 0 {
		return nil, nil
	}

	// Read key length and value length
	_, err = cdb.file.Seek(int64(position), 0)
	if err != nil {
		return nil, err
	}

	var klen uint32
	var vlen uint32
	err = binary.Read(cdb.file, binary.LittleEndian, &klen)
	if err != nil {
		return nil, err
	}
	err = binary.Read(cdb.file, binary.LittleEndian, &vlen)
	if err != nil {
		return nil, err
	}

	// Read and compare key
	k := make([]byte, klen)
	if _, err := io.ReadFull(cdb.file, k); err != nil {
		return nil, err
	}

	if string(k) != string(key) {
		return nil, nil
	}

	// Read value
	v := make([]byte, vlen)
	if _, err := io.ReadFull(cdb.file, v); err != nil {
		return nil, err
	}

	return v, nil
}
