package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// CacheDir is the directory path under the repo root for cache files.
	CacheDir = ".kotoshiro/tokotachi/.cache"
	// CacheFileName is the name of the catalog cache file.
	CacheFileName = "catalog.yaml"
)

// CachedCatalog represents the cached catalog.yaml with a timestamp for validity checks.
type CachedCatalog struct {
	UpdatedAt   string `yaml:"updated_at"`
	CatalogData []byte `yaml:"catalog_data"`
}

// CacheStore manages reading and writing of cached catalog data.
type CacheStore struct {
	repoRoot string
}

// NewCacheStore creates a new CacheStore for the given repository root.
func NewCacheStore(repoRoot string) *CacheStore {
	return &CacheStore{repoRoot: repoRoot}
}

// CachePath returns the absolute path to the cache file.
func (s *CacheStore) CachePath() string {
	return filepath.Join(s.repoRoot, CacheDir, CacheFileName)
}

// Load reads the cached catalog. Returns nil, nil if the cache file does not exist.
func (s *CacheStore) Load() (*CachedCatalog, error) {
	data, err := os.ReadFile(s.CachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cached CachedCatalog
	if err := yaml.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}
	return &cached, nil
}

// Save writes the cached catalog to disk and ensures .gitignore is set up.
func (s *CacheStore) Save(catalog *CachedCatalog) error {
	cacheDir := filepath.Join(s.repoRoot, CacheDir)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(s.CachePath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return s.EnsureGitignore()
}

// IsValid checks if the cached catalog is still valid by comparing updated_at timestamps.
func (s *CacheStore) IsValid(remoteUpdatedAt string) bool {
	cached, err := s.Load()
	if err != nil || cached == nil {
		return false
	}
	return cached.UpdatedAt == remoteUpdatedAt
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
