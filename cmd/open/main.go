package main

import (
	"errors"
	"fmt"
	"mime-dispatch/lib"
	"os"
	"os/exec"
)

// open looks up and executes a MIME-type-specific open handler for each
// given file. Handlers inherit stdin/stdout/stderr so they can be
// interactive (e.g. launching an editor or viewer). Unlike metadata
// plugins, open handlers are NOT sandboxed — they need full system
// access to open files in arbitrary applications.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: open <file>...")
		os.Exit(1)
	}

	var failed bool
	for _, filePath := range os.Args[1:] {
		if err := openFile(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", filePath, err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func openFile(filePath string) error {
	mimeType, err := lib.GetMimeType(filePath)
	if err != nil {
		return err
	}

	handlerPath, err := lib.FindPluginForCommand(mimeType, lib.PluginOpen)
	if err != nil {
		var noPlugin lib.ErrNoPluginFound
		if errors.As(err, &noPlugin) {
			return fmt.Errorf("no open handler for %s", mimeType)
		}
		return err
	}

	if err := lib.RunOpenHandler(handlerPath, filePath); err != nil {
		// The handler's stderr was already printed to the terminal.
		// Suppress the raw "exit status N" message from exec.ExitError.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("handler exited with status %d", exitErr.ExitCode())
		}
		return err
	}

	return nil
}
