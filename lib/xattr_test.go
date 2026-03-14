package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/xattr"
)

func TestXattrKeyToDisplayName(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantOk   bool
	}{
		{"user.mime_type", "mime_type", true},
		{"user.metadata.date", "date", true},
		{"user.metadata.author", "author", true},
		{"user.metadata.title", "title", true},
		{"security.selinux", "", false},
		{"system.posix_acl_access", "", false},
		{"trusted.glusterfs.volume-id", "", false},
		{"user.unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			name, ok := xattrKeyToDisplayName(tt.key)
			if ok != tt.wantOk {
				t.Errorf("xattrKeyToDisplayName(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if name != tt.wantName {
				t.Errorf("xattrKeyToDisplayName(%q) = %q, want %q", tt.key, name, tt.wantName)
			}
		})
	}
}

func TestDisplayNameToXattrKey(t *testing.T) {
	tests := []struct {
		name    string
		wantKey string
	}{
		{"mime_type", "user.mime_type"},
		{"date", "user.metadata.date"},
		{"author", "user.metadata.author"},
		{"title", "user.metadata.title"},
		{"custom", "user.metadata.custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := displayNameToXattrKey(tt.name)
			if key != tt.wantKey {
				t.Errorf("displayNameToXattrKey(%q) = %q, want %q", tt.name, key, tt.wantKey)
			}
		})
	}
}

func TestEncodeCSV(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "single value",
			values: []string{"value1"},
			want:   "value1",
		},
		{
			name:   "multiple values",
			values: []string{"value1", "value2", "value3"},
			want:   "value1,value2,value3",
		},
		{
			name:   "value with comma",
			values: []string{"val,ue1", "value2"},
			want:   `"val,ue1",value2`,
		},
		{
			name:   "value with quotes",
			values: []string{"val\"ue1", "value2"},
			want:   `"val""ue1",value2`,
		},
		{
			name:   "empty list",
			values: []string{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeCSV(tt.values)
			if got != tt.want {
				t.Errorf("encodeCSV(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestDecodeCSV(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{
			name:  "single value",
			value: "value1",
			want:  []string{"value1"},
		},
		{
			name:  "multiple values",
			value: "value1,value2,value3",
			want:  []string{"value1", "value2", "value3"},
		},
		{
			name:  "value with comma",
			value: `"val,ue1",value2`,
			want:  []string{"val,ue1", "value2"},
		},
		{
			name:  "value with quotes",
			value: `"val""ue1",value2`,
			want:  []string{"val\"ue1", "value2"},
		},
		{
			name:  "empty value",
			value: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeCSV(tt.value)
			if len(got) != len(tt.want) {
				t.Errorf("decodeCSV(%q) = %v, want %v", tt.value, got, tt.want)
				return
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("decodeCSV(%q)[%d] = %q, want %q", tt.value, i, got[i], v)
				}
			}
		})
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{
			name:   "single value",
			values: []string{"value1"},
		},
		{
			name:   "multiple values",
			values: []string{"value1", "value2", "value3"},
		},
		{
			name:   "value with comma",
			values: []string{"val,ue1", "value2"},
		},
		{
			name:   "value with quotes",
			values: []string{"val\"ue1", "value2"},
		},
		{
			name:   "mixed special chars",
			values: []string{"val,ue\"1", "val\nue2", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeCSV(tt.values)
			decoded := decodeCSV(encoded)
			if len(decoded) != len(tt.values) {
				t.Errorf("roundtrip failed: got %d values, want %d", len(decoded), len(tt.values))
				return
			}
			for i, v := range tt.values {
				if decoded[i] != v {
					t.Errorf("roundtrip failed at index %d: got %q, want %q", i, decoded[i], v)
				}
			}
		})
	}
}

func TestXattrKeyToDisplayName_NamespaceFiltering(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantOk   bool
	}{
		// Valid namespaces
		{"user.mime_type", "mime_type", true},
		{"user.metadata.title", "title", true},
		{"user.metadata.author", "author", true},
		{"user.metadata.custom_key", "custom_key", true},

		// Invalid namespaces (should be ignored)
		{"security.selinux", "", false},
		{"system.posix_acl_access", "", false},
		{"trusted.glusterfs.volume-id", "", false},
		{"user.other_namespace", "", false},
		{"user.metadata", "", false}, // No key after namespace

		// Edge cases
		{"", "", false},
		{"user.", "", false},
		{"user.metadata.", "", true}, // Empty key name - implementation accepts this
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			name, ok := xattrKeyToDisplayName(tt.key)
			if ok != tt.wantOk {
				t.Errorf("xattrKeyToDisplayName(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if name != tt.wantName {
				t.Errorf("xattrKeyToDisplayName(%q) = %q, want %q", tt.key, name, tt.wantName)
			}
		})
	}
}

