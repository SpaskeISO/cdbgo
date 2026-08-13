package cdb64

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to generate random keys
func generateRandomKey(b *testing.B, size int) []byte {
	b.Helper()
	key := make([]byte, size)
	_, err := rand.Read(key)
	if err != nil {
		b.Log(err)
	}
	return key
}

// Helper function to generate sequential keys
func generateSequentialKeyB(b *testing.B, i int) []byte {
	b.Helper()
	return fmt.Appendf(nil, "key_%010d", i)
}

// Helper function to generate sequential keys
func generateSequentialKeyT(t *testing.T, i int) []byte {
	t.Helper()
	return fmt.Appendf(nil, "key_%010d", i)
}

// BenchmarkWrite measures write performance
func BenchmarkWrite(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("records_%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				dbPath := filepath.Join(tmpDir, fmt.Sprintf("bench_%d.cdb64", i))
				writer, err := Create(dbPath, "")
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()

				for j := 0; j < size; j++ {
					key := generateSequentialKeyB(b, j)
					value := fmt.Appendf(nil, "value_%d", j)
					if err := writer.Put(key, value); err != nil {
						b.Fatal(err)
					}
				}

				if err := writer.Finalize(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRead measures read performance
func BenchmarkRead(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("records_%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.cdb64")

			// Create database
			writer, err := Create(dbPath, "")
			if err != nil {
				b.Fatal(err)
			}

			for j := 0; j < size; j++ {
				key := generateSequentialKeyB(b, j)
				value := fmt.Appendf(nil, "value_%d", j)
				if err := writer.Put(key, value); err != nil {
					b.Fatal(err)
				}
			}

			if err := writer.Finalize(); err != nil {
				b.Fatal(err)
			}

			// Open for reading
			db, err := Open(dbPath)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Log(err)
				}
			}()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := generateSequentialKeyB(b, i%size)
				_, err := db.Get(key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkReadRandom measures random read performance
func BenchmarkReadRandom(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("records_%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.cdb64")

			// Create database
			writer, err := Create(dbPath, "")
			if err != nil {
				b.Fatal(err)
			}

			keys := make([][]byte, size)
			for j := 0; j < size; j++ {
				keys[j] = generateSequentialKeyB(b, j)
				value := fmt.Appendf(nil, "value_%d", j)
				if err := writer.Put(keys[j], value); err != nil {
					b.Fatal(err)
				}
			}

			if err := writer.Finalize(); err != nil {
				b.Fatal(err)
			}

			// Open for reading
			db, err := Open(dbPath)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Log(err)
				}
			}()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Access keys in pseudo-random order
				idx := (i * 7919) % size
				_, err := db.Get(keys[idx])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkIteration measures iteration performance
func BenchmarkIteration(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("records_%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.cdb64")

			// Create database
			writer, err := Create(dbPath, "")
			if err != nil {
				b.Fatal(err)
			}

			for j := 0; j < size; j++ {
				key := generateSequentialKeyB(b, j)
				value := fmt.Appendf(nil, "value_%d", j)
				if err := writer.Put(key, value); err != nil {
					b.Fatal(err)
				}
			}

			if err := writer.Finalize(); err != nil {
				b.Fatal(err)
			}

			// Open for reading
			db, err := Open(dbPath)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					b.Log(err)
				}
			}()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				it := NewIterator(db)
				count := 0
				for {
					_, _, ok := it.Next()
					if !ok {
						break
					}
					count++
				}
				if count != size {
					b.Fatalf("Expected %d records, got %d", size, count)
				}
			}
		})
	}
}

// BenchmarkHashFunction measures hash function performance
func BenchmarkHashFunction(b *testing.B) {
	keySizes := []int{8, 16, 32, 64, 128, 256}

	for _, size := range keySizes {
		b.Run(fmt.Sprintf("keysize_%d", size), func(b *testing.B) {
			key := generateRandomKey(b, size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = hash(key)
			}
		})
	}
}

// Collision analysis (not a benchmark, but useful for comparison)
func TestCollisionAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping collision analysis in short mode")
	}

	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("records_%d", size), func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "collision.cdb64")

			// Create database
			writer, err := Create(dbPath, "")
			if err != nil {
				t.Fatal(err)
			}

			for j := 0; j < size; j++ {
				key := generateSequentialKeyT(t, j)
				value := fmt.Appendf(nil, "value_%d", j)
				if err := writer.Put(key, value); err != nil {
					t.Fatal(err)
				}
			}

			if err := writer.Finalize(); err != nil {
				t.Fatal(err)
			}

			// Analyze collisions
			file, err := os.Open(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := file.Close(); err != nil {
					t.Log(err)
				}
			}()

			// Read header
			header := make([]byte, HeaderSize)
			if _, err := file.Read(header); err != nil {
				t.Fatal(err)
			}

			totalSlots := 0
			usedSlots := 0
			collisions := 0
			maxDistance := 0
			distances := make(map[int]int)

			for i := 0; i < NumTables; i++ {
				pos := uint64(0)
				nslots := uint64(0)

				for j := 0; j < 8; j++ {
					pos |= uint64(header[i*16+j]) << (j * 8)
					nslots |= uint64(header[i*16+8+j]) << (j * 8)
				}

				if nslots == 0 {
					continue
				}

				totalSlots += int(nslots)

				// Read hash table
				tableData := make([]byte, nslots*16)
				if _, err := file.ReadAt(tableData, int64(pos)); err != nil {
					t.Fatal(err)
				}

				for j := uint64(0); j < nslots; j++ {
					slotHash := uint64(0)
					slotPos := uint64(0)

					for k := uint64(0); k < 8; k++ {
						slotHash |= uint64(tableData[j*16+k]) << (k * 8)
						slotPos |= uint64(tableData[j*16+8+k]) << (k * 8)
					}

					if slotPos != 0 {
						usedSlots++

						// Calculate ideal position
						idealSlot := (slotHash / NumTables) % nslots
						distance := int(j) - int(idealSlot)
						if distance < 0 {
							distance += int(nslots)
						}

						distances[distance]++
						if distance > maxDistance {
							maxDistance = distance
						}
						if distance > 0 {
							collisions++
						}
					}
				}
			}

			collisionRate := float64(collisions) / float64(usedSlots) * 100
			loadFactor := float64(usedSlots) / float64(totalSlots) * 100

			t.Logf("Records: %d", size)
			t.Logf("Total slots: %d", totalSlots)
			t.Logf("Used slots: %d", usedSlots)
			t.Logf("Load factor: %.2f%%", loadFactor)
			t.Logf("Collisions: %d (%.2f%%)", collisions, collisionRate)
			t.Logf("Max distance: %d", maxDistance)
			t.Logf("Distance distribution:")
			for i := 0; i <= maxDistance && i < 10; i++ {
				if count, ok := distances[i]; ok {
					pct := float64(count) / float64(usedSlots) * 100
					t.Logf("  Distance %d: %d (%.2f%%)", i, count, pct)
				}
			}
		})
	}
}
