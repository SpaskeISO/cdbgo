package main

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/SpaskeISO/cdbgo/cdb64"
)

type stats struct {
	numRecords  int
	minKeyLen   int
	maxKeyLen   int
	avgKeyLen   float64
	minValueLen int
	maxValueLen int
	avgValueLen float64

	tablesUsed int
	totalSlots int
	usedSlots  int

	minTableSize int
	maxTableSize int
	avgTableSize float64

	collisions int
	distances  [11]int
}

func statsMode(cfg *config) error {
	db, cleanup, err := openDatabase(cfg.dbfile)
	if err != nil {
		return err
	}
	defer cleanup()

	st := &stats{
		minKeyLen:    math.MaxInt32,
		minValueLen:  math.MaxInt32,
		minTableSize: math.MaxInt32,
	}

	it := cdb64.NewIterator(db)
	totalKeyLen := 0
	totalValueLen := 0

	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}

		st.numRecords++

		keyLen := len(key)
		valueLen := len(value)

		totalKeyLen += keyLen
		totalValueLen += valueLen

		if keyLen < st.minKeyLen {
			st.minKeyLen = keyLen
		}
		if keyLen > st.maxKeyLen {
			st.maxKeyLen = keyLen
		}

		if valueLen < st.minValueLen {
			st.minValueLen = valueLen
		}
		if valueLen > st.maxValueLen {
			st.maxValueLen = valueLen
		}
	}

	if err := it.Err(); err != nil {
		return err
	}

	if st.numRecords > 0 {
		st.avgKeyLen = float64(totalKeyLen) / float64(st.numRecords)
		st.avgValueLen = float64(totalValueLen) / float64(st.numRecords)
	} else {
		st.minKeyLen = 0
		st.minValueLen = 0
	}

	if err := analyzeHashTables(db, st); err != nil {
		return err
	}

	printStats(st)
	return nil
}

func analyzeHashTables(db *cdb64.CDB, st *stats) error {
	file := db.GetFile()
	header := make([]byte, cdb64.HeaderSize)
	if _, err := file.ReadAt(header, 0); err != nil {
		return err
	}

	totalTableSize := 0
	fileSize := uint64(db.Size())

	for i := 0; i < cdb64.NumTables; i++ {
		offset := i * 16
		pos := binary.LittleEndian.Uint64(header[offset : offset+8])
		nslots := binary.LittleEndian.Uint64(header[offset+8 : offset+16])

		if nslots == 0 {
			continue
		}

		if nslots > math.MaxUint64/16 {
			return fmt.Errorf("invalid hash table %d", i)
		}
		tableBytes := nslots * 16
		if pos+tableBytes > fileSize || pos+tableBytes < pos || tableBytes > uint64(math.MaxInt) {
			return fmt.Errorf("invalid hash table %d", i)
		}

		st.tablesUsed++
		st.totalSlots += int(nslots)
		totalTableSize += int(nslots)

		if int(nslots) < st.minTableSize {
			st.minTableSize = int(nslots)
		}
		if int(nslots) > st.maxTableSize {
			st.maxTableSize = int(nslots)
		}

		tableData := make([]byte, tableBytes)
		if _, err := file.ReadAt(tableData, int64(pos)); err != nil {
			return err
		}

		usedInTable := 0
		for j := uint64(0); j < nslots; j++ {
			slotOffset := j * 16
			slotHash := binary.LittleEndian.Uint64(tableData[slotOffset : slotOffset+8])
			slotPos := binary.LittleEndian.Uint64(tableData[slotOffset+8 : slotOffset+16])

			if slotPos != 0 {
				usedInTable++

				idealSlot := (slotHash / uint64(cdb64.NumTables)) % nslots
				distance := int(j) - int(idealSlot)
				if distance < 0 {
					distance += int(nslots)
				}

				if distance > 0 {
					st.collisions++
				}

				if distance > 10 {
					st.distances[10]++
				} else {
					st.distances[distance]++
				}
			}
		}

		st.usedSlots += usedInTable
	}

	if st.tablesUsed > 0 {
		st.avgTableSize = float64(totalTableSize) / float64(st.tablesUsed)
	}

	if st.tablesUsed == 0 {
		st.minTableSize = 0
	}

	return nil
}

func printStats(st *stats) {
	fmt.Printf("number of records: %d\n", st.numRecords)
	fmt.Printf("key min/avg/max length: %d/%g/%d\n", st.minKeyLen, st.avgKeyLen, st.maxKeyLen)
	fmt.Printf("val min/avg/max length: %d/%g/%d\n", st.minValueLen, st.avgValueLen, st.maxValueLen)
	fmt.Printf("hash tables used: %d of %d\n", st.tablesUsed, cdb64.NumTables)
	fmt.Printf("hash table slots used: %d of %d\n", st.usedSlots, st.totalSlots)
	fmt.Printf("hash table min/avg/max size: %d/%g/%d\n", st.minTableSize, st.avgTableSize, st.maxTableSize)
	fmt.Printf("hash collisions: %d\n", st.collisions)
	fmt.Println("distance distribution:")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %d: %d\n", i, st.distances[i])
	}
	fmt.Printf("  10+: %d\n", st.distances[10])
}
