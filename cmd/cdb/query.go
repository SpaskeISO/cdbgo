package main

import (
	"fmt"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func queryMode(cfg *config) error {
	if cfg.dbfile == "-" {
		return fmt.Errorf("query mode does not support stdin")
	}
	
	db, err := cdb.Open(cfg.dbfile)
	if err != nil {
		return err
	}
	defer cdb.Close(db)
	
	key := []byte(cfg.key)
	
	var values [][]byte
	
	if cfg.recno > 0 {
		// Get specific record number
		value, err := db.GetN(key, cfg.recno)
		if err != nil {
			if err == cdb.ErrNotFound {
				return fmt.Errorf("key not found")
			}
			return err
		}
		values = append(values, value)
	} else {
		// Get all records
		allValues, err := db.GetAll(key)
		if err != nil {
			return err
		}
		if len(allValues) == 0 {
			return fmt.Errorf("key not found")
		}
		values = allValues
	}
	
	// Output values
	for _, value := range values {
		os.Stdout.Write(value)
		if cfg.mapFormat {
			os.Stdout.Write([]byte{'\n'})
		}
	}
	
	return nil
}

