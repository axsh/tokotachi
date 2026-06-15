package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axsh/tokotachi/features/tt/internal/agent/intake"
	"github.com/axsh/tokotachi/features/tt/internal/agent/knowledge"
)

// TestFarKnowledge_Phase0_KnowledgeAdd tests adding new knowledge to a fresh store.
func TestFarKnowledge_Phase0_KnowledgeAdd(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	// Create content file
	contentFile := filepath.Join(contentDir, "content.md")
	require.NoError(t, os.WriteFile(contentFile, []byte("# Error Handling\n\nUse apierror types for all API handlers.\n"), 0o644))

	// Phase 0: Add first knowledge to new category
	err := store.Add("error-handling", "API Error Responses", "Error patterns for API handlers", contentFile, []string{"E-001"})
	require.NoError(t, err)

	// Verify category was created
	catMeta, err := knowledge.ReadCategoryMeta(filepath.Join(root, "knowledge", "error-handling"))
	require.NoError(t, err)
	assert.Equal(t, "error-handling", catMeta.CategoryID)
	assert.Equal(t, "API Error Responses", catMeta.Title)

	// Verify knowledge file was created
	mdPath := filepath.Join(root, "knowledge", "error-handling", "api-error-responses.md")
	meta, body, err := knowledge.ReadFrontmatter(mdPath)
	require.NoError(t, err)
	assert.Equal(t, "api-error-responses", meta.KnowledgeID)
	assert.Equal(t, "api-error-responses", meta.ID)
	assert.Equal(t, "current", meta.Status)
	assert.Contains(t, body, "apierror types")
}

// TestFarKnowledge_Phase1_MultipleRecordsAndCategories tests multiple knowledge items across categories.
func TestFarKnowledge_Phase1_MultipleRecordsAndCategories(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	// Create content files
	cf1 := filepath.Join(contentDir, "c1.md")
	cf2 := filepath.Join(contentDir, "c2.md")
	cf3 := filepath.Join(contentDir, "c3.md")
	require.NoError(t, os.WriteFile(cf1, []byte("Error content"), 0o644))
	require.NoError(t, os.WriteFile(cf2, []byte("Testing content"), 0o644))
	require.NoError(t, os.WriteFile(cf3, []byte("Logging content"), 0o644))

	// Add to multiple categories
	require.NoError(t, store.Add("error-handling", "API Errors", "Error patterns", cf1, []string{"E-001"}))
	require.NoError(t, store.Add("testing", "Test Naming", "Test conventions", cf2, []string{"E-002"}))
	require.NoError(t, store.Add("logging", "Log Format", "Logging rules", cf3, []string{"E-003"}))

	// Verify all categories exist
	categories, err := store.List()
	require.NoError(t, err)
	assert.Len(t, categories, 3)
}

// TestFarKnowledge_Phase2_AppendToExisting tests appending knowledge to an existing category.
func TestFarKnowledge_Phase2_AppendToExisting(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	cf1 := filepath.Join(contentDir, "c1.md")
	cf2 := filepath.Join(contentDir, "c2.md")
	require.NoError(t, os.WriteFile(cf1, []byte("Initial error content"), 0o644))
	require.NoError(t, os.WriteFile(cf2, []byte("Appended validation content"), 0o644))

	// Create category
	require.NoError(t, store.Add("error-handling", "API Errors", "Error patterns", cf1, []string{"E-001"}))

	// Append to existing category
	require.NoError(t, store.Append("error-handling", "Validation Errors", cf2, []string{"E-002"}))

	// Verify both files exist
	categories, err := store.List()
	require.NoError(t, err)
	require.Len(t, categories, 1)
	assert.Equal(t, 2, categories[0].FileCount)
}

