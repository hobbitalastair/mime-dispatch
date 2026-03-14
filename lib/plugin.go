package lib

import (
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var PluginSearchPaths []string

func init() {
	userConfigDir := os.Getenv("XDG_CONFIG_HOME")
	if userConfigDir == "" {
		usr, err := user.Current()
		if err == nil {
			userConfigDir = filepath.Join(usr.HomeDir, ".config")
		}
	}

	PluginSearchPaths = []string{
		filepath.Join(userConfigDir, "metadata", "plugins"),
		"/etc/metadata/plugins",
		"/usr/lib/metadata/plugins",
	}
}

func FindPlugin(mimeType string) (string, error) {
	for _, baseDir := range PluginSearchPaths {
		pluginDir := filepath.Join(baseDir, mimeType)
		info, err := os.Lstat(pluginDir)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(pluginDir)
			if err != nil {
				continue
			}
			if filepath.IsAbs(target) {
				return target, nil
			}
			absTarget, err := filepath.Abs(filepath.Join(pluginDir, target))
			if err != nil {
				continue
			}
			return absTarget, nil
		}

		if info.IsDir() {
			entries, err := os.ReadDir(pluginDir)
			if err != nil || len(entries) == 0 {
				continue
			}
			for _, entry := range entries {
				if entry.Type()&os.ModeSymlink != 0 {
					entryPath := filepath.Join(pluginDir, entry.Name())
					target, err := os.Readlink(entryPath)
					if err != nil {
						continue
					}
					var absTarget string
					if filepath.IsAbs(target) {
						absTarget = target
					} else {
						absTarget, err = filepath.Abs(filepath.Join(pluginDir, target))
						if err != nil {
							continue
						}
					}
					return absTarget, nil
				}
			}
		}
	}

	return "", ErrNoPluginFound{MimeType: mimeType}
}

type ErrNoPluginFound struct {
	MimeType string
}

func (e ErrNoPluginFound) Error() string {
	return "no plugin found for mime type: " + e.MimeType
}

func RunPlugin(pluginPath, command, filePath, key, value string) (map[string][]string, error) {
	var args []string
	switch command {
	case "list":
		args = []string{"list", filePath}
	case "set":
		args = []string{"set", filePath, key, value}
	case "delete":
		args = []string{"delete", filePath, key}
	}

	cmd := exec.Command(pluginPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

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

	stdin.Close()
	output, err := io.ReadAll(stdout)
	if err != nil {
		cmd.Wait()
		return nil, err
	}

	errBytes, _ := io.ReadAll(stderr)
	cmd.Wait()

	if cmd.ProcessState.ExitCode() != 0 {
		return nil, PluginError{
			Err:    string(errBytes),
			Stderr: string(errBytes),
		}
	}

	return ParsePluginOutput(string(output)), nil
}

type PluginError struct {
	Err    string
	Stderr string
}

func (e PluginError) Error() string {
	return e.Stderr
}

func ParsePluginOutput(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var currentKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "  - ") {
			value := strings.TrimPrefix(line, "  - ")
			if currentKey != "" {
				result[currentKey] = append(result[currentKey], value)
			}
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.TrimSpace(parts[1])
			currentKey = key
			if value != "" {
				result[key] = append(result[key], value)
			} else {
				result[key] = []string{}
			}
		}
	}

	return result
}
