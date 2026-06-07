package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// FileStore handles atomic file writes to the pending directory.
type FileStore struct {
	baseDir string // e.g. "prompts/memory/var/intake"
}

// NewFileStore creates a new FileStore.
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

// Write atomically writes an IntakeEvent to pending/{YYYY}/{MM}/{DD}/{event_id}.json.
// Steps: marshal JSON -> write to _tmp/ -> fsync -> rename to pending/
// Returns the relative path from baseDir where the file was stored.
func (fs *FileStore) Write(event *agent.IntakeEvent) (string, error) {
	// Compute relative path
	ts := event.Timestamps.CreatedAt
	relPath := filepath.Join("pending",
		ts.Format("2006"), ts.Format("01"), ts.Format("02"),
		event.EventID+".json")
	finalPath := filepath.Join(fs.baseDir, relPath)

	// Marshal JSON
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create _tmp directory
	tmpDir := filepath.Join(fs.baseDir, "_tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp(tmpDir, "intake-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	// fsync
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to sync temp file: %w", err)
	}
	tmpFile.Close()

	// Create final directory
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create final directory: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return relPath, nil
}
