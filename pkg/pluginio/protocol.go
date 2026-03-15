package pluginio

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func SerializeMetadata(metadata map[string][]string) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}

	intermediate := make(map[string]interface{}, len(metadata))
	for key, values := range metadata {
		if len(values) == 1 {
			intermediate[key] = values[0]
			continue
		}
		intermediate[key] = values
	}

	data, err := yaml.Marshal(intermediate)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

func DeserializeMetadata(input string) (map[string][]string, error) {
	if strings.TrimSpace(input) == "" {
		return map[string][]string{}, nil
	}

	var intermediate map[string]interface{}
	if err := yaml.Unmarshal([]byte(input), &intermediate); err != nil {
		return nil, err
	}

	if intermediate == nil {
		return map[string][]string{}, nil
	}

	result := make(map[string][]string, len(intermediate))
	for key, value := range intermediate {
		if value == nil {
			result[key] = []string{}
			continue
		}

		switch parsed := value.(type) {
		case []interface{}:
			values := make([]string, len(parsed))
			for i, item := range parsed {
				values[i] = fmt.Sprintf("%v", item)
			}
			result[key] = values
		default:
			result[key] = []string{fmt.Sprintf("%v", parsed)}
		}
	}

	return result, nil
}
