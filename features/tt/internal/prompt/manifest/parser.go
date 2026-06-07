package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseMarkdownWithFrontmatter(data []byte) (frontmatter []byte, body string, hasFrontmatter bool) {
	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(content, "---\n") {
		return nil, content, false
	}

	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) < 3 {
		return nil, content, false
	}

	return []byte(parts[1]), parts[2], true
}

// ParseEntity parses a single YAML/Markdown file and returns an Entity
func ParseEntity(path string) (*Entity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity file: %s: %w", path, err)
	}

	var frontmatterBytes []byte
	var bodyText string
	var hasFrontmatter bool

	if filepath.Ext(path) == ".md" {
		frontmatterBytes, bodyText, hasFrontmatter = parseMarkdownWithFrontmatter(data)
		if !hasFrontmatter {
			return nil, fmt.Errorf("%s: markdown file must start with frontmatter (---)", path)
		}
	} else {
		frontmatterBytes = data
	}

	// Parse into raw map for schema validation
	var raw map[string]any
	if err := yaml.Unmarshal(frontmatterBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %s: %w", path, err)
	}

	if hasFrontmatter {
		if strings.TrimSpace(bodyText) != "" {
			raw["body"] = bodyText
		}
	}

	// Parse into Entity struct
	var entity Entity
	if err := yaml.Unmarshal(frontmatterBytes, &entity); err != nil {
		return nil, fmt.Errorf("failed to parse entity: %s: %w", path, err)
	}

	entity.FilePath = path
	entity.Raw = raw

	if entity.ID == "" {
		return nil, fmt.Errorf("%s: id is required", path)
	}
	if entity.Kind == "" {
		return nil, fmt.Errorf("%s: kind is required", path)
	}

	return &entity, nil
}

// ExpandGlob returns a list of file paths matching a glob pattern
// Uses filepath.WalkDir for expansion since filepath.Glob does not support double-star
func ExpandGlob(rootDir, pattern string) ([]string, error) {
	// Split pattern at **
	// e.g. "testdata/valid/policies/**/*.yaml" -> base="testdata/valid/policies", suffix="*.yaml"
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		// No ** in pattern, use simple glob
		fullPattern := filepath.Join(rootDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %s: %w", pattern, err)
		}
		return matches, nil
	}

	baseDir := filepath.Join(rootDir, filepath.Clean(parts[0]))
	suffix := strings.TrimPrefix(parts[1], "/")
	suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil // No match, return empty (not an error per R6)
	}

	var matches []string
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Match the filename against the suffix pattern
		matched, matchErr := filepath.Match(suffix, d.Name())
		if matchErr != nil {
			return matchErr
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", baseDir, err)
	}

	return matches, nil
}

// ParseAllEntities parses all entities from the sources section of project.yaml
func ParseAllEntities(cfg *ProjectConfig, rootDir string) ([]*Entity, []ValidationError) {
	var entities []*Entity
	var errors []ValidationError

	for sourceName, pattern := range cfg.Sources {
		// Skip non-entity sources (memory docs are handled separately)
		if sourceName == "memory_docs" || sourceName == "memory_config" {
			continue
		}
		// All other sources (policies, procedures, capabilities, refs, guards,
		// workers, bundles, targets) are parsed as entities.

		files, err := ExpandGlob(rootDir, pattern)
		if err != nil {
			errors = append(errors, ValidationError{
				File:    pattern,
				Message: fmt.Sprintf("glob expansion failed for source '%s': %v", sourceName, err),
			})
			continue
		}

		for _, file := range files {
			entity, err := ParseEntity(file)
			if err != nil {
				errors = append(errors, ValidationError{
					File:    file,
					Message: err.Error(),
				})
				continue
			}
			entities = append(entities, entity)
		}
	}

	return entities, errors
}
