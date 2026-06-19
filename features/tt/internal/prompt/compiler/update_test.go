package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate_AlwaysRuns(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	// Initialize git repo to make any git commands happy (though no longer used by CheckForChanges)
	cmdInit := exec.Command("git", "init")
	cmdInit.Dir = tmpDir
	err := cmdInit.Run()
	require.NoError(t, err)

	cmdConfigName := exec.Command("git", "config", "user.name", "test")
	cmdConfigName.Dir = tmpDir
	_ = cmdConfigName.Run()

	cmdConfigEmail := exec.Command("git", "config", "user.email", "test@test.com")
	cmdConfigEmail.Dir = tmpDir
	_ = cmdConfigEmail.Run()

	cmdAdd := exec.Command("git", "add", ".")
	cmdAdd.Dir = tmpDir
	err = cmdAdd.Run()
	require.NoError(t, err)

	cmdCommit := exec.Command("git", "commit", "-m", "initial")
	cmdCommit.Dir = tmpDir
	err = cmdCommit.Run()
	require.NoError(t, err)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

	// 1. First update (should run deploy, Skipped=false)
	res1, err := Update(UpdateOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	require.NoError(t, err)
	require.NotNil(t, res1)
	require.Contains(t, res1.TargetResults, "antigravity")
	assert.False(t, res1.TargetResults["antigravity"].Skipped)

	// Verify target file was created
	targetFile := filepath.Join(tmpDir, ".agents", "rules", "test-compile-policy.md")
	_, err = os.Stat(targetFile)
	require.NoError(t, err)

	// 2. Second update (should STILL run deploy, Skipped=false)
	res2, err := Update(UpdateOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	require.NoError(t, err)
	require.NotNil(t, res2)
	assert.False(t, res2.TargetResults["antigravity"].Skipped)
}
