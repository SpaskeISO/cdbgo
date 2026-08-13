package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/SpaskeISO/cdbgo/cdb64"
)

type config struct {
	query  bool
	dump   bool
	list   bool
	create bool
	stats  bool
	help   bool

	mapFormat  bool
	allRecords bool

	recno int

	tempFile string
	perms    string
	warn     bool
	errDup   bool
	replace  bool
	unique   bool
	zeroFill bool

	dbfile  string
	key     string
	infiles []string
}

func main() {
	cfg := parseFlags()

	if cfg.help {
		printHelp()
		return
	}

	var err error

	switch {
	case cfg.query:
		err = queryMode(cfg)
	case cfg.dump:
		err = dumpMode(cfg)
	case cfg.list:
		err = listMode(cfg)
	case cfg.create:
		err = createMode(cfg)
	case cfg.stats:
		err = statsMode(cfg)
	default:
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "cdb64: %v\n", err)
		os.Exit(exitStatus(err))
	}
}

func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, cdb64.ErrNotFound) {
		return 100
	}
	return 1
}

func parseFlags() *config {
	cfg := &config{
		recno: -1,
	}

	flag.BoolVar(&cfg.query, "q", false, "query mode")
	flag.BoolVar(&cfg.dump, "d", false, "dump mode")
	flag.BoolVar(&cfg.list, "l", false, "list mode")
	flag.BoolVar(&cfg.create, "c", false, "create mode")
	flag.BoolVar(&cfg.stats, "s", false, "statistics mode")
	flag.BoolVar(&cfg.help, "h", false, "print help")

	flag.BoolVar(&cfg.mapFormat, "m", false, "use map format (not native cdb format)")
	flag.BoolVar(&cfg.allRecords, "a", false, "find and print all records")

	flag.IntVar(&cfg.recno, "n", -1, "find and print nth record")

	flag.StringVar(&cfg.tempFile, "t", "", "temporary file for creation")
	flag.StringVar(&cfg.perms, "p", "", "file permissions (octal)")
	flag.BoolVar(&cfg.warn, "w", false, "warn about duplicate keys")
	flag.BoolVar(&cfg.errDup, "e", false, "error on duplicate keys")
	flag.BoolVar(&cfg.replace, "r", false, "replace duplicate keys")
	flag.BoolVar(&cfg.unique, "u", false, "do not add duplicate keys")
	flag.BoolVar(&cfg.zeroFill, "0", false, "zero-fill duplicate records")

	flag.Parse()

	args := flag.Args()

	if cfg.query {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "cdb64: query mode requires dbfile and key\n")
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
			fmt.Fprintf(os.Stderr, "cdb64: create mode requires dbfile\n")
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
	fmt.Println("cdb64: 64-bit Constant DataBase (CDB) tool. Usage is:")
	fmt.Println(" query:  cdb64 -q [-m] [-n recno|-a] cdbfile key")
	fmt.Println(" dump:   cdb64 -d [-m] [cdbfile|-]")
	fmt.Println(" list:   cdb64 -l [-m] [cdbfile|-]")
	fmt.Println(" create: cdb64 -c [-m] [-wrue0] [-t tempfile|-] [-p perms] cdbfile [infile...]")
	fmt.Println(" stats:  cdb64 -s [cdbfile|-]")
	fmt.Println(" help:   cdb64 -h")
	fmt.Println()
}

func parsePerms(permsStr string) (os.FileMode, error) {
	if permsStr == "" {
		return 0666, nil
	}

	perms, err := strconv.ParseUint(permsStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permissions: %s", permsStr)
	}

	return os.FileMode(perms), nil
}
