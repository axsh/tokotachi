package scaffold

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFiles_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("# Hello")},
		{RelativePath: "subdir/file.txt", Content: []byte("content")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "skip"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Hello", string(content))

	content, err = os.ReadFile(filepath.Join(tmpDir, "subdir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
}

func TestApplyFiles_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	existingContent := "original content"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(existingContent), 0o644))

	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("new content")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "skip"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

func TestApplyFiles_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("old"), 0o644))

	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("new")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "overwrite"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(content))
}

func TestApplyFiles_AppendExisting(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.txt"), []byte("line1\n"), 0o644))

	files := []DownloadedFile{
		{RelativePath: "config.txt", Content: []byte("line2\n")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "append"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "config.txt"))
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\n", string(content))
}

func TestApplyFiles_ErrorOnExisting(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("exists"), 0o644))

	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("new")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "error"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "README.md")
}

func TestApplyFiles_WithBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	files := []DownloadedFile{
		{RelativePath: "main.go", Content: []byte("package main")},
	}
	placement := &Placement{BaseDir: "features/myapp", ConflictPolicy: "skip"}

	err := ApplyFiles(files, placement, tmpDir, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "features", "myapp", "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main", string(content))
}

func TestApplyFiles_WithTemplateVars(t *testing.T) {
	tmpDir := t.TempDir()
	files := []DownloadedFile{
		{RelativePath: "main.go.tmpl", Content: []byte("package {{.Name}}")},
	}
	placement := &Placement{
		BaseDir:        "features/{{.Name}}",
		ConflictPolicy: "skip",
		TemplateConfig: TemplateConfig{
			TemplateExtension: ".tmpl",
			StripExtension:    true,
		},
	}
	values := map[string]string{"Name": "myapp"}

	err := ApplyFiles(files, placement, tmpDir, values)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "features", "myapp", "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package myapp", string(content))
}

func TestApplyGitignore_AddEntries(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log\n"), 0o644))

	actions := PostActions{GitignoreEntries: []string{"work/*", "tmp/"}}
	err := ApplyPostActions(actions, tmpDir, ".")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "work/*")
	assert.Contains(t, string(content), "tmp/")
	assert.Contains(t, string(content), "*.log")
}

func TestApplyGitignore_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("work/*\n*.log\n"), 0o644))

	actions := PostActions{GitignoreEntries: []string{"work/*"}}
	err := ApplyPostActions(actions, tmpDir, ".")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	require.NoError(t, err)
	count := 0
	for _, line := range splitLines(string(content)) {
		if line == "work/*" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestApplyGitignore_CreateFile(t *testing.T) {
	tmpDir := t.TempDir()

	actions := PostActions{GitignoreEntries: []string{"work/*"}}
	err := ApplyPostActions(actions, tmpDir, ".")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "work/*")
}

func TestApplyGitignore_EmptyEntries(t *testing.T) {
	tmpDir := t.TempDir()

	actions := PostActions{GitignoreEntries: nil}
	err := ApplyPostActions(actions, tmpDir, ".")
	assert.NoError(t, err)
}

func TestBuildPlan_NewFiles(t *testing.T) {
	tmpDir := t.TempDir()
	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("# Hello")},
		{RelativePath: "subdir/file.txt", Content: []byte("content")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "skip"}

	plan, err := BuildPlan(files, placement, tmpDir, "default", nil)
	require.NoError(t, err)
	assert.Equal(t, "default", plan.ScaffoldName)
	assert.Len(t, plan.FilesToCreate, 2)
	assert.Equal(t, "create", plan.FilesToCreate[0].Action)
}

func TestBuildPlan_WithConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("old"), 0o644))

	files := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("new")},
		{RelativePath: "NEW.md", Content: []byte("brand new")},
	}
	placement := &Placement{BaseDir: ".", ConflictPolicy: "skip"}

	plan, err := BuildPlan(files, placement, tmpDir, "default", nil)
	require.NoError(t, err)
	assert.Len(t, plan.FilesToCreate, 1)
	assert.Len(t, plan.FilesToSkip, 1)
	assert.Equal(t, "skip", plan.FilesToSkip[0].Action)
}

func TestApplyFilePermissions_Executable(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "setup.sh"), []byte("#!/bin/bash"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "build.sh"), []byte("#!/bin/bash"), 0o644))

	perms := []FilePermission{
		{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)},
	}

	err := applyFilePermissions(perms, tmpDir, ".")
	require.NoError(t, err)

	// On Unix, verify executable bits are set.
	// On Windows, os.Chmod has limited support for Unix permission bits,
	// so we only verify the operation completed without error.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(scriptsDir, "setup.sh"))
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "setup.sh should be executable")

		info, err = os.Stat(filepath.Join(scriptsDir, "build.sh"))
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "build.sh should be executable")
	}
}

func TestApplyFilePermissions_Mode0600(t *testing.T) {
	tmpDir := t.TempDir()
	secretsDir := filepath.Join(tmpDir, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(secretsDir, "api-key.txt"), []byte("secret"), 0o644))

	perms := []FilePermission{
		{Pattern: "secrets/**/*", Mode: "0600"},
	}

	err := applyFilePermissions(perms, tmpDir, ".")
	require.NoError(t, err)

	// On Unix, verify exact mode. On Windows, only verify no error.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(secretsDir, "api-key.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestApplyFilePermissions_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0o644))

	perms := []FilePermission{
		{Pattern: "*.py", Executable: boolPtr(true)},
	}

	err := applyFilePermissions(perms, tmpDir, ".")
	assert.NoError(t, err)
}

func TestApplyFilePermissions_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "run.sh"), []byte("#!/bin/bash"), 0o644))

	perms := []FilePermission{
		{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)},
	}

	// First application
	err := applyFilePermissions(perms, tmpDir, ".")
	require.NoError(t, err)

	// Second application - should not error
	err = applyFilePermissions(perms, tmpDir, ".")
	assert.NoError(t, err)
}

func TestApplyPostActions_WithFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "run.sh"), []byte("#!/bin/bash"), 0o644))

	actions := PostActions{
		GitignoreEntries: []string{"work/*"},
		FilePermissions: []FilePermission{
			{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)},
		},
	}

	err := ApplyPostActions(actions, tmpDir, ".")
	require.NoError(t, err)

	// Check gitignore was applied
	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "work/*")

	// Check file permission was applied (platform-specific)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(scriptsDir, "run.sh"))
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o111 != 0, "run.sh should be executable")
	}
}

func TestBuildPlan_WithPermissionActions(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "build.sh"), []byte("#!/bin/bash"), 0o644))

	files := []DownloadedFile{
		{RelativePath: "scripts/build.sh", Content: []byte("#!/bin/bash")},
	}
	placement := &Placement{
		BaseDir:        ".",
		ConflictPolicy: "skip",
		PostActions: PostActions{
			FilePermissions: []FilePermission{
				{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)},
			},
		},
	}

	plan, err := BuildPlan(files, placement, tmpDir, "default", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, plan.PermissionActions)
	assert.Equal(t, "0755", plan.PermissionActions[0].Mode)
	assert.Contains(t, plan.PermissionActions[0].Path, "build.sh")
}
