# cdbgo

A complete implementation of the Constant DataBase (CDB) format in Go, including both 32-bit and 64-bit versions with libraries and full-featured command-line tools.

## Features

- **32-bit CDB (cdb)**: Classic implementation compatible with DJB's original format
  - 256 hash tables, 2KB header
  - 4GB maximum file size
  - Standard `cdb` package and CLI tool
  
- **64-bit CDB (cdb64)**: Extended implementation for large-scale databases
  - 1024 hash tables, 16KB header
  - Exabyte-scale file support
  - 64-bit hash function for better collision resistance
  - Separate `cdb64` package and CLI tool
  
- **Common Features**:
  - Simple API for reading and writing databases
  - Complete CLI tools with all modes (query, dump, list, create, stats)
  - Native CDB format and map format support
  - Multiple duplicate key handling strategies
  - Efficient hash-based lookups with minimal overhead
  - Comprehensive unit tests

## Installation

```bash
# Install the 32-bit library
go get github.com/SpaskeISO/cdbgo/cdb

# Install the 64-bit library
go get github.com/SpaskeISO/cdbgo/cdb/cdb64

# Install the 32-bit binary
go install github.com/SpaskeISO/cdbgo/cmd/cdb@latest

# Install the 64-bit binary
go install github.com/SpaskeISO/cdbgo/cmd/cdb64@latest

# Or build from source using Makefile
make all              # Build for Linux, Windows, and macOS
make windows          # Build for Windows only
make help             # See all available targets
```

## Library Usage

### Opening and Reading from a CDB Database

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/SpaskeISO/cdbgo/cdb"
)

func main() {
    // Open a CDB database
    db, err := cdb.Open("data.cdb")
    if err != nil {
        log.Fatal(err)
    }
    defer cdb.Close(db)
    
    // Get a single value
    value, err := db.Get([]byte("mykey"))
    if err != nil {
        if err == cdb.ErrNotFound {
            fmt.Println("Key not found")
        } else {
            log.Fatal(err)
        }
    }
    fmt.Printf("Value: %s\n", value)
    
    // Get all values for a key (if there are duplicates)
    values, err := db.GetAll([]byte("mykey"))
    if err != nil {
        log.Fatal(err)
    }
    for i, v := range values {
        fmt.Printf("Value %d: %s\n", i+1, v)
    }
    
    // Get the nth value for a key (1-based)
    value, err = db.GetN([]byte("mykey"), 2)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Second value: %s\n", value)
}
```

### Iterating Through All Records

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/SpaskeISO/cdbgo/cdb"
)

func main() {
    db, err := cdb.Open("data.cdb")
    if err != nil {
        log.Fatal(err)
    }
    defer cdb.Close(db)
    
    it := cdb.NewIterator(db)
    for {
        key, value, ok := it.Next()
        if !ok {
            break
        }
        fmt.Printf("%s -> %s\n", key, value)
    }
    
    if err := it.Err(); err != nil {
        log.Fatal(err)
    }
}
```

### Creating a CDB Database (Not recommended for read-only use cases)

While the library includes writer functionality for internal use by the CLI tool, it's exposed for advanced use cases:

```go
package main

import (
    "log"
    
    "github.com/SpaskeISO/cdbgo/cdb"
)

func main() {
    // Create a new database
    writer, err := cdb.Create("data.cdb", "")
    if err != nil {
        log.Fatal(err)
    }
    
    // Add records
    if err := writer.PutString("key1", "value1"); err != nil {
        writer.Abort()
        log.Fatal(err)
    }
    
    if err := writer.PutString("key2", "value2"); err != nil {
        writer.Abort()
        log.Fatal(err)
    }
    
    // Finalize the database
    if err := writer.Finalize(); err != nil {
        log.Fatal(err)
    }
}
```

## CLI Tool Usage

The CLI tool provides complete compatibility with the original CDB tool.

### Query Mode

Find and print values for a given key:

```bash
# Query for a key
cdb -q database.cdb mykey

# Query with newline after value (useful for scripting)
cdb -q -m database.cdb mykey

# Get the nth record for a key (if there are duplicates)
cdb -q -n 2 database.cdb mykey
```

### Dump Mode

Output all records in the database:

```bash
# Dump in native CDB format
cdb -d database.cdb

# Dump in map format (one line per record)
cdb -d -m database.cdb

# Dump from stdin
cat database.cdb | cdb -d -
```

### List Mode

Output only keys (no values):

```bash
# List keys in native format
cdb -l database.cdb

# List keys in map format
cdb -l -m database.cdb
```

### Create Mode

Create a new CDB database from input:

```bash
# Create from stdin (native format)
cdb -c database.cdb < input.txt

# Create from stdin (map format)
cdb -c -m database.cdb < input.txt

# Create from one or more files
cdb -c database.cdb input1.txt input2.txt

# Create with custom temp file
cdb -c -t /tmp/mytemp.cdb database.cdb < input.txt

# Create in-place (no temp file)
cdb -c -t - database.cdb < input.txt

# Create with custom permissions
cdb -c -p 0600 database.cdb < input.txt

# Duplicate key handling options
cdb -c -w database.cdb < input.txt      # Warn about duplicates
cdb -c -e database.cdb < input.txt      # Error on duplicates
cdb -c -r database.cdb < input.txt      # Replace duplicates
cdb -c -u database.cdb < input.txt      # Unique (skip duplicates)
cdb -c -0 database.cdb < input.txt      # Zero-fill duplicates
```

### Stats Mode

Display database statistics:

```bash
cdb -s database.cdb
```