func TestGetXattr_OnlyReturnsMetadataNamespace(t *testing.T) {
	if !xattr.XATTR_SUPPORTED {
		t.Skip("xattr not supported")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Set valid metadata
	err = SetXattr(testFile, "title", "Test Title")
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("failed to set xattr: %v", err)
	}

	if err == ErrXattrNotSupported {
		t.Skip("xattr not supported on this filesystem")
	}

	// Try to set an xattr outside our namespaces (may fail due to permissions, that's ok)
	// This tests that even if such attributes exist, we don't return them
	xattr.Set(testFile, "user.other", []byte("other value"))

	// Get metadata
	metadata, err := GetXattr(testFile)
	if err != nil {
		t.Fatalf("GetXattr failed: %v", err)
	}

	// Should only have title and mime_type (if set)
	if _, ok := metadata["title"]; !ok {
		t.Error("expected title in metadata")
	}

	// Should NOT have "other" since it's not in our namespace
	if _, ok := metadata["other"]; ok {
		t.Error("expected 'other' to be filtered out (not in metadata namespace)")
	}
}

func TestDisplayNameToXattrKey_Consistency(t *testing.T) {
	tests := []struct {
		displayName string
		expectedKey string
	}{
		{"mime_type", "user.mime_type"},
		{"title", "user.metadata.title"},
		{"author", "user.metadata.author"},
		{"custom_key", "user.metadata.custom_key"},
		{"key.with.dots", "user.metadata.key.with.dots"},
	}

	for _, tt := range tests {
		t.Run(tt.displayName, func(t *testing.T) {
			key := displayNameToXattrKey(tt.displayName)
			if key != tt.expectedKey {
				t.Errorf("displayNameToXattrKey(%q) = %q, want %q", tt.displayName, key, tt.expectedKey)
			}

			// Verify round-trip
			name, ok := xattrKeyToDisplayName(key)
			if !ok {
				t.Errorf("xattrKeyToDisplayName(%q) failed, expected ok=true", key)
			}
			if name != tt.displayName {
				t.Errorf("round-trip failed: got %q, want %q", name, tt.displayName)
			}
		})
	}
}

func TestDeleteXattr_MultipleValues(t *testing.T) {
	if !xattr.XATTR_SUPPORTED {
		t.Skip("xattr not supported")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Set multiple values
	values := []string{"value1", "value2", "value3"}
	encoded := encodeCSV(values)
	err = SetXattr(testFile, "tags", encoded)
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("failed to set xattr: %v", err)
	}

	if err == ErrXattrNotSupported {
		t.Skip("xattr not supported on this filesystem")
	}

	// Delete one value
	err = DeleteXattr(testFile, "tags", "value2")
	if err != nil {
		t.Fatalf("DeleteXattr failed: %v", err)
	}

	// Check remaining values
	rawValue, err := GetXattrValue(testFile, "tags")
	if err != nil {
		t.Fatalf("GetXattrValue failed: %v", err)
	}

	remaining := decodeCSV(rawValue)
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining values, got %d: %v", len(remaining), remaining)
	}

	if remaining[0] != "value1" || remaining[1] != "value3" {
		t.Errorf("expected [value1, value3], got %v", remaining)
	}
}

func TestDeleteXattr_LastValue(t *testing.T) {
	if !xattr.XATTR_SUPPORTED {
		t.Skip("xattr not supported")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Set single value
	err = SetXattr(testFile, "key", "value")
	if err != nil && err != ErrXattrNotSupported {
		t.Fatalf("failed to set xattr: %v", err)
	}

	if err == ErrXattrNotSupported {
		t.Skip("xattr not supported on this filesystem")
	}

	// Delete the only value
	err = DeleteXattr(testFile, "key", "value")
	if err != nil {
		t.Fatalf("DeleteXattr failed: %v", err)
	}

	// Xattr should be completely removed
	rawValue, err := GetXattrValue(testFile, "key")
	if err != nil {
		t.Fatalf("GetXattrValue failed: %v", err)
	}

	if rawValue != "" {
		t.Errorf("expected xattr to be removed, got value: %q", rawValue)
	}
}
