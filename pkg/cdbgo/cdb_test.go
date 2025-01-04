package cdbgo

import "testing"

func TestGet(t *testing.T) {

	cdb, err := Open("test.cdb")
	if err != nil {
		t.Fatal(err)
	}

	defer cdb.Close()

	value, err := cdb.Get([]byte("one"))
	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "here" {
		t.Fatalf("expected here, got %s", value)
	}

	t.Log("value:", string(value))
}
