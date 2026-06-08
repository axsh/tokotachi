package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEnsureBranchManifest_NewManifest(t *testing.T) {
	tmpDir := t.TempDir()
	memoryRoot := filepath.Join(tmpDir, "memory")

	task := agent.AgentTask{
		BranchPackageID:  "BR-test-branch-abcdef12",
		BranchPackageKey: "owner/repo:test-branch:abcdef1234567890",
	}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	err := EnsureBranchManifest(memoryRoot, task, now)
	require.NoError(t, err)

	// Verify manifest exists
	manifestPath := filepath.Join(memoryRoot, "branches", "BR-test-branch-abcdef12", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest agent.BranchManifest
	err = yaml.Unmarshal(data, &manifest)
	require.NoError(t, err)

	assert.Equal(t, "BR-test-branch-abcdef12", manifest.ID)
	assert.Equal(t, "owner/repo:test-branch:abcdef1234567890", manifest.Key)
	assert.Equal(t, "test-branch", manifest.Branch)
	assert.Equal(t, "abcdef12", manifest.MergeBase)
	assert.Equal(t, "main", manifest.DefaultBranch)
	assert.Equal(t, "active", manifest.Status)
}

func TestEnsureBranchManifest_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	memoryRoot := filepath.Join(tmpDir, "memory")

	task := agent.AgentTask{
		BranchPackageID: "BR-test-branch-abcdef12",
	}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create first
	err := EnsureBranchManifest(memoryRoot, task, now)
	require.NoError(t, err)

	// Create again - should not error
	err = EnsureBranchManifest(memoryRoot, task, now.Add(time.Hour))
	require.NoError(t, err)

	// Verify the first one is still there (not overwritten)
	manifestPath := filepath.Join(memoryRoot, "branches", "BR-test-branch-abcdef12", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest agent.BranchManifest
	err = yaml.Unmarshal(data, &manifest)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01T12:00:00Z", manifest.CreatedAt, "Should keep original timestamp")
}

func TestExtractBranchFromBPID(t *testing.T) {
	tests := []struct {
		bpid     string
		expected string
	}{
		{"BR-fix-memory-compiling-4a67ef5a", "fix-memory-compiling"},
		{"BR-main-abcdef12", "main"},
		{"BR-feature-long-branch-name-12345678", "feature-long-branch-name"},
		{"single", "single"},
	}
	for _, tc := range tests {
		t.Run(tc.bpid, func(t *testing.T) {
			got := ExtractBranchFromBPID(tc.bpid)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExtractMergeBaseFromBPID(t *testing.T) {
	tests := []struct {
		bpid     string
		expected string
	}{
		{"BR-fix-memory-compiling-4a67ef5a", "4a67ef5a"},
		{"BR-main-abcdef12", "abcdef12"},
		{"nohyphen", ""},
	}
	for _, tc := range tests {
		t.Run(tc.bpid, func(t *testing.T) {
			got := ExtractMergeBaseFromBPID(tc.bpid)
			assert.Equal(t, tc.expected, got)
		})
	}
}
