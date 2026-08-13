package main

import (
	"fmt"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func dumpMode(cfg *config) error {
	db, cleanup, err := openDatabase(cfg.dbfile)
	if err != nil {
		return err
	}
	defer cleanup()

	it := cdb.NewIterator(db)

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

	if !cfg.mapFormat {
		fmt.Println()
	}

	return nil
}
