package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEntity(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantID   string
		wantKind string
		wantErr  bool
	}{
		{
			name:     "valid policy",
			file:     "testdata/valid/policies/test-policy.yaml",
			wantID:   "test-policy",
			wantKind: "policy",
		},
		{
			name:     "valid procedure",
			file:     "testdata/valid/procedures/test-procedure.yaml",
			wantID:   "test-procedure",
			wantKind: "procedure",
		},
		{
			name:    "missing kind",
			file:    "testdata/invalid/missing-kind.yaml",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := ParseEntity(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if entity.ID != tt.wantID {
				t.Errorf("ParseEntity() ID = %q, want %q", entity.ID, tt.wantID)
			}
			if entity.Kind != tt.wantKind {
				t.Errorf("ParseEntity() Kind = %q, want %q", entity.Kind, tt.wantKind)
			}
			if entity.FilePath != tt.file {
				t.Errorf("ParseEntity() FilePath = %q, want %q", entity.FilePath, tt.file)
			}
			if entity.Raw == nil {
				t.Error("ParseEntity() Raw should not be nil")
			}
		})
	}
}

func TestParseEntity_FromBytes(t *testing.T) {
	yamlContent := "apiVersion: agent.meta/v1\nkind: policy\nid: inline-test\ntitle: Inline\nscope: project\nactivation:\n  mode: always\nbody: test\n"
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "inline.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	entity, err := ParseEntity(path)
	if err != nil {
		t.Fatalf("ParseEntity() error = %v", err)
	}
	if entity.ID != "inline-test" {
		t.Errorf("ParseEntity() ID = %q, want %q", entity.ID, "inline-test")
	}
}

func TestParseEntity_MissingID(t *testing.T) {
	yamlContent := "apiVersion: agent.meta/v1\nkind: policy\ntitle: No ID\n"
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "no-id.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseEntity(path)
	if err == nil {
		t.Error("ParseEntity() expected error for missing id")
	}
}

func TestExpandGlob(t *testing.T) {
	tests := []struct {
		name      string
		rootDir   string
		pattern   string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "match yaml files in policies",
			rootDir:   ".",
			pattern:   "testdata/valid/policies/**/*.yaml",
			wantCount: 1,
		},
		{
			name:      "match yaml files in procedures",
			rootDir:   ".",
			pattern:   "testdata/valid/procedures/**/*.yaml",
			wantCount: 1,
		},
		{
			name:      "no match returns empty",
			rootDir:   ".",
			pattern:   "testdata/nonexistent/**/*.yaml",
			wantCount: 0,
		},
		{
			name:      "match all valid yamls",
			rootDir:   ".",
			pattern:   "testdata/valid/**/*.yaml",
			wantCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := ExpandGlob(tt.rootDir, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandGlob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(files) != tt.wantCount {
				t.Errorf("ExpandGlob() matched %d files, want %d; files=%v", len(files), tt.wantCount, files)
			}
		})
	}
}

func TestParseAllEntities_Valid(t *testing.T) {
	cfg := &ProjectConfig{
		Version:   1,
		ProjectID: "test",
		Sources: map[string]string{
			"policies":   "testdata/valid/policies/**/*.yaml",
			"procedures": "testdata/valid/procedures/**/*.yaml",
		},
	}
	entities, errs := ParseAllEntities(cfg, ".")
	if len(errs) > 0 {
		t.Errorf("ParseAllEntities() errors = %v", errs)
	}
	if len(entities) != 2 {
		t.Errorf("ParseAllEntities() got %d entities, want 2", len(entities))
	}
}

func TestParseEntity_Frontmatter(t *testing.T) {
	mdContent := "---\napiVersion: agent.meta/v1\nkind: policy\nid: md-test\ntitle: MD Test\nscope: project\nactivation:\n  mode: always\n---\n# My Policy Body\nThis is the markdown body.\n"
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "policy.md")
	if err := os.WriteFile(path, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	entity, err := ParseEntity(path)
	if err != nil {
		t.Fatalf("ParseEntity() error = %v", err)
	}

	if entity.ID != "md-test" {
		t.Errorf("ParseEntity() ID = %q, want %q", entity.ID, "md-test")
	}
	if entity.Kind != "policy" {
		t.Errorf("ParseEntity() Kind = %q, want %q", entity.Kind, "policy")
	}
	if entity.Raw == nil {
		t.Fatal("ParseEntity() Raw is nil")
	}

	bodyVal, ok := entity.Raw["body"]
	if !ok {
		t.Fatal("ParseEntity() Raw should contain 'body' key")
	}
	bodyStr, ok := bodyVal.(string)
	if !ok {
		t.Fatal("ParseEntity() body is not a string")
	}
	expectedBody := "# My Policy Body\nThis is the markdown body.\n"
	if bodyStr != expectedBody {
		t.Errorf("ParseEntity() body = %q, want %q", bodyStr, expectedBody)
	}
}
