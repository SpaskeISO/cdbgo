# cdbgo

A Go implementation of DJB's Constant Database (CDB) format, aiming to reimplement TinyCDB functionality in pure Go.

## Current Features
- Thread-safe CDB file reading
- Fast lookups with O(1) complexity
- Full database iteration support

## Roadmap
- [x] CDB Reader implementation (v1.0.0)
- [ ] CDB Writer implementation
- [ ] Full TinyCDB feature parity:
  - [ ] Database creation
  - [ ] Record insertion
  - [ ] Atomic updates
  - [ ] Command-line tools

## Requirements
 - Go 1.22 or later

## Installation

```bash
go get github.com/SpaskeISO/cdbgo@latest
```

## Usage
### cdbreader
```go
package main

import (
    "fmt"
    "github.com/SpaskeISO/cdbgo/pkg/cdbreader"
)

func main() {
    // Open CDB file
    db, err := cdbreader.Open("data.cdb")
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
