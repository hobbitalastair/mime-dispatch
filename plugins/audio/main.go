package main

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: metadata-audio <command> <file>")
		os.Exit(1)
	}

	command := os.Args[1]
	filePath := os.Args[2]

	switch command {
	case "list":
		err := listMetadata(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "set":
		fmt.Fprintln(os.Stderr, "error: this plugin is read-only, use xattr for writing")
		os.Exit(1)
	case "delete":
		fmt.Fprintln(os.Stderr, "error: this plugin is read-only, use xattr for deleting")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
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
