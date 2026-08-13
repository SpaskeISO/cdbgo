package main

import (
	"fmt"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func listMode(cfg *config) error {
	db, cleanup, err := openDatabase(cfg.dbfile)
	if err != nil {
		return err
	}
	defer cleanup()

	it := cdb.NewIterator(db)

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

	if !cfg.mapFormat {
		fmt.Println()
	}

	return nil
}
