// Package registry loads and validates the knowledge DB registry (registry.yaml).
package registry

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is the in-memory model of registry.yaml.
type File struct {
	Version int     `yaml:"version" json:"version"`
	KDBs    []Entry `yaml:"kdbs" json:"kdbs"`
}

// Entry describes one registered knowledge DB.
type Entry struct {
	ID         string `yaml:"id" json:"id"`
	ModulePath string `yaml:"module_path" json:"module_path"`
	CorpusPath string `yaml:"corpus_path" json:"corpus_path"`
	StatusCmd  string `yaml:"status_cmd" json:"status_cmd"`
	DistillCmd string `yaml:"distill_cmd" json:"distill_cmd"`
	DumpCmd    string `yaml:"dump_cmd" json:"dump_cmd"`
	MergeCmd   string `yaml:"merge_cmd" json:"merge_cmd"`
	PackCmd    string `yaml:"pack_cmd" json:"pack_cmd"`
}

// Load reads and parses a registry file. Unknown keys are ignored.
// It does not validate; call Validate on the result.
func Load(path string) (*File, error) {
	f, missing, err := LoadOptional(path)
	if err != nil {
		return nil, err
	}
	if missing {
		return nil, fmt.Errorf("read registry %s: %w", path, os.ErrNotExist)
	}
	return f, nil
}

// LoadOptional reads a registry file when present.
// If the file does not exist, it returns an empty File with Version 1, missing=true, and err=nil.
// Parse or other I/O errors still return err.
func LoadOptional(path string) (f *File, missing bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: 1}, true, nil
		}
		return nil, false, fmt.Errorf("read registry %s: %w", path, err)
	}
	var out File
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, false, fmt.Errorf("parse registry %s: %w", path, err)
	}
	return &out, false, nil
}

// Validate checks version, required fields of every entry, and id uniqueness.
func (f *File) Validate() error {
	if f.Version < 1 {
		return fmt.Errorf("registry: version must be >= 1 (got %d)", f.Version)
	}
	seen := map[string]bool{}
	for i, e := range f.KDBs {
		fields := []struct{ name, val string }{
			{"id", e.ID},
			{"module_path", e.ModulePath},
			{"corpus_path", e.CorpusPath},
			{"status_cmd", e.StatusCmd},
			{"distill_cmd", e.DistillCmd},
			{"dump_cmd", e.DumpCmd},
			{"merge_cmd", e.MergeCmd},
			{"pack_cmd", e.PackCmd},
		}
		for _, fld := range fields {
			if strings.TrimSpace(fld.val) == "" {
				return fmt.Errorf("registry: incomplete entry #%d (id=%q): missing %s", i+1, e.ID, fld.name)
			}
		}
		if seen[e.ID] {
			return fmt.Errorf("registry: duplicate id %q", e.ID)
		}
		seen[e.ID] = true
	}
	return nil
}

// ByID returns the entry with the given id.
func (f *File) ByID(id string) (Entry, bool) {
	for _, e := range f.KDBs {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}
