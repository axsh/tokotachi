package knowledge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const categoryMetaFile = "_category.yaml"
const frontmatterDelimiter = "---"

// ReadCategoryMeta reads _category.yaml from the given directory path.
func ReadCategoryMeta(dirPath string) (*CategoryMeta, error) {
	p := filepath.Join(dirPath, categoryMetaFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read category meta: %w", err)
	}
	var meta CategoryMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse category meta: %w", err)
	}
	return &meta, nil
}

// WriteCategoryMeta writes _category.yaml to the given directory path.
func WriteCategoryMeta(dirPath string, meta *CategoryMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal category meta: %w", err)
	}
	p := filepath.Join(dirPath, categoryMetaFile)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("failed to write category meta: %w", err)
	}
	return nil
}

// ReadFrontmatter reads YAML frontmatter from a markdown file.
// Returns the parsed metadata, the body content after the frontmatter, and any error.
func ReadFrontmatter(path string) (*KnowledgeFileMeta, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Expect first line to be ---
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != frontmatterDelimiter {
		return nil, "", fmt.Errorf("file does not start with frontmatter delimiter")
	}

	// Read until closing ---
	var fmLines []string
	foundEnd := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == frontmatterDelimiter {
			foundEnd = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !foundEnd {
		return nil, "", fmt.Errorf("frontmatter not terminated")
	}

	// Parse YAML frontmatter
	fmData := strings.Join(fmLines, "\n")
	var meta KnowledgeFileMeta
	if err := yaml.Unmarshal([]byte(fmData), &meta); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Read remaining body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading file: %w", err)
	}

	body := strings.Join(bodyLines, "\n")
	// Trim leading blank line that typically separates frontmatter from body
	body = strings.TrimLeft(body, "\n")

	return &meta, body, nil
}

// WriteFrontmatter writes YAML frontmatter + body to a markdown file.
func WriteFrontmatter(path string, meta *KnowledgeFileMeta, body string) error {
	fmData, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(frontmatterDelimiter + "\n")
	sb.Write(fmData)
	sb.WriteString(frontmatterDelimiter + "\n\n")
	sb.WriteString(body)

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
