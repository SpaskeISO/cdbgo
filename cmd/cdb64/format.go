package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
)

func writeNativeRecord(w io.Writer, key, value []byte) error {
	_, err := fmt.Fprintf(w, "+%d,%d:", len(key), len(value))
	if err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "->"); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

func writeNativeKey(w io.Writer, key []byte) error {
	_, err := fmt.Fprintf(w, "+%d:", len(key))
	if err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

func writeMapRecord(w io.Writer, key, value []byte) error {
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := io.WriteString(w, " "); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func writeMapKey(w io.Writer, key []byte) error {
	if _, err := w.Write(key); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

type nativeFormatReader struct {
	r *bufio.Reader
}

func newNativeFormatReader(r io.Reader) *nativeFormatReader {
	return &nativeFormatReader{
		r: bufio.NewReader(r),
	}
}

func (nfr *nativeFormatReader) next() (key, value []byte, err error) {
	b, err := nfr.r.ReadByte()
	if err != nil {
		return nil, nil, err
	}

	if b == '\n' {
		return nil, nil, io.EOF
	}

	if b != '+' {
		return nil, nil, fmt.Errorf("invalid format: expected '+', got '%c'", b)
	}

	var klen, vlen int
	if _, err := fmt.Fscanf(nfr.r, "%d,%d:", &klen, &vlen); err != nil {
		return nil, nil, fmt.Errorf("invalid format: %w", err)
	}

	if klen < 0 || vlen < 0 {
		return nil, nil, fmt.Errorf("invalid format: negative length")
	}
	if klen > math.MaxInt-1 || vlen > math.MaxInt-1 {
		return nil, nil, fmt.Errorf("invalid format: length too large")
	}

	key = make([]byte, klen)
	if _, err := io.ReadFull(nfr.r, key); err != nil {
		return nil, nil, fmt.Errorf("failed to read key: %w", err)
	}

	arrow := make([]byte, 2)
	if _, err := io.ReadFull(nfr.r, arrow); err != nil {
		return nil, nil, err
	}
	if arrow[0] != '-' || arrow[1] != '>' {
		return nil, nil, fmt.Errorf("invalid format: expected '->'")
	}

	value = make([]byte, vlen)
	if _, err := io.ReadFull(nfr.r, value); err != nil {
		return nil, nil, fmt.Errorf("failed to read value: %w", err)
	}

	b, err = nfr.r.ReadByte()
	if err != nil {
		return nil, nil, err
	}
	if b != '\n' {
		return nil, nil, fmt.Errorf("invalid format: expected newline")
	}

	return key, value, nil
}

type mapFormatReader struct {
	r *bufio.Reader
}

func newMapFormatReader(r io.Reader) *mapFormatReader {
	return &mapFormatReader{
		r: bufio.NewReader(r),
	}
}

func (mfr *mapFormatReader) next() (key, value []byte, err error) {
	for {
		line, err := mfr.r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) == 0 {
					return nil, nil, io.EOF
				}
			} else {
				return nil, nil, err
			}
		}

		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 {
			if err == io.EOF {
				return nil, nil, io.EOF
			}
			continue
		}

		start := 0
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}

		if start >= len(line) || line[start] == '#' {
			if err == io.EOF {
				return nil, nil, io.EOF
			}
			continue
		}

		keyEnd := start
		for keyEnd < len(line) && line[keyEnd] != ' ' && line[keyEnd] != '\t' {
			keyEnd++
		}

		key = make([]byte, keyEnd-start)
		copy(key, line[start:keyEnd])

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
}
