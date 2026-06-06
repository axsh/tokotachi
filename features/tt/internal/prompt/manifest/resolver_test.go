package manifest

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		entities    []*Entity
		memDocs    []*MemoryDoc
		wantKinds   int // number of distinct kinds
		wantTotal   int // total entity count
		wantArchLen int
	}{
		{
			name: "basic resolve with two kinds",
			entities: []*Entity{
				{ID: "policy-1", Kind: "policy", Title: "Policy 1"},
				{ID: "proc-1", Kind: "procedure", Title: "Proc 1"},
			},
			memDocs: []*MemoryDoc{
				{ID: "current", Status: "current", Title: "Current"},
			},
			wantKinds:   2,
			wantTotal:   2,
			wantArchLen: 1,
		},
		{
			name: "multiple entities same kind",
			entities: []*Entity{
				{ID: "policy-1", Kind: "policy", Title: "Policy 1"},
				{ID: "policy-2", Kind: "policy", Title: "Policy 2"},
				{ID: "cap-1", Kind: "capability", Title: "Cap 1"},
			},
			memDocs:    []*MemoryDoc{},
			wantKinds:   2,
			wantTotal:   3,
			wantArchLen: 0,
		},
		{
			name:        "empty input",
			entities:    []*Entity{},
			memDocs:    []*MemoryDoc{},
			wantKinds:   0,
			wantTotal:   0,
			wantArchLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProjectConfig{Version: 1, ProjectID: "test"}
			resolved, err := Resolve(cfg, tt.entities, tt.memDocs)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resolved.Version != 1 {
				t.Errorf("Version = %d, want 1", resolved.Version)
			}
			if resolved.ProjectID != "test" {
				t.Errorf("ProjectID = %q, want %q", resolved.ProjectID, "test")
			}
			if len(resolved.Entities) != tt.wantKinds {
				t.Errorf("distinct kinds = %d, want %d", len(resolved.Entities), tt.wantKinds)
			}
			totalEntities := 0
			for _, ents := range resolved.Entities {
				totalEntities += len(ents)
			}
			if totalEntities != tt.wantTotal {
				t.Errorf("total entities = %d, want %d", totalEntities, tt.wantTotal)
			}
			if len(resolved.MemoryDocs) != tt.wantArchLen {
				t.Errorf("memDocs = %d, want %d", len(resolved.MemoryDocs), tt.wantArchLen)
			}
		})
	}
}

func TestMarshalResolvedManifest(t *testing.T) {
	resolved := &ResolvedManifest{
		Version:   1,
		ProjectID: "test",
		Entities: map[string][]*Entity{
			"policy": {{ID: "p1", Kind: "policy", Title: "P1"}},
		},
		MemoryDocs: []*MemoryDoc{{ID: "a1", Status: "current", Title: "A1"}},
	}
	data, err := MarshalResolvedManifest(resolved)
	if err != nil {
		t.Fatalf("MarshalResolvedManifest() error = %v", err)
	}
	if data == "" {
		t.Error("MarshalResolvedManifest() returned empty string")
	}

	// Verify it is valid YAML by unmarshaling back
	var check map[string]any
	if err := yaml.Unmarshal([]byte(data), &check); err != nil {
		t.Errorf("MarshalResolvedManifest() produced invalid YAML: %v", err)
	}
}
