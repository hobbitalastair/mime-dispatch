package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func main() {
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: metadata-markdown <command> <file> [args...]")
		fmt.Fprintln(os.Stderr, "Commands: list, add, delete")
		fmt.Fprintln(os.Stderr, "")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	args := pflag.Args()

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: metadata-markdown <command> <file> [args...]")
		fmt.Fprintln(os.Stderr, "Commands: list, add, delete")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	command := args[0]
	filePath := args[1]
	remainingArgs := args[2:]

	var err error

	switch command {
	case "list":
		err = extractMetadata(filePath)
	case "add":
		if len(remainingArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: metadata-markdown add <file> <key> <value>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		value := remainingArgs[1]
		err = addMetadata(filePath, key, value)
	case "delete":
		if len(remainingArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: metadata-markdown delete <file> <key> <value>")
			os.Exit(1)
		}
		key := remainingArgs[0]
		value := remainingArgs[1]
		err = deleteMetadata(filePath, key, value)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func extractMetadata(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	frontmatter, _, hasFrontmatter := extractFrontmatter(string(content))
	if !hasFrontmatter {
		return nil
	}

	metadata, err := parseFrontmatter(frontmatter)
	if err != nil {
		return err
	}

	for k, v := range metadata {
		switch val := v.(type) {
		case string:
			fmt.Printf("%s: %s\n", k, val)
		case []interface{}:
			fmt.Printf("%s:\n", k)
			for _, item := range val {
				fmt.Printf("  - %s\n", item)
			}
		}
	}

	return nil
}

func addMetadata(filePath, key, value string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	frontmatter, body, hasFrontmatter := extractFrontmatter(string(content))

	metadata := make(map[string]interface{})
	if hasFrontmatter {
		metadata, _ = parseFrontmatter(frontmatter)
	}

	// If key already exists, convert to list and append
	if existing, ok := metadata[key]; ok {
		switch v := existing.(type) {
		case string:
			// Convert single value to list and append
			metadata[key] = []interface{}{v, value}
		case []interface{}:
			// Append to existing list
			metadata[key] = append(v, value)
		}
	} else {
		// New key
		metadata[key] = value
	}

	newContent, err := serializeFrontmatter(metadata, body, hasFrontmatter)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func deleteMetadata(filePath, key, value string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	frontmatter, body, hasFrontmatter := extractFrontmatter(string(content))
	if !hasFrontmatter {
		return nil
	}

	metadata, err := parseFrontmatter(frontmatter)
	if err != nil {
		return err
	}

	if existing, ok := metadata[key]; ok {
		switch v := existing.(type) {
		case string:
			// Single value - if it matches, delete the key
			if v == value {
				delete(metadata, key)
			}
		case []interface{}:
			// List - remove matching value
			var newList []interface{}
			found := false
			for _, item := range v {
				if fmt.Sprintf("%v", item) != value {
					newList = append(newList, item)
				} else {
					found = true
				}
			}
			if found {
				if len(newList) == 0 {
					delete(metadata, key)
				} else if len(newList) == 1 {
					// Convert back to single value
					metadata[key] = newList[0]
				} else {
					metadata[key] = newList
				}
			}
		}
	}

	newContent, err := serializeFrontmatter(metadata, body, hasFrontmatter)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func extractFrontmatter(content string) (string, string, bool) {
	lines := strings.Split(content, "\n")

	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", content, false
	}

	var frontmatterLines []string
	bodyStart := 1

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			bodyStart = i + 1
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")
	body := strings.Join(lines[bodyStart:], "\n")

	return frontmatter, body, true
}

func parseFrontmatter(frontmatter string) (map[string]interface{}, error) {
	if strings.TrimSpace(frontmatter) == "" {
		return make(map[string]interface{}), nil
	}

	decoder := yaml.NewDecoder(strings.NewReader(frontmatter))
	var result map[string]interface{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	if result == nil {
		result = make(map[string]interface{})
	}

	return result, nil
}

func serializeFrontmatter(metadata map[string]interface{}, body string, hasFrontmatter bool) (string, error) {
	if len(metadata) == 0 {
		if hasFrontmatter {
			return strings.TrimPrefix(body, "\n"), nil
		}
		return body, nil
	}

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return "", err
	}

	frontmatter := string(data)

	if hasFrontmatter {
		return "---\n" + frontmatter + body, nil
	}

	return "---\n" + frontmatter + "---\n\n" + body, nil
}
