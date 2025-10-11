package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

const version = "0.80"

type config struct {
	// Mode flags
	query   bool
	dump    bool
	list    bool
	create  bool
	stats   bool
	help    bool
	
	// Common flags
	mapFormat bool
	
	// Query flags
	recno int
	
	// Create flags
	tempFile string
	perms    string
	warn     bool
	errDup   bool
	replace  bool
	unique   bool
	zeroFill bool
	
	// Positional arguments
	dbfile string
	key    string
	infiles []string
}

func main() {
	cfg := parseFlags()
	
	if cfg.help {
		printHelp()
		os.Exit(0)
	}
	
	var err error
	exitCode := 0
	
	if cfg.query {
		err = queryMode(cfg)
		if err != nil {
			exitCode = 1
		}
	} else if cfg.dump {
		err = dumpMode(cfg)
	} else if cfg.list {
		err = listMode(cfg)
	} else if cfg.create {
		err = createMode(cfg)
	} else if cfg.stats {
		err = statsMode(cfg)
	} else {
		printHelp()
		os.Exit(1)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "cdb: %v\n", err)
		os.Exit(exitCode)
	}
}

func parseFlags() *config {
	cfg := &config{
		recno: -1,
	}
	
	// Mode flags
	flag.BoolVar(&cfg.query, "q", false, "query mode")
	flag.BoolVar(&cfg.dump, "d", false, "dump mode")
	flag.BoolVar(&cfg.list, "l", false, "list mode")
	flag.BoolVar(&cfg.create, "c", false, "create mode")
	flag.BoolVar(&cfg.stats, "s", false, "statistics mode")
	flag.BoolVar(&cfg.help, "h", false, "print help")
	
	// Common flags
	flag.BoolVar(&cfg.mapFormat, "m", false, "use map format (not native cdb format)")
	
	// Query flags
	flag.IntVar(&cfg.recno, "n", -1, "find and print nth record")
	
	// Create flags
	flag.StringVar(&cfg.tempFile, "t", "", "temporary file for creation")
	flag.StringVar(&cfg.perms, "p", "", "file permissions (octal)")
	flag.BoolVar(&cfg.warn, "w", false, "warn about duplicate keys")
	flag.BoolVar(&cfg.errDup, "e", false, "error on duplicate keys")
	flag.BoolVar(&cfg.replace, "r", false, "replace duplicate keys")
	flag.BoolVar(&cfg.unique, "u", false, "do not add duplicate keys")
	flag.BoolVar(&cfg.zeroFill, "0", false, "zero-fill duplicate records")
	
	flag.Parse()
	
	// Parse positional arguments
	args := flag.Args()
	
	if cfg.query {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "cdb: query mode requires dbfile and key\n")
			os.Exit(1)
		}
		cfg.dbfile = args[0]
		cfg.key = args[1]
	} else if cfg.dump || cfg.list || cfg.stats {
		if len(args) > 0 {
			cfg.dbfile = args[0]
		} else {
			cfg.dbfile = "-"
		}
	} else if cfg.create {
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "cdb: create mode requires dbfile\n")
			os.Exit(1)
		}
		cfg.dbfile = args[0]
		if len(args) > 1 {
			cfg.infiles = args[1:]
		}
	}
	
	return cfg
}

func printHelp() {
	fmt.Printf("cdb: Constant DataBase (CDB) tool version %s. Usage is:\n", version)
	fmt.Println(" query:  cdb -q [-m] [-n recno|-a] cdbfile key")
	fmt.Println(" dump:   cdb -d [-m] [cdbfile|-]")
	fmt.Println(" list:   cdb -l [-m] [cdbfile|-]")
	fmt.Println(" create: cdb -c [-m] [-wrue0] [-t tempfile|-] [-p perms] cdbfile [infile...]")
	fmt.Println(" stats:  cdb -s [cdbfile|-]")
	fmt.Println(" help:   cdb -h")
	fmt.Println()
}

func parsePerms(permsStr string) (os.FileMode, error) {
	if permsStr == "" {
		return 0666, nil
	}
	
	// Parse octal
	perms, err := strconv.ParseUint(permsStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permissions: %s", permsStr)
	}
	
	return os.FileMode(perms), nil
}

