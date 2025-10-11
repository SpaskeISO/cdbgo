package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/SpaskeISO/cdbgo/cdb"
)

type stats struct {
	numRecords    int
	minKeyLen     int
	maxKeyLen     int
	avgKeyLen     float64
	minValueLen   int
	maxValueLen   int
	avgValueLen   float64
	
	tablesUsed    int
	totalSlots    int
	usedSlots     int
	
	minTableSize  int
	maxTableSize  int
	avgTableSize  float64
	
	collisions    int
	distances     [11]int // 0-9 and 10+
}

func statsMode(cfg *config) error {
	if cfg.dbfile == "-" {
		return fmt.Errorf("stats mode does not support stdin (yet)")
	}
	
	db, err := cdb.Open(cfg.dbfile)
	if err != nil {
		return err
	}
	defer cdb.Close(db)
	
	st := &stats{
		minKeyLen:    math.MaxInt32,
		minValueLen:  math.MaxInt32,
		minTableSize: math.MaxInt32,
	}
	
	// Collect statistics by iterating through records
	it := cdb.NewIterator(db)
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
	
	// Analyze hash tables
	if err := analyzeHashTables(db, st); err != nil {
		return err
	}
	
	// Print statistics
	printStats(st)
	
	return nil
}

func analyzeHashTables(db *cdb.CDB, st *stats) error {
	// Read header to get table information
	file := db.GetFile()
	header := make([]byte, 2048)
	if _, err := file.ReadAt(header, 0); err != nil {
		return err
	}
	
	totalTableSize := 0
	
	for i := 0; i < 256; i++ {
		offset := i * 8
		pos := binary.LittleEndian.Uint32(header[offset : offset+4])
		nslots := binary.LittleEndian.Uint32(header[offset+4 : offset+8])
		
		if nslots == 0 {
			continue
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
		
		// Read hash table
		tableData := make([]byte, nslots*8)
		if _, err := file.ReadAt(tableData, int64(pos)); err != nil {
			if err != io.EOF {
				return err
			}
		}
		
		// Count used slots and collisions
		usedInTable := 0
		for j := uint32(0); j < nslots; j++ {
			slotOffset := j * 8
			slotHash := binary.LittleEndian.Uint32(tableData[slotOffset : slotOffset+4])
			slotPos := binary.LittleEndian.Uint32(tableData[slotOffset+4 : slotOffset+8])
			
			if slotPos != 0 {
				usedInTable++
				
				// Calculate ideal slot position
				idealSlot := (slotHash / 256) % nslots
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
	fmt.Printf("hash tables used: %d of 256\n", st.tablesUsed)
	fmt.Printf("hash table slots used: %d of %d\n", st.usedSlots, st.totalSlots)
	fmt.Printf("hash table min/avg/max size: %d/%g/%d\n", st.minTableSize, st.avgTableSize, st.maxTableSize)
	fmt.Printf("hash collisions: %d\n", st.collisions)
	fmt.Println("distance distribution:")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %d: %d\n", i, st.distances[i])
	}
	fmt.Printf("  10+: %d\n", st.distances[10])
}

