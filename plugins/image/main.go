package main

import (
	"fmt"
	"metadata/pkg/pluginio"
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
	if command != "metadata-list" || len(args) != 1 {
		usage()
		os.Exit(1)
	}

	if err := listMetadata(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: metadata-list <file>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "This is a read-only plugin. Create a symlink named 'metadata-list' pointing to this binary.")
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

	metadata := make(map[string][]string)

	date := exif.DateTimeOriginal()
	if date.IsZero() {
		date = exif.ModifyDate()
	}
	if date.IsZero() {
		date = exif.CreateDate()
	}
	if !date.IsZero() {
		metadata["date"] = []string{date.Format("2006:01:02 15:04:05")}
	}

	lat := exif.GPS.Latitude()
	lon := exif.GPS.Longitude()
	if lat != 0 || lon != 0 {
		metadata["location"] = []string{fmt.Sprintf("%v,%v", lat, lon)}
	}

	if exif.ImageDescription != "" {
		metadata["caption"] = []string{exif.ImageDescription}
	}

	if len(metadata) > 0 {
		data, err := pluginio.SerializeMetadata(metadata)
		if err != nil {
			return err
		}
		fmt.Print(data)
	}

	return nil
}
