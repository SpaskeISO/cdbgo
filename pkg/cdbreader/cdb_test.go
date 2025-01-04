package cdbreader

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

func TestIterator(t *testing.T) {

	cdb, err := Open("test.cdb")
	if err != nil {
		t.Fatal(err)
	}

	defer cdb.Close()

	iterator := cdb.Iterator()
	for {
		key, value, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		if key == nil {
			break
		}
		t.Logf("key: %s, value: %s", key, value)
	}

	// Check if normal Get still works
	value, err := cdb.Get([]byte("one"))
	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "here" {
		t.Fatalf("expected here, got %s", value)
	}

	t.Log("value:", string(value))

	// Try iterator once again
	iterator = cdb.Iterator()
	for {
		key, value, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		if key == nil {
			break
		}
		t.Logf("key: %s, value: %s", key, value)
	}
}
