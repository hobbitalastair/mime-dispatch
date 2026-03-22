package main

import (
	"bytes"
	"fmt"
	"mime-dispatch/pkg/pluginio"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

func main() {
	userLevel := pflag.Bool("user", false, "Install to $XDG_CONFIG_HOME/mimetype/")
	systemLevel := pflag.Bool("system", false, "Install to /etc/mimetype/")
	vendorLevel := pflag.Bool("vendor", false, "Install to /usr/lib/mimetype/")
	mimetypes := pflag.StringArray("mimetype", nil, "Override MIME types (can be specified multiple times)")
	uninstall := pflag.Bool("uninstall", false, "Remove symlinks instead of creating them")

	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: mime-dispatch-install [--user|--system|--vendor] [--mimetype <type>]... [--uninstall] <binary-path>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Install or uninstall MIME type handler symlinks for a plugin binary.")
		fmt.Fprintln(os.Stderr, "The plugin must support --capabilities to declare its MIME types and commands.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Levels:")
		fmt.Fprintln(os.Stderr, "  --user     $XDG_CONFIG_HOME/mimetype/  (current user)")
		fmt.Fprintln(os.Stderr, "  --system   /etc/mimetype/              (system administrator)")
		fmt.Fprintln(os.Stderr, "  --vendor   /usr/lib/mimetype/          (distribution packages)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Environment:")
		fmt.Fprintln(os.Stderr, "  DESTDIR    Prefix for all filesystem operations")
		fmt.Fprintln(os.Stderr, "")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	if pflag.NArg() != 1 {
		pflag.Usage()
		os.Exit(1)
	}

	binaryPath := pflag.Arg(0)

	// Validate exactly one level is selected
	levelCount := boolCount(*userLevel, *systemLevel, *vendorLevel)
	if levelCount != 1 {
		fmt.Fprintln(os.Stderr, "Error: exactly one of --user, --system, or --vendor must be specified")
		os.Exit(1)
	}

	mimetypeBase, err := mimetypeBaseDir(*userLevel, *systemLevel, *vendorLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Get capabilities from the plugin
	caps, err := queryCapabilities(binaryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error querying capabilities:", err)
		os.Exit(1)
	}

	// Determine MIME types to install for
	targetMimetypes := caps.Mimetypes
	if len(*mimetypes) > 0 {
		targetMimetypes = *mimetypes
	}

	destdir := os.Getenv("DESTDIR")

	// Compute the logical binary path (strip DESTDIR prefix)
	logicalBinaryPath := binaryPath
	if destdir != "" {
		stripped, err := stripPrefix(binaryPath, destdir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		logicalBinaryPath = stripped
	}

	if *uninstall {
		err = doUninstall(destdir, mimetypeBase, logicalBinaryPath, targetMimetypes, caps.Commands)
	} else {
		err = doInstall(destdir, mimetypeBase, logicalBinaryPath, targetMimetypes, caps.Commands)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func boolCount(values ...bool) int {
	count := 0
	for _, v := range values {
		if v {
			count++
		}
	}
	return count
}

func mimetypeBaseDir(user, system, vendor bool) (string, error) {
	switch {
	case user:
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			usr, err := osuser.Current()
			if err != nil {
				return "", fmt.Errorf("cannot determine home directory: %w", err)
			}
			configDir = filepath.Join(usr.HomeDir, ".config")
		}
		return filepath.Join(configDir, "mimetype"), nil
	case system:
		return "/etc/mimetype", nil
	case vendor:
		return "/usr/lib/mimetype", nil
	default:
		return "", fmt.Errorf("no level specified")
	}
}

func queryCapabilities(binaryPath string) (pluginio.DeserializedCapabilities, error) {
	cmd := exec.Command(binaryPath, "--capabilities")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return pluginio.DeserializedCapabilities{}, fmt.Errorf("%s: %s", binaryPath, msg)
		}
		return pluginio.DeserializedCapabilities{}, fmt.Errorf("%s: %w", binaryPath, err)
	}

	return pluginio.DeserializeCapabilities(stdout.String())
}

func stripPrefix(path, prefix string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absPrefix, err := filepath.Abs(prefix)
	if err != nil {
		return "", err
	}

	// Ensure the prefix matches at a path boundary
	if absPath != absPrefix && !strings.HasPrefix(absPath, absPrefix+"/") {
		return "", fmt.Errorf("binary path %q is not under DESTDIR %q", path, prefix)
	}

	result := strings.TrimPrefix(absPath, absPrefix)
	if result == "" {
		return "/", nil
	}
	return result, nil
}

func doInstall(destdir, mimetypeBase, logicalBinaryPath string, mimetypes, commands []string) error {
	if err := validateNames(mimetypes, commands); err != nil {
		return err
	}

	for _, mimetype := range mimetypes {
		dir := filepath.Join(destdir, mimetypeBase, mimetype)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}

		for _, command := range commands {
			linkPath := filepath.Join(dir, command)

			// Remove existing symlink if present
			if _, err := os.Lstat(linkPath); err == nil {
				if err := os.Remove(linkPath); err != nil {
					return fmt.Errorf("remove existing %s: %w", linkPath, err)
				}
			}

			if err := os.Symlink(logicalBinaryPath, linkPath); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", linkPath, logicalBinaryPath, err)
			}
		}
	}
	return nil
}

func doUninstall(destdir, mimetypeBase, logicalBinaryPath string, mimetypes, commands []string) error {
	if err := validateNames(mimetypes, commands); err != nil {
		return err
	}

	stopAt := filepath.Join(destdir, mimetypeBase)

	for _, mimetype := range mimetypes {
		dir := filepath.Join(destdir, mimetypeBase, mimetype)

		for _, command := range commands {
			linkPath := filepath.Join(dir, command)

			// Only remove if it's a symlink pointing to our binary
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue // not a symlink or doesn't exist, skip
			}
			if target != logicalBinaryPath {
				continue // symlink points elsewhere, don't touch it
			}

			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("remove %s: %w", linkPath, err)
			}
		}

		// Remove directory if empty, but never above the mimetype base
		removeEmptyDirs(dir, stopAt)
	}
	return nil
}

// validateNames checks that command names and MIME types don't contain
// path traversal components.
func validateNames(mimetypes, commands []string) error {
	for _, cmd := range commands {
		if filepath.Base(cmd) != cmd {
			return fmt.Errorf("invalid command name %q: must not contain path separators", cmd)
		}
		if cmd == ".." || cmd == "." {
			return fmt.Errorf("invalid command name %q", cmd)
		}
	}
	for _, mt := range mimetypes {
		if strings.Contains(mt, "..") {
			return fmt.Errorf("invalid MIME type %q: must not contain '..'", mt)
		}
	}
	return nil
}

// removeEmptyDirs removes the directory and its empty parents,
// stopping at (and never removing) the stopAt directory.
func removeEmptyDirs(dir, stopAt string) {
	for dir != stopAt {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			break
		}
		dir = filepath.Dir(dir)
	}
}
