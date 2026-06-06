package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSourceDigest(t *testing.T) {
	projectPath := filepath.Join("testdata", "valid", "prompts", "manifest", "project.yaml")
	cfg, err := LoadConfig(projectPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	rootDir, err := ResolveProjectRoot(projectPath)
	if err != nil {
		t.Fatalf("ResolveProjectRoot() error = %v", err)
	}

	t.Run("stability", func(t *testing.T) {
		d1, err := ComputeSourceDigest(cfg, rootDir)
		if err != nil {
			t.Fatalf("ComputeSourceDigest() error = %v", err)
		}
		d2, err := ComputeSourceDigest(cfg, rootDir)
		if err != nil {
			t.Fatalf("ComputeSourceDigest() error = %v", err)
		}
		if d1 != d2 {
			t.Errorf("expected same digest for same input, got %s and %s", d1, d2)
		}
		if d1 == "" {
			t.Error("digest should not be empty")
		}
	})

	t.Run("change_detection", func(t *testing.T) {
		// Use a temp copy to modify files
		tmpDir := t.TempDir()
		copyTestdata(t, filepath.Join("testdata", "valid"), tmpDir)

		tmpProjectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")
		tmpCfg, err := LoadConfig(tmpProjectPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		tmpRootDir, err := ResolveProjectRoot(tmpProjectPath)
		if err != nil {
			t.Fatalf("ResolveProjectRoot() error = %v", err)
		}

		d1, err := ComputeSourceDigest(tmpCfg, tmpRootDir)
		if err != nil {
			t.Fatalf("ComputeSourceDigest() error = %v", err)
		}

		// Modify a source file
		policyFile := filepath.Join(tmpDir, "prompts", "manifest", "code_content", "policies", "test.yaml")
		original, err := os.ReadFile(policyFile)
		if err != nil {
			t.Fatalf("failed to read policy file: %v", err)
		}
		if err := os.WriteFile(policyFile, append(original, []byte("\n# modified\n")...), 0644); err != nil {
			t.Fatalf("failed to modify policy file: %v", err)
		}

		d2, err := ComputeSourceDigest(tmpCfg, tmpRootDir)
		if err != nil {
			t.Fatalf("ComputeSourceDigest() error = %v", err)
		}
		if d1 == d2 {
			t.Error("digest should change when source file is modified")
		}
	})
}

func TestLoadDigest(t *testing.T) {
	t.Run("file_not_found", func(t *testing.T) {
		info, err := LoadDigest(filepath.Join(t.TempDir(), "nonexistent"))
		if err != nil {
			t.Fatalf("LoadDigest() unexpected error = %v", err)
		}
		if info.Digest != "" {
			t.Errorf("expected empty digest, got %s", info.Digest)
		}
	})

	t.Run("valid_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		digestFile := filepath.Join(tmpDir, ".compile-digest")
		content := "digest: abc123\ntarget: antigravity\ntimestamp: \"2026-01-01T00:00:00Z\"\n"
		if err := os.WriteFile(digestFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write digest file: %v", err)
		}

		info, err := LoadDigest(digestFile)
		if err != nil {
			t.Fatalf("LoadDigest() error = %v", err)
		}
		if info.Digest != "abc123" {
			t.Errorf("expected digest 'abc123', got %s", info.Digest)
		}
		if info.Target != "antigravity" {
			t.Errorf("expected target 'antigravity', got %s", info.Target)
		}
	})

	t.Run("invalid_yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		digestFile := filepath.Join(tmpDir, ".compile-digest")
		// Use content with duplicate keys and tab indentation that causes parse error
		if err := os.WriteFile(digestFile, []byte("digest:\n\t- bad:\n\t\t- worse\ndigest: dup\n\t- broken"), 0644); err != nil {
			t.Fatalf("failed to write digest file: %v", err)
		}

		_, err := LoadDigest(digestFile)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestSaveDigest(t *testing.T) {
	t.Run("normal_write", func(t *testing.T) {
		tmpDir := t.TempDir()
		digestFile := filepath.Join(tmpDir, ".compile-digest")
		info := &DigestInfo{
			Digest:    "abc123",
			Target:    "antigravity",
			Timestamp: "2026-01-01T00:00:00Z",
		}
		if err := SaveDigest(digestFile, info); err != nil {
			t.Fatalf("SaveDigest() error = %v", err)
		}

		// Verify by loading back
		loaded, err := LoadDigest(digestFile)
		if err != nil {
			t.Fatalf("LoadDigest() error = %v", err)
		}
		if loaded.Digest != "abc123" {
			t.Errorf("expected digest 'abc123', got %s", loaded.Digest)
		}
	})

	t.Run("creates_parent_dirs", func(t *testing.T) {
		tmpDir := t.TempDir()
		digestFile := filepath.Join(tmpDir, "sub", "dir", ".compile-digest")
		info := &DigestInfo{
			Digest:    "xyz789",
			Target:    "antigravity",
			Timestamp: "2026-01-01T00:00:00Z",
		}
		if err := SaveDigest(digestFile, info); err != nil {
			t.Fatalf("SaveDigest() error = %v", err)
		}
		if _, err := os.Stat(digestFile); os.IsNotExist(err) {
			t.Error("digest file was not created")
		}
	})
}

func TestDigestPath(t *testing.T) {
	// Legacy (no target)
	got := DigestPath("/tmp/build")
	expected := filepath.Join("/tmp/build", ".compile-digest")
	if got != expected {
		t.Errorf("DigestPath() = %s, want %s", got, expected)
	}
	// With target "antigravity" (backward compat)
	got = DigestPath("/tmp/build", "antigravity")
	if got != expected {
		t.Errorf("DigestPath(antigravity) = %s, want %s", got, expected)
	}
	// With target "cursor"
	got = DigestPath("/tmp/build", "cursor")
	expected = filepath.Join("/tmp/build", ".compile-digest-cursor")
	if got != expected {
		t.Errorf("DigestPath(cursor) = %s, want %s", got, expected)
	}
}
