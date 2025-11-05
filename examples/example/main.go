package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func main() {
	// Create a sample database
	dbPath := "example.cdb"

	fmt.Println("Creating CDB database...")
	writer, err := cdb.Create(dbPath, "")
	if err != nil {
		slog.Error("failed to create database", "error", err)
		os.Exit(1)
	}

	// Add some sample data
	data := map[string]string{
		"name":    "Alice",
		"age":     "30",
		"city":    "New York",
		"country": "USA",
		"job":     "Engineer",
	}

	for key, value := range data {
		if err := writer.PutString(key, value); err != nil {
			if abortErr := writer.Abort(); abortErr != nil {
				slog.Warn("failed to abort after error", "abort_error", abortErr)
			}
			slog.Error("failed to put record", "key", key, "error", err)
			os.Exit(1)
		}
		fmt.Printf("  Added: %s -> %s\n", key, value)
	}

	if err := writer.Finalize(); err != nil {
		slog.Error("failed to finalize database", "error", err)
		os.Exit(1)
	}
	fmt.Println("Database created successfully!")

	// Open and read from the database
	fmt.Println("\nReading from database...")
	db, err := cdb.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("failed to close database", "error", err)
		}
	}()

	// Query specific keys
	keys := []string{"name", "age", "city"}
	for _, key := range keys {
		value, err := db.Get([]byte(key))
		if err != nil {
			if err == cdb.ErrNotFound {
				fmt.Printf("  %s: not found\n", key)
			} else {
				slog.Error("failed to get key", "key", key, "error", err)
				os.Exit(1)
			}
			continue
		}
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Iterate through all records
	fmt.Println("\nAll records in database:")
	it := cdb.NewIterator(db)
	count := 0
	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}
		count++
		fmt.Printf("  %s: %s\n", key, value)
	}

	if err := it.Err(); err != nil {
		slog.Error("iterator error", "error", err)
		os.Exit(1)
	}

	fmt.Printf("\nTotal records: %d\n", count)

	// Clean up
	if err := os.Remove(dbPath); err != nil {
		slog.Warn("failed to remove example database", "error", err)
	}
	fmt.Println("\nExample completed!")
}
