package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func TestEmit_Cursor(t *testing.T) {
	tempDir := t.TempDir()

	// Setup mock ResolvedManifest
	resolved := &manifest.ResolvedManifest{
		Version:   1,
		ProjectID: "test-project",
		Entities: map[string][]*manifest.Entity{
			"target": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "target",
					ID:         "cursor",
					Title:      "Cursor Target",
					Raw: map[string]any{
						"paths": map[string]any{
							"rules":  ".cursor/rules/",
							"skills": ".cursor/skills/",
						},
					},
				},
			},
			"policy": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "policy",
					ID:         "project-instructions",
					Title:      "Project Instructions",
					FilePath:   filepath.Join(tempDir, "policy1.yaml"),
					Raw: map[string]any{
						"activation": map[string]any{
							"mode": "always",
						},
						"body": "Always follow rules.",
					},
				},
				{
					APIVersion: "agent.meta/v1",
					Kind:       "policy",
					ID:         "coding-rules",
					Title:      "Coding Rules",
					FilePath:   filepath.Join(tempDir, "policy2.yaml"),
					Raw: map[string]any{
						"activation": map[string]any{
							"mode": "manual",
						},
						"body": "Coding is fun.",
						"paths": []any{
							"features/**/*.go",
							"shared/**/*.go",
						},
					},
				},
			},
			"capability": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "capability",
					ID:         "test-skill",
					Title:      "Test Skill",
					FilePath:   filepath.Join(tempDir, "skill.yaml"),
					Raw: map[string]any{
						"description": "A skill for testing",
						"body":        "This is body content.\n",
						"manual_only": true,
					},
				},
			},
			"procedure": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "procedure",
					ID:         "test-proc",
					Title:      "Test Procedure",
					FilePath:   filepath.Join(tempDir, "proc.yaml"),
					Raw: map[string]any{
						"body": "Procedure body.\n",
						"trigger": map[string]any{
							"command":     "test-proc",
							"manual_only": true,
						},
					},
				},
				{
					APIVersion: "agent.meta/v1",
					Kind:       "procedure",
					ID:         "test-auto-proc",
					Title:      "Test Auto Procedure",
					FilePath:   filepath.Join(tempDir, "auto-proc.yaml"),
					Raw: map[string]any{
						"body": "Auto procedure body.\n",
						"trigger": map[string]any{
							"command": "test-auto-proc",
						},
					},
				},
			},
		},
	}

	emitter := NewCursorEmitter(tempDir)

	// Test dry-run Emit (apply = false)
	buildDir := filepath.Join(tempDir, "build_output")
	if err := emitter.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (dry-run) failed: %v", err)
	}

	// Expected dry-run files
	expectedDryRunFiles := []struct {
		path    string
		contain []string
	}{
		{
			path:    filepath.Join(buildDir, "cursor", ".cursor", "rules", "project-instructions.mdc"),
			contain: []string{"alwaysApply: true", "description: Project Instructions", "Always follow rules."},
		},
		{
			path:    filepath.Join(buildDir, "cursor", ".cursor", "rules", "coding-rules.mdc"),
			contain: []string{"alwaysApply: false", "description: Coding Rules", "Coding is fun.", "globs:"},
		},
		{
			path:    filepath.Join(buildDir, "cursor", ".cursor", "skills", "test-skill", "SKILL.md"),
			contain: []string{"name: test-skill", "description: A skill for testing", "disable-model-invocation: true", "This is body content."},
		},
		{
			path: filepath.Join(buildDir, "cursor", ".cursor", "skills", "test-proc", "SKILL.md"),
			contain: []string{
				"name: test-proc",
				"description: Test Procedure",
				"disable-model-invocation: true",
				"Procedure body.",
			},
		},
		{
			path: filepath.Join(buildDir, "cursor", ".cursor", "skills", "test-auto-proc", "SKILL.md"),
			contain: []string{
				"name: test-auto-proc",
				"description: Test Auto Procedure",
				"disable-model-invocation: false",
				"Auto procedure body.",
			},
		},
	}

	for _, tc := range expectedDryRunFiles {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Errorf("expected file %s to be written, but got error: %v", tc.path, err)
			continue
		}
		content := string(data)
		for _, s := range tc.contain {
			if !strings.Contains(content, s) {
				t.Errorf("file %s expected to contain %q, but content was:\n%s", tc.path, s, content)
			}
		}
	}

	// Verify procedures are NOT emitted as rules (.mdc)
	procRulePath := filepath.Join(buildDir, "cursor", ".cursor", "rules", "test-proc.mdc")
	if _, err := os.Stat(procRulePath); !os.IsNotExist(err) {
		t.Errorf("expected procedure rule file %s to NOT exist, but it does", procRulePath)
	}

	// Test apply Emit (apply = true)
	if err := emitter.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (apply) failed: %v", err)
	}

	expectedApplyFiles := []string{
		filepath.Join(tempDir, ".cursor", "rules", "project-instructions.mdc"),
		filepath.Join(tempDir, ".cursor", "rules", "coding-rules.mdc"),
		filepath.Join(tempDir, ".cursor", "skills", "test-skill", "SKILL.md"),
		filepath.Join(tempDir, ".cursor", "skills", "test-proc", "SKILL.md"),
		filepath.Join(tempDir, ".cursor", "skills", "test-auto-proc", "SKILL.md"),
	}

	for _, p := range expectedApplyFiles {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected apply file %s to exist, but it does not", p)
		}
	}

	// Test Check when no drift is present
	ok, err := emitter.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !ok {
		t.Errorf("expected Check to pass (no drift), but it returned false")
	}

	// Test Check when a file is modified (drift)
	instructionsPath := filepath.Join(tempDir, ".cursor", "rules", "project-instructions.mdc")
	if err := os.WriteFile(instructionsPath, []byte("Modified contents"), 0644); err != nil {
		t.Fatalf("failed to modify instructions file: %v", err)
	}

	ok, err = emitter.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if ok {
		t.Errorf("expected Check to detect drift, but it passed")
	}

	// Reset and test Check when an untracked file is present (drift)
	if err := emitter.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("re-emit failed: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, ".cursor", "rules", "untracked.mdc")
	if err := os.WriteFile(untrackedPath, []byte("Untracked rule"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	ok, err = emitter.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if ok {
		t.Errorf("expected Check to detect untracked file drift, but it passed")
	}
}
