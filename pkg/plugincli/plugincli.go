package plugincli

import (
	"errors"
	"fmt"
	"metadata/pkg/pluginio"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/pflag"
)

// ErrUsage is returned by command handlers to indicate bad arguments.
// Run prints the usage function and exits with code 1 when it sees this error.
var ErrUsage = errors.New("bad arguments")

// Capabilities describes which MIME types a plugin handles
// and which commands it supports. Commands maps symlink names
// (e.g. "metadata-list") to their handler functions.
// This is the single source of truth — serialization derives
// the command name list from the map keys.
type Capabilities struct {
	Mimetypes []string
	Commands  map[string]func() error
}

// Run is the main entry point for plugins. It dispatches to the
// matching command handler based on argv[0], or handles --capabilities
// when the binary is called directly (not via a command symlink).
func Run(caps Capabilities, usage func()) {
	command := filepath.Base(os.Args[0])

	if handler, ok := caps.Commands[command]; ok {
		if handler == nil {
			fmt.Fprintf(os.Stderr, "nil handler for command %q\n", command)
			os.Exit(1)
		}
		if err := handler(); err != nil {
			if errors.Is(err, ErrUsage) {
				usage()
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// Called directly — only --capabilities is valid
	fs := pflag.NewFlagSet("plugin", pflag.ContinueOnError)
	showCaps := fs.Bool("capabilities", false, "Print supported MIME types and commands")
	fs.Usage = usage
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *showCaps {
		names := make([]string, 0, len(caps.Commands))
		for name := range caps.Commands {
			names = append(names, name)
		}
		sort.Strings(names)

		output, err := pluginio.SerializeCapabilities(caps.Mimetypes, names)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(output)
		return
	}

	usage()
	os.Exit(1)
}

// ParseArgs parses os.Args[1:] with an empty flag set and returns
// the positional arguments. If parsing fails, it prints usage and exits.
func ParseArgs(command string, usage func()) []string {
	fs := pflag.NewFlagSet(command, pflag.ContinueOnError)
	fs.Usage = usage
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}
	return fs.Args()
}
