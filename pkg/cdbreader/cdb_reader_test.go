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

	value, err = cdb.Get([]byte("badKey"))
	if err != nil {
		t.Fatal(err)
	}

	if value != nil {
		t.Fatalf("expected nil, got %s", value)
	}
}

func TestOpenNonExistentFile(t *testing.T) {
	_, err := Open("non_existent.cdb")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func BenchmarkGet(b *testing.B) {
	cdb, err := Open("test.cdb")
	if err != nil {
		b.Fatal(err)
	}
	defer cdb.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cdb.Get([]byte("one"))
		if err != nil {
			b.Fatal(err)
		}
	}
}
