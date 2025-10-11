package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func createMode(cfg *config) error {
	// Create writer
	writer, err := cdb.Create(cfg.dbfile, cfg.tempFile)
	if err != nil {
		return err
	}
	
	// Set duplicate handling mode
	mode := cdb.DuplicateModeAllow
	if cfg.errDup {
		mode = cdb.DuplicateModeError
	} else if cfg.replace {
		mode = cdb.DuplicateModeReplace
	} else if cfg.unique {
		mode = cdb.DuplicateModeUnique
	} else if cfg.zeroFill {
		mode = cdb.DuplicateModeZeroFill
	} else if cfg.warn {
		mode = cdb.DuplicateModeWarn
	}
	writer.SetDuplicateMode(mode)
	
	// Read input files or stdin
	if len(cfg.infiles) == 0 {
		// Read from stdin
		if err := readInput(writer, os.Stdin, cfg.mapFormat); err != nil {
			if abortErr := writer.Abort(); abortErr != nil {
				return fmt.Errorf("input error: %w; abort failed: %v", err, abortErr)
			}
			return err
		}
	} else {
		// Read from files
		for _, filename := range cfg.infiles {
			if err := readFile(writer, filename, cfg.mapFormat); err != nil {
				if abortErr := writer.Abort(); abortErr != nil {
					return fmt.Errorf("input error: %w; abort failed: %v", err, abortErr)
				}
				return err
			}
		}
	}
	
	// Finalize database
	if err := writer.Finalize(); err != nil {
		return err
	}
	
	// Set permissions if specified
	if cfg.perms != "" {
		perms, err := parsePerms(cfg.perms)
		if err != nil {
			return err
		}
		if err := cdb.SetPermissions(cfg.dbfile, perms); err != nil {
			return err
		}
	}
	
	return nil
}

func readInput(writer *cdb.Writer, r io.Reader, mapFormat bool) error {
	if mapFormat {
		reader := newMapFormatReader(r)
		for {
			key, value, err := reader.next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if err := writer.Put(key, value); err != nil {
				return err
			}
		}
	} else {
		reader := newNativeFormatReader(r)
		for {
			key, value, err := reader.next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if err := writer.Put(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func readFile(writer *cdb.Writer, filename string, mapFormat bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Warn("failed to close input file", "filename", filename, "error", err)
		}
	}()
	
	return readInput(writer, file, mapFormat)
}

