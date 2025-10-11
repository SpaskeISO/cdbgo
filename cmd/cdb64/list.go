package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb/cdb64"
)

func listMode(cfg *config) error {
	if cfg.dbfile == "-" {
		return fmt.Errorf("list mode does not support stdin (yet)")
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

	it := cdb64.NewIterator(db)

	for {
		key, _, ok := it.Next()
		if !ok {
			break
		}

		if cfg.mapFormat {
			if err := writeMapKey(os.Stdout, key); err != nil {
				return err
			}
		} else {
			if err := writeNativeKey(os.Stdout, key); err != nil {
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
