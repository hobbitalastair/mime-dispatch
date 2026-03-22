// Package e2e contains end-to-end tests that build all binaries (metadata, open,
// mime-dispatch-install) and all plugins, install them, and test the full CLI.
// Run with: go test ./e2e/ -v
package e2e

import (
	"bytes"
	"fmt"
	"mime-dispatch/pkg/pluginio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// testDir holds all compiled test binaries and plugin symlinks.
// It is created in TestMain and cleaned up after tests finish.
var testDir string

// cliBinary is the path to the compiled metadata CLI binary.
var cliBinary string

// openBinary is the path to the compiled open binary.
var openBinary string

func TestMain(m *testing.M) {
	var err error
	testDir, err = os.MkdirTemp("", "metadata-e2e-*")
	if err != nil {
		panic(err)
	}

	if err := buildAll(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(testDir)
		os.Exit(1)
	}

	if err := installPlugins(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(testDir)
		os.Exit(1)
	}

	// Point plugin search at our test directory.
	os.Setenv("XDG_CONFIG_HOME", testDir)

	code := m.Run()

	os.RemoveAll(testDir)
	os.Exit(code)
}

func buildAll() error {
	builds := []struct {
		name string
		dir  string
	}{
		{"cli", "../cmd/metadata"},
		{"open", "../cmd/open"},
		{"mime-dispatch-install", "../cmd/mime-dispatch-install"},
		{"yaml-frontmatter", "../plugins/yaml-frontmatter"},
		{"audio", "../plugins/audio"},
		{"image", "../plugins/image"},
	}

	var wg sync.WaitGroup
	errs := make([]error, len(builds))
	for i, b := range builds {
		wg.Add(1)
		go func(i int, name, dir string) {
			defer wg.Done()
			binary := filepath.Join(testDir, name)
			cmd := exec.Command("go", "build", "-o", binary, ".")
			cmd.Dir = dir
			if output, err := cmd.CombinedOutput(); err != nil {
				errs[i] = fmt.Errorf("failed to build %s: %v\n%s", name, err, output)
			}
		}(i, b.name, b.dir)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	cliBinary = filepath.Join(testDir, "cli")
	openBinary = filepath.Join(testDir, "open")
	return nil
}

func installPlugins() error {
	installer := filepath.Join(testDir, "mime-dispatch-install")
	for _, plugin := range []string{"yaml-frontmatter", "audio", "image"} {
		binary := filepath.Join(testDir, plugin)
		cmd := exec.Command(installer, "--user", binary)
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+testDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("install %s: %v\n%s", plugin, err, output)
		}
	}
	return nil
}

func runCLI(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command(cliBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runOpen(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(openBinary, args...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+testDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func parseMetadataOutput(t *testing.T, output string) map[string][]string {
	t.Helper()

	metadata, err := pluginio.DeserializeMetadata(output)
	if err != nil {
		t.Fatalf("failed to parse metadata output %q: %v", output, err)
	}

	return metadata
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

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	output, err := runCLI(t, "add", "--file-only", testFile, "title", "Test Title")
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

	_, err = runCLI(t, "delete", "--file-only", testFile, "title", "Test Title")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	output, _ = runCLI(t, "list", "--file-only", testFile)
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected empty output after delete, got: %s", output)
	}
}

func TestMarkdownPluginWithExistingFrontmatter(t *testing.T) {

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

	_, err = runCLI(t, "add", "--file-only", testFile, "title", "New Title")
	if err != nil {
		t.Skip("plugin not configured")
		return
	}

	output, err := runCLI(t, "list", "--file-only", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "- New Title") {
		t.Errorf("expected updated title in output, got: %s", output)
	}
	if !strings.Contains(output, "author: Test Author") {
		t.Errorf("expected author in output, got: %s", output)
	}
}

func TestXattrOnlyList(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
author: File Author
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	output, err := runCLI(t, "list", "--xattr-only", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "xattr-key: xattr-value") {
		t.Errorf("expected xattr in output, got: %s", output)
	}
	if strings.Contains(output, "title:") {
		t.Errorf("expected no file metadata in output when using --xattr-only, got: %s", output)
	}
}

func TestXattrOnlySet(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
author: File Author
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: File Title") {
		t.Errorf("expected file metadata in output, got: %s", output)
	}
	if !strings.Contains(output, "xattr-key: xattr-value") {
		t.Errorf("expected xattr in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if strings.Contains(string(fileContent), "xattr-key") {
		t.Errorf("expected xattr not to be in file content when using --xattr-only, got: %s", fileContent)
	}
}

func TestXattrOnlyPreservesUnusualCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "xattr-special.txt")
	err := os.WriteFile(testFile, []byte("hello\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	key := "xattr_special"
	value := "line1\nline2: \"quoted\",comma, and trailing space "

	_, err = runCLI(t, "add", "--xattr-only", testFile, key, value)
	if err != nil {
		t.Fatalf("add --xattr-only failed: %v", err)
	}

	output, err := runCLI(t, "list", "--xattr-only", testFile)
	if err != nil {
		t.Fatalf("list --xattr-only failed: %v, output: %s", err, output)
	}

	metadata := parseMetadataOutput(t, output)
	values, ok := metadata[key]
	if !ok {
		t.Fatalf("expected key %q in output, got: %s", key, output)
	}
	if len(values) != 1 {
		t.Fatalf("expected one value for %q, got %v", key, values)
	}
	if values[0] != value {
		t.Fatalf("xattr value mismatch: got %q, expected %q", values[0], value)
	}
}

func TestPluginPreservesUnusualCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "plugin-special.md")
	err := os.WriteFile(testFile, []byte("# Hello\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	key := "plugin:key with spaces"
	value := "line1\nline2: \"quoted\" and #hash"

	_, err = runCLI(t, "add", "--file-only", testFile, key, value)
	if err != nil {
		t.Fatalf("add --file-only failed: %v", err)
	}

	output, err := runCLI(t, "list", "--file-only", testFile)
	if err != nil {
		t.Fatalf("list --file-only failed: %v, output: %s", err, output)
	}

	metadata := parseMetadataOutput(t, output)
	values, ok := metadata[key]
	if !ok {
		t.Fatalf("expected key %q in output, got: %s", key, output)
	}
	if len(values) != 1 {
		t.Fatalf("expected one value for %q, got %v", key, values)
	}
	if values[0] != value {
		t.Fatalf("plugin value mismatch: got %q, expected %q", values[0], value)
	}
}

func TestXattrOnlyDeleteFile(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
author: File Author
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("delete xattr failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: File Title") {
		t.Errorf("expected file metadata in output, got: %s", output)
	}
	if strings.Contains(output, "xattr-key:") {
		t.Errorf("expected xattr to be deleted, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "title:") {
		t.Errorf("expected file metadata to still exist after --xattr-only delete, got: %s", fileContent)
	}
}

func TestSetDefaultBehaviorOnFreshFile(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", testFile, "title", "Test Title")
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: Test Title") {
		t.Errorf("expected title in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "title: Test Title") {
		t.Errorf("expected metadata in file content, got: %s", fileContent)
	}
}

func TestAddAppendsToExistingFileValue(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: Original Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", testFile, "title", "Updated Title")
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title:") {
		t.Errorf("expected title in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "- Updated Title") {
		t.Errorf("expected updated title in file content, got: %s", fileContent)
	}
}

func TestAddAppendsToExistingXattrValue(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "add", testFile, "title", "New Title")
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "- New Title") {
		t.Errorf("expected new title in output, got: %s", output)
	}
}

func TestDeleteDefaultBehaviorDeletesBoth(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add the same value to xattr as well
	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "File Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	// Delete the value from both (default behavior deletes from both locations)
	_, err = runCLI(t, "delete", testFile, "title", "File Title")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	// Title should be completely gone from both locations
	if strings.Contains(output, "title:") {
		t.Errorf("expected title to be deleted from both locations, got: %s", output)
	}
}

func TestDeleteSpecificValueFromMultiValuedKey(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: First Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add a second title value (creates multi-valued)
	_, err = runCLI(t, "add", "--file-only", testFile, "title", "Second Title")
	if err != nil {
		t.Fatalf("add second title failed: %v", err)
	}

	// Delete only the first value
	_, err = runCLI(t, "delete", "--file-only", testFile, "title", "First Title")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	// Second title should remain (normalized to scalar since only one value), first should be gone
	if !strings.Contains(output, "title: Second Title") {
		t.Errorf("expected second title to remain, got: %s", output)
	}
	if strings.Contains(output, "First Title") {
		t.Errorf("expected first title to be deleted, got: %s", output)
	}
}

func TestListDefaultBehaviorMergedWithPrecedence(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
author: File Author
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "xattr-only-key", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title:") {
		t.Errorf("expected title in output, got: %s", output)
	}
	if !strings.Contains(output, "author: File Author") {
		t.Errorf("expected file author in output, got: %s", output)
	}
	if !strings.Contains(output, "xattr-only-key: xattr-value") {
		t.Errorf("expected xattr-only key in output, got: %s", output)
	}
}

func TestMultiValuedKeysYAMLFormat(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
tags:
  - file-value1
  - file-value2
---

# Hello World
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "tags", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "tags:") {
		t.Errorf("expected tags in output, got: %s", output)
	}
	if !strings.Contains(output, "- file-value1") || !strings.Contains(output, "- file-value2") || !strings.Contains(output, "- xattr-value") {
		t.Errorf("expected multi-valued YAML sequence format with merged values, got: %s", output)
	}
}

func TestMimeTypeInListOutput(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: Test File
---

# Hello World
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "mime_type:") {
		t.Errorf("expected mime_type in output, got: %s", output)
	}
	if !strings.Contains(output, "text/markdown") {
		t.Errorf("expected text/markdown in output, got: %s", output)
	}
}

