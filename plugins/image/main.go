package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/evanoberholster/imagemeta"
	"github.com/spf13/pflag"
)

func main() {
	command := filepath.Base(os.Args[0])

	flagSet := pflag.NewFlagSet(command, pflag.ContinueOnError)
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(1)
	}

	args := flagSet.Args()
	if command != "list" || len(args) != 1 {
		usage()
		os.Exit(1)
	}

	if err := listMetadata(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: list <file>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "This is a read-only plugin. Create a symlink named 'list' pointing to this binary.")
}

func listMetadata(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	exif, err := imagemeta.Decode(f)
	if err != nil {
		return err
	}

	// Extract date from EXIF - try DateTimeOriginal first, then ModifyDate, then CreateDate
	date := exif.DateTimeOriginal()
	if date.IsZero() {
		date = exif.ModifyDate()
	}
	if date.IsZero() {
		date = exif.CreateDate()
	}
	if !date.IsZero() {
		fmt.Printf("date: %s\n", date.Format("2006:01:02 15:04:05"))
	}

	// Extract location from EXIF GPS
	lat := exif.GPS.Latitude()
	lon := exif.GPS.Longitude()
	if lat != 0 || lon != 0 {
		fmt.Printf("location: %v,%v\n", lat, lon)
	}

	// Extract caption from EXIF ImageDescription
	if exif.ImageDescription != "" {
		fmt.Printf("caption: %s\n", exif.ImageDescription)
	}

	return nil
}
