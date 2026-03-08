package scaffold

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintPlan_CreateOnly(t *testing.T) {
	plan := &Plan{
		ScaffoldName: "default",
		FilesToCreate: []FileAction{
			{Path: "README.md", Action: "create"},
			{Path: "scripts/.gitkeep", Action: "create"},
		},
		PostActions: PostActions{
			GitignoreEntries: []string{"work/*"},
		},
	}

	var buf bytes.Buffer
	PrintPlan(plan, &buf)

	output := buf.String()
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "[CREATE]")
	assert.Contains(t, output, "README.md")
	assert.Contains(t, output, "work/*")
}

func TestPrintPlan_WithConflicts(t *testing.T) {
	plan := &Plan{
		ScaffoldName: "default",
		FilesToCreate: []FileAction{
			{Path: "NEW.md", Action: "create"},
		},
		FilesToSkip: []FileAction{
			{Path: "README.md", Action: "skip", Exists: true, ConflictPolicy: "skip"},
		},
		FilesToModify: []FileAction{
			{Path: "config.txt", Action: "overwrite", Exists: true, ConflictPolicy: "overwrite"},
		},
	}

	var buf bytes.Buffer
	PrintPlan(plan, &buf)

	output := buf.String()
	assert.Contains(t, output, "[CREATE]")
	assert.Contains(t, output, "[SKIP]")
	assert.Contains(t, output, "[OVERWRITE]")
}

func TestPrintPlan_WithWarnings(t *testing.T) {
	plan := &Plan{
		ScaffoldName: "features",
		Warnings:     []string{"conflict: README.md already exists (policy: error)"},
	}

	var buf bytes.Buffer
	PrintPlan(plan, &buf)

	output := buf.String()
	assert.Contains(t, output, "WARNING")
}

func TestPrintPlan_Empty(t *testing.T) {
	plan := &Plan{ScaffoldName: "default"}

	var buf bytes.Buffer
	PrintPlan(plan, &buf)

	output := buf.String()
	assert.Contains(t, output, "default")
}

func TestPrintPlan_WithPermissions(t *testing.T) {
	plan := &Plan{
		ScaffoldName: "default",
		FilesToCreate: []FileAction{
			{Path: "scripts/build.sh", Action: "create"},
		},
		PermissionActions: []PermissionAction{
			{Path: "scripts/build.sh", Mode: "0755"},
			{Path: "secrets/key.txt", Mode: "0600"},
		},
	}

	var buf bytes.Buffer
	PrintPlan(plan, &buf)

	output := buf.String()
	assert.Contains(t, output, "[CHMOD 0755] scripts/build.sh")
	assert.Contains(t, output, "[CHMOD 0600] secrets/key.txt")
	assert.Contains(t, output, "Post-actions:")
}
