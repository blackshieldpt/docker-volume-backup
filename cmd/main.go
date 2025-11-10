package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"docker-volume-backup/internal/operation"
)

var (
	progress  bool
	compress  string
	overwrite bool
)

func usage() {
	fmt.Println(`Usage:
  docker-volume-backup backup [--progress] [--compress gz|zstd|none] <volume> <dest>
  docker-volume-backup restore [--progress] [--overwrite] <src> <volume>

Flags:
  --progress          Show progress bar during backup/restore
  --compress <type>   Compression type: none|gz|zstd (default: gz) [backup only]
  --overwrite         Clear existing volume before restore [restore only]`)
	os.Exit(1)
}

// checkErr logs a fatal error message and exits the program if the provided error is non-nil.
func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalf("ERROR: %s: %v", msg, err)
	}
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.BoolVar(&progress, "progress", false, "show progress bar")
	fs.StringVar(&compress, "compress", "gz", "compression type: none|gz|zstd")
	fs.BoolVar(&overwrite, "overwrite", false, "clear existing volume before restore")

	// parse flags starting from second arg (after command)
	fs.Parse(os.Args[2:])
	args := fs.Args()

	switch cmd {
	case "backup":
		if len(args) != 2 {
			usage()
		}
		volume, dest := args[0], args[1]
		op, err := operation.NewBackup(volume, compress, progress)
		checkErr(err, "Backup failed")

		if strings.HasPrefix(dest, "s3://") {
			checkErr(op.BackupToS3(dest), "Backup failed")
		} else {
			checkErr(op.BackupToFile(dest), "Backup failed")
		}

	case "restore":
		if len(args) != 2 {
			usage()
		}
		src, volume := args[0], args[1]
		op, err := operation.NewRestore(volume, progress)
		checkErr(err, "Restore failed")

		if strings.HasPrefix(src, "s3://") {
			err := op.RestoreFromS3(src, overwrite)
			checkErr(err, "Restore failed")
		} else {
			err := op.RestoreFromFile(src, overwrite)
			checkErr(err, "Restore failed")
		}
	default:
		usage()
	}
}
