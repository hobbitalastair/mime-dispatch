package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func main() {
	command := filepath.Base(os.Args[0])

	flagSet := pflag.NewFlagSet(command, pflag.ContinueOnError)
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(1)
	}

	args := flagSet.Args()
	var err error
	switch command {
	case "list":
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}
		err = extractMetadata(args[0])
	case "add":
		if len(args) < 3 {
			usage()
			os.Exit(1)
		}
		err = addMetadata(args[0], args[1], args[2])
	case "delete":
		if len(args) < 3 {
			usage()
			os.Exit(1)
		}
		err = deleteMetadata(args[0], args[1], args[2])
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage (run via a command-specific symlink):")
	fmt.Fprintln(os.Stderr, "  list <file>")
	fmt.Fprintln(os.Stderr, "  add <file> <key> <value>")
	fmt.Fprintln(os.Stderr, "  delete <file> <key> <value>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Available commands are invoked by creating symlinks named list/add/delete pointing to this binary.")
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
		parsed, err := parseFrontmatter(frontmatter)
		if err != nil {
			return err
		}
		metadata = parsed
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
				} else {
					// Keep as list even if only one element remains
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
	bodyStart := -1

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			bodyStart = i + 1
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	// If no closing --- found, this is not valid frontmatter
	if bodyStart == -1 {
		return "", content, false
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

	// Always include closing --- separator
	return "---\n" + string(data) + "---\n" + body, nil
}