func TestFileOnlySet(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--file-only", testFile, "title", "File Title")
	if err != nil {
		t.Fatalf("add --file-only failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: File Title") {
		t.Errorf("expected title in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "title: File Title") {
		t.Errorf("expected title in file content, got: %s", fileContent)
	}

	xattrOutput, err := runCLI(t, "list", "--xattr-only", testFile)
	if err != nil {
		t.Fatalf("list --xattr-only failed: %v, output: %s", err, xattrOutput)
	}
	if strings.Contains(xattrOutput, "title:") {
		t.Errorf("expected no title in xattr when using --file-only, got: %s", xattrOutput)
	}
}

func TestFileOnlyDelete(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--file-only", testFile, "title", "File Title")
	if err != nil {
		t.Fatalf("delete --file-only failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if strings.Contains(output, "title: File Title") {
		t.Errorf("expected file title to be deleted, got: %s", output)
	}
	if !strings.Contains(output, "title: Xattr Title") {
		t.Errorf("expected xattr title to remain, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if strings.Contains(string(fileContent), "title:") {
		t.Errorf("expected title to be removed from file content, got: %s", fileContent)
	}
}

func TestDeleteXattrOnlyPreservesFile(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("delete --xattr-only failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: File Title") {
		t.Errorf("expected file title to remain, got: %s", output)
	}
	if strings.Contains(output, "title: Xattr Title") {
		t.Errorf("expected xattr title to be deleted, got: %s", output)
	}

	xattrOutput, err := runCLI(t, "list", "--xattr-only", testFile)
	if err != nil {
		t.Fatalf("list --xattr-only failed: %v, output: %s", err, xattrOutput)
	}
	if strings.Contains(xattrOutput, "title:") {
		t.Errorf("expected title to be removed from xattr, got: %s", xattrOutput)
	}
}

func TestDeleteWithBothLocationsAndFlags(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: File Title
---

# Hello
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "xattr-only-key", "xattr-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("delete --xattr-only failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: File Title") {
		t.Errorf("expected file title to remain, got: %s", output)
	}
	if strings.Contains(output, "title: Xattr Title") {
		t.Errorf("expected xattr title to be deleted, got: %s", output)
	}
	if !strings.Contains(output, "xattr-only-key: xattr-value") {
		t.Errorf("expected xattr-only-key to remain, got: %s", output)
	}
}

func TestAudioPluginList(t *testing.T) {

	tmpDir := t.TempDir()

	testMp3 := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testMp3)
	if err != nil {
		t.Fatalf("failed to copy mp3 file: %v", err)
	}

	output, err := runCLI(t, "list", "--file-only", testMp3)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title:") {
		t.Errorf("expected title in output, got: %s", output)
	}
	if !strings.Contains(output, "artist:") {
		t.Errorf("expected artist in output, got: %s", output)
	}
	if !strings.Contains(output, "album:") {
		t.Errorf("expected album in output, got: %s", output)
	}

	testOgg := filepath.Join(tmpDir, "test.ogg")
	err = copyFile("samples/sample1.ogg", testOgg)
	if err != nil {
		t.Fatalf("failed to copy ogg file: %v", err)
	}

	output, err = runCLI(t, "list", "--file-only", testOgg)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: Everdusk Rescue") {
		t.Errorf("expected title in output, got: %s", output)
	}
}

