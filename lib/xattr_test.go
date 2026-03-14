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
