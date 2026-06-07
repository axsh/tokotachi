package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func buildClaudeCodeTestManifest(tempDir string) *manifest.ResolvedManifest {
	return &manifest.ResolvedManifest{
		Version:   1,
		ProjectID: "test-project",
		Entities: map[string][]*manifest.Entity{
			"target": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "target",
					ID:         "claude-code",
					Title:      "Claude Code Target",
					Raw: map[string]any{
						"paths": map[string]any{
							"rules":  ".claude/rules/",
							"skills": ".claude/skills/",
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
}

func TestEmit_ClaudeCode(t *testing.T) {
	tempDir := t.TempDir()
	resolved := buildClaudeCodeTestManifest(tempDir)
	e := NewClaudeCodeEmitter(tempDir)

	// Test dry-run Emit (apply = false)
	buildDir := filepath.Join(tempDir, "build_output")
	if err := e.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (dry-run) failed: %v", err)
	}

	// Expected dry-run files
	expectedDryRunFiles := []struct {
		path       string
		contain    []string
		notContain []string
	}{
		{
			// Policy without paths -> no frontmatter
			path:       filepath.Join(buildDir, "claude-code", ".claude", "rules", "project-instructions.md"),
			contain:    []string{"Always follow rules."},
			notContain: []string{"---"},
		},
		{
			// Policy with paths -> frontmatter with paths
			path:    filepath.Join(buildDir, "claude-code", ".claude", "rules", "coding-rules.md"),
			contain: []string{"---", "paths:", "features/**/*.go", "shared/**/*.go", "Coding is fun."},
		},
		{
			// Capability as SKILL.md
			path:    filepath.Join(buildDir, "claude-code", ".claude", "skills", "test-skill", "SKILL.md"),
			contain: []string{"name: test-skill", "description: A skill for testing", "disable-model-invocation: true", "This is body content."},
		},
		{
			// Procedure as skill
			path: filepath.Join(buildDir, "claude-code", ".claude", "skills", "test-proc", "SKILL.md"),
			contain: []string{
				"name: test-proc",
				"description: Test Procedure",
				"disable-model-invocation: true",
				"Procedure body.",
			},
		},
		{
			// Auto procedure as skill (manual_only = false)
			path: filepath.Join(buildDir, "claude-code", ".claude", "skills", "test-auto-proc", "SKILL.md"),
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
		for _, s := range tc.notContain {
			if strings.Contains(content, s) {
				t.Errorf("file %s expected NOT to contain %q, but content was:\n%s", tc.path, s, content)
			}
		}
	}

	// Verify .md extension (not .mdc)
	mdcPath := filepath.Join(buildDir, "claude-code", ".claude", "rules", "project-instructions.mdc")
	if _, err := os.Stat(mdcPath); !os.IsNotExist(err) {
		t.Errorf("expected .mdc file NOT to exist, but it does: %s", mdcPath)
	}

	// Test apply Emit (apply = true)
	if err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (apply) failed: %v", err)
	}

	expectedApplyFiles := []string{
		filepath.Join(tempDir, ".claude", "rules", "project-instructions.md"),
		filepath.Join(tempDir, ".claude", "rules", "coding-rules.md"),
		filepath.Join(tempDir, ".claude", "skills", "test-skill", "SKILL.md"),
		filepath.Join(tempDir, ".claude", "skills", "test-proc", "SKILL.md"),
		filepath.Join(tempDir, ".claude", "skills", "test-auto-proc", "SKILL.md"),
	}

	for _, p := range expectedApplyFiles {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected apply file %s to exist, but it does not", p)
		}
	}
}

func TestCheck_ClaudeCode(t *testing.T) {
	tempDir := t.TempDir()
	resolved := buildClaudeCodeTestManifest(tempDir)
	e := NewClaudeCodeEmitter(tempDir)

	buildDir := filepath.Join(tempDir, "build_output")

	// Deploy files
	if err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (apply) failed: %v", err)
	}

	// Test Check when no drift
	ok, err := e.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !ok {
		t.Errorf("expected Check to pass (no drift), but it returned false")
	}

	// Test Check when content is modified (drift)
	rulePath := filepath.Join(tempDir, ".claude", "rules", "project-instructions.md")
	if err := os.WriteFile(rulePath, []byte("Modified contents"), 0644); err != nil {
		t.Fatalf("failed to modify rule file: %v", err)
	}

	ok, err = e.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if ok {
		t.Errorf("expected Check to detect drift, but it passed")
	}

	// Reset and test untracked file drift
	if err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("re-emit failed: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, ".claude", "rules", "untracked.md")
	if err := os.WriteFile(untrackedPath, []byte("Untracked rule"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	ok, err = e.Check(resolved, buildDir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if ok {
		t.Errorf("expected Check to detect untracked file drift, but it passed")
	}
}
