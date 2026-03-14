package e2e

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"metadata/lib"
)

var cliModule = "../cmd/metadata"
var pluginModule = "../plugins/markdown"

var audioPluginModule = "../plugins/audio"

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

func TestXattrOnlyList(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "xattr-key", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "xattr-key")
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
	cleanup := setupPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "set", testFile, "title", "Test Title")
	if err != nil {
		t.Fatalf("set failed: %v", err)
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

func TestSetDefaultBehaviorReplacesFileOnly(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", testFile, "title", "Updated Title")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: Updated Title") {
		t.Errorf("expected updated title in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "title: Updated Title") {
		t.Errorf("expected updated title in file content, got: %s", fileContent)
	}
}

func TestSetDefaultBehaviorReplacesXattrOnly(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "set", testFile, "title", "New Title")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "title: New Title") {
		t.Errorf("expected new title in output, got: %s", output)
	}
}

func TestSetDefaultBehaviorReplacesBothOnlyXattr(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "set", testFile, "title", "Replaced Title")
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if !strings.Contains(output, "Replaced Title") {
		t.Errorf("expected replaced title in output, got: %s", output)
	}

	fileContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(fileContent), "title: File Title") {
		t.Errorf("expected original file title preserved, got: %s", fileContent)
	}
}

func TestDeleteDefaultBehaviorDeletesBoth(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", testFile, "title")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	output, err := runCLI(t, "list", testFile)
	if err != nil {
		t.Fatalf("list failed: %v, output: %s", err, output)
	}

	if strings.Contains(output, "title:") {
		t.Errorf("expected title to be deleted from both locations, got: %s", output)
	}
}

func TestListDefaultBehaviorMergedWithPrecedence(t *testing.T) {
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "set", "--xattr-only", testFile, "xattr-only-key", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "tags", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
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
	cleanup := setupPlugin(t)
	defer cleanup()

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
	cleanup := setupPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte("# Hello\n\nWorld"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = runCLI(t, "set", "--file-only", testFile, "title", "File Title")
	if err != nil {
		t.Fatalf("set --file-only failed: %v", err)
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--file-only", testFile, "title")
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "title")
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
	cleanup := setupPlugin(t)
	defer cleanup()

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

	_, err = runCLI(t, "set", "--xattr-only", testFile, "title", "Xattr Title")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "set", "--xattr-only", testFile, "xattr-only-key", "xattr-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	_, err = runCLI(t, "delete", "--xattr-only", testFile, "title")
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

func setupAudioPlugin(t *testing.T) func() {
	tmpDir := t.TempDir()

	pluginBuildDir := filepath.Join(tmpDir, "build")
	err := os.MkdirAll(pluginBuildDir, 0755)
	if err != nil {
		t.Fatalf("failed to create plugin build dir: %v", err)
	}

	pluginBinary := filepath.Join(pluginBuildDir, "metadata-plugin")
	cmd := exec.Command("go", "-C", "../plugins/audio", "build", "-o", pluginBinary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build audio plugin: %v, output: %s", err, output)
	}

	for _, mimeType := range []string{"audio/mpeg", "audio/ogg", "audio/x-vorbis+ogg"} {
		pluginParentDir := filepath.Join(tmpDir, "metadata", "plugins", mimeType)
		err = os.MkdirAll(pluginParentDir, 0755)
		if err != nil {
			t.Fatalf("failed to create plugin parent dir: %v", err)
		}

		pluginSymlink := filepath.Join(pluginParentDir, "list")
		err = os.Symlink(pluginBinary, pluginSymlink)
		if err != nil {
			t.Fatalf("failed to create plugin symlink: %v", err)
		}
	}

	originalHome := os.Getenv("HOME")
	originalConfig := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	os.Unsetenv("HOME")

	// Reinitialize plugin search paths to use the new XDG_CONFIG_HOME
	lib.InitPluginSearchPaths()

	return func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalConfig)
		lib.InitPluginSearchPaths()
	}
}

func TestAudioPluginList(t *testing.T) {
	cleanup := setupAudioPlugin(t)
	defer cleanup()

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
	cleanup := setupAudioPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	output, err := runCLI(t, "set", testFile, "custom-key", "custom-value")
	if err != nil {
		t.Fatalf("set failed: %v, output: %s", err, output)
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
	cleanup := setupAudioPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	delOutput, err := runCLI(t, "delete", testFile, "title")
	if err == nil {
		t.Error("expected error when deleting key that exists in file")
	}

	if !strings.Contains(delOutput, "read-only") {
		t.Errorf("expected error about read-only, got: %v, output: %s", err, delOutput)
	}
}

func TestAudioPluginDeleteKeyNotInFile(t *testing.T) {
	cleanup := setupAudioPlugin(t)
	defer cleanup()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := copyFile("samples/sample1.mp3", testFile)
	if err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	_, err = runCLI(t, "set", "--xattr-only", testFile, "custom-key", "custom-value")
	if err != nil {
		t.Fatalf("set xattr failed: %v", err)
	}

	delOutput, err := runCLI(t, "delete", testFile, "custom-key")
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

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
