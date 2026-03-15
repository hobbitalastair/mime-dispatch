package pluginio

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrNonScalarValue = errors.New("metadata values must be strings or sequences of strings")

func SerializeMetadata(metadata map[string][]string) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: make([]*yaml.Node, 0, len(metadata)*2)}
	for _, key := range keys {
		values := metadata[key]
		sortedValues := append([]string(nil), values...)
		slices.Sort(sortedValues)

		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}

		var valueNode *yaml.Node
		if len(sortedValues) == 1 {
			valueNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sortedValues[0]}
		} else {
			valueNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for _, value := range sortedValues {
				valueNode.Content = append(valueNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
			}
		}

		mapping.Content = append(mapping.Content, keyNode, valueNode)
	}

	doc := yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}

	data, err := yaml.Marshal(&doc)
	if err != nil {
		return "", err
	}

	return string(data), nil
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
		case map[string]interface{}:
			return nil, fmt.Errorf("key %q: %w", key, ErrNonScalarValue)
		case []interface{}:
			values := make([]string, len(parsed))
			for i, item := range parsed {
				if _, ok := item.(map[string]interface{}); ok {
					return nil, fmt.Errorf("key %q[%d]: %w", key, i, ErrNonScalarValue)
				}
				values[i] = fmt.Sprintf("%v", item)
			}
			result[key] = values
		default:
			result[key] = []string{fmt.Sprintf("%v", parsed)}
		}
	}

	return result, nil
}
