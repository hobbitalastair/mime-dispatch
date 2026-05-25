package pluginio

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SerializeCapabilities returns the YAML representation of plugin capabilities.
func SerializeCapabilities(mimetypes, commands []string) (string, error) {
	wire := struct {
		Mimetypes []string `yaml:"mimetypes"`
		Commands  []string `yaml:"commands"`
	}{
		Mimetypes: mimetypes,
		Commands:  commands,
	}

	data, err := yaml.Marshal(wire)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializedCapabilities is the data-only form returned by
// DeserializeCapabilities, used by tools like mime-dispatch-install
// that need the declared names without handler functions.
type DeserializedCapabilities struct {
	Mimetypes []string `yaml:"mimetypes"`
	Commands  []string `yaml:"commands"`
}

func DeserializeCapabilities(input string) (DeserializedCapabilities, error) {
	if strings.TrimSpace(input) == "" {
		return DeserializedCapabilities{}, fmt.Errorf("empty capabilities output")
	}

	var caps DeserializedCapabilities
	if err := yaml.Unmarshal([]byte(input), &caps); err != nil {
		return DeserializedCapabilities{}, err
	}

	if len(caps.Mimetypes) == 0 {
		return DeserializedCapabilities{}, fmt.Errorf("capabilities: no mimetypes declared")
	}
	if len(caps.Commands) == 0 {
		return DeserializedCapabilities{}, fmt.Errorf("capabilities: no commands declared")
	}

	return caps, nil
}
