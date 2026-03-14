package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// CacheDir is the directory path under the repo root for cache files.
	CacheDir = ".kotoshiro/tokotachi/.cache"
	// CacheCategory is the category name for repository data cache.
	CacheCategory = "repository_data"
	// CacheItemName is the name of the cached item (used as folder name).
	CacheItemName = "catalog.yaml"
	// MetaFileName is the name of the cache metadata file.
	MetaFileName = "meta.yaml"
	// DataFileName is the name of the cached data file.
	DataFileName = "data"
)

// CacheMeta represents the meta.yaml file for a cached item.
type CacheMeta struct {
	UpdatedAt string `yaml:"updated_at"` // remote timestamp for validity check
	CachedAt  string `yaml:"cached_at"`  // local timestamp of when the cache was saved
}

// CacheStore manages reading and writing of cached catalog data.
type CacheStore struct {
	repoRoot string
}

// NewCacheStore creates a new CacheStore for the given repository root.
func NewCacheStore(repoRoot string) *CacheStore {
	return &CacheStore{repoRoot: repoRoot}
}

// CachePath returns the absolute path to the cache item directory.
func (s *CacheStore) CachePath() string {
	return filepath.Join(s.repoRoot, CacheDir, CacheCategory, CacheItemName)
}

// metaPath returns the absolute path to the meta.yaml file.
func (s *CacheStore) metaPath() string {
	return filepath.Join(s.CachePath(), MetaFileName)
}

// dataPath returns the absolute path to the data file.
func (s *CacheStore) dataPath() string {
	return filepath.Join(s.CachePath(), DataFileName)
}

// Load reads the cached catalog meta and data.
// Returns nil, nil, nil if the cache does not exist.
func (s *CacheStore) Load() (*CacheMeta, []byte, error) {
	metaData, err := os.ReadFile(s.metaPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read cache meta: %w", err)
	}

	var meta CacheMeta
	if err := yaml.Unmarshal(metaData, &meta); err != nil {
		return nil, nil, fmt.Errorf("failed to parse cache meta: %w", err)
	}

	data, err := os.ReadFile(s.dataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read cache data: %w", err)
	}

	return &meta, data, nil
}

// Save writes the cached catalog meta and data to disk.
func (s *CacheStore) Save(updatedAt string, data []byte) error {
	itemDir := s.CachePath()
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write meta.yaml
	meta := CacheMeta{
		UpdatedAt: updatedAt,
		CachedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	metaBytes, err := yaml.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("failed to marshal cache meta: %w", err)
	}
	if err := os.WriteFile(s.metaPath(), metaBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write cache meta: %w", err)
	}

	// Write data file (raw, unmodified bytes)
	if err := os.WriteFile(s.dataPath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache data: %w", err)
	}

	return s.EnsureGitignore()
}

// IsValid checks if the cached catalog is still valid by comparing updated_at timestamps.
func (s *CacheStore) IsValid(remoteUpdatedAt string) bool {
	metaData, err := os.ReadFile(s.metaPath())
	if err != nil {
		return false
	}

	var meta CacheMeta
	if err := yaml.Unmarshal(metaData, &meta); err != nil {
		return false
	}

	return meta.UpdatedAt == remoteUpdatedAt
}

// EnsureGitignore adds ".cache/" to the .gitignore in the .kotoshiro/tokotachi/ directory.
func (s *CacheStore) EnsureGitignore() error {
	kotoshiroDir := filepath.Join(s.repoRoot, ".kotoshiro", "tokotachi")
	if err := os.MkdirAll(kotoshiroDir, 0o755); err != nil {
		return fmt.Errorf("failed to create kotoshiro directory: %w", err)
	}

	gitignorePath := filepath.Join(kotoshiroDir, ".gitignore")
	gi, err := LoadGitignore(gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to load .gitignore: %w", err)
	}

	gi.AddEntries([]string{".cache/"})
	return gi.Save(gitignorePath)
}
