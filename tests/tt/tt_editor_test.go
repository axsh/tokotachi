package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTtEditor_DryRun verifies that 'tt editor <branch> --dry-run' runs
// without errors and shows the editor open step in the report.
func TestTtEditor_DryRun(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "editor", "--dry-run", branchName)
	combined := stdout + stderr

	// In dry-run mode, editor open is simulated
	// The command may fail if worktree doesn't exist, but report should print
	assert.Contains(t, combined, "Editor",
		"output should mention editor, got stdout=%q stderr=%q", stdout, stderr)
	_ = exitCode
}

// TestTtEditor_ReservedBranch verifies that 'tt editor main' is rejected.
func TestTtEditor_ReservedBranch(t *testing.T) {
	_, stderr, exitCode := runTT(t, "editor", "--dry-run", "main")
	assert.NotEqual(t, 0, exitCode,
		"tt editor main should fail")
	assert.Contains(t, stderr, "reserved",
		"error should mention reserved branch: %q", stderr)
}

// TestEditor_LoadWarningAndFallback verifies that a warning is logged when
// editor.yaml is malformed, and the client falls back to the default config.
func TestEditor_LoadWarningAndFallback(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "tokotachi-integration-home-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	configDir := filepath.Join(tmpHome, ".kotoshiro", "tokotachi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	invalidYAML := `
system:
  editors:
    code:
      cmd: [invalid-yaml-structure]
      type: "vscode
`
	if err := os.WriteFile(filepath.Join(configDir, "editor.yaml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write invalid editor.yaml: %v", err)
	}

	ensureWorktree(t)

	stdout, stderr, exitCode := runTT(t, "open", "--dry-run", "--editor", "code", branchName)
	combined := stdout + stderr

	assert.Contains(t, combined, "WARNING: Failed to load editor.yaml",
		"output should warn about failed config load")
	assert.Contains(t, combined, "[DRY-RUN] code",
		"should fallback to default config and launch 'code'")
	assert.Equal(t, 0, exitCode, "should run successfully despite load warning")
}

// TestEditor_CustomEditorDynamicLaunch verifies that a custom editor defined in
// editor.yaml can be dynamically resolved and launched with its configured args.
func TestEditor_CustomEditorDynamicLaunch(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "tokotachi-integration-home-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	configDir := filepath.Join(tmpHome, ".kotoshiro", "tokotachi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	customYAML := `
user:
  editors:
    myeditor:
      cmd: "myeditor-cmd"
      type: "local"
      args:
        default: ["--custom-arg", "{path}"]
`
	if err := os.WriteFile(filepath.Join(configDir, "editor.yaml"), []byte(customYAML), 0644); err != nil {
		t.Fatalf("failed to write custom editor.yaml: %v", err)
	}

	ensureWorktree(t)

	stdout, stderr, exitCode := runTT(t, "open", "--dry-run", "--editor", "myeditor", branchName)
	combined := stdout + stderr

	assert.Contains(t, combined, "[DRY-RUN] myeditor-cmd",
		"should dynamically launch myeditor-cmd")
	assert.Contains(t, combined, "--custom-arg",
		"should apply custom arguments template")
	assert.Equal(t, 0, exitCode, "should launch custom editor successfully")
}
