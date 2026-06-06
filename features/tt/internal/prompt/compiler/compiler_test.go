package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompile_Valid(t *testing.T) {
	projectPath := filepath.Join("testdata", "valid", "prompts", "manifest", "project.yaml")
	result, err := Compile(CompileOptions{
		ProjectPath: projectPath,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Logf("validation error: %s", e.Error())
		}
		t.Fatalf("Compile() got %d validation errors, want 0", len(result.Errors))
	}
	if result.IndexContent == "" {
		t.Error("Compile() IndexContent is empty")
	}
	if result.ResolvedYAML == "" {
		t.Error("Compile() ResolvedYAML is empty")
	}

	// Verify index.md content
	if !strings.Contains(result.IndexContent, "GENERATED FILE -- DO NOT EDIT") {
		t.Error("IndexContent missing generated banner")
	}
	if !strings.Contains(result.IndexContent, "test-current") {
		t.Error("IndexContent missing test-current entry")
	}
}

func TestCompile_DryRun(t *testing.T) {
	projectPath := filepath.Join("testdata", "valid", "prompts", "manifest", "project.yaml")

	result, err := Compile(CompileOptions{
		ProjectPath: projectPath,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Verify files were NOT written
	rootDir := filepath.Join("testdata", "valid")
	indexPath := filepath.Join(rootDir, "prompts", "memory", "index.md")
	if _, err := os.Stat(indexPath); err == nil {
		// Clean up if it exists from a previous test run
		os.Remove(indexPath)
	}

	// DryRun should produce content but not write files
	if result.IndexContent == "" {
		t.Error("DryRun should still produce IndexContent")
	}
}

func TestCompile_WriteFiles(t *testing.T) {
	// Use a temp copy to avoid polluting testdata
	tmpDir := t.TempDir()
	copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

	projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")
	result, err := Compile(CompileOptions{
		ProjectPath: projectPath,
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("Compile() got %d errors", len(result.Errors))
	}

	// Verify index.md was written
	indexPath := filepath.Join(tmpDir, "prompts", "memory", "index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.md not written: %v", err)
	}
	if !strings.Contains(string(data), "GENERATED FILE -- DO NOT EDIT") {
		t.Error("written index.md missing generated banner")
	}

	// Verify resolved manifest was written
	resolvedPath := filepath.Join(tmpDir, "tmp", "dist", "manifest.resolved.yaml")
	data, err = os.ReadFile(resolvedPath)
	if err != nil {
		t.Fatalf("resolved manifest not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("resolved manifest is empty")
	}
}

func TestCompile_WithValidationErrors(t *testing.T) {
	projectPath := filepath.Join("testdata", "invalid", "prompts", "manifest", "project.yaml")
	result, err := Compile(CompileOptions{
		ProjectPath: projectPath,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Compile() unexpected error = %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("Compile() expected validation errors for invalid data")
	}
	// Index and resolved should be empty when validation fails
	if result.IndexContent != "" {
		t.Error("Compile() should not produce IndexContent when validation fails")
	}
	if result.ResolvedYAML != "" {
		t.Error("Compile() should not produce ResolvedYAML when validation fails")
	}
}

// copyTestdata recursively copies src to dst
func copyTestdata(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0644)
	})
	if err != nil {
		t.Fatalf("copyTestdata: %v", err)
	}
}
