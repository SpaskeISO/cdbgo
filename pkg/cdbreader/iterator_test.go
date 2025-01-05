package cdbreader

import (
	"testing"
)

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

func BenchmarkIterator(b *testing.B) {
	// Open the test database
	cdb, err := Open("test.cdb")
	if err != nil {
		b.Fatal(err)
	}
	defer cdb.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iterator := cdb.Iterator()
		for {
			key, value, err := iterator.Next()
			if err != nil {
				b.Fatal(err)
			}
			if key == nil {
				break
			}
			// Prevent compiler optimization by doing something with the values
			b.SetBytes(int64(len(key) + len(value)))
		}
	}
}

// BenchmarkIteratorParallel tests how well the iterator performs under concurrent access
func BenchmarkIteratorParallel(b *testing.B) {
	cdb, err := Open("test.cdb")
	if err != nil {
		b.Fatal(err)
	}
	defer cdb.Close()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iterator := cdb.Iterator()
			for {
				key, value, err := iterator.Next()
				if err != nil {
					b.Fatal(err)
				}
				if key == nil {
					break
				}
				// Prevent compiler optimization by doing something with the values
				b.SetBytes(int64(len(key) + len(value)))
			}
		}
	})
}
