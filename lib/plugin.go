package lib

import (
	"bytes"
	"mime-dispatch/pkg/pluginio"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

type PluginCommand int

const (
	PluginList PluginCommand = iota
	PluginAdd
	PluginDelete
	PluginOpen
)

func (c PluginCommand) String() string {
	switch c {
	case PluginList:
		return "metadata-list"
	case PluginAdd:
		return "metadata-add"
	case PluginDelete:
		return "metadata-delete"
	case PluginOpen:
		return "open"
	default:
		return "unknown"
	}
}

func PluginSearchPaths() []string {
	userConfigDir := os.Getenv("XDG_CONFIG_HOME")
	if userConfigDir == "" {
		usr, err := user.Current()
		if err == nil {
			userConfigDir = filepath.Join(usr.HomeDir, ".config")
		}
	}

	return []string{
		filepath.Join(userConfigDir, "mimetype"),
		"/etc/mimetype",
		"/usr/lib/mimetype",
	}
}

type ErrNoPluginFound struct {
	MimeType string
	Command  PluginCommand
}

var pluginSearchPathsFn = PluginSearchPaths

func (e ErrNoPluginFound) Error() string {
	return "no plugin found for mime type: " + e.MimeType + " (command: " + e.Command.String() + ")"
}

func FindPluginForCommand(mimeType string, command PluginCommand) (string, error) {
	pluginPath := filepath.Join(mimeType, command.String())
	for _, baseDir := range pluginSearchPathsFn() {
		fullPath := filepath.Join(baseDir, pluginPath)
		info, err := os.Lstat(fullPath)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || info.Mode().IsRegular() {
				return fullPath, nil
			}
		}
	}

	return "", ErrNoPluginFound{MimeType: mimeType, Command: command}
}

func RunPlugin(pluginPath string, command PluginCommand, filePath, key, value string) (map[string][]string, error) {
	var args []string
	switch command {
	case PluginList:
		args = []string{filePath}
	case PluginAdd, PluginDelete:
		args = []string{filePath, key, value}
	}

	cmd := exec.Command(pluginPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, PluginError{
			PluginPath: pluginPath,
			Stderr:     stderr.String(),
		}
	}

	return ParsePluginOutput(stdout.String())
}

type PluginError struct {
	PluginPath string
	Stderr     string
}

func (e PluginError) Error() string {
	if e.PluginPath != "" {
		return e.PluginPath + ": " + e.Stderr
	}
	return e.Stderr
}

func ParsePluginOutput(output string) (map[string][]string, error) {
	return pluginio.DeserializeMetadata(output)
}

// RunOpenHandler executes an open handler for a file. The handler inherits
// stdin, stdout, and stderr so it can interact with the user.
func RunOpenHandler(handlerPath, filePath string) error {
	cmd := exec.Command(handlerPath, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