// TestFarKnowledge_Phase3_SplitCategory tests splitting a category into subcategories.
func TestFarKnowledge_Phase3_SplitCategory(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	cf1 := filepath.Join(contentDir, "c1.md")
	cf2 := filepath.Join(contentDir, "c2.md")
	require.NoError(t, os.WriteFile(cf1, []byte("API error content"), 0o644))
	require.NoError(t, os.WriteFile(cf2, []byte("Validation content"), 0o644))

	require.NoError(t, store.Add("error-handling", "API Errors", "Error patterns", cf1, []string{"E-001"}))
	require.NoError(t, store.Append("error-handling", "Validation Errors", cf2, []string{"E-002"}))

	// Create split plan
	planDir := t.TempDir()
	plan := knowledge.SplitPlan{
		Assignments: map[string]string{
			"api-errors.md":        "api",
			"validation-errors.md": "validation",
		},
	}
	planData, _ := json.Marshal(plan)
	planFile := filepath.Join(planDir, "plan.json")
	require.NoError(t, os.WriteFile(planFile, planData, 0o644))

	// Split
	err := store.Split("error-handling", []string{"api", "validation"}, planFile)
	require.NoError(t, err)

	// Verify subcategories
	_, err = os.Stat(filepath.Join(root, "knowledge", "error-handling", "api", "api-errors.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "knowledge", "error-handling", "validation", "validation-errors.md"))
	assert.NoError(t, err)
}

// TestFarKnowledge_Phase4_MergeCategories tests merging two categories.
func TestFarKnowledge_Phase4_MergeCategories(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	cf1 := filepath.Join(contentDir, "c1.md")
	cf2 := filepath.Join(contentDir, "c2.md")
	require.NoError(t, os.WriteFile(cf1, []byte("Error content"), 0o644))
	require.NoError(t, os.WriteFile(cf2, []byte("Log content"), 0o644))

	require.NoError(t, store.Add("error-handling", "Errors", "Desc1", cf1, []string{"E-001"}))
	require.NoError(t, store.Add("logging", "Logs", "Desc2", cf2, []string{"E-002"}))

	// Create merge plan
	planDir := t.TempDir()
	plan := knowledge.MergePlan{Title: "Observability", Description: "Merged"}
	planData, _ := json.Marshal(plan)
	planFile := filepath.Join(planDir, "plan.json")
	require.NoError(t, os.WriteFile(planFile, planData, 0o644))

	err := store.Merge([]string{"error-handling", "logging"}, "observability", planFile)
	require.NoError(t, err)

	// Verify
	categories, err := store.List()
	require.NoError(t, err)
	assert.Len(t, categories, 1)
	assert.Equal(t, "observability", categories[0].Path)
}

// TestFarKnowledge_Phase5_IntakeProcessed tests the intake pending -> processed flow.
func TestFarKnowledge_Phase5_IntakeProcessed(t *testing.T) {
	varDir := t.TempDir()

	// Create pending event
	pendingDir := filepath.Join(varDir, "intake", "pending", "2026-06-15")
	require.NoError(t, os.MkdirAll(pendingDir, 0o755))
	event := map[string]any{
		"event_id":     "E-01TESTPROCESSED",
		"task_summary": "test event for processing",
		"raw_notes":    []string{"note 1"},
	}
	data, _ := json.Marshal(event)
	require.NoError(t, os.WriteFile(filepath.Join(pendingDir, "E-01TESTPROCESSED.json"), data, 0o644))

	// Move to processed
	err := intake.MoveToProcessed(varDir, "E-01TESTPROCESSED")
	require.NoError(t, err)

	// Verify in processed
	processedFound := false
	processedRoot := filepath.Join(varDir, "intake", "processed")
	_ = filepath.WalkDir(processedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.Contains(filepath.Base(path), "E-01TESTPROCESSED") {
			processedFound = true
		}
		return nil
	})
	assert.True(t, processedFound, "event should be in processed directory")

	// Verify not in pending
	pendingFound := false
	pendingRoot := filepath.Join(varDir, "intake", "pending")
	_ = filepath.WalkDir(pendingRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.Contains(filepath.Base(path), "E-01TESTPROCESSED") {
			pendingFound = true
		}
		return nil
	})
	assert.False(t, pendingFound, "event should not be in pending directory")
}

// TestFarKnowledge_Phase6_RenameAndMove tests rename and move operations.
func TestFarKnowledge_Phase6_RenameAndMove(t *testing.T) {
	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge"))
	contentDir := t.TempDir()

	cf1 := filepath.Join(contentDir, "c1.md")
	cf2 := filepath.Join(contentDir, "c2.md")
	require.NoError(t, os.WriteFile(cf1, []byte("Docker content"), 0o644))
	require.NoError(t, os.WriteFile(cf2, []byte("Infra placeholder"), 0o644))

	require.NoError(t, store.Add("backend", "Docker Setup", "Docker desc", cf1, []string{"E-001"}))
	require.NoError(t, store.Add("infra", "Placeholder", "Infra desc", cf2, []string{"E-002"}))

	// Rename
	err := store.Rename("backend", "services", "Service Layer")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(root, "knowledge", "backend"))
	assert.True(t, os.IsNotExist(err))

	catMeta, err := knowledge.ReadCategoryMeta(filepath.Join(root, "knowledge", "services"))
	require.NoError(t, err)
	assert.Equal(t, "Service Layer", catMeta.Title)

	// Move knowledge file
	err = store.Move("services/docker-setup.md", "infra")
	require.NoError(t, err)

	meta, _, err := knowledge.ReadFrontmatter(filepath.Join(root, "knowledge", "infra", "docker-setup.md"))
	require.NoError(t, err)
	assert.Equal(t, "infra", meta.CategoryPath)
}
