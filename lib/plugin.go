package lib

import (
	"io"
	"metadata/pkg/pluginio"
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
)

func (c PluginCommand) String() string {
	switch c {
	case PluginList:
		return "metadata-list"
	case PluginAdd:
		return "metadata-add"
	case PluginDelete:
		return "metadata-delete"
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

func (e ErrNoPluginFound) Error() string {
	return "no plugin found for mime type: " + e.MimeType + " (command: " + e.Command.String() + ")"
}

func FindPluginForCommand(mimeType string, command PluginCommand) (string, error) {
	pluginPath := filepath.Join(mimeType, command.String())
	for _, baseDir := range PluginSearchPaths() {
		fullPath := filepath.Join(baseDir, pluginPath)
		info, err := os.Lstat(fullPath)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || info.Mode().IsRegular() {
				return fullPath, nil
			}
		}
	}

	if command == PluginList {
		genericPath := mimeType
		for _, baseDir := range PluginSearchPaths() {
			fullPath := filepath.Join(baseDir, genericPath)
			info, err := os.Lstat(fullPath)
			if err != nil {
				continue
			}

			if info.Mode()&os.ModeSymlink != 0 {
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	errBytes, _ := io.ReadAll(stderr)
	cmd.Wait()

	if cmd.ProcessState.ExitCode() != 0 {
		return nil, PluginError{
			Stderr: string(errBytes),
		}
	}

	return ParsePluginOutput(string(output))
}

type PluginError struct {
	Stderr string
}

func (e PluginError) Error() string {
	return e.Stderr
}

func ParsePluginOutput(output string) (map[string][]string, error) {
	return pluginio.DeserializeMetadata(output)
}
