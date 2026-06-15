package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeContentFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestStore_Add(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	contentFile := writeContentFile(t, contentDir, "content.md", "# Error Handling\n\nUse apierror types.")

	err := s.Add("error-handling", "API Error Responses", "Error patterns for API handlers", contentFile, []string{"E-001"})
	require.NoError(t, err)

	// Verify _category.yaml
	catMeta, err := ReadCategoryMeta(filepath.Join(root, "error-handling"))
	require.NoError(t, err)
	assert.Equal(t, "error-handling", catMeta.CategoryID)
	assert.Equal(t, "API Error Responses", catMeta.Title)

	// Verify knowledge file
	mdPath := filepath.Join(root, "error-handling", "api-error-responses.md")
	meta, body, err := ReadFrontmatter(mdPath)
	require.NoError(t, err)
	assert.Equal(t, "api-error-responses", meta.KnowledgeID)
	assert.Equal(t, "error-handling", meta.CategoryPath)
	assert.Contains(t, body, "apierror types")
	assert.Equal(t, []string{"E-001"}, meta.SourceEventIDs)
}

func TestStore_Add_ExistingCategory(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "First knowledge")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Second knowledge")

	require.NoError(t, s.Add("testing", "Naming Conventions", "Test naming rules", cf1, []string{"E-001"}))
	require.NoError(t, s.Add("testing", "Table Driven Tests", "TDT patterns", cf2, []string{"E-002"}))

	// Both files should exist
	entries, err := os.ReadDir(filepath.Join(root, "testing"))
	require.NoError(t, err)
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && e.Name() != categoryMetaFile {
			mdCount++
		}
	}
	assert.Equal(t, 2, mdCount)
}

func TestStore_Append(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "Initial knowledge")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Appended knowledge")

	// First create the category
	require.NoError(t, s.Add("error-handling", "API Errors", "Error patterns", cf1, []string{"E-001"}))

	// Then append
	err := s.Append("error-handling", "Validation Errors", cf2, []string{"E-002"})
	require.NoError(t, err)

	// Verify the appended file exists
	mdPath := filepath.Join(root, "error-handling", "validation-errors.md")
	_, _, err = ReadFrontmatter(mdPath)
	require.NoError(t, err)
}

func TestStore_Append_NonexistentCategory(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf := writeContentFile(t, contentDir, "c.md", "content")

	err := s.Append("nonexistent", "Title", cf, []string{"E-001"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestStore_List_Empty(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)

	result, err := s.List()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestStore_List_WithCategories(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "Knowledge 1")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Knowledge 2")
	cf3 := writeContentFile(t, contentDir, "c3.md", "Knowledge 3")

	require.NoError(t, s.Add("error-handling", "Errors", "Error desc", cf1, []string{"E-001"}))
	require.NoError(t, s.Add("testing", "Tests", "Test desc", cf2, []string{"E-002"}))
	require.NoError(t, s.Add("logging", "Logs", "Log desc", cf3, []string{"E-003"}))

	result, err := s.List()
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestStore_Split(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "API error content")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Validation content")

	require.NoError(t, s.Add("error-handling", "API Errors", "Error patterns", cf1, []string{"E-001"}))
	require.NoError(t, s.Append("error-handling", "Validation Errors", cf2, []string{"E-002"}))

	// Create split plan
	planDir := t.TempDir()
	plan := SplitPlan{
		Assignments: map[string]string{
			"api-errors.md":        "api",
			"validation-errors.md": "validation",
		},
	}
	planData, _ := json.Marshal(plan)
	planFile := filepath.Join(planDir, "plan.json")
	require.NoError(t, os.WriteFile(planFile, planData, 0o644))

	err := s.Split("error-handling", []string{"api", "validation"}, planFile)
	require.NoError(t, err)

	// Verify subcategories created
	_, err = ReadCategoryMeta(filepath.Join(root, "error-handling", "api"))
	require.NoError(t, err)
	_, err = ReadCategoryMeta(filepath.Join(root, "error-handling", "validation"))
	require.NoError(t, err)

	// Verify files moved
	_, err = os.Stat(filepath.Join(root, "error-handling", "api", "api-errors.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "error-handling", "validation", "validation-errors.md"))
	assert.NoError(t, err)
}

func TestStore_Merge(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "Error content")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Log content")

	require.NoError(t, s.Add("error-handling", "Errors", "Desc1", cf1, []string{"E-001"}))
	require.NoError(t, s.Add("logging", "Logs", "Desc2", cf2, []string{"E-002"}))

	// Create merge plan
	planDir := t.TempDir()
	plan := MergePlan{
		Title:       "Observability",
		Description: "Error handling and logging patterns",
	}
	planData, _ := json.Marshal(plan)
	planFile := filepath.Join(planDir, "plan.json")
	require.NoError(t, os.WriteFile(planFile, planData, 0o644))

	err := s.Merge([]string{"error-handling", "logging"}, "observability", planFile)
	require.NoError(t, err)

	// Verify target category
	catMeta, err := ReadCategoryMeta(filepath.Join(root, "observability"))
	require.NoError(t, err)
	assert.Equal(t, "Observability", catMeta.Title)

	// Verify source categories removed
	_, err = os.Stat(filepath.Join(root, "error-handling"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root, "logging"))
	assert.True(t, os.IsNotExist(err))

	// Verify files exist in merged category
	result, err := s.List()
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "observability", result[0].Path)
}

func TestStore_Rename(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf := writeContentFile(t, contentDir, "c.md", "Some knowledge")
	require.NoError(t, s.Add("programming", "Prog", "Programming", cf, []string{"E-001"}))

	err := s.Rename("programming", "frontend", "Frontend Development")
	require.NoError(t, err)

	// Old dir should not exist
	_, err = os.Stat(filepath.Join(root, "programming"))
	assert.True(t, os.IsNotExist(err))

	// New dir should exist with correct metadata
	catMeta, err := ReadCategoryMeta(filepath.Join(root, "frontend"))
	require.NoError(t, err)
	assert.Equal(t, "frontend", catMeta.CategoryID)
	assert.Equal(t, "Frontend Development", catMeta.Title)
}

func TestStore_Move(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	contentDir := t.TempDir()

	cf1 := writeContentFile(t, contentDir, "c1.md", "Docker setup content")
	cf2 := writeContentFile(t, contentDir, "c2.md", "Placeholder")

	require.NoError(t, s.Add("backend", "Docker Setup", "Docker desc", cf1, []string{"E-001"}))
	require.NoError(t, s.Add("infrastructure", "Infra Placeholder", "Infra desc", cf2, []string{"E-002"}))

	err := s.Move("backend/docker-setup.md", "infrastructure")
	require.NoError(t, err)

	// File should be in target
	meta, _, err := ReadFrontmatter(filepath.Join(root, "infrastructure", "docker-setup.md"))
	require.NoError(t, err)
	assert.Equal(t, "infrastructure", meta.CategoryPath)

	// File should not be in source
	_, err = os.Stat(filepath.Join(root, "backend", "docker-setup.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"API Error Responses", "api-error-responses"},
		{"test-naming conventions", "test-naming-conventions"},
		{"Some/Path/Title", "some-path-title"},
		{"Hello  World", "hello-world"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := slugify(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
