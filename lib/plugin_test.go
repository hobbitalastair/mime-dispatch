package lib

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setPluginSearchPathsForTest(t *testing.T, paths []string) {
	t.Helper()
	original := pluginSearchPathsFn
	pluginSearchPathsFn = func() []string {
		return paths
	}
	t.Cleanup(func() {
		pluginSearchPathsFn = original
	})
}

func TestPluginSearchPaths(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	paths := PluginSearchPaths()

	if len(paths) != 3 {
		t.Errorf("expected 3 search paths, got %d", len(paths))
	}

	if paths[0] != "/custom/config/mimetype" {
		t.Errorf("expected first path to be XDG_CONFIG_HOME based, got %s", paths[0])
	}

	if paths[1] != "/etc/mimetype" {
		t.Errorf("expected second path to be /etc/mimetype, got %s", paths[1])
	}

	if paths[2] != "/usr/lib/mimetype" {
		t.Errorf("expected third path to be /usr/lib/mimetype, got %s", paths[2])
	}
}

func TestFindPluginForCommand_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create three levels of plugin directories
	userDir := filepath.Join(tmpDir, "user", "mimetype", "text", "markdown")
	adminDir := filepath.Join(tmpDir, "etc", "mimetype", "text", "markdown")
	distroDir := filepath.Join(tmpDir, "lib", "mimetype", "text", "markdown")

	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatalf("mkdir user dir: %v", err)
	}
	if err := os.MkdirAll(adminDir, 0755); err != nil {
		t.Fatalf("mkdir admin dir: %v", err)
	}
	if err := os.MkdirAll(distroDir, 0755); err != nil {
		t.Fatalf("mkdir distro dir: %v", err)
	}

	// Create dummy binaries
	userBin := filepath.Join(tmpDir, "user-plugin")
	adminBin := filepath.Join(tmpDir, "admin-plugin")
	distroBin := filepath.Join(tmpDir, "distro-plugin")

	if err := os.WriteFile(userBin, []byte("#!/bin/sh\necho user"), 0755); err != nil {
		t.Fatalf("write user binary: %v", err)
	}
	if err := os.WriteFile(adminBin, []byte("#!/bin/sh\necho admin"), 0755); err != nil {
		t.Fatalf("write admin binary: %v", err)
	}
	if err := os.WriteFile(distroBin, []byte("#!/bin/sh\necho distro"), 0755); err != nil {
		t.Fatalf("write distro binary: %v", err)
	}

	// Create symlinks
	if err := os.Symlink(userBin, filepath.Join(userDir, "metadata-list")); err != nil {
		t.Fatalf("symlink user plugin: %v", err)
	}
	if err := os.Symlink(adminBin, filepath.Join(adminDir, "metadata-list")); err != nil {
		t.Fatalf("symlink admin plugin: %v", err)
	}
	if err := os.Symlink(distroBin, filepath.Join(distroDir, "metadata-list")); err != nil {
		t.Fatalf("symlink distro plugin: %v", err)
	}

	paths := []string{
		filepath.Join(tmpDir, "user", "mimetype"),
		filepath.Join(tmpDir, "etc", "mimetype"),
		filepath.Join(tmpDir, "lib", "mimetype"),
	}
	setPluginSearchPathsForTest(t, paths)

	foundPath, err := FindPluginForCommand("text/markdown", PluginList)
	if err != nil {
		t.Fatalf("find list plugin: %v", err)
	}
	if foundPath != filepath.Join(userDir, "metadata-list") {
		t.Errorf("expected user plugin path, got %s", foundPath)
	}

	// Test: admin plugin when user plugin doesn't exist
	if err := os.Remove(filepath.Join(userDir, "metadata-list")); err != nil {
		t.Fatalf("remove user plugin symlink: %v", err)
	}

	foundPath, err = FindPluginForCommand("text/markdown", PluginList)
	if err != nil {
		t.Fatalf("find admin plugin: %v", err)
	}

	if foundPath != filepath.Join(adminDir, "metadata-list") {
		t.Errorf("expected admin plugin when user missing, got %s", foundPath)
	}
}

func TestFindPluginForCommand_NoPlugin(t *testing.T) {
	tmpDir := t.TempDir()

	// Override env to use temp directory
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	setPluginSearchPathsForTest(t, []string{filepath.Join(tmpDir, "mimetype")})

	_, err := FindPluginForCommand("nonexistent/mimetype", PluginList)

	var noPluginErr ErrNoPluginFound
	if err == nil {
		t.Error("expected error when plugin not found")
	}

	if err != nil && err.Error() != "no plugin found for mime type: nonexistent/mimetype (command: metadata-list)" {
		t.Errorf("expected ErrNoPluginFound, got %v", err)
	}

	if !errors.As(err, &noPluginErr) {
		t.Errorf("expected ErrNoPluginFound type, got %T", err)
	}
}

func TestFindPluginForCommand_CommandSpecific(t *testing.T) {
	tmpDir := t.TempDir()

	pluginDir := filepath.Join(tmpDir, "mimetype", "text", "markdown")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	// Create only list command (no add or delete)
	listBin := filepath.Join(tmpDir, "list-plugin")
	if err := os.WriteFile(listBin, []byte("#!/bin/sh\necho list"), 0755); err != nil {
		t.Fatalf("write list binary: %v", err)
	}
	if err := os.Symlink(listBin, filepath.Join(pluginDir, "metadata-list")); err != nil {
		t.Fatalf("symlink list plugin: %v", err)
	}

	setPluginSearchPathsForTest(t, []string{filepath.Join(tmpDir, "mimetype")})

	// list should be found
	pluginPath, err := FindPluginForCommand("text/markdown", PluginList)
	if err != nil {
		t.Errorf("expected to find list plugin, got error: %v", err)
	}
	if pluginPath != filepath.Join(pluginDir, "metadata-list") {
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

func TestParsePluginOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected map[string][]string
	}{
		{
			name:     "empty output",
			output:   "",
			expected: map[string][]string{},
		},
		{
			name:     "single key-value",
			output:   "key: value",
			expected: map[string][]string{"key": {"value"}},
		},
		{
			name:   "multiple key-values",
			output: "key1: value1\nkey2: value2",
			expected: map[string][]string{
				"key1": {"value1"},
				"key2": {"value2"},
			},
		},
		{
			name:     "multi-valued key",
			output:   "key:\n  - value1\n  - value2",
			expected: map[string][]string{"key": {"value1", "value2"}},
		},
		{
			name:   "ignore empty lines",
			output: "key: value\n\nkey2: value2\n",
			expected: map[string][]string{
				"key":  {"value"},
				"key2": {"value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePluginOutput(tt.output)
			if err != nil {
				t.Fatalf("ParsePluginOutput error: %v", err)
			}
			for k, v := range tt.expected {
				if len(result[k]) != len(v) {
					t.Errorf("expected %v, got %v", v, result[k])
					continue
				}
				for i, val := range v {
					if result[k][i] != val {
						t.Errorf("expected %v, got %v", v, result[k])
					}
				}
			}
		})
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
			result, err := ParsePluginOutput(tt.output)
			if err != nil {
				t.Fatalf("ParsePluginOutput error: %v", err)
			}

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
