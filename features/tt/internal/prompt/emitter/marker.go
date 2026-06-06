package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MarkerBegin is the start marker for the auto-managed section.
	MarkerBegin = "<!-- AGENT-MANAGED:BEGIN -->"
	// MarkerEnd is the end marker for the auto-managed section.
	MarkerEnd = "<!-- AGENT-MANAGED:END -->"
)

// ReplaceMarkerSection replaces content between AGENT-MANAGED markers in a file.
// Behavior:
//   - File does not exist: create file with marker section only
//   - File exists but no markers: append marker section at end
//   - Markers exist: replace content between markers (inclusive), preserve everything outside
//
// The newContent string should NOT include the markers themselves; they are added automatically.
func ReplaceMarkerSection(filePath string, newContent string) error {
	var existing string

	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		// File does not exist - start with empty content
		existing = ""
	} else {
		existing = string(data)
	}

	block := buildMarkerBlock(newContent)

	beginIdx := strings.Index(existing, MarkerBegin)
	endIdx := strings.Index(existing, MarkerEnd)

	var result string
	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		// Both markers found - replace the section (inclusive of markers)
		before := existing[:beginIdx]
		after := existing[endIdx+len(MarkerEnd):]
		result = before + block + after
	} else {
		// No markers found - append to end
		if existing == "" {
			result = block + "\n"
		} else {
			result = existing + "\n\n" + block + "\n"
		}
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(filePath, []byte(result), 0644)
}

// ExtractMarkerSection extracts the content between AGENT-MANAGED markers.
// Returns the content between markers and true if found, or empty string and false if not.
func ExtractMarkerSection(content string) (string, bool) {
	beginIdx := strings.Index(content, MarkerBegin)
	endIdx := strings.Index(content, MarkerEnd)

	if beginIdx < 0 || endIdx < 0 || endIdx <= beginIdx {
		return "", false
	}

	// Extract content after the MarkerBegin line
	sectionStart := beginIdx + len(MarkerBegin)
	// Skip the newline after MarkerBegin if present
	if sectionStart < len(content) && content[sectionStart] == '\n' {
		sectionStart++
	}

	section := content[sectionStart:endIdx]
	return section, true
}

// buildMarkerBlock constructs the full marker block with warning comments.
func buildMarkerBlock(content string) string {
	return MarkerBegin + "\n" +
		"<!-- WARNING: This section is auto-generated. Do not edit manually. -->\n" +
		"<!-- Changes between AGENT-MANAGED:BEGIN and AGENT-MANAGED:END will be overwritten. -->\n" +
		"<!-- To modify, update source files in prompts/manifest/ and re-run the deploy command. -->\n" +
		content +
		MarkerEnd
}
