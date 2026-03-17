package writer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer handles release note file output.
type Writer struct {
	notesDir string
}

// New creates a new Writer that outputs to the given notes directory.
func New(notesDir string) *Writer {
	return &Writer{notesDir: notesDir}
}

// Write saves the release note content.
//  1. Writes to {notesDir}/latest.md
//  2. Writes to {notesDir}/{version}.md as archive (R10)
func (w *Writer) Write(content string, version string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(w.notesDir, 0o755); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}

	// Format the release note
	formatted := formatReleaseNote(content)

	// Write latest.md
	latestPath := filepath.Join(w.notesDir, "latest.md")
	if err := os.WriteFile(latestPath, []byte(formatted), 0o644); err != nil {
		return fmt.Errorf("failed to write latest.md: %w", err)
	}

	// Write version archive
	archivePath := filepath.Join(w.notesDir, version+".md")
	if err := os.WriteFile(archivePath, []byte(formatted), 0o644); err != nil {
		return fmt.Errorf("failed to write %s.md: %w", version, err)
	}

	return nil
}

// formatReleaseNote wraps the changelog content in the standard
// release note format.
func formatReleaseNote(content string) string {
	return fmt.Sprintf("# Release Notes\n\n## What's New\n\n%s\n", content)
}
