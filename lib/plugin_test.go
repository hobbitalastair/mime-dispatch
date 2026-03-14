package lib

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginSearchPaths(t *testing.T) {
	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test with XDG_CONFIG_HOME set
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	paths := PluginSearchPaths()

	if len(paths) != 3 {
		t.Errorf("expected 3 search paths, got %d", len(paths))
	}

	if paths[0] != "/custom/config/metadata/plugins" {
		t.Errorf("expected first path to be XDG_CONFIG_HOME based, got %s", paths[0])
	}

	if paths[1] != "/etc/metadata/plugins" {
		t.Errorf("expected second path to be /etc/metadata/plugins, got %s", paths[1])
	}

	if paths[2] != "/usr/lib/metadata/plugins" {
		t.Errorf("expected third path to be /usr/lib/metadata/plugins, got %s", paths[2])
	}
}

func TestFindPluginForCommand_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create three levels of plugin directories
	userDir := filepath.Join(tmpDir, "user", "metadata", "plugins", "text", "markdown")
	adminDir := filepath.Join(tmpDir, "etc", "metadata", "plugins", "text", "markdown")
	distroDir := filepath.Join(tmpDir, "lib", "metadata", "plugins", "text", "markdown")

	os.MkdirAll(userDir, 0755)
	os.MkdirAll(adminDir, 0755)
	os.MkdirAll(distroDir, 0755)

	// Create dummy binaries
	userBin := filepath.Join(tmpDir, "user-plugin")
	adminBin := filepath.Join(tmpDir, "admin-plugin")
	distroBin := filepath.Join(tmpDir, "distro-plugin")

	os.WriteFile(userBin, []byte("#!/bin/sh\necho user"), 0755)
	os.WriteFile(adminBin, []byte("#!/bin/sh\necho admin"), 0755)
	os.WriteFile(distroBin, []byte("#!/bin/sh\necho distro"), 0755)

	// Create symlinks
	os.Symlink(userBin, filepath.Join(userDir, "list"))
	os.Symlink(adminBin, filepath.Join(adminDir, "list"))
	os.Symlink(distroBin, filepath.Join(distroDir, "list"))

	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test: user plugin takes precedence
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "user"))

	// Manually create test with overridden paths
	paths := []string{
		filepath.Join(tmpDir, "user", "metadata", "plugins"),
		filepath.Join(tmpDir, "etc", "metadata", "plugins"),
		filepath.Join(tmpDir, "lib", "metadata", "plugins"),
	}

	// Test user plugin is found
	var foundPath string
	for _, baseDir := range paths {
		fullPath := filepath.Join(baseDir, "text", "markdown", "list")
		info, err := os.Lstat(fullPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(fullPath)
			if filepath.IsAbs(target) {
				foundPath = target
			} else {
				foundPath, _ = filepath.Abs(filepath.Join(fullPath, target))
			}
			break
		}
	}

	if foundPath != userBin {
		t.Errorf("expected user plugin to take precedence, got %s", foundPath)
	}

	// Test: admin plugin when user plugin doesn't exist
	os.Remove(filepath.Join(userDir, "list"))
	foundPath = ""
	for _, baseDir := range paths {
		fullPath := filepath.Join(baseDir, "text", "markdown", "list")
		info, err := os.Lstat(fullPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(fullPath)
			if filepath.IsAbs(target) {
				foundPath = target
			} else {
				foundPath, _ = filepath.Abs(filepath.Join(fullPath, target))
			}
			break
		}
	}

	if foundPath != adminBin {
		t.Errorf("expected admin plugin when user missing, got %s", foundPath)
	}
}

func TestFindPluginForCommand_NoPlugin(t *testing.T) {
	tmpDir := t.TempDir()

	// Override env to use temp directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := FindPluginForCommand("nonexistent/mimetype", PluginList)

	var noPluginErr ErrNoPluginFound
	if err == nil {
		t.Error("expected error when plugin not found")
	}

	if err != nil && err.Error() != "no plugin found for mime type: nonexistent/mimetype (command: list)" {
		t.Errorf("expected ErrNoPluginFound, got %v", err)
	}

	if !errors.As(err, &noPluginErr) {
		t.Errorf("expected ErrNoPluginFound type, got %T", err)
	}
}

func TestFindPluginForCommand_CommandSpecific(t *testing.T) {
	tmpDir := t.TempDir()

	pluginDir := filepath.Join(tmpDir, "metadata", "plugins", "text", "markdown")
	os.MkdirAll(pluginDir, 0755)

	// Create only list command (no add or delete)
	listBin := filepath.Join(tmpDir, "list-plugin")
	os.WriteFile(listBin, []byte("#!/bin/sh\necho list"), 0755)
	os.Symlink(listBin, filepath.Join(pluginDir, "list"))

	// Override env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// list should be found
	pluginPath, err := FindPluginForCommand("text/markdown", PluginList)
	if err != nil {
		t.Errorf("expected to find list plugin, got error: %v", err)
	}
	if pluginPath != filepath.Join(pluginDir, "list") {
		t.Errorf("expected plugin path %s, got %s", listBin, pluginPath)
	}

	// add should not be found
	_, err = FindPluginForCommand("text/markdown", PluginAdd)
	if err == nil {
		t.Error("expected error when add plugin not found")
	}

	var noPluginErr ErrNoPluginFound
	if !errors.As(err, &noPluginErr) {
		t.Errorf("expected ErrNoPluginFound, got %v", err)
	}
}

func TestParsePluginOutput_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string][]string
	}{
		{
			name:     "key with colon in value",
			output:   "url: https://example.com",
			expected: map[string][]string{"url": {"https://example.com"}},
		},
		{
			name:     "multi-valued with empty initial",
			output:   "tags:\n  - tag1\n  - tag2",
			expected: map[string][]string{"tags": {"tag1", "tag2"}},
		},
		{
			name:   "mixed single and multi",
			output: "title: Test\ntags:\n  - tag1\nauthor: Me",
			expected: map[string][]string{
				"title":  {"Test"},
				"tags":   {"tag1"},
				"author": {"Me"},
			},
		},
		{
			name:   "whitespace handling",
			output: "  key:   value  \n\n  key2:  value2  ",
			expected: map[string][]string{
				"key":  {"value"},
				"key2": {"value2"},
			},
		},
		{
			name:     "empty multi-valued key",
			output:   "tags:",
			expected: map[string][]string{"tags": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePluginOutput(tt.output)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(result))
			}

			for k, expectedValues := range tt.expected {
				actualValues, ok := result[k]
				if !ok {
					t.Errorf("expected key %q not found in result", k)
					continue
				}

				if len(actualValues) != len(expectedValues) {
					t.Errorf("key %q: expected %d values, got %d", k, len(expectedValues), len(actualValues))
					continue
				}

				for i, expectedValue := range expectedValues {
					if actualValues[i] != expectedValue {
						t.Errorf("key %q value[%d]: expected %q, got %q", k, i, expectedValue, actualValues[i])
					}
				}
			}
		})
	}
}
