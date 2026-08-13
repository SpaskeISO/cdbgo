package main

import (
	"fmt"
	"io"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func openDatabase(path string) (*cdb.CDB, func(), error) {
	if path != "-" {
		db, err := cdb.Open(path)
		if err != nil {
			return nil, nil, err
		}
		return db, func() { _ = db.Close() }, nil
	}

	tmp, err := os.CreateTemp("", "cdb-stdin-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file for stdin: %w", err)
	}
	tmpName := tmp.Name()

	cleanupTmp := func() {
		_ = os.Remove(tmpName)
	}

	if _, err := io.Copy(tmp, os.Stdin); err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return nil, nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanupTmp()
		return nil, nil, err
	}

	db, err := cdb.Open(tmpName)
	if err != nil {
		cleanupTmp()
		return nil, nil, err
	}

	return db, func() {
		_ = db.Close()
		cleanupTmp()
	}, nil
}
