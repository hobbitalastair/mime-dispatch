package pluginio

import (
	"reflect"
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
		})
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := DeserializeMetadata(tt.input)
			if err != nil {
				t.Fatalf("DeserializeMetadata error: %v", err)
			}
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Fatalf("got %#v, expected %#v", actual, tt.expected)
			}
		})
	}
}
