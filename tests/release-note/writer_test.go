package release_note_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriter_GenerateReleaseNote(t *testing.T) {
	tmpDir := t.TempDir()

	content := "【新規】テスト機能が追加されました。\n【変更】設定形式が変更されました。"
	formatted := fmt.Sprintf("# Release Notes\n\n## What's New\n\n%s\n", content)
	version := "v1.0.0"

	// Write latest.md
	latestPath := filepath.Join(tmpDir, "latest.md")
	if err := os.WriteFile(latestPath, []byte(formatted), 0o644); err != nil {
		t.Fatalf("failed to write latest.md: %v", err)
	}

	// Write version archive
	archivePath := filepath.Join(tmpDir, version+".md")
	if err := os.WriteFile(archivePath, []byte(formatted), 0o644); err != nil {
		t.Fatalf("failed to write %s.md: %v", version, err)
	}

	// Verify latest.md
	latestData, err := os.ReadFile(latestPath)
	if err != nil {
		t.Fatalf("failed to read latest.md: %v", err)
	}
	latestContent := string(latestData)

	if !strings.Contains(latestContent, "# Release Notes") {
		t.Error("latest.md should contain '# Release Notes' header")
	}
	if !strings.Contains(latestContent, "## What's New") {
		t.Error("latest.md should contain '## What's New' section")
	}
	if !strings.Contains(latestContent, "テスト機能が追加されました") {
		t.Error("latest.md should contain the changelog content")
	}

	// Verify archive file
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read %s.md: %v", version, err)
	}

	if string(archiveData) != string(latestData) {
		t.Error("archive file should have the same content as latest.md")
	}
}

func TestWriter_OutputDirectoryStructure(t *testing.T) {
	notesDir := filepath.Join(projectRoot(), "releases", "notes")

	// Verify releases/notes/ directory exists
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		t.Fatalf("releases/notes/ directory not found at %s", notesDir)
	}

	// Verify templates/ subdirectory exists
	templatesDir := filepath.Join(notesDir, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Fatalf("releases/notes/templates/ directory not found at %s", templatesDir)
	}

	// Verify release-note.md.tmpl file exists
	tmplPath := filepath.Join(templatesDir, "release-note.md.tmpl")
	if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
		t.Fatalf("release-note.md.tmpl not found at %s", tmplPath)
	}
}
