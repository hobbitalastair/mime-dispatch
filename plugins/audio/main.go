package main

import (
	"fmt"
	"metadata/pkg/pluginio"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
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

	metadata, err := tag.ReadFrom(f)
	if err != nil {
		return err
	}

	result := make(map[string][]string)

	if title := metadata.Title(); title != "" {
		result["title"] = []string{title}
	}
	if album := metadata.Album(); album != "" {
		result["album"] = []string{album}
	}
	if artist := metadata.Artist(); artist != "" {
		result["artist"] = []string{artist}
	}
	if albumArtist := metadata.AlbumArtist(); albumArtist != "" {
		result["album_artist"] = []string{albumArtist}
	}
	if composer := metadata.Composer(); composer != "" {
		result["composer"] = []string{composer}
	}
	if genre := metadata.Genre(); genre != "" {
		result["genre"] = []string{genre}
	}
	if year := metadata.Year(); year != 0 {
		result["year"] = []string{fmt.Sprintf("%d", year)}
	}
	if comment := metadata.Comment(); comment != "" {
		result["comment"] = []string{comment}
	}

	if len(result) > 0 {
		output, err := pluginio.SerializeMetadata(result)
		if err != nil {
			return err
		}
		fmt.Print(output)
	}

	return nil
}
