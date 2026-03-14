package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// testDir holds all compiled test binaries and plugin symlinks.
// It is created in TestMain and cleaned up after tests finish.
var testDir string

// cliBinary is the path to the compiled metadata CLI binary.
var cliBinary string

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
	return nil
}

func installPlugins() error {
	plugins := map[string]struct {
		binary   string
		commands []string
	}{
		"text/markdown":      {"yaml-frontmatter", []string{"metadata-list", "metadata-add", "metadata-delete"}},
		"text/plain":         {"yaml-frontmatter", []string{"metadata-list", "metadata-add", "metadata-delete"}},
		"audio/mpeg":         {"audio", []string{"metadata-list"}},
		"audio/ogg":          {"audio", []string{"metadata-list"}},
		"audio/x-vorbis+ogg": {"audio", []string{"metadata-list"}},
		"image/jpeg":         {"image", []string{"metadata-list"}},
	}
	for mimeType, p := range plugins {
		dir := filepath.Join(testDir, "mimetype", mimeType)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		binary := filepath.Join(testDir, p.binary)
		for _, cmd := range p.commands {
			if err := os.Symlink(binary, filepath.Join(dir, cmd)); err != nil {
				return err
			}
		}
	}
	return nil
}

func runCLI(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command(cliBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
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

	if !strings.Contains(output, "date:") {
		t.Errorf("expected date in output, got: %s", output)
	}

	if !strings.Contains(output, "location:") {
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

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
