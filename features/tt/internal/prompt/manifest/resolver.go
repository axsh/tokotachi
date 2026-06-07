package manifest

import (
	"gopkg.in/yaml.v3"
)

// ResolvedManifest represents the fully resolved manifest
type ResolvedManifest struct {
	Version   int                  `yaml:"version"`
	ProjectID string               `yaml:"project_id"`
	Entities   map[string][]*Entity    `yaml:"entities"`     // kind -> entities
	MemoryDocs []*MemoryDoc            `yaml:"memory_docs"`
}

// Resolve integrates entities and memory docs into a ResolvedManifest
func Resolve(cfg *ProjectConfig, entities []*Entity, memDocs []*MemoryDoc) (*ResolvedManifest, error) {
	resolved := &ResolvedManifest{
		Version:    cfg.Version,
		ProjectID:  cfg.ProjectID,
		Entities:   make(map[string][]*Entity),
		MemoryDocs: memDocs,
	}

	// Classify entities by kind
	for _, e := range entities {
		resolved.Entities[e.Kind] = append(resolved.Entities[e.Kind], e)
	}

	return resolved, nil
}

// MarshalResolvedManifest serializes a ResolvedManifest to YAML string
func MarshalResolvedManifest(resolved *ResolvedManifest) (string, error) {
	data, err := yaml.Marshal(resolved)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
