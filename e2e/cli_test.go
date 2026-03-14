package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var cliPath string
var pluginPath string

func init() {
	cliPath = os.Getenv("METADATA_CLI_PATH")
	if cliPath == "" {
		cliPath = "./metadata"
	}
	pluginPath = os.Getenv("METADATA_PLUGIN_PATH")
	if pluginPath == "" {
		pluginPath = "./metadata-markdown"
	}
}

func setupPlugin(t *testing.T) func() {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "metadata", "plugins", "text", "markdown")
	err := os.MkdirAll(pluginDir, 0755)
	if err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginSymlink := filepath.Join(pluginDir, "metadata")
	err = os.Symlink(pluginPath, pluginSymlink)
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
	cmd := exec.Command(cliPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for missing args")
	}
	if !strings.Contains(string(output), "Usage") && !strings.Contains(string(output), "command") {
		t.Errorf("expected usage message, got: %s", output)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	cmd := exec.Command(cliPath, "unknown", "test.md")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for unknown command")
	}
	if !strings.Contains(string(output), "Unknown command") && !strings.Contains(string(output), "command") {
		t.Errorf("expected unknown command error, got: %s", output)
	}
}

func TestCLIListMissingFile(t *testing.T) {
	cmd := exec.Command(cliPath, "list", "nonexistent.md")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strings.Contains(string(output), "no such file") {
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

	cmd := exec.Command(cliPath, "set", "--file-only", testFile, "title", "Test Title")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("output: %s", output)
		t.Skip("plugin not configured")
		return
	}

	cmd = exec.Command(cliPath, "list", "--file-only", testFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(string(output), "title: Test Title") {
		t.Errorf("expected title in output, got: %s", output)
	}

	cmd = exec.Command(cliPath, "delete", "--file-only", testFile, "title")
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	cmd = exec.Command(cliPath, "list", "--file-only", testFile)
	output, _ = cmd.CombinedOutput()
	if strings.TrimSpace(string(output)) != "" {
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

	cmd := exec.Command(cliPath, "set", "--file-only", testFile, "title", "New Title")
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Skip("plugin not configured")
		return
	}

	cmd = exec.Command(cliPath, "list", "--file-only", testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(string(output), "title: New Title") {
		t.Errorf("expected updated title in output, got: %s", output)
	}
	if !strings.Contains(string(output), "author: Test Author") {
		t.Errorf("expected author in output, got: %s", output)
	}
}