Output includes:
- Total number of records
- Key length statistics (min/avg/max)
- Value length statistics (min/avg/max)
- Hash table utilization
- Collision statistics
- Distance distribution (lookup efficiency)

### Help Mode

```bash
cdb -h
```

## File Formats

### Native CDB Format

Records are represented as:
```
+klen,vlen:key->val
```

For example:
```
+4,5:name->Alice
+3,2:age->25

```

The data ends with an empty line.

### Map Format

One line per record with key and value separated by whitespace:
```
name Alice
age 25
```

Lines starting with `#` are treated as comments and empty lines are ignored.

## CDB File Format

The CDB file format consists of:

1. **Header (2048 bytes)**: 256 hash table pointers, each containing:
   - Position in file (4 bytes, little-endian)
   - Number of slots (4 bytes, little-endian)

2. **Records**: Each record contains:
   - Key length (4 bytes, little-endian)
   - Value length (4 bytes, little-endian)
   - Key data
   - Value data

3. **Hash Tables**: Positioned after records, each slot contains:
   - Hash value (4 bytes, little-endian)
   - Record position (4 bytes, little-endian)

## Performance

CDB provides constant-time lookups using a simple but effective hashing scheme:
- DJB hash function for key hashing
- 256 separate hash tables to distribute load
- Linear probing for collision resolution
- No locks needed (readers don't block creators)

### Atomic Updates

One of CDB's key features is **atomic database replacement** on Unix-like systems:

1. **Writers create a new file** - The database is built in a temporary file
2. **Atomic rename** - `Finalize()` atomically renames the temp file to replace the old one
3. **Old readers continue** - Programs with the old database open keep working with the old data
4. **New readers get new data** - Programs opening after the rename see the new version
5. **No locks or downtime** - The switch is instantaneous and transparent

This means you can safely update a CDB database while it's being read by other processes. The old file's data remains accessible until all readers close it.

**Example:**
```go
// Reader keeps working through the update
db, _ := cdb.Open("data.cdb")
defer cdb.Close(db)

// Another process updates the database
// writer.Finalize() renames atomically

// This reader still sees old data until Close()
value, _ := db.Get([]byte("key"))
```

## CDB64 - 64-bit Version

The `cdb64` package provides a 64-bit implementation of CDB for large-scale databases.

### Key Differences from 32-bit CDB

- **File Format**: 64-bit offsets and lengths throughout
- **Header**: 16 KB (1024 tables × 16 bytes per table pointer)
- **Hash Tables**: 1024 tables (vs 256 in 32-bit)
- **Hash Function**: 64-bit DJB hash for better collision resistance
- **Capacity**: Supports exabyte-scale files vs 4GB limit in 32-bit
- **No Compatibility**: Cannot read/write 32-bit CDB files

### CDB64 Library Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/SpaskeISO/cdbgo/cdb/cdb64"
)

func main() {
    // Create a database
    writer, err := cdb64.Create("data.cdb64", "")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := writer.PutString("key", "value"); err != nil {
        writer.Abort()
        log.Fatal(err)
    }
    
    if err := writer.Finalize(); err != nil {
        log.Fatal(err)
    }
    
    // Open and read
    db, err := cdb64.Open("data.cdb64")
    if err != nil {
        log.Fatal(err)
    }
    defer cdb64.Close(db)
    
    value, err := db.Get([]byte("key"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Value: %s\n", value)
    
    // Iterate through all records
    it := cdb64.NewIterator(db)
    for {
        key, value, ok := it.Next()
        if !ok {
            break
        }
        fmt.Printf("%s -> %s\n", key, value)
    }
}
```

### CDB64 CLI Tool

The `cdb64` CLI tool has the same interface as `cdb`:

```bash
# Create a database
echo "+3,5:key->value" | cdb64 -c data.cdb64

# Query
cdb64 -q data.cdb64 key

# Dump
cdb64 -d data.cdb64

# List keys
cdb64 -l data.cdb64

# Statistics
cdb64 -s data.cdb64
```

All flags and modes work identically to the 32-bit version.

### When to Use CDB64

Use **cdb64** when:
- Your database will exceed 4GB
- You need more than 256 hash tables for better distribution
- You're working with billions or trillions of records
- You want better collision resistance with 64-bit hashing

Use **cdb** (32-bit) when:
- You need compatibility with existing CDB tools
- Your database is under 4GB
- You want minimal overhead (2KB header vs 16KB)

## Testing

Run the test suites:

```bash
# Test 32-bit CDB
go test ./cdb/... -v

# Test 64-bit CDB
go test ./cdb/cdb64/... -v

# Test all
go test ./... -v
```

## Benchmarks

**Quick Summary:**
- **Read Performance**: CDB64 matches CDB32 (within 5%)
- **Write Performance**: CDB64 is ~3% slower (larger data structures)
- **Collision Resistance**: CDB64 has 35% fewer collisions
- **Latency**: Both achieve sub-microsecond lookups
- **Space**: CDB64 uses ~2x disk space

Run benchmarks yourself:

```bash
# Benchmark 32-bit CDB
go test -bench=. -benchtime=1s ./cdb/ -run=^$

# Benchmark 64-bit CDB
go test -bench=. -benchtime=1s ./cdb/cdb64/ -run=^$

# Collision analysis
go test -v -run=TestCollisionAnalysis ./cdb/
go test -v -run=TestCollisionAnalysis ./cdb/cdb64/
```

## References

- [Original CDB by Dan Bernstein](http://cr.yp.to/cdb.html)
- [tinycdb by Michael Tokarev](http://www.corpit.ru/mjt/tinycdb.html)

## Contributing

This implementation follows the CDB format specification and maintains compatibility with other CDB implementations. Bug reports and contributions are welcome.
