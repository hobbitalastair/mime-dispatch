package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: metadata <command> <file>")
		os.Exit(1)
	}

	command := os.Args[1]
	filePath := os.Args[2]

	key := os.Getenv("METADATA_KEY")
	value := os.Getenv("METADATA_VALUE")

	var err error
	switch command {
	case "EXTRACT":
		err = extractMetadata(filePath)
	case "SET":
		err = setMetadata(filePath, key, value)
	case "DELETE":
		err = deleteMetadata(filePath, key)
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
		fmt.Printf("%s: %s\n", k, v)
	}

	return nil
}

func setMetadata(filePath, key, value string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	frontmatter, body, hasFrontmatter := extractFrontmatter(string(content))

	metadata := make(map[string]string)
	if hasFrontmatter {
		metadata, _ = parseFrontmatter(frontmatter)
	}

	metadata[key] = value

	newContent, err := serializeFrontmatter(metadata, body, hasFrontmatter)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func deleteMetadata(filePath, key string) error {
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

	delete(metadata, key)

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

func parseFrontmatter(frontmatter string) (map[string]string, error) {
	if strings.TrimSpace(frontmatter) == "" {
		return make(map[string]string), nil
	}

	decoder := yaml.NewDecoder(strings.NewReader(frontmatter))
	var result map[string]string
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	if result == nil {
		result = make(map[string]string)
	}

	return result, nil
}

func serializeFrontmatter(metadata map[string]string, body string, hasFrontmatter bool) (string, error) {
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
