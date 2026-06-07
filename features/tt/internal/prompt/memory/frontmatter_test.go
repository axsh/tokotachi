package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantID     string
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "valid current doc",
			file:       "testdata/valid/current.md",
			wantID:     "test-current",
			wantStatus: "current",
		},
		{
			name:       "valid inbox doc",
			file:       "testdata/valid/inbox.md",
			wantID:     "test-inbox",
			wantStatus: "current",
		},
		{
			name:    "missing id",
			file:    "testdata/invalid/missing-id.md",
			wantErr: true,
		},
		{
			name:    "invalid status",
			file:    "testdata/invalid/bad-status.md",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseFrontmatter(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if doc.ID != tt.wantID {
				t.Errorf("ParseFrontmatter() ID = %q, want %q", doc.ID, tt.wantID)
			}
			if doc.Status != tt.wantStatus {
				t.Errorf("ParseFrontmatter() Status = %q, want %q", doc.Status, tt.wantStatus)
			}
			if doc.FilePath != tt.file {
				t.Errorf("ParseFrontmatter() FilePath = %q, want %q", doc.FilePath, tt.file)
			}
		})
	}
}

func TestParseFrontmatter_FromContent(t *testing.T) {
	content := "---\nid: inline-test\nkind: memory\ntitle: Inline\nstatus: current\ntopics:\n  - test\ntriggers:\n  - testing\ndepends_on: []\nlast_reviewed: 2026-06-04\n---\n# Inline Test\n\nBody."
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "inline.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
	if doc.ID != "inline-test" {
		t.Errorf("ID = %q, want %q", doc.ID, "inline-test")
	}
	if len(doc.Topics) != 1 || doc.Topics[0] != "test" {
		t.Errorf("Topics = %v, want [test]", doc.Topics)
	}
	if len(doc.Triggers) != 1 || doc.Triggers[0] != "testing" {
		t.Errorf("Triggers = %v, want [testing]", doc.Triggers)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := "# No Frontmatter\n\nJust a regular markdown file."
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "no-fm.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Error("ParseFrontmatter() expected error for file without frontmatter")
	}
}

func TestParseAllMemoryDocs(t *testing.T) {
	docs, errs := ParseAllMemoryDocs(".", "testdata/valid/**/*.md")
	if len(errs) > 0 {
		t.Errorf("ParseAllMemoryDocs() errors = %v", errs)
	}
	if len(docs) != 2 {
		t.Errorf("ParseAllMemoryDocs() got %d docs, want 2", len(docs))
	}
}

func TestParseAllMemoryDocs_WithInvalid(t *testing.T) {
	_, errs := ParseAllMemoryDocs(".", "testdata/invalid/**/*.md")
	if len(errs) == 0 {
		t.Error("ParseAllMemoryDocs() expected errors for invalid docs")
	}
}

func TestShouldSkipMemoryPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "var directory should be skipped",
			path:     "prompts/memory/var/intake/pending/2026/06/07/E-test.json",
			expected: true,
		},
		{
			name:     "schemas directory should be skipped",
			path:     "prompts/memory/schemas/agent-notify-payload.schema.json",
			expected: true,
		},
		{
			name:     "normal memory doc should not be skipped",
			path:     "prompts/memory/current.md",
			expected: false,
		},
		{
			name:     "nested memory doc should not be skipped",
			path:     "prompts/memory/adr/001-decision.md",
			expected: false,
		},
		{
			name:     "var in windows path should be skipped",
			path:     "prompts\\memory\\var\\intake\\file.json",
			expected: true,
		},
		{
			name:     "schemas in windows path should be skipped",
			path:     "prompts\\memory\\schemas\\schema.json",
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipMemoryPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipMemoryPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseAllMemoryDocs_SkipsVarAndSchemas(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid memory doc
	validDir := filepath.Join(tmpDir, "memory")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatal(err)
	}
	validContent := "---\nid: test-doc\nkind: memory\ntitle: Test\nstatus: current\n---\n# Test\n"
	if err := os.WriteFile(filepath.Join(validDir, "test.md"), []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file under var/ that would cause a parse error if not skipped
	varDir := filepath.Join(tmpDir, "memory", "var", "intake")
	if err := os.MkdirAll(varDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(varDir, "bad.md"), []byte("not valid frontmatter"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file under schemas/ that would cause a parse error if not skipped
	schemasDir := filepath.Join(tmpDir, "memory", "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemasDir, "schema.md"), []byte("not valid"), 0644); err != nil {
		t.Fatal(err)
	}

	docs, errs := ParseAllMemoryDocs(tmpDir, "memory/**/*.md")
	if len(errs) > 0 {
		t.Errorf("ParseAllMemoryDocs() unexpected errors: %v", errs)
	}
	if len(docs) != 1 {
		t.Errorf("ParseAllMemoryDocs() got %d docs, want 1 (only the valid doc)", len(docs))
	}
}
