package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb/cdb64"
)

func main() {
	dbPath := "example64.cdb64"

	// Create a database
	fmt.Println("Creating database...")
	if err := createDatabase(dbPath); err != nil {
		log.Fatal(err)
	}

	// Read from database
	fmt.Println("\nReading from database...")
	if err := readDatabase(dbPath); err != nil {
		log.Fatal(err)
	}

	// Iterate through all records
	fmt.Println("\nIterating through all records...")
	if err := iterateDatabase(dbPath); err != nil {
		log.Fatal(err)
	}

	// Clean up
	if err := os.Remove(dbPath); err != nil {
		log.Printf("Warning: failed to remove database: %v", err)
	}

	fmt.Println("\nExample completed successfully!")
}

func createDatabase(path string) error {
	writer, err := cdb64.Create(path, "")
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Add some records
	records := map[string]string{
		"name":    "Alice",
		"age":     "30",
		"city":    "San Francisco",
		"country": "USA",
		"email":   "alice@example.com",
	}

	for key, value := range records {
		if err := writer.PutString(key, value); err != nil {
			if abortErr := writer.Abort(); abortErr != nil {
				return fmt.Errorf("put failed: %w; abort failed: %v", err, abortErr)
			}
			return fmt.Errorf("failed to put record: %w", err)
		}
		fmt.Printf("  Added: %s -> %s\n", key, value)
	}

	// Add duplicate keys to demonstrate multiple values
	if err := writer.PutString("hobby", "reading"); err != nil {
		return err
	}
	fmt.Println("  Added: hobby -> reading")

	if err := writer.PutString("hobby", "hiking"); err != nil {
		return err
	}
	fmt.Println("  Added: hobby -> hiking")

	if err := writer.PutString("hobby", "coding"); err != nil {
		return err
	}
	fmt.Println("  Added: hobby -> coding")

	if err := writer.Finalize(); err != nil {
		return fmt.Errorf("failed to finalize database: %w", err)
	}

	return nil
}

func readDatabase(path string) error {
	db, err := cdb64.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := cdb64.Close(db); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Get single value
	name, err := db.Get([]byte("name"))
	if err != nil {
		return fmt.Errorf("failed to get name: %w", err)
	}
	fmt.Printf("  Name: %s\n", string(name))

	// Get all values for a key with duplicates
	hobbies, err := db.GetAll([]byte("hobby"))
	if err != nil {
		return fmt.Errorf("failed to get hobbies: %w", err)
	}
	fmt.Printf("  Hobbies: ")
	for i, hobby := range hobbies {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(string(hobby))
	}
	fmt.Println()

	// Get specific record number
	secondHobby, err := db.GetN([]byte("hobby"), 2)
	if err != nil {
		return fmt.Errorf("failed to get second hobby: %w", err)
	}
	fmt.Printf("  Second hobby: %s\n", string(secondHobby))

	// Try to get non-existent key
	_, err = db.Get([]byte("nonexistent"))
	if err == cdb64.ErrNotFound {
		fmt.Println("  Key 'nonexistent' not found (as expected)")
	}

	return nil
}

func iterateDatabase(path string) error {
	db, err := cdb64.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := cdb64.Close(db); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	it := cdb64.NewIterator(db)
	count := 0

	for {
		key, value, ok := it.Next()
		if !ok {
			break
		}
		count++
		fmt.Printf("  %s -> %s\n", string(key), string(value))
	}

	if err := it.Err(); err != nil {
		return fmt.Errorf("iteration error: %w", err)
	}

	fmt.Printf("  Total records: %d\n", count)
	return nil
}
