package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DownloadHistoryDir is the base directory for the history file.
	DownloadHistoryDir = ".kotoshiro/tokotachi"
	// DownloadHistoryFileName is the name of the download history file.
	DownloadHistoryFileName = "downloaded.yaml"
)

// DownloadRecord represents a single scaffold download record.
type DownloadRecord struct {
	DownloadedAt string `yaml:"downloaded_at"`
}

// DownloadHistory represents the download history file structure.
type DownloadHistory struct {
	History map[string]map[string]DownloadRecord `yaml:"history"`
}

// DownloadHistoryStore manages reading and writing of scaffold download history.
type DownloadHistoryStore struct {
	repoRoot string
}

// NewDownloadHistoryStore creates a new DownloadHistoryStore for the given repository root.
func NewDownloadHistoryStore(repoRoot string) *DownloadHistoryStore {
	return &DownloadHistoryStore{repoRoot: repoRoot}
}

// historyPath returns the absolute path to the download history file.
func (s *DownloadHistoryStore) historyPath() string {
	return filepath.Join(s.repoRoot, DownloadHistoryDir, DownloadHistoryFileName)
}

// Load reads the download history from disk.
// Returns an empty DownloadHistory if the file does not exist.
func (s *DownloadHistoryStore) Load() (*DownloadHistory, error) {
	data, err := os.ReadFile(s.historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &DownloadHistory{
				History: make(map[string]map[string]DownloadRecord),
			}, nil
		}
		return nil, fmt.Errorf("failed to read download history: %w", err)
	}

	var history DownloadHistory
	if err := yaml.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse download history: %w", err)
	}

	if history.History == nil {
		history.History = make(map[string]map[string]DownloadRecord)
	}

	return &history, nil
}

// Save writes the download history to disk, creating directories as needed.
func (s *DownloadHistoryStore) Save(history *DownloadHistory) error {
	dir := filepath.Join(s.repoRoot, DownloadHistoryDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	data, err := yaml.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal download history: %w", err)
	}

	if err := os.WriteFile(s.historyPath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write download history: %w", err)
	}

	return nil
}

// IsDownloaded checks whether the given category/name has been downloaded before.
func (s *DownloadHistoryStore) IsDownloaded(category, name string) bool {
	history, err := s.Load()
	if err != nil {
		return false
	}

	categoryMap, ok := history.History[category]
	if !ok {
		return false
	}

	_, ok = categoryMap[name]
	return ok
}

// RecordDownload records a successful download with the current UTC timestamp.
func (s *DownloadHistoryStore) RecordDownload(category, name string) error {
	history, err := s.Load()
	if err != nil {
		return err
	}

	if history.History[category] == nil {
		history.History[category] = make(map[string]DownloadRecord)
	}

	history.History[category][name] = DownloadRecord{
		DownloadedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return s.Save(history)
}

// IsDynamic returns true if the placement's base_dir contains template variables.
// Dynamic scaffolds are intended for repeated use with different parameters
// and should not be recorded in the download history.
func IsDynamic(placement *Placement) bool {
	return strings.Contains(placement.BaseDir, "{{")
}
