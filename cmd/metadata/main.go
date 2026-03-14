package main

import (
	"fmt"
	"os"

	"metadata/lib"

	"github.com/spf13/pflag"
)

func main() {
	xattrOnly := pflag.BoolP("xattr-only", "x", false, "Only access extended attributes")
	fileOnly := pflag.BoolP("file-only", "f", false, "Only access file contents")

	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: metadata <command> <file> [args...]")
		fmt.Fprintln(os.Stderr, "Commands: list, add, delete")
		fmt.Fprintln(os.Stderr, "")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	args := pflag.Args()

	if len(args) < 2 {
		pflag.Usage()
		os.Exit(1)
	}

	opts := lib.Options{
		XattrOnly: *xattrOnly,
		FileOnly:  *fileOnly,
	}

	command := args[0]
	filePath := args[1]
	remainingArgs := args[2:]

	// Apply Landlock sandbox restrictions
	if err := lib.SetupSandbox(filePath); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to setup sandbox:", err)
		os.Exit(1)
	}

	var err error

	switch command {
	case "list":
		metadata, e := lib.GetMetadata(filePath, opts)
		if e != nil {
			err = e
		} else {
			fmt.Print(metadata.ToYAML())
		}
	case "add", "delete":
		if len(remainingArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: metadata %s <file> <key> <value>\n", command)
			os.Exit(1)
		}
		key, value := remainingArgs[0], remainingArgs[1]
		if command == "add" {
			err = lib.AddMetadata(filePath, key, value, opts)
		} else {
			err = lib.DeleteMetadata(filePath, key, value, opts)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
