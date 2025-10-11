package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb/cdb64"
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
		if err := cdb64.Close(db); err != nil {
			slog.Warn("failed to close database", "error", err)
		}
	}()

	key := []byte(cfg.key)

	var values [][]byte

	if cfg.recno > 0 {
		// Get specific record number
		value, err := db.GetN(key, cfg.recno)
		if err != nil {
			if err == cdb64.ErrNotFound {
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
