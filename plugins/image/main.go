package main

import (
	"fmt"
	"metadata/pkg/plugincli"
	"metadata/pkg/pluginio"
	"os"
	"time"

	"github.com/evanoberholster/imagemeta"
)

func main() {
	caps := plugincli.Capabilities{
		Mimetypes: []string{"image/jpeg"},
		Commands: map[string]func() error{
			"metadata-list": func() error {
				args := plugincli.ParseArgs("metadata-list", usage)
				if len(args) != 1 {
					return plugincli.ErrUsage
				}
				return listMetadata(args[0])
			},
		},
	}
	plugincli.Run(caps, usage)
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
		metadata["datetime"] = []string{formatISO8601DateTime(date)}
	}

	lat := exif.GPS.Latitude()
	lon := exif.GPS.Longitude()
	if lat != 0 || lon != 0 {
		metadata["location"] = []string{fmt.Sprintf("%v,%v", lat, lon)}
	}

	if exif.ImageDescription != "" {
		metadata["comment"] = []string{exif.ImageDescription}
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

func formatISO8601DateTime(t time.Time) string {
	_, offset := t.Zone()
	if offset != 0 {
		return t.Format("2006-01-02T15:04:05-07:00")
	}
	return t.Format("2006-01-02T15:04:05")
}
