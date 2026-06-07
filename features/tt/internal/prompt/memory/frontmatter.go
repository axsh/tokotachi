package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// ParseFrontmatter parses frontmatter from a Markdown file into a MemoryDoc
func ParseFrontmatter(path string) (*manifest.MemoryDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", path, err)
	}

	// Use goldmark with meta extension to parse frontmatter
	md := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	ctx := parser.NewContext()
	reader := text.NewReader(data)
	md.Parser().Parse(reader, parser.WithContext(ctx))

	metaData := meta.Get(ctx)
	if metaData == nil || len(metaData) == 0 {
		return nil, fmt.Errorf("%s: no frontmatter found", path)
	}

	doc := &manifest.MemoryDoc{
		FilePath: path,
	}

	// Extract required fields
	id, ok := metaData["id"]
	if !ok {
		return nil, fmt.Errorf("%s: required field 'id' is missing in frontmatter", path)
	}
	doc.ID = fmt.Sprintf("%v", id)

	if kind, ok := metaData["kind"]; ok {
		doc.Kind = fmt.Sprintf("%v", kind)
	}

	title, ok := metaData["title"]
	if !ok {
		return nil, fmt.Errorf("%s: required field 'title' is missing in frontmatter", path)
	}
	doc.Title = fmt.Sprintf("%v", title)

	status, ok := metaData["status"]
	if !ok {
		return nil, fmt.Errorf("%s: required field 'status' is missing in frontmatter", path)
	}
	doc.Status = fmt.Sprintf("%v", status)

	// Extract optional fields
	if topics, ok := metaData["topics"]; ok {
		doc.Topics = toStringSlice(topics)
	}
	if triggers, ok := metaData["triggers"]; ok {
		doc.Triggers = toStringSlice(triggers)
	}
	if dependsOn, ok := metaData["depends_on"]; ok {
		doc.DependsOn = toStringSlice(dependsOn)
	}
	if evidence, ok := metaData["evidence"]; ok {
		doc.Evidence = toStringSlice(evidence)
	}
	if lastReviewed, ok := metaData["last_reviewed"]; ok {
		doc.LastReviewed = fmt.Sprintf("%v", lastReviewed)
	}

	// Validate status
	if err := doc.ValidateStatus(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return doc, nil
}

// ParseAllMemoryDocs parses frontmatter from all Markdown files matching the glob pattern.
// Generated files (containing "GENERATED FILE" banner) are skipped.
func ParseAllMemoryDocs(rootDir, pattern string) ([]*manifest.MemoryDoc, []manifest.ValidationError) {
	files, err := manifest.ExpandGlob(rootDir, pattern)
	if err != nil {
		return nil, []manifest.ValidationError{
			{File: pattern, Message: fmt.Sprintf("glob expansion failed: %v", err)},
		}
	}

	var docs []*manifest.MemoryDoc
	var errors []manifest.ValidationError

	for _, file := range files {
		// Skip non-memory paths (var/, schemas/) before reading file content
		if shouldSkipMemoryPath(file) {
			continue
		}

		// Skip generated files
		if isGeneratedFile(file) {
			continue
		}

		doc, err := ParseFrontmatter(file)
		if err != nil {
			errors = append(errors, manifest.ValidationError{
				File:    file,
				Message: err.Error(),
			})
			continue
		}
		docs = append(docs, doc)
	}

	return docs, errors
}

// shouldSkipMemoryPath returns true if the path should be excluded from
// memory document parsing. Paths under var/ and schemas/ are excluded.
func shouldSkipMemoryPath(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/var/") ||
		strings.Contains(normalized, "/schemas/")
}

// isGeneratedFile checks if a file contains the generated file banner
func isGeneratedFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "GENERATED FILE -- DO NOT EDIT")
}

// toStringSlice converts an interface{} to a string slice
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case []string:
		return val
	default:
		return nil
	}
}
