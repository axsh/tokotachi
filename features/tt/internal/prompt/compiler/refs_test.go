package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func TestBuildRefsCatalog_FiltersAndSorts(t *testing.T) {
	entities := []*manifest.Entity{
		{Kind: "procedure", ID: "zzz-proc", FilePath: "/tmp/zzz-proc.md"},
		{Kind: "policy", ID: "bbb-policy", FilePath: "/tmp/bbb-policy.md"},
		{Kind: "capability", ID: "aaa-cap", FilePath: "/tmp/aaa-cap.md"},
		{Kind: "policy", ID: "aaa-policy", FilePath: "/tmp/aaa-policy.md"},
		{Kind: "target", ID: "cursor", FilePath: "/tmp/cursor.yaml"},
	}

	catalog := BuildRefsCatalog(entities)
	if len(catalog.Refs) != 4 {
		t.Fatalf("len(Refs) = %d, want 4", len(catalog.Refs))
	}

	wantOrder := []struct {
		kind string
		id   string
	}{
		{"capability", "aaa-cap"},
		{"policy", "aaa-policy"},
		{"policy", "bbb-policy"},
		{"procedure", "zzz-proc"},
	}
	for i, want := range wantOrder {
		got := catalog.Refs[i]
		if got.Kind != want.kind || got.ID != want.id {
			t.Errorf("Refs[%d] = {%s %s}, want {%s %s}", i, got.Kind, got.ID, want.kind, want.id)
		}
	}
}

func TestBuildRefsCatalog_RefUsesKindAndID(t *testing.T) {
	entities := []*manifest.Entity{
		{Kind: "policy", ID: "coding-rules", FilePath: "/ws/prompts/manifest/code_content/policies/coding-rules.md"},
	}
	catalog := BuildRefsCatalog(entities)
	if len(catalog.Refs) != 1 {
		t.Fatalf("len(Refs) = %d, want 1", len(catalog.Refs))
	}
	got := catalog.Refs[0]
	if got.File != "coding-rules.md" {
		t.Errorf("File = %q, want coding-rules.md", got.File)
	}
	if got.Kind != "policy" {
		t.Errorf("Kind = %q, want policy", got.Kind)
	}
	if got.ID != "coding-rules" {
		t.Errorf("ID = %q, want coding-rules", got.ID)
	}
	if got.Ref != "{{policy:coding-rules}}" {
		t.Errorf("Ref = %q, want {{policy:coding-rules}}", got.Ref)
	}

	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	s := string(raw)
	if strings.Contains(s, ".claude/") || strings.Contains(s, "SKILL.md") {
		t.Errorf("JSON must not contain resolved deploy paths: %s", s)
	}
}

func TestBuildRefsCatalog_IgnoresNonRefKinds(t *testing.T) {
	entities := []*manifest.Entity{
		{Kind: "target", ID: "cursor", FilePath: "cursor.yaml"},
		{Kind: "guard", ID: "g1", FilePath: "g1.yaml"},
		{Kind: "worker", ID: "w1", FilePath: "w1.yaml"},
		{Kind: "bundle", ID: "b1", FilePath: "b1.yaml"},
		{Kind: "skip", ID: "s1", FilePath: "s1.yaml"},
	}
	catalog := BuildRefsCatalog(entities)
	if len(catalog.Refs) != 0 {
		t.Fatalf("len(Refs) = %d, want 0", len(catalog.Refs))
	}
	if catalog.Refs == nil {
		t.Error("Refs must be empty slice, not nil")
	}
}

func TestListRefs_HappyPath(t *testing.T) {
	ws, err := filepath.Abs(filepath.Join("testdata", "catalog_template"))
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	paths := &PathConfig{
		Workspace:   ws,
		ProjectYAML: filepath.Join(ws, "prompts", "manifest", "project.yaml"),
	}
	catalog, err := ListRefs(paths)
	if err != nil {
		t.Fatalf("ListRefs() error = %v", err)
	}
	byID := map[string]RefEntry{}
	for _, e := range catalog.Refs {
		byID[e.ID] = e
	}
	coding, ok := byID["coding-rules"]
	if !ok {
		t.Fatal("missing coding-rules")
	}
	if coding.Ref != "{{policy:coding-rules}}" || coding.Kind != "policy" || coding.File != "coding-rules.md" {
		t.Errorf("coding-rules entry = %+v", coding)
	}
	pipe, ok := byID["build-pipeline"]
	if !ok {
		t.Fatal("missing build-pipeline")
	}
	if pipe.Ref != "{{procedure:build-pipeline}}" || pipe.Kind != "procedure" {
		t.Errorf("build-pipeline entry = %+v", pipe)
	}
}

func TestListRefs_ParseErrorFails(t *testing.T) {
	tmp := t.TempDir()
	policies := filepath.Join(tmp, "prompts", "manifest", "code_content", "policies")
	if err := os.MkdirAll(policies, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	projectYAML := filepath.Join(tmp, "prompts", "manifest", "project.yaml")
	projectBody := `version: 1
project_id: refs-parse-error
sources:
  policies: prompts/manifest/code_content/policies/**/*.md
`
	if err := os.WriteFile(projectYAML, []byte(projectBody), 0o644); err != nil {
		t.Fatalf("WriteFile project: %v", err)
	}
	// Missing id in frontmatter — ParseEntity must fail.
	badMD := "---\nkind: policy\ntitle: Broken\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(policies, "broken.md"), []byte(badMD), 0o644); err != nil {
		t.Fatalf("WriteFile broken: %v", err)
	}

	paths := &PathConfig{
		Workspace:   tmp,
		ProjectYAML: projectYAML,
	}
	catalog, err := ListRefs(paths)
	if err == nil {
		t.Fatalf("ListRefs() expected error, got catalog=%+v", catalog)
	}
	if catalog != nil {
		t.Errorf("ListRefs() catalog = %+v, want nil on error", catalog)
	}
}
