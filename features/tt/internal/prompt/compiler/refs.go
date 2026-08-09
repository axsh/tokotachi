package compiler

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// RefEntry is one catalog row for a file-backed template reference.
type RefEntry struct {
	File string `json:"file"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Ref  string `json:"ref"`
}

// RefsCatalog is the stdout JSON document for `tt prompt refs`.
type RefsCatalog struct {
	Refs []RefEntry `json:"refs"`
}

var refCatalogKinds = map[string]bool{
	"policy":     true,
	"procedure":  true,
	"capability": true,
}

// BuildRefsCatalog maps parsed entities to a stable refs list.
// Only policy, procedure, and capability are included.
func BuildRefsCatalog(entities []*manifest.Entity) RefsCatalog {
	entries := make([]RefEntry, 0)
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		if !refCatalogKinds[entity.Kind] {
			continue
		}
		entries = append(entries, RefEntry{
			File: filepath.Base(entity.FilePath),
			Kind: entity.Kind,
			ID:   entity.ID,
			Ref:  "{{" + entity.Kind + ":" + entity.ID + "}}",
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].ID < entries[j].ID
	})
	return RefsCatalog{Refs: entries}
}

// ListRefs loads project config, parses entities, and builds the catalog.
// Any parse/validation error from ParseAllEntities causes a non-nil error (R7).
func ListRefs(paths *PathConfig) (*RefsCatalog, error) {
	if paths == nil {
		return nil, fmt.Errorf("prompt refs: paths is nil")
	}
	cfg, err := LoadConfig(paths.ProjectYAML)
	if err != nil {
		return nil, err
	}
	entities, parseErrors := manifest.ParseAllEntities(cfg, paths.Workspace)
	if len(parseErrors) > 0 {
		msgs := make([]string, 0, len(parseErrors))
		for _, pe := range parseErrors {
			msgs = append(msgs, pe.Error())
		}
		return nil, fmt.Errorf("prompt refs: %d parse error(s): %s", len(parseErrors), strings.Join(msgs, "; "))
	}
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		if !refCatalogKinds[entity.Kind] {
			continue
		}
		if entity.ID == "" {
			return nil, fmt.Errorf("prompt refs: empty id in %s", entity.FilePath)
		}
	}
	catalog := BuildRefsCatalog(entities)
	return &catalog, nil
}
