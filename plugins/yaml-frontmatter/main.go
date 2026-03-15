package main

import (
	"fmt"
	"metadata/pkg/pluginio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// formatValue converts any YAML-decoded value to its string representation.
// This handles types that gopkg.in/yaml.v3 automatically parses from scalars,
// such as time.Time (from dates like 2026-03-05), bool, int, float64, etc.
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case time.Time:
		if val.Hour() == 0 && val.Minute() == 0 && val.Second() == 0 && val.Nanosecond() == 0 {
			return val.Format("2006-01-02")
		}
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}

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
	case "metadata-list":
		if len(args) != 1 {
			usage()
			os.Exit(1)
		}
		err = extractMetadata(args[0])
	case "metadata-add":
		if len(args) < 3 {
			usage()
			os.Exit(1)
		}
		err = addMetadata(args[0], args[1], args[2])
	case "metadata-delete":
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
	fmt.Fprintln(os.Stderr, "  metadata-list <file>")
	fmt.Fprintln(os.Stderr, "  metadata-add <file> <key> <value>")
	fmt.Fprintln(os.Stderr, "  metadata-delete <file> <key> <value>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Available commands are invoked by creating symlinks named metadata-list/metadata-add/metadata-delete pointing to this binary.")
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

	result := make(map[string][]string)
	for k, v := range metadata {
		switch val := v.(type) {
		case []interface{}:
			values := make([]string, 0, len(val))
			for _, item := range val {
				values = append(values, formatValue(item))
			}
			result[k] = values
		default:
			result[k] = []string{formatValue(val)}
		}
	}

	if len(result) > 0 {
		output, err := pluginio.SerializeMetadata(result)
		if err != nil {
			return err
		}
		fmt.Print(output)
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
		case []interface{}:
			// Append to existing list
			metadata[key] = append(v, value)
		default:
			// Convert single value to list and append
			metadata[key] = []interface{}{v, value}
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
		case []interface{}:
			// List - remove matching value
			var newList []interface{}
			found := false
			for _, item := range v {
				if formatValue(item) != value {
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
		default:
			// Single value - if it matches, delete the key
			if formatValue(v) == value {
				delete(metadata, key)
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
