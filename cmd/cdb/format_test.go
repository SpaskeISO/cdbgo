package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func TestExitStatus(t *testing.T) {
	if got := exitStatus(nil); got != 0 {
		t.Errorf("nil err: got %d, want 0", got)
	}
	if got := exitStatus(cdb.ErrNotFound); got != 100 {
		t.Errorf("not found: got %d, want 100", got)
	}
	if got := exitStatus(errors.New("boom")); got != 1 {
		t.Errorf("other error: got %d, want 1", got)
	}
}

func TestNativeFormatRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeNativeRecord(&buf, []byte("key"), []byte("val\nue")); err != nil {
		t.Fatal(err)
	}

	r := newNativeFormatReader(&buf)
	key, value, err := r.next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if string(key) != "key" || string(value) != "val\nue" {
		t.Errorf("got %q=%q", key, value)
	}
}

func TestNativeFormatNegativeLength(t *testing.T) {
	r := newNativeFormatReader(strings.NewReader("+-1,5:xxxxx\n"))
	_, _, err := r.next()
	if err == nil {
		t.Fatal("expected error for negative length")
	}
}

func TestMapFormatLongLine(t *testing.T) {
	value := strings.Repeat("x", 70*1024)
	input := "key " + value + "\n"
	r := newMapFormatReader(strings.NewReader(input))
	k, v, err := r.next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if string(k) != "key" {
		t.Errorf("key = %q", k)
	}
	if string(v) != value {
		t.Errorf("value length = %d, want %d", len(v), len(value))
	}
	_, _, err = r.next()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestMapFormatCommentAndEmpty(t *testing.T) {
	input := "# comment\n\nfoo bar\n"
	r := newMapFormatReader(strings.NewReader(input))
	k, v, err := r.next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if string(k) != "foo" || string(v) != "bar" {
		t.Errorf("got %q=%q", k, v)
	}
}
