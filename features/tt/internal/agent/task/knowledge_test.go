package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteKnowledgeAtom(t *testing.T) {
	tmpDir := t.TempDir()
	memoryRoot := filepath.Join(tmpDir, "memory")

	atom := &agent.KnowledgeAtom{
		ID:         "K-01JABC123456789012345678",
		Version:    1,
		Type:       agent.KnowledgeTypeFact,
		Title:      "Test Knowledge",
		Body:       "This is a test body.",
		Status:     "draft",
		Importance: "medium",
		Confidence: 0.9,
		ActivationHints: agent.ActivationHints{
			Positive: []string{"when testing"},
		},
		Source: agent.KnowledgeSource{
			EventIDs:        []string{"E-01JABC123456789012345678"},
			BranchPackageID: "BR-test-branch-abcdef12",
			Agent:           "coding_agent",
			GitBranch:       "test-branch",
		},
		Timestamps: agent.KnowledgeTS{
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	relPath, err := WriteKnowledgeAtom(memoryRoot, "BR-test-branch-abcdef12", atom)
	require.NoError(t, err)

	// Verify file exists
	expectedAbsPath := filepath.Join(memoryRoot, "branches", "BR-test-branch-abcdef12", "knowledge", "K-01JABC123456789012345678.yaml")
	_, err = os.Stat(expectedAbsPath)
	require.NoError(t, err, "YAML file should exist")

	// Verify relative path
	assert.Equal(t, filepath.Join("prompts", "memory", "branches", "BR-test-branch-abcdef12", "knowledge", "K-01JABC123456789012345678.yaml"), relPath)

	// Verify content is valid YAML
	data, err := os.ReadFile(expectedAbsPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "id: K-01JABC123456789012345678")
	assert.Contains(t, string(data), "title: Test Knowledge")
	assert.Contains(t, string(data), "status: draft")
}
