package main

import (
	"fmt"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb64"
)

func queryMode(cfg *config) error {
	if cfg.dbfile == "-" {
		return fmt.Errorf("query mode does not support stdin")
	}

	db, err := cdb64.Open(cfg.dbfile)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "cdb64: warning: failed to close database: %v\n", err)
		}
	}()

	key := []byte(cfg.key)

	var values [][]byte

	switch {
	case cfg.recno > 0:
		value, err := db.GetN(key, cfg.recno)
		if err != nil {
			return err
		}
		values = append(values, value)
	case cfg.allRecords:
		allValues, err := db.GetAll(key)
		if err != nil {
			return err
		}
		if len(allValues) == 0 {
			return cdb64.ErrNotFound
		}
		values = allValues
	default:
		value, err := db.Get(key)
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	for _, value := range values {
		if _, err := os.Stdout.Write(value); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		if cfg.mapFormat {
			if _, err := os.Stdout.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
		}
	}

	return nil
}
