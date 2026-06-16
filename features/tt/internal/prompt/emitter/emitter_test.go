package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func TestEmit_Antigravity(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy body_file for capability and procedure
	dummyBodyPath := filepath.Join(tempDir, "dummy_body.md")
	if err := os.WriteFile(dummyBodyPath, []byte("This is body content.\n"), 0644); err != nil {
		t.Fatalf("failed to write dummy body file: %v", err)
	}

	// Setup mock ResolvedManifest
	resolved := &manifest.ResolvedManifest{
		Version:   1,
		ProjectID: "test-project",
		Entities: map[string][]*manifest.Entity{
			"target": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "target",
					ID:         "antigravity",
					Title:      "Antigravity Target",
					Raw: map[string]any{
						"paths": map[string]any{
							"rules":  "rules_dir",
							"skills": "skills_dir",
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
						"body": "Coding is fun. See {{procedure:test-proc-body}} for workflow.",
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
						"paths": []any{
							"features/**/*.go",
						},
						"manual_only": true,
					},
				},
			},
			"procedure": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "procedure",
					ID:         "test-proc-body",
					Title:      "Procedure with body",
					FilePath:   filepath.Join(tempDir, "proc1.yaml"),
					Raw: map[string]any{
						"body": "This is body content.\n",
					},
				},
				{
					APIVersion: "agent.meta/v1",
					Kind:       "procedure",
					ID:         "test-proc-steps",
					Title:      "Procedure with steps",
					FilePath:   filepath.Join(tempDir, "proc2.yaml"),
					Raw: map[string]any{
						"steps": []any{
							"step-one",
							"step-two",
						},
					},
				},
			},
			"skip": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "skip",
					ID:         "vibe-coding-standard",
					Title:      "Vibe Coding Standard",
					Raw: map[string]any{
						"body": "This is a skip document.\n",
					},
				},
			},
		},
	}

	emitter := NewAntigravityEmitter(tempDir)

	// Test dry-run Emit (apply = false) -> should write to tempDir/build_dir/antigravity/...
	buildDir := filepath.Join(tempDir, "build_output")
	if _, err := emitter.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (dry-run) failed: %v", err)
	}

	// Expected dry-run files
	expectedDryRunFiles := []struct {
		path    string
		contain []string
	}{
		{
			path:    filepath.Join(buildDir, "antigravity", "rules_dir", "instructions.md"),
			contain: []string{"trigger: always_on", "Always follow rules."},
		},
		{
			path:    filepath.Join(buildDir, "antigravity", "rules_dir", "coding-rules.md"),
			contain: []string{"Coding is fun.", "skills_dir/test-proc-body/SKILL.md"},
		},
		{
			path:    filepath.Join(buildDir, "antigravity", "skills_dir", "test-skill", "SKILL.md"),
			contain: []string{"name: test-skill", "description: A skill for testing", "disable-model-invocation: true", "This is body content."},
		},
		{
			path:    filepath.Join(buildDir, "antigravity", "skills_dir", "test-proc-body", "SKILL.md"),
			contain: []string{"name: test-proc-body", "description: Procedure with body", "This is body content."},
		},
		{
			path:    filepath.Join(buildDir, "antigravity", "skills_dir", "test-proc-steps", "SKILL.md"),
			contain: []string{"name: test-proc-steps", "description: Procedure with steps", "1. step-one", "2. step-two"},
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

	// Test apply Emit (apply = true) -> should write to tempDir/rules_dir/...
	if _, err := emitter.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (apply) failed: %v", err)
	}

	expectedApplyFiles := []string{
		filepath.Join(tempDir, "rules_dir", "instructions.md"),
		filepath.Join(tempDir, "rules_dir", "coding-rules.md"),
		filepath.Join(tempDir, "skills_dir", "test-skill", "SKILL.md"),
		filepath.Join(tempDir, "skills_dir", "test-proc-body", "SKILL.md"),
		filepath.Join(tempDir, "skills_dir", "test-proc-steps", "SKILL.md"),
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
	instructionsPath := filepath.Join(tempDir, "rules_dir", "instructions.md")
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

	// Reset instructions and test Check when an untracked file is present (drift)
	if _, err := emitter.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("re-emit failed: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, "rules_dir", "untracked.md")
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

	// Verify skip entities are NOT emitted
	refPath := filepath.Join(buildDir, "antigravity", "rules_dir", "vibe-coding-standard.md")
	if _, err := os.Stat(refPath); !os.IsNotExist(err) {
		t.Errorf("skip entity should not be emitted, but file exists: %s", refPath)
	}
}

func TestEmit_Antigravity_WithLimits(t *testing.T) {
	tempDir := t.TempDir()

	// Sub-test: skip
	t.Run("skip", func(t *testing.T) {
		resolved := &manifest.ResolvedManifest{
			Version:   1,
			ProjectID: "test-project",
			Entities: map[string][]*manifest.Entity{
				"target": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "target",
						ID:         "antigravity",
						Raw: map[string]any{
							"paths": map[string]any{
								"rules":  "rules_dir",
								"skills": "skills_dir",
							},
							"limits": map[string]any{
								"rules": map[string]any{
									"max_file_size": 10,
									"on_exceed":     "skip",
								},
							},
						},
					},
				},
				"policy": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "policy",
						ID:         "big-policy",
						Title:      "Big Policy",
						Raw: map[string]any{
							"activation": map[string]any{"mode": "manual"},
							"body":       strings.Repeat("x", 100),
						},
					},
				},
			},
		}

		buildDir := filepath.Join(tempDir, "build_skip")
		emitter := NewAntigravityEmitter(tempDir)
		if _, err := emitter.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}

		skippedPath := filepath.Join(buildDir, "antigravity", "rules_dir", "big-policy.md")
		if _, err := os.Stat(skippedPath); !os.IsNotExist(err) {
			t.Errorf("expected skipped policy not to be emitted, but file exists: %s", skippedPath)
		}
	})

	// Sub-test: warn (file should still be written)
	t.Run("warn", func(t *testing.T) {
		resolved := &manifest.ResolvedManifest{
			Version:   1,
			ProjectID: "test-project",
			Entities: map[string][]*manifest.Entity{
				"target": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "target",
						ID:         "antigravity",
						Raw: map[string]any{
							"paths": map[string]any{
								"rules":  "rules_dir",
								"skills": "skills_dir",
							},
							"limits": map[string]any{
								"rules": map[string]any{
									"max_file_size": 10,
									"on_exceed":     "warn",
								},
							},
						},
					},
				},
				"policy": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "policy",
						ID:         "big-policy",
						Title:      "Big Policy",
						Raw: map[string]any{
							"activation": map[string]any{"mode": "manual"},
							"body":       strings.Repeat("x", 100),
						},
					},
				},
			},
		}

		buildDir := filepath.Join(tempDir, "build_warn")
		emitter := NewAntigravityEmitter(tempDir)
		if _, err := emitter.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}

		warnPath := filepath.Join(buildDir, "antigravity", "rules_dir", "big-policy.md")
		if _, err := os.Stat(warnPath); os.IsNotExist(err) {
			t.Errorf("expected warned policy to be emitted, but file does not exist: %s", warnPath)
		}
	})

	// Sub-test: truncate
	t.Run("truncate", func(t *testing.T) {
		resolved := &manifest.ResolvedManifest{
			Version:   1,
			ProjectID: "test-project",
			Entities: map[string][]*manifest.Entity{
				"target": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "target",
						ID:         "antigravity",
						Raw: map[string]any{
							"paths": map[string]any{
								"rules":  "rules_dir",
								"skills": "skills_dir",
							},
							"limits": map[string]any{
								"rules": map[string]any{
									"max_file_size": 50,
									"on_exceed":     "truncate",
								},
							},
						},
					},
				},
				"policy": {
					{
						APIVersion: "agent.meta/v1",
						Kind:       "policy",
						ID:         "big-policy",
						Title:      "Big Policy",
						Raw: map[string]any{
							"activation": map[string]any{"mode": "manual"},
							"body":       strings.Repeat("x", 200),
						},
					},
				},
			},
		}

		buildDir := filepath.Join(tempDir, "build_truncate")
		emitter := NewAntigravityEmitter(tempDir)
		if _, err := emitter.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}

		truncPath := filepath.Join(buildDir, "antigravity", "rules_dir", "big-policy.md")
		data, err := os.ReadFile(truncPath)
		if err != nil {
			t.Fatalf("expected truncated policy to be emitted, but got error: %v", err)
		}
		if len(data) > 50 {
			t.Errorf("expected truncated content <= 50 bytes, got %d", len(data))
		}
	})
}
