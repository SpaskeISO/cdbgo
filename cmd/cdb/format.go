package main

import (
	"bufio"
	"fmt"
	"io"
)

// writeNativeRecord writes a key-value pair in native CDB format
func writeNativeRecord(w io.Writer, key, value []byte) error {
	_, err := fmt.Fprintf(w, "+%d,%d:%s->%s\n", len(key), len(value), key, value)
	return err
}

// writeNativeKey writes a key in native CDB format (for list mode)
func writeNativeKey(w io.Writer, key []byte) error {
	_, err := fmt.Fprintf(w, "+%d:%s\n", len(key), key)
	return err
}

// writeMapRecord writes a key-value pair in map format
func writeMapRecord(w io.Writer, key, value []byte) error {
	_, err := fmt.Fprintf(w, "%s %s\n", key, value)
	return err
}

// writeMapKey writes a key in map format (for list mode)
func writeMapKey(w io.Writer, key []byte) error {
	_, err := fmt.Fprintf(w, "%s\n", key)
	return err
}

// readNativeFormat reads records from native CDB format
type nativeFormatReader struct {
	r *bufio.Reader
}

func newNativeFormatReader(r io.Reader) *nativeFormatReader {
	return &nativeFormatReader{
		r: bufio.NewReader(r),
	}
}

func (nfr *nativeFormatReader) next() (key, value []byte, err error) {
	// Read '+'
	b, err := nfr.r.ReadByte()
	if err != nil {
		return nil, nil, err
	}

	if b == '\n' {
		// Empty line means end of data
		return nil, nil, io.EOF
	}

	if b != '+' {
		return nil, nil, fmt.Errorf("invalid format: expected '+', got '%c'", b)
	}

	// Read klen
	var klen, vlen int
	if _, err := fmt.Fscanf(nfr.r, "%d,%d:", &klen, &vlen); err != nil {
		return nil, nil, fmt.Errorf("invalid format: %w", err)
	}

	// Read key
	key = make([]byte, klen)
	if _, err := io.ReadFull(nfr.r, key); err != nil {
		return nil, nil, fmt.Errorf("failed to read key: %w", err)
	}

	// Read '->'
	arrow := make([]byte, 2)
	if _, err := io.ReadFull(nfr.r, arrow); err != nil {
		return nil, nil, err
	}
	if arrow[0] != '-' || arrow[1] != '>' {
		return nil, nil, fmt.Errorf("invalid format: expected '->'")
	}

	// Read value
	value = make([]byte, vlen)
	if _, err := io.ReadFull(nfr.r, value); err != nil {
		return nil, nil, fmt.Errorf("failed to read value: %w", err)
	}

	// Read newline
	b, err = nfr.r.ReadByte()
	if err != nil {
		return nil, nil, err
	}
	if b != '\n' {
		return nil, nil, fmt.Errorf("invalid format: expected newline")
	}

	return key, value, nil
}

// readMapFormat reads records from map format
type mapFormatReader struct {
	scanner *bufio.Scanner
}

func newMapFormatReader(r io.Reader) *mapFormatReader {
	return &mapFormatReader{
		scanner: bufio.NewScanner(r),
	}
}

func (mfr *mapFormatReader) next() (key, value []byte, err error) {
	for mfr.scanner.Scan() {
		line := mfr.scanner.Bytes()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Trim leading whitespace
		start := 0
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}

		// Skip comment lines
		if start >= len(line) || line[start] == '#' {
			continue
		}

		// Find key end (first whitespace)
		keyEnd := start
		for keyEnd < len(line) && line[keyEnd] != ' ' && line[keyEnd] != '\t' {
			keyEnd++
		}

		key = make([]byte, keyEnd-start)
		copy(key, line[start:keyEnd])

		// Find value start (skip whitespace)
		valueStart := keyEnd
		for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
			valueStart++
		}

		if valueStart < len(line) {
			value = make([]byte, len(line)-valueStart)
			copy(value, line[valueStart:])
		} else {
			value = []byte{}
		}

		return key, value, nil
	}

	if err := mfr.scanner.Err(); err != nil {
		return nil, nil, err
	}

	return nil, nil, io.EOF
}
