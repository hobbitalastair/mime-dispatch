package lib

import (
	"strings"
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
			result := ParsePluginOutput(tt.output)
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

func TestMetadataToYAML(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected []string // Expected lines (in any order for multiple key case)
	}{
		{
			name:     "empty",
			metadata: Metadata{},
			expected: []string{},
		},
		{
			name:     "single key-value",
			metadata: Metadata{"key": {"value"}},
			expected: []string{"key: value"},
		},
		{
			name:     "multi-valued key",
			metadata: Metadata{"key": {"value1", "value2"}},
			expected: []string{"key:", "  - value1", "  - value2"},
		},
		{
			name:     "multiple keys",
			metadata: Metadata{"key1": {"value1"}, "key2": {"value2"}},
			expected: []string{"key1: value1", "key2: value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.ToYAML()
			resultLines := strings.Split(strings.TrimSpace(result), "\n")
			if len(resultLines) == 1 && resultLines[0] == "" {
				resultLines = []string{}
			}

			if len(resultLines) != len(tt.expected) {
				t.Errorf("expected %d lines, got %d: %v", len(tt.expected), len(resultLines), resultLines)
				return
			}

			// For tests with more than one key, check that all expected lines are present
			if tt.name == "multiple keys" {
				resultStr := strings.Join(resultLines, "\n")
				for _, exp := range tt.expected {
					if !strings.Contains(resultStr, exp) {
						t.Errorf("expected line %q not found in result: %q", exp, resultStr)
					}
				}
			} else {
				// For other tests, check exact match
				for i, exp := range tt.expected {
					if i < len(resultLines) && resultLines[i] != exp {
						t.Errorf("line %d: expected %q, got %q", i, exp, resultLines[i])
					}
				}
			}
		})
	}
}
