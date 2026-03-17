package writer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/writer"
)

func TestWriteReleaseNote(t *testing.T) {
	dir := t.TempDir()

	w := writer.New(dir)
	content := "【新規】Feature A was added.\n【変更】Feature B was changed."

	err := w.Write(content, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check latest.md
	latestPath := filepath.Join(dir, "latest.md")
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
	if !strings.Contains(latestContent, "Feature A was added") {
		t.Error("latest.md should contain the changelog content")
	}

	// Check version archive (R10)
	archivePath := filepath.Join(dir, "v1.0.0.md")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read v1.0.0.md: %v", err)
	}
	if string(archiveData) != string(latestData) {
		t.Error("archive should have the same content as latest.md")
	}
}

func TestWriteReleaseNote_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	notesDir := filepath.Join(dir, "nested", "notes")

	w := writer.New(notesDir)
	err := w.Write("Test content", "v0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that directory was created
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		t.Error("expected notes directory to be created")
	}

	// Check that files exist
	if _, err := os.Stat(filepath.Join(notesDir, "latest.md")); os.IsNotExist(err) {
		t.Error("expected latest.md to be created")
	}
	if _, err := os.Stat(filepath.Join(notesDir, "v0.1.0.md")); os.IsNotExist(err) {
		t.Error("expected v0.1.0.md to be created")
	}
}

func TestWriteReleaseNote_OverwriteLatest(t *testing.T) {
	dir := t.TempDir()

	w := writer.New(dir)

	// Write first version
	err := w.Write("Version 1 content", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error on first write: %v", err)
	}

	// Write second version (should overwrite latest.md)
	err = w.Write("Version 2 content", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error on second write: %v", err)
	}

	// latest.md should have v2 content
	latestData, err := os.ReadFile(filepath.Join(dir, "latest.md"))
	if err != nil {
		t.Fatalf("failed to read latest.md: %v", err)
	}
	if !strings.Contains(string(latestData), "Version 2 content") {
		t.Error("latest.md should contain updated content")
	}

	// Both archives should exist
	if _, err := os.Stat(filepath.Join(dir, "v1.0.0.md")); os.IsNotExist(err) {
		t.Error("v1.0.0.md archive should still exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "v2.0.0.md")); os.IsNotExist(err) {
		t.Error("v2.0.0.md archive should exist")
	}
}
