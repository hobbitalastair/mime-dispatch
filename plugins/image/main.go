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
	if !isValidCommand(command) {
		usage()
		os.Exit(1)
	}

	flagSet := pflag.NewFlagSet(command, pflag.ContinueOnError)
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(1)
	}

	args := flagSet.Args()
	switch command {
	case "list":
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}
		if err := listMetadata(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "set":
		fmt.Fprintln(os.Stderr, "error: this plugin is read-only, use xattr for writing")
		os.Exit(1)
	case "delete":
		fmt.Fprintln(os.Stderr, "error: this plugin is read-only, use xattr for deleting")
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage (run via a command-specific symlink):")
	fmt.Fprintln(os.Stderr, "  list <file>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Available commands are invoked by creating symlinks named list/set/delete pointing to this binary.")
}

func isValidCommand(command string) bool {
	switch command {
	case "list", "set", "delete":
		return true
	default:
		return false
	}
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

	// Extract EXIF metadata
	hasMetadata := false

	// Extract date from EXIF - try DateTimeOriginal first, then ModifyDate, then CreateDate
	date := exif.DateTimeOriginal()
	if date.IsZero() {
		date = exif.ModifyDate()
	}
	if date.IsZero() {
		date = exif.CreateDate()
	}

	if !date.IsZero() {
		// Format as YYYY:MM:DD HH:MM:SS (standard EXIF format)
		fmt.Printf("date: %s\n", date.Format("2006:01:02 15:04:05"))
		hasMetadata = true
	}

	// Extract location from EXIF GPS
	lat := exif.GPS.Latitude()
	lon := exif.GPS.Longitude()
	if lat != 0 || lon != 0 {
		// Format as compact CSV: latitude,longitude
		fmt.Printf("location: %v,%v\n", lat, lon)
		hasMetadata = true
	}

	// Extract caption from EXIF ImageDescription
	if exif.ImageDescription != "" {
		fmt.Printf("caption: %s\n", exif.ImageDescription)
		hasMetadata = true
	}

	// Note: imagemeta library focuses on EXIF and does not provide direct access to
	// XMP or IPTC keywords. To support these, additional libraries would be needed.
	// For now, we only extract EXIF data.

	// If no metadata was found, output nothing (silent, exit 0)
	if !hasMetadata {
		return nil
	}

	return nil
}