func TestAudioPluginSetFallback(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	output, err := runCLI(t, "add", testFile, "custom-key", "custom-value")
	if err != nil {
		t.Fatalf("add failed: %v, output: %s", err, output)
	}

	listOutput, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, listOutput)
	}

	if !strings.Contains(listOutput, "custom-key: custom-value") {
		t.Errorf("expected custom-key in output (xattr fallback), got: %s", listOutput)
	}
}

func TestAudioPluginDeleteKeyExistsInFile(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	delOutput, err := runCLI(t, "delete", testFile, "title", "Test Title")
	if err == nil {
		t.Error("expected error when deleting key that exists in file")
	}

	if !strings.Contains(delOutput, "read-only") {
		t.Errorf("expected error about read-only, got: %v, output: %s", err, delOutput)
	}
}

func TestAudioPluginDeleteKeyNotInFile(t *testing.T) {

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	_, err = runCLI(t, "add", "--xattr-only", testFile, "custom-key", "custom-value")
	if err != nil {
		t.Fatalf("add xattr failed: %v", err)
	}

	delOutput, err := runCLI(t, "delete", testFile, "custom-key", "custom-value")
	if err != nil {
		t.Fatalf("delete failed: %v, output: %s", err, delOutput)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if strings.Contains(output, "custom-key:") {
		t.Errorf("expected custom-key to be deleted, got: %s", output)
	}
}

func TestImagePluginListMetadata(t *testing.T) {

	output, err := runCLI(t, "list", "--file-only", "samples/image.jpg")
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	metadata := parseMetadataOutput(t, output)

	datetimeValues, ok := metadata["datetime"]
	if !ok || len(datetimeValues) != 1 {
		t.Fatalf("expected single datetime value in output, got: %v", metadata["datetime"])
	}

	isoDateTimePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([+-]\d{2}:\d{2})?$`)
	if !isoDateTimePattern.MatchString(datetimeValues[0]) {
		t.Errorf("expected ISO 8601 datetime, got: %q", datetimeValues[0])
	}

	if _, datePresent := metadata["date"]; datePresent {
		t.Errorf("did not expect date field, got output: %s", output)
	}

	if _, captionPresent := metadata["caption"]; captionPresent {
		t.Errorf("did not expect caption field, got output: %s", output)
	}

	if _, hasLocation := metadata["location"]; !hasLocation {
		t.Errorf("expected location in output, got: %s", output)
	}
}

func TestImagePluginReadOnly(t *testing.T) {

	// Test that add command fails for image plugins (read-only, no add plugin)
	_, err := runCLI(t, "add", "--file-only", "samples/image.jpg", "test", "value")
	if err == nil {
		t.Error("expected add to fail on read-only image plugin")
	}
}

// runPlugin runs a plugin binary directly (not via symlink) with the given args.
func runPlugin(t *testing.T, plugin string, args ...string) (string, string, error) {
	t.Helper()
	binary := filepath.Join(testDir, plugin)
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestPluginCapabilities(t *testing.T) {
	stdout, _, err := runPlugin(t, "yaml-frontmatter", "--capabilities")
	if err != nil {
		t.Fatalf("--capabilities failed: %v", err)
	}

	caps, err := pluginio.DeserializeCapabilities(stdout)
	if err != nil {
		t.Fatalf("failed to parse capabilities: %v", err)
	}

	expectedMimetypes := []string{"text/markdown", "text/plain"}
	for _, mt := range expectedMimetypes {
		found := false
		for _, got := range caps.Mimetypes {
			if got == mt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected mimetype %q in capabilities, got: %v", mt, caps.Mimetypes)
		}
	}

	expectedCommands := []string{"metadata-add", "metadata-delete", "metadata-list"}
	for _, cmd := range expectedCommands {
		found := false
		for _, got := range caps.Commands {
			if got == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %q in capabilities, got: %v", cmd, caps.Commands)
		}
	}
}

func TestPluginCapabilitiesAudio(t *testing.T) {
	stdout, _, err := runPlugin(t, "audio", "--capabilities")
	if err != nil {
		t.Fatalf("--capabilities failed: %v", err)
	}

	caps, err := pluginio.DeserializeCapabilities(stdout)
	if err != nil {
		t.Fatalf("failed to parse capabilities: %v", err)
	}

	expectedMimetypes := []string{"audio/mpeg", "audio/ogg", "audio/x-vorbis+ogg"}
	for _, mt := range expectedMimetypes {
		found := false
		for _, got := range caps.Mimetypes {
			if got == mt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected mimetype %q in capabilities, got: %v", mt, caps.Mimetypes)
		}
	}

	if len(caps.Commands) != 1 || caps.Commands[0] != "metadata-list" {
		t.Errorf("expected commands [metadata-list], got: %v", caps.Commands)
	}
}

func TestPluginCapabilitiesImage(t *testing.T) {
	stdout, _, err := runPlugin(t, "image", "--capabilities")
	if err != nil {
		t.Fatalf("--capabilities failed: %v", err)
	}

	caps, err := pluginio.DeserializeCapabilities(stdout)
	if err != nil {
		t.Fatalf("failed to parse capabilities: %v", err)
	}

	if len(caps.Mimetypes) != 1 || caps.Mimetypes[0] != "image/jpeg" {
		t.Errorf("expected mimetypes [image/jpeg], got: %v", caps.Mimetypes)
	}

	if len(caps.Commands) != 1 || caps.Commands[0] != "metadata-list" {
		t.Errorf("expected commands [metadata-list], got: %v", caps.Commands)
	}
}

func TestMimetypeInstallAndUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	pluginBinary := filepath.Join(testDir, "yaml-frontmatter")
	installer := filepath.Join(testDir, "mime-dispatch-install")

	// Install with --user, using tmpDir as XDG_CONFIG_HOME
	cmd := exec.Command(installer, "--user", pluginBinary)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install failed: %v\n%s", err, output)
	}

	// Verify symlinks exist for each mimetype/command combination
	expectedMimetypes := []string{"text/markdown", "text/plain"}
	expectedCommands := []string{"metadata-add", "metadata-delete", "metadata-list"}
	mimetypeBase := filepath.Join(tmpDir, "mimetype")

	for _, mt := range expectedMimetypes {
		for _, command := range expectedCommands {
			linkPath := filepath.Join(mimetypeBase, mt, command)
			target, err := os.Readlink(linkPath)
			if err != nil {
				t.Errorf("expected symlink at %s, got error: %v", linkPath, err)
				continue
			}
			if target != pluginBinary {
				t.Errorf("symlink %s points to %q, expected %q", linkPath, target, pluginBinary)
			}
		}
	}

	// Uninstall
	cmd = exec.Command(installer, "--user", "--uninstall", pluginBinary)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("uninstall failed: %v\n%s", err, output)
	}

	// Verify symlinks are removed
	for _, mt := range expectedMimetypes {
		for _, command := range expectedCommands {
			linkPath := filepath.Join(mimetypeBase, mt, command)
			if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
				t.Errorf("expected symlink %s to be removed after uninstall", linkPath)
			}
		}
	}

	// Verify empty mimetype dirs were cleaned up
	for _, mt := range expectedMimetypes {
		dir := filepath.Join(mimetypeBase, mt)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("expected directory %s to be removed after uninstall", dir)
		}
	}
}

func TestPluginUnknownSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	pluginBinary := filepath.Join(testDir, "yaml-frontmatter")
	bogusLink := filepath.Join(tmpDir, "metadata-bogus")

	if err := os.Symlink(pluginBinary, bogusLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	cmd := exec.Command(bogusLink)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		t.Error("expected non-zero exit code for unknown symlink name")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("expected usage message on stderr, got: %s", stderr.String())
	}
}

func TestPluginDirectInvocation(t *testing.T) {
	_, stderr, err := runPlugin(t, "yaml-frontmatter")
	if err == nil {
		t.Error("expected non-zero exit code for direct invocation without --capabilities")
	}
	if !strings.Contains(stderr, "Usage") {
		t.Errorf("expected usage message on stderr, got: %s", stderr)
	}
}

func TestOpenNoArgs(t *testing.T) {
	_, stderr, err := runOpen(t)
	if err == nil {
		t.Error("expected non-zero exit code for open with no args")
	}
	if !strings.Contains(stderr, "Usage") {
		t.Errorf("expected usage message on stderr, got: %s", stderr)
	}
}

func TestOpenWithHandler(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test handler script that writes a marker file
	markerFile := filepath.Join(tmpDir, "opened.txt")
	handlerScript := filepath.Join(tmpDir, "test-open-handler")
	err := os.WriteFile(handlerScript, []byte(fmt.Sprintf("#!/bin/sh\necho \"$1\" > %s\n", markerFile)), 0755)
	if err != nil {
		t.Fatalf("failed to create handler script: %v", err)
	}

	// Install handler: create mimetype/text/markdown/open -> handlerScript
	mimetypeDir := filepath.Join(testDir, "mimetype", "text", "markdown")
	openLink := filepath.Join(mimetypeDir, "open")

	// mimetypeDir should already exist from plugin installation
	if err := os.Symlink(handlerScript, openLink); err != nil {
		t.Fatalf("failed to create open symlink: %v", err)
	}
	defer os.Remove(openLink)

	// Create a test markdown file
	testFile := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(testFile, []byte("# Hello\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, stderr, err := runOpen(t, testFile)
	if err != nil {
		t.Fatalf("open failed: %v, stderr: %s", err, stderr)
	}

	// Verify the handler was executed
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("marker file not created — handler was not executed: %v", err)
	}
	if strings.TrimSpace(string(content)) != testFile {
		t.Errorf("handler received wrong path: got %q, expected %q", strings.TrimSpace(string(content)), testFile)
	}
}

func TestOpenNoHandler(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with a type that has no open handler
	testFile := filepath.Join(tmpDir, "test.bin")
	err := os.WriteFile(testFile, []byte{0x00, 0x01, 0x02}, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, stderr, err := runOpen(t, testFile)
	if err == nil {
		t.Error("expected non-zero exit for file with no open handler")
	}
	if !strings.Contains(stderr, "no open handler") {
		t.Errorf("expected 'no open handler' in stderr, got: %s", stderr)
	}
}

func TestOpenMultipleFilesMixedSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a handler for text/markdown
	markerFile := filepath.Join(tmpDir, "opened.txt")
	handlerScript := filepath.Join(tmpDir, "test-open-handler")
	err := os.WriteFile(handlerScript, []byte(fmt.Sprintf("#!/bin/sh\necho \"$1\" >> %s\n", markerFile)), 0755)
	if err != nil {
		t.Fatalf("failed to create handler script: %v", err)
	}

	mimetypeDir := filepath.Join(testDir, "mimetype", "text", "markdown")
	openLink := filepath.Join(mimetypeDir, "open")
	if err := os.Symlink(handlerScript, openLink); err != nil {
		t.Fatalf("failed to create open symlink: %v", err)
	}
	defer os.Remove(openLink)

	// Create a markdown file (handler exists)
	mdFile := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(mdFile, []byte("# Hello\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create md file: %v", err)
	}

	// Create a binary file (no handler)
	binFile := filepath.Join(tmpDir, "test.bin")
	err = os.WriteFile(binFile, []byte{0x00, 0x01, 0x02}, 0644)
	if err != nil {
		t.Fatalf("failed to create bin file: %v", err)
	}

	_, stderr, err := runOpen(t, mdFile, binFile)
	if err == nil {
		t.Error("expected non-zero exit when some files have no handler")
	}
	if !strings.Contains(stderr, "no open handler") {
		t.Errorf("expected 'no open handler' in stderr, got: %s", stderr)
	}

	// Verify the markdown file was still opened
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("marker file not created — handler was not executed for md file: %v", err)
	}
	if !strings.Contains(string(content), mdFile) {
		t.Errorf("handler did not receive the md file path")
	}
}

func TestOpenNonexistentFile(t *testing.T) {
	_, stderr, err := runOpen(t, "/nonexistent/path/to/file.md")
	if err == nil {
		t.Error("expected non-zero exit for nonexistent file")
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestOpenHandlerFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a handler that exits with status 1
	handlerScript := filepath.Join(tmpDir, "failing-handler")
	err := os.WriteFile(handlerScript, []byte("#!/bin/sh\nexit 1\n"), 0755)
	if err != nil {
		t.Fatalf("failed to create handler script: %v", err)
	}

	mimetypeDir := filepath.Join(testDir, "mimetype", "text", "markdown")
	openLink := filepath.Join(mimetypeDir, "open")
	if err := os.Symlink(handlerScript, openLink); err != nil {
		t.Fatalf("failed to create open symlink: %v", err)
	}
	defer os.Remove(openLink)

	// Create a markdown file
	testFile := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(testFile, []byte("# Hello\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, stderr, err := runOpen(t, testFile)
	if err == nil {
		t.Error("expected non-zero exit when handler fails")
	}
	if !strings.Contains(stderr, "handler exited with status") {
		t.Errorf("expected 'handler exited with status' in stderr, got: %s", stderr)
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
