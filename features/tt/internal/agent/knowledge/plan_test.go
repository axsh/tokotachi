package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSplitPlan(t *testing.T) {
	dir := t.TempDir()
	plan := SplitPlan{
		Assignments: map[string]string{
			"file1.md": "sub-a",
			"file2.md": "sub-b",
		},
	}
	data, _ := json.Marshal(plan)
	path := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	parsed, err := ParseSplitPlan(path)
	require.NoError(t, err)
	assert.Len(t, parsed.Assignments, 2)
	assert.Equal(t, "sub-a", parsed.Assignments["file1.md"])
}

func TestParseSplitPlan_EmptyAssignments(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(SplitPlan{})
	path := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	_, err := ParseSplitPlan(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no assignments")
}

func TestParseMergePlan(t *testing.T) {
	dir := t.TempDir()
	plan := MergePlan{
		Title:       "Observability",
		Description: "Merged category",
	}
	data, _ := json.Marshal(plan)
	path := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	parsed, err := ParseMergePlan(path)
	require.NoError(t, err)
	assert.Equal(t, "Observability", parsed.Title)
}

func TestParseMergePlan_NoTitle(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(MergePlan{Description: "desc"})
	path := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	_, err := ParseMergePlan(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a title")
}

func TestParseSplitPlan_FileNotFound(t *testing.T) {
	_, err := ParseSplitPlan("/nonexistent/plan.json")
	require.Error(t, err)
}
