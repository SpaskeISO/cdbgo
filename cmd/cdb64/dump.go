package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb64"
)

func dumpMode(cfg *config) error {
	if cfg.dbfile == "-" {
		return fmt.Errorf("dump mode does not support stdin (yet)")
	}

	db, err := cdb64.Open(cfg.dbfile)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("failed to close database", "error", err)
		}
	}()

	it := cdb64.NewIterator(db)

	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}

		if cfg.mapFormat {
			if err := writeMapRecord(os.Stdout, key, value); err != nil {
				return err
			}
		} else {
			if err := writeNativeRecord(os.Stdout, key, value); err != nil {
				return err
			}
		}
	}

	if err := it.Err(); err != nil {
		return err
	}

	// Write empty line for native format
	if !cfg.mapFormat {
		fmt.Println()
	}

	return nil
}
