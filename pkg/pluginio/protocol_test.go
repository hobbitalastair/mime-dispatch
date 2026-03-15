package pluginio

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSerializeMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string][]string
		expected map[string][]string
	}{
		{
			name:     "empty",
			metadata: map[string][]string{},
			expected: map[string][]string{},
		},
		{
			name:     "single value",
			metadata: map[string][]string{"key": {"value"}},
			expected: map[string][]string{"key": {"value"}},
		},
		{
			name:     "multi-valued",
			metadata: map[string][]string{"key": {"value1", "value2"}},
			expected: map[string][]string{"key": {"value1", "value2"}},
		},
		{
			name:     "special characters",
			metadata: map[string][]string{"key": {"line1\nline2: \"quoted\""}},
			expected: map[string][]string{"key": {"line1\nline2: \"quoted\""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := SerializeMetadata(tt.metadata)
			if err != nil {
				t.Fatalf("SerializeMetadata error: %v", err)
			}

			roundtrip, err := DeserializeMetadata(serialized)
			if err != nil {
				t.Fatalf("DeserializeMetadata error: %v", err)
			}

			if !reflect.DeepEqual(roundtrip, tt.expected) {
				t.Fatalf("roundtrip mismatch: got %#v, expected %#v", roundtrip, tt.expected)
			}

			if serialized != "" && !strings.HasSuffix(serialized, "\n") {
				t.Fatalf("expected serialized metadata to end with newline, got %q", serialized)
			}
		})
	}
}

func TestSerializeMetadataSortsKeysAndValues(t *testing.T) {
	metadata := map[string][]string{
		"zeta":  {"2", "10", "1"},
		"alpha": {"b", "a"},
	}

	serialized, err := SerializeMetadata(metadata)
	if err != nil {
		t.Fatalf("SerializeMetadata error: %v", err)
	}

	expected := "alpha:\n    - a\n    - b\nzeta:\n    - \"1\"\n    - \"10\"\n    - \"2\"\n"
	if serialized != expected {
		t.Fatalf("unexpected serialization order:\n got: %q\nwant: %q", serialized, expected)
	}
}

func TestDeserializeMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:     "empty",
			input:    "",
			expected: map[string][]string{},
		},
		{
			name:     "single value",
			input:    "key: value\n",
			expected: map[string][]string{"key": {"value"}},
		},
		{
			name:     "multi-valued",
			input:    "key:\n  - value1\n  - value2\n",
			expected: map[string][]string{"key": {"value1", "value2"}},
		},
		{
			name:     "single value as sequence",
			input:    "key:\n  - value\n",
			expected: map[string][]string{"key": {"value"}},
		},
		{
			name:     "empty multi-valued key",
			input:    "tags:\n",
			expected: map[string][]string{"tags": {}},
		},
		{
			name:  "reject nested map",
			input: "obj:\n  a: b\n",
		},
		{
			name:  "reject nested map in sequence",
			input: "list:\n  - a\n  - k: v\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := DeserializeMetadata(tt.input)
			if tt.expected == nil {
				if !errors.Is(err, ErrNonScalarValue) {
					t.Fatalf("expected ErrNonScalarValue, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeserializeMetadata error: %v", err)
			}
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Fatalf("got %#v, expected %#v", actual, tt.expected)
			}
		})
	}
}
