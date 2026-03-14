package main

import (
	"fmt"
	"os"

	"metadata/lib"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: metadata <command> <file> [args...]")
		fmt.Fprintln(os.Stderr, "Commands: list, set, delete")
		os.Exit(1)
	}

	opts := lib.Options{
		XattrOnly: contains(os.Args, "--xattr-only"),
		FileOnly:  contains(os.Args, "--file-only"),
	}

	args := os.Args[1:]

	var command string
	var filePath string
	var remainingArgs []string

	for i, arg := range args {
		if len(arg) > 0 && arg[0] == '-' {
			continue
		}
		command = arg
		if i+1 < len(args) {
			allRemaining := args[i+1:]
			filePath = findFileArg(allRemaining)
			nonFlagArgs := filterFlags(allRemaining)
			if filePath != "" {
				remainingArgs = removeFileArg(nonFlagArgs, filePath)
			}
		}
		break
	}

	if command == "" {
		fmt.Fprintln(os.Stderr, "Usage: metadata <command> <file> [args...]")
		os.Exit(1)
	}

	var err error

	switch command {
	case "list":
		if filePath == "" {
			fmt.Fprintln(os.Stderr, "Usage: metadata list <file>")
			os.Exit(1)
		}
		err = handleList(filePath, opts)
	case "set":
		if filePath == "" || len(remainingArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: metadata set <file> <key> <value>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		value := remainingArgs[1]
		err = handleSet(filePath, key, value, opts)
	case "delete":
		if filePath == "" || len(remainingArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: metadata delete <file> <key>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		err = handleDelete(filePath, key, opts)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func findFileArg(args []string) string {
	for _, arg := range args {
		if len(arg) > 0 && arg[0] != '-' {
			return arg
		}
	}
	return ""
}

func filterFlags(args []string) []string {
	var result []string
	for _, arg := range args {
		if len(arg) > 0 && arg[0] != '-' {
			result = append(result, arg)
		}
	}
	return result
}

func removeFileArg(args []string, filePath string) []string {
	var result []string
	for _, arg := range args {
		if arg != filePath {
			result = append(result, arg)
		}
	}
	return result
}

func handleList(filePath string, opts lib.Options) error {
	metadata, err := lib.GetMetadata(filePath, opts)
	if err != nil {
		return err
	}
	fmt.Print(metadata.ToYAML())
	return nil
}

func handleSet(filePath, key, value string, opts lib.Options) error {
	return lib.SetMetadata(filePath, key, value, opts)
}

func handleDelete(filePath, key string, opts lib.Options) error {
	return lib.DeleteMetadata(filePath, key, opts)
}
