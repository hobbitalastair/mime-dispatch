package lib

import (
	"testing"
)

func TestMergeMetadata(t *testing.T) {
	tests := []struct {
		name     string
		fileMeta map[string][]string
		xattrMap map[string][]string
		expected map[string][]string
	}{
		{
			name:     "empty maps",
			fileMeta: map[string][]string{},
			xattrMap: map[string][]string{},
			expected: map[string][]string{},
		},
		{
			name:     "xattr takes precedence",
			fileMeta: map[string][]string{"key": {"file-value"}},
			xattrMap: map[string][]string{"key": {"xattr-value"}},
			expected: map[string][]string{"key": {"file-value", "xattr-value"}},
		},
		{
			name:     "merge both",
			fileMeta: map[string][]string{"key1": {"value1"}},
			xattrMap: map[string][]string{"key2": {"value2"}},
			expected: map[string][]string{
				"key1": {"value1"},
				"key2": {"value2"},
			},
		},
		{
			name:     "multi-valued keys",
			fileMeta: map[string][]string{"key": {"value1", "value2"}},
			xattrMap: map[string][]string{"key": {"value3"}},
			expected: map[string][]string{"key": {"value1", "value2", "value3"}},
		},
		{
			name:     "multi-valued keys with duplicates",
			fileMeta: map[string][]string{"key": {"value1", "value3"}},
			xattrMap: map[string][]string{"key": {"value3"}},
			expected: map[string][]string{"key": {"value1", "value3"}},
		},
		{
			name:     "mime_type from xattr included",
			fileMeta: map[string][]string{"title": {"File Title"}},
			xattrMap: map[string][]string{"mime_type": {"text/markdown"}, "author": {"Test Author"}},
			expected: map[string][]string{
				"title":     {"File Title"},
				"mime_type": {"text/markdown"},
				"author":    {"Test Author"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMetadata(tt.fileMeta, tt.xattrMap)
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
