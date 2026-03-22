package pluginio

import (
	"reflect"
	"testing"
)

func TestSerializeDeserializeCapabilitiesRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		mimetypes []string
		commands  []string
	}{
		{
			name:      "single mimetype and command",
			mimetypes: []string{"text/markdown"},
			commands:  []string{"metadata-list"},
		},
		{
			name:      "multiple mimetypes and commands",
			mimetypes: []string{"audio/mpeg", "audio/ogg", "audio/x-vorbis+ogg"},
			commands:  []string{"metadata-list", "metadata-add", "metadata-delete"},
		},
		{
			name:      "open command",
			mimetypes: []string{"image/jpeg"},
			commands:  []string{"open"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := SerializeCapabilities(tt.mimetypes, tt.commands)
			if err != nil {
				t.Fatalf("SerializeCapabilities error: %v", err)
			}

			caps, err := DeserializeCapabilities(serialized)
			if err != nil {
				t.Fatalf("DeserializeCapabilities error: %v", err)
			}

			if !reflect.DeepEqual(caps.Mimetypes, tt.mimetypes) {
				t.Errorf("mimetypes: got %v, want %v", caps.Mimetypes, tt.mimetypes)
			}
			if !reflect.DeepEqual(caps.Commands, tt.commands) {
				t.Errorf("commands: got %v, want %v", caps.Commands, tt.commands)
			}
		})
	}
}

func TestDeserializeCapabilities_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "whitespace only",
			input: "   \n\n  ",
		},
		{
			name:  "no mimetypes",
			input: "commands:\n    - metadata-list\n",
		},
		{
			name:  "no commands",
			input: "mimetypes:\n    - text/markdown\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeCapabilities(tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
