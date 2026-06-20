package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeploy_FirstRun(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if result.Skipped {
		t.Error("expected Skipped=false on first run")
	}
	if result.CompileResult == nil {
		t.Error("expected non-nil CompileResult")
	}
}

func TestDeploy_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

	// First deploy
	_, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("first Deploy() error = %v", err)
	}

	// Second deploy without changes (should still deploy, Skipped=false)
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("second Deploy() error = %v", err)
	}
	if result.Skipped {
		t.Error("expected Skipped=false even when no changes")
	}
}

func TestDeploy_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

	// First deploy
	_, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("first Deploy() error = %v", err)
	}

	// Modify a source file
	policyFile := filepath.Join(tmpDir, "prompts", "manifest", "code_content", "policies", "test.yaml")
	data, err := os.ReadFile(policyFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if err := os.WriteFile(policyFile, append(data, []byte("\n# changed\n")...), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Second deploy with changes
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("second Deploy() error = %v", err)
	}
	if result.Skipped {
		t.Error("expected Skipped=false when source changed")
	}
}

func TestDeploy_Force(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

	// First deploy
	_, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("first Deploy() error = %v", err)
	}

	// Force deploy (no changes)
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       true,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("force Deploy() error = %v", err)
	}
	if result.Skipped {
		t.Error("expected Skipped=false with --force")
	}
}

func TestDeploy_ValidationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "invalid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("Deploy() unexpected fatal error = %v", err)
	}
	if result.CompileResult == nil || len(result.CompileResult.Errors) == 0 {
		t.Error("expected validation errors in CompileResult")
	}
}

func TestDeploy_Drift(t *testing.T) {
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

	// First deploy
	_, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("first Deploy() error = %v", err)
	}

	// Verify target file was created
	targetFile := filepath.Join(tmpDir, ".agent", "rules", "test-compile-policy.md")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Fatalf("expected target file to be created: %s", targetFile)
	}

	// Delete target file to introduce drift
	if err := os.Remove(targetFile); err != nil {
		t.Fatalf("failed to delete target file: %v", err)
	}

	// Second deploy (should redeploy, Skipped=false)
	result, err := Deploy(DeployOptions{
		ProjectPath: projectPath,
		Target:      "antigravity",
		Force:       false,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("second Deploy() error = %v", err)
	}
	if result.Skipped {
		t.Error("expected Skipped=false due to drift")
	}

	// Verify target file was recreated
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("expected target file to be recreated after redeploy")
	}
}
