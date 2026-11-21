package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogCodeContentKdbSuite(t *testing.T) {
	root := projectRoot()
	base := filepath.Join(root, "catalog", "originals", "axsh", "go-standard-project", "base", "prompts", "manifest", "code_content")
	repo := filepath.Join(root, "prompts", "manifest", "code_content")

	exactPairs := []string{
		"capabilities/knowledge-database.md",
		"capabilities/committed-doc-links.md",
		"capabilities/portable-file-references.md",
		"capabilities/pre-sync-knowledge-compile.md",
		"procedures/create-backlog-item.md",
		"procedures/systematize-knowledge-database.md",
		"procedures/verify-and-specify-next.md",
		"procedures/create-reproducible-report.md",
	}

	for _, rel := range exactPairs {
		baseText := mustReadNorm(t, filepath.Join(base, filepath.FromSlash(rel)))
		repoText := mustReadNorm(t, filepath.Join(repo, filepath.FromSlash(rel)))
		if baseText != repoText {
			t.Fatalf("BASE/REPO mismatch for %s (after EOL normalize)", rel)
		}
		assertForbidden(t, rel, baseText)
		assertExpectedShared(t, rel, baseText)
	}

	createSpec := mustReadNorm(t, filepath.Join(base, "procedures", "create-specification.md"))
	if !strings.Contains(createSpec, "create-backlog-item") || !strings.Contains(createSpec, "backlog:") {
		t.Fatalf("BASE create-specification missing backlog wiring")
	}

	coding := mustReadNorm(t, filepath.Join(base, "policies", "coding-rules.md"))
	if strings.Contains(coding, "legalchk") {
		t.Fatalf("BASE coding-rules must not contain legalchk product paths")
	}

	scaffold := mustReadNorm(t, filepath.Join(root, "catalog", "originals", "axsh", "go-standard-project", "scaffold.yaml"))
	if !strings.Contains(scaffold, "prompts/phases/*/branches/*/_runs/") {
		t.Fatalf("scaffold.yaml missing _runs gitignore_entries pattern")
	}

	rootGI := mustReadNorm(t, filepath.Join(root, ".gitignore"))
	if !strings.Contains(rootGI, "prompts/phases/*/branches/*/_runs/") {
		t.Fatalf("root .gitignore missing _runs pattern")
	}
}

func mustReadNorm(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func assertForbidden(t *testing.T, rel, text string) {
	t.Helper()
	for _, f := range []string{"./scripts/kdb/", "features/kdb/registry.yaml", "legalchk"} {
		if strings.Contains(text, f) {
			t.Fatalf("%s contains forbidden substring %q", rel, f)
		}
	}
}

func assertExpectedShared(t *testing.T, rel, text string) {
	t.Helper()
	baseName := filepath.Base(rel)

	switch baseName {
	case "knowledge-database.md", "pre-sync-knowledge-compile.md", "systematize-knowledge-database.md":
		if !strings.Contains(text, "tt kdb-track") {
			t.Fatalf("%s missing tt kdb-track", rel)
		}
		if !strings.Contains(text, "shared/kdb") {
			t.Fatalf("%s missing shared/kdb", rel)
		}
	case "create-backlog-item.md":
		if !strings.Contains(text, "tt kdb backlog") {
			t.Fatalf("%s missing tt kdb backlog", rel)
		}
		if !strings.Contains(text, "systematize-knowledge-database") {
			t.Fatalf("%s must reference systematize-knowledge-database", rel)
		}
	case "create-reproducible-report.md":
		if !strings.Contains(text, "_runs/procedures") {
			t.Fatalf("create-reproducible-report missing _runs/procedures")
		}
		if strings.Contains(text, "git commit -m") {
			t.Fatalf("create-reproducible-report must not instruct git commit of the report")
		}
		if !strings.Contains(text, "git commit しない") {
			t.Fatalf("create-reproducible-report must state not to git commit")
		}
	case "committed-doc-links.md":
		if !strings.Contains(text, "_runs/") {
			t.Fatalf("committed-doc-links missing _runs/")
		}
	}
}
