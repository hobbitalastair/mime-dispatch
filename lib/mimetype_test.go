package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectMimetype(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("Hello, World!"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mimeType, err := DetectMimetype(testFile)
	if err != nil {
		t.Fatalf("DetectMimetype failed: %v", err)
	}

	if !strings.HasPrefix(mimeType, "text/") {
		t.Errorf("expected text/* mime type, got %s", mimeType)
	}
}

func TestDetectMimetype_NonexistentFile(t *testing.T) {
	mimeType, err := DetectMimetype("/nonexistent/file/that/does/not/exist.txt")
	// mimetype command may or may not error on nonexistent files depending on version
	// If it doesn't error, it typically returns inode/x-empty or similar
	if err == nil && mimeType == "" {
		t.Error("expected either error or mime type for nonexistent file")
	}
	// Test passes if we get an error OR a mime type (some versions of mimetype
	// detect the type based on extension even for nonexistent files)
}

func TestGetMimeType_FromXattr(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	err := os.WriteFile(testFile, []byte("# Test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Set mime type in xattr
	expectedMime := "text/markdown"
	err = SetXattr(testFile, "mime_type", expectedMime)
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("failed to set xattr: %v", err)
	}

	if err == ErrXattrNotSupported {
		t.Skip("xattr not supported on this system")
	}

	// getMimeType should read from xattr without calling mimetype command
	mimeType, err := getMimeType(testFile, Options{})
	if err != nil {
		t.Fatalf("getMimeType failed: %v", err)
	}

	if mimeType != expectedMime {
		t.Errorf("expected %s from xattr, got %s", expectedMime, mimeType)
	}
}

func TestGetMimeType_DetectAndCache(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// First call should detect and cache
	mimeType, err := getMimeType(testFile, Options{})
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("getMimeType failed: %v", err)
	}

	if err == ErrXattrNotSupported {
		t.Skip("xattr not supported on this system")
	}

	if mimeType == "" {
		t.Error("expected mime type to be detected")
	}

	// Verify it was cached in xattr
	cachedMime, err := GetXattrValue(testFile, "mime_type")
	if err != nil {
		t.Fatalf("failed to read xattr: %v", err)
	}

	if cachedMime != mimeType {
		t.Errorf("expected mime type %s to be cached in xattr, got %s", mimeType, cachedMime)
	}
}

func TestGetMimeType_FileOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("Test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Set a different mime type in xattr
	err = SetXattr(testFile, "mime_type", "application/fake")
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("failed to set xattr: %v", err)
	}

	// With FileOnly, should detect from file, not read xattr
	mimeType, err := getMimeType(testFile, Options{FileOnly: true})
	if err != nil {
		t.Fatalf("getMimeType with FileOnly failed: %v", err)
	}

	// Should be text/*, not the fake mime type from xattr
	if !strings.HasPrefix(mimeType, "text/") {
		t.Errorf("expected text/* mime type with FileOnly, got %s", mimeType)
	}
}
