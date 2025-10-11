package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SpaskeISO/cdbgo/cdb"
)

func main() {
	// Create a sample database
	dbPath := "example.cdb"
	
	fmt.Println("Creating CDB database...")
	writer, err := cdb.Create(dbPath, "")
	if err != nil {
		log.Fatal(err)
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
			writer.Abort()
			log.Fatal(err)
		}
		fmt.Printf("  Added: %s -> %s\n", key, value)
	}
	
	if err := writer.Finalize(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database created successfully!")
	
	// Open and read from the database
	fmt.Println("\nReading from database...")
	db, err := cdb.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer cdb.Close(db)
	
	// Query specific keys
	keys := []string{"name", "age", "city"}
	for _, key := range keys {
		value, err := db.Get([]byte(key))
		if err != nil {
			if err == cdb.ErrNotFound {
				fmt.Printf("  %s: not found\n", key)
			} else {
				log.Fatal(err)
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
		log.Fatal(err)
	}
	
	fmt.Printf("\nTotal records: %d\n", count)
	
	// Clean up
	os.Remove(dbPath)
	fmt.Println("\nExample completed!")
}

