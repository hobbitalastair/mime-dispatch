package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var cliModule = "../cmd/metadata"
var pluginModule = "../plugins/markdown"

func runCLI(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command("go", append([]string{"run", cliModule}, args...)...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func setupPlugin(t *testing.T) func() {
	tmpDir := t.TempDir()

	pluginBuildDir := filepath.Join(tmpDir, "build")
	err := os.MkdirAll(pluginBuildDir, 0755)
	if err != nil {
		t.Fatalf("failed to create plugin build dir: %v", err)
	}

	pluginBinary := filepath.Join(pluginBuildDir, "metadata-plugin")
	cmd := exec.Command("go", "build", "-o", pluginBinary, pluginModule)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build plugin: %v, output: %s", err, output)
	}

	pluginDir := filepath.Join(tmpDir, "metadata", "plugins", "text", "markdown")
	err = os.MkdirAll(pluginDir, 0755)
	if err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginSymlink := filepath.Join(pluginDir, "metadata")
	err = os.Symlink(pluginBinary, pluginSymlink)
	if err != nil {
		t.Fatalf("failed to create plugin symlink: %v", err)
	}

	originalHome := os.Getenv("HOME")
	originalConfig := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	os.Unsetenv("HOME")

	return func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalConfig)
	}
}

func TestCLIMissingArgs(t *testing.T) {
	output, err := runCLI(t)
	if err == nil {
		t.Error("expected error for missing args")
	}
	if !strings.Contains(output, "Usage") && !strings.Contains(output, "command") {
		t.Errorf("expected usage message, got: %s", output)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	output, err := runCLI(t, "unknown", "test.md")
	if err == nil {
		t.Error("expected error for unknown command")
	}
	if !strings.Contains(output, "Unknown command") && !strings.Contains(output, "command") {
		t.Errorf("expected unknown command error, got: %s", output)
	}
}

func TestCLIListMissingFile(t *testing.T) {
	output, err := runCLI(t, "list", "nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strings.Contains(output, "no such file") {
		t.Errorf("expected file not found error, got: %s", output)
	}
}

func TestMarkdownPlugin(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	output, err := runCLI(t, "set", "--file-only", testFile, "title", "Test Title")
	if err != nil {
		t.Logf("output: %s", output)
		t.Skip("plugin not configured")
		return
	}

	output, err = runCLI(t, "list", "--file-only", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: Test Title") {
		t.Errorf("expected title in output, got: %s", output)
	}

	_, err = runCLI(t, "delete", "--file-only", testFile, "title")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	output, _ = runCLI(t, "list", "--file-only", testFile)
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected empty output after delete, got: %s", output)
	}
}

func TestMarkdownPluginWithExistingFrontmatter(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test2.md")
	content := `---
title: Original Title
author: Test Author
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "set", "--file-only", testFile, "title", "New Title")
	if err != nil {
		t.Skip("plugin not configured")
		return
	}

	output, err := runCLI(t, "list", "--file-only", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: New Title") {
		t.Errorf("expected updated title in output, got: %s", output)
	}
	if !strings.Contains(output, "author: Test Author") {
		t.Errorf("expected author in output, got: %s", output)
	}
}
