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
		fmt.Fprintln(os.Stderr, "Usage: metadata <command> <file> [args...]")
		fmt.Fprintln(os.Stderr, "Commands: list, add, delete")
		pflag.PrintDefaults()
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
		err = handleList(filePath, opts)
	case "add":
		if len(remainingArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: metadata add <file> <key> <value>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		value := remainingArgs[1]
		err = handleAdd(filePath, key, value, opts)
	case "delete":
		if len(remainingArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: metadata delete <file> <key> <value>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		value := remainingArgs[1]
		err = handleDelete(filePath, key, value, opts)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleList(filePath string, opts lib.Options) error {
	metadata, err := lib.GetMetadata(filePath, opts)
	if err != nil {
		return err
	}
	fmt.Print(metadata.ToYAML())
	return nil
}

func handleAdd(filePath, key, value string, opts lib.Options) error {
	return lib.AddMetadata(filePath, key, value, opts)
}

func handleDelete(filePath, key, value string, opts lib.Options) error {
	return lib.DeleteMetadata(filePath, key, value, opts)
}
