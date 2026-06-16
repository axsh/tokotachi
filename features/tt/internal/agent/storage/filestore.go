package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// MoveToProcessed moves an event file from pending/ to processed/.
// The relative path structure (YYYY/MM/DD/event_id.json) is preserved.
func (fs *FileStore) MoveToProcessed(eventID string, createdAt time.Time) error {
	relPath := filepath.Join(
		createdAt.Format("2006"), createdAt.Format("01"), createdAt.Format("02"),
		eventID+".json")
	srcPath := filepath.Join(fs.baseDir, "pending", relPath)
	dstPath := filepath.Join(fs.baseDir, "processed", relPath)

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move event to processed: %w", err)
	}
	return nil
}

// ReadEvent reads and unmarshals an IntakeEvent from the pending directory.
func (fs *FileStore) ReadEvent(eventID string, createdAt time.Time) (*agent.IntakeEvent, error) {
	relPath := filepath.Join(
		createdAt.Format("2006"), createdAt.Format("01"), createdAt.Format("02"),
		eventID+".json")
	filePath := filepath.Join(fs.baseDir, "pending", relPath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event file: %w", err)
	}
	var event agent.IntakeEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

