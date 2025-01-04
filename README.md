# cdbgo
A Go library that allows you to read data from Constant Database (CDB) file

## What is CDB?

CDB (Constant Database) is a fast, reliable, and simple package designed by Daniel J. Bernstein for creating and reading constant key/value databases. Its main features are:

- Fast lookups: Only two disk accesses per lookup
- Low overhead: Keys and values are stored efficiently
- No crash recovery needed: Reader processes are never corrupted by writer processes
- Atomic updates: Database replacement is atomic

## About cdbgo

cdbgo is a pure Go implementation of a **reader** of the CDB format. It provides:
- Reading CDB files
- Thread-safe concurrent access
- Iterator support for full database scanning

## Requirements
 - Go 1.22 or later

## Installation

```bash
go get github.com/SpaskeISO/cdbgo
```

## Usage
```go
package main

import (
    "fmt"
    "github.com/yourusername/cdbgo"
)

func main() {
    // Open CDB file
    db, err := cdbgo.Open("data.cdb")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Get value by key
    value, err := db.Get([]byte("mykey"))
    if err != nil {
        panic(err)
    }
    if value != nil {
        fmt.Printf("Value: %s\n", value)
    }

    // Iterationg over all records
    iter := db.Iterator()
    for {
        key, value, err := iter.Next()
        if err != nil {
            panic(err)
        }
        if key == nil {
            // End of database
            break
        }
        fmt.Printf("Key: %s, Value: %s\n", key, value)
    }
}
```

## Thread Safety
All operations are thread-safe and can be used concurrently from multiple goroutines.