package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceMarkerSection(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")

		err := ReplaceMarkerSection(path, "\nHello World\n")
		if err != nil {
			t.Fatalf("ReplaceMarkerSection() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		content := string(data)

		if !strings.Contains(content, MarkerBegin) {
			t.Errorf("expected file to contain MarkerBegin, got:\n%s", content)
		}
		if !strings.Contains(content, MarkerEnd) {
			t.Errorf("expected file to contain MarkerEnd, got:\n%s", content)
		}
		if !strings.Contains(content, "Hello World") {
			t.Errorf("expected file to contain 'Hello World', got:\n%s", content)
		}
		if !strings.Contains(content, "WARNING: This section is auto-generated") {
			t.Errorf("expected file to contain warning comment, got:\n%s", content)
		}
	})

	t.Run("file exists without markers", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")

		existingContent := "# My Project\n\nThis is my project.\n"
		if err := os.WriteFile(path, []byte(existingContent), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		err := ReplaceMarkerSection(path, "\nNew Section\n")
		if err != nil {
			t.Fatalf("ReplaceMarkerSection() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		content := string(data)

		// Existing content must be preserved
		if !strings.Contains(content, "# My Project") {
			t.Errorf("expected existing content to be preserved, got:\n%s", content)
		}
		if !strings.Contains(content, "This is my project.") {
			t.Errorf("expected existing content to be preserved, got:\n%s", content)
		}
		// Marker section must be appended
		if !strings.Contains(content, MarkerBegin) {
			t.Errorf("expected file to contain MarkerBegin, got:\n%s", content)
		}
		if !strings.Contains(content, "New Section") {
			t.Errorf("expected file to contain 'New Section', got:\n%s", content)
		}

		// Existing content should appear before marker
		beginIdx := strings.Index(content, MarkerBegin)
		projectIdx := strings.Index(content, "# My Project")
		if projectIdx > beginIdx {
			t.Errorf("expected existing content to appear before marker section")
		}
	})

	t.Run("file exists with markers - replace section", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")

		existingContent := "# My Project\n\nBefore marker.\n\n" +
			MarkerBegin + "\n" +
			"Old content\n" +
			MarkerEnd + "\n\n" +
			"After marker.\n"
		if err := os.WriteFile(path, []byte(existingContent), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		err := ReplaceMarkerSection(path, "\nReplaced content\n")
		if err != nil {
			t.Fatalf("ReplaceMarkerSection() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		content := string(data)

		// Content before marker preserved
		if !strings.Contains(content, "Before marker.") {
			t.Errorf("expected content before marker to be preserved, got:\n%s", content)
		}
		// Content after marker preserved
		if !strings.Contains(content, "After marker.") {
			t.Errorf("expected content after marker to be preserved, got:\n%s", content)
		}
		// Old content replaced
		if strings.Contains(content, "Old content") {
			t.Errorf("expected old content to be replaced, got:\n%s", content)
		}
		// New content present
		if !strings.Contains(content, "Replaced content") {
			t.Errorf("expected new content to be present, got:\n%s", content)
		}
	})

	t.Run("update existing marker content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")

		// First write
		err := ReplaceMarkerSection(path, "\nVersion 1\n")
		if err != nil {
			t.Fatalf("first ReplaceMarkerSection() error = %v", err)
		}

		// Second write (update)
		err = ReplaceMarkerSection(path, "\nVersion 2\n")
		if err != nil {
			t.Fatalf("second ReplaceMarkerSection() error = %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		content := string(data)

		if strings.Contains(content, "Version 1") {
			t.Errorf("expected old version to be replaced, got:\n%s", content)
		}
		if !strings.Contains(content, "Version 2") {
			t.Errorf("expected new version to be present, got:\n%s", content)
		}

		// Should have exactly one pair of markers
		beginCount := strings.Count(content, MarkerBegin)
		endCount := strings.Count(content, MarkerEnd)
		if beginCount != 1 {
			t.Errorf("expected exactly 1 MarkerBegin, got %d", beginCount)
		}
		if endCount != 1 {
			t.Errorf("expected exactly 1 MarkerEnd, got %d", endCount)
		}
	})
}

func TestExtractMarkerSection(t *testing.T) {
	t.Run("markers present", func(t *testing.T) {
		content := "Before\n" +
			MarkerBegin + "\n" +
			"Marker body\n" +
			MarkerEnd + "\n" +
			"After\n"

		section, found := ExtractMarkerSection(content)
		if !found {
			t.Fatalf("expected to find marker section")
		}
		if !strings.Contains(section, "Marker body") {
			t.Errorf("expected section to contain 'Marker body', got: %q", section)
		}
	})

	t.Run("no markers", func(t *testing.T) {
		content := "Just some content\nwithout markers\n"

		section, found := ExtractMarkerSection(content)
		if found {
			t.Errorf("expected not to find marker section")
		}
		if section != "" {
			t.Errorf("expected empty section, got: %q", section)
		}
	})

	t.Run("empty content between markers", func(t *testing.T) {
		content := "Before\n" +
			MarkerBegin + "\n" +
			MarkerEnd + "\n" +
			"After\n"

		section, found := ExtractMarkerSection(content)
		if !found {
			t.Fatalf("expected to find marker section")
		}
		if strings.TrimSpace(section) != "" {
			t.Errorf("expected empty section, got: %q", section)
		}
	})
}
