package lib

import "testing"

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
