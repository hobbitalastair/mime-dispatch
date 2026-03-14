package main

import (
	"fmt"
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

	metadata, err := tag.ReadFrom(f)
	if err != nil {
		return err
	}

	if title := metadata.Title(); title != "" {
		fmt.Printf("title: %s\n", title)
	}
	if album := metadata.Album(); album != "" {
		fmt.Printf("album: %s\n", album)
	}
	if artist := metadata.Artist(); artist != "" {
		fmt.Printf("artist: %s\n", artist)
	}
	if albumArtist := metadata.AlbumArtist(); albumArtist != "" {
		fmt.Printf("album_artist: %s\n", albumArtist)
	}
	if composer := metadata.Composer(); composer != "" {
		fmt.Printf("composer: %s\n", composer)
	}
	if genre := metadata.Genre(); genre != "" {
		fmt.Printf("genre: %s\n", genre)
	}
	if year := metadata.Year(); year != 0 {
		fmt.Printf("year: %d\n", year)
	}
	if comment := metadata.Comment(); comment != "" {
		fmt.Printf("comment: %s\n", comment)
	}

	return nil
}
