package intake

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MoveToProcessed moves an intake event from pending/ to processed/.
// It searches for the event file by event ID, moves it to the processed
// directory under a date-based subdirectory, and cleans up empty directories.
func MoveToProcessed(varDir, eventID string) error {
	pendingRoot := filepath.Join(varDir, "intake", "pending")
	processedRoot := filepath.Join(varDir, "intake", "processed")

	// Find the event file in pending/
	var foundPath string
	var foundDateDir string
	err := filepath.WalkDir(pendingRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == eventID+".json" {
			foundPath = path
			foundDateDir = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk pending directory: %w", err)
	}
	if foundPath == "" {
		return fmt.Errorf("event %s not found in pending", eventID)
	}

	// Create processed date directory
	dateStr := time.Now().UTC().Format("2006-01-02")
	processedDateDir := filepath.Join(processedRoot, dateStr)
	if err := os.MkdirAll(processedDateDir, 0o755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Move the file
	destPath := filepath.Join(processedDateDir, filepath.Base(foundPath))
	if err := os.Rename(foundPath, destPath); err != nil {
		return fmt.Errorf("failed to move event file: %w", err)
	}

	// Clean up empty date directory in pending
	cleanupEmptyDir(foundDateDir)

	return nil
}

// cleanupEmptyDir removes the directory if it is empty.
func cleanupEmptyDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}
