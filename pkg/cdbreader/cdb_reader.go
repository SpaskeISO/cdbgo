package cdbreader

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sync"
)

type CDBReader struct {
	file        *os.File
	mutex       sync.Mutex
	tables      [256]table
	endPosition uint32
}

type table struct {
	position uint32
	slots    uint32
}

func Open(filepath string) (*CDBReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		fileCloseErr := file.Close()
		if fileCloseErr != nil {
			return nil, fileCloseErr
		}
		return nil, err
	}

	cdb := &CDBReader{file: file, endPosition: uint32(fileInfo.Size())}

	// Read all table data at once (256 tables * 8 bytes per table = 2048 bytes)
	buf := make([]byte, 256*8)
	_, err = io.ReadFull(file, buf)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Parse the buffer into tables
	for i := 0; i < 256; i++ {
		offset := i * 8
		position := binary.LittleEndian.Uint32(buf[offset : offset+4])
		slots := binary.LittleEndian.Uint32(buf[offset+4 : offset+8])
		cdb.tables[i] = table{position: position, slots: slots}
	}

	return cdb, nil
}

func (cdb *CDBReader) Close() error {
	return cdb.file.Close()
}

// Get retrieves a value from the CDB file for the given key. It returns the value as a byte slice
// if found, or (nil, nil) if the key doesn't exist. Returns (nil, error) if an error occurs during reading.
//
// This function is thread-safe as it uses a mutex to synchronize access to the file.
func (cdb *CDBReader) Get(key []byte) ([]byte, error) {
	cdb.mutex.Lock()
	defer cdb.mutex.Unlock()
	hashedKey := hash(key)

	// Calculate table number
	tableNumber := hashedKey & 0xff
	table := cdb.tables[tableNumber]

	if table.slots == 0 {
		return nil, nil
	}

	// Calculate slot number
	slotNumber := ((hashedKey >> 8) % table.slots)

	buf := make([]byte, 8)

	// Seek and read position in one operation
	_, err := cdb.file.ReadAt(buf[:4], int64(table.position+slotNumber*8+4))
	if err != nil {
		return nil, err
	}
	position := binary.LittleEndian.Uint32(buf[:4])

	if position == 0 {
		return nil, nil
	}

	// Read key length and value length in one operation
	_, err = cdb.file.ReadAt(buf, int64(position))
	if err != nil {
		return nil, err
	}
	klen := binary.LittleEndian.Uint32(buf[:4])
	vlen := binary.LittleEndian.Uint32(buf[4:])

	// Read key and value in one operation
	combined := make([]byte, klen+vlen)
	_, err = cdb.file.ReadAt(combined, int64(position+8))
	if err != nil {
		return nil, err
	}

	// Compare key
	if !bytes.Equal(combined[:klen], key) {
		return nil, nil
	}

	// Return value portion
	return combined[klen:], nil
}
