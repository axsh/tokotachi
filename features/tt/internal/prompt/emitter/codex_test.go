package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func buildCodexTestManifest(tempDir string) *manifest.ResolvedManifest {
	return &manifest.ResolvedManifest{
		Version:   1,
		ProjectID: "test-project",
		Entities: map[string][]*manifest.Entity{
			"target": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "target",
					ID:         "codex",
					Title:      "Codex Target",
					Raw: map[string]any{
						"paths": map[string]any{
							"rules":  ".codex/rules/",
							"skills": ".codex/skills/",
						},
						"index_file": "AGENTS.md",
					},
				},
			},
			"policy": {
				{
					APIVersion: "agent.meta/v1",
					Kind:       "policy",
					ID:         "architecture-memory",
					Title:      "Architecture Memory",
					FilePath:   filepath.Join(tempDir, "policy1.yaml"),
					Raw: map[string]any{
						"activation": map[string]any{
							"mode": "always",
						},
						"body":         "Record far-knowledge before push.",
						"applies_when": "Applies when changing architecture-sensitive code",
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
						"body": "Follow coding standards.",
						"paths": []any{
							"features/**/*.go",
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
						"description": "A test skill",
						"body":        "Skill body content.\n",
						"manual_only": false,
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
			},
		},
	}
}

func TestEmit_Codex(t *testing.T) {
	tempDir := t.TempDir()
	resolved := buildCodexTestManifest(tempDir)
	e := NewCodexEmitter(tempDir)

	buildDir := filepath.Join(tempDir, "build_output")
	if _, err := e.Emit(resolved, buildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		t.Fatalf("Emit (dry-run) failed: %v", err)
	}

	// Verify policy files have NO frontmatter
	policyPath := filepath.Join(buildDir, "codex", ".codex", "rules", "architecture-memory.md")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("expected file %s to be written, but got error: %v", policyPath, err)
	}
	content := string(data)

	// Must NOT contain YAML frontmatter delimiters
	if strings.Contains(content, "---") {
		t.Errorf("Codex policy file should NOT contain frontmatter (---), got:\n%s", content)
	}
	if !strings.Contains(content, "Record far-knowledge before push.") {
		t.Errorf("expected body content to be present, got:\n%s", content)
	}

	// Verify capability as SKILL.md
	skillPath := filepath.Join(buildDir, "codex", ".codex", "skills", "test-skill", "SKILL.md")
	data, err = os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected skill file %s, got error: %v", skillPath, err)
	}
	skillContent := string(data)
	for _, s := range []string{"name: test-skill", "description: A test skill", "Skill body content."} {
		if !strings.Contains(skillContent, s) {
			t.Errorf("skill file expected to contain %q, got:\n%s", s, skillContent)
		}
	}

	// Verify procedure as skill
	procPath := filepath.Join(buildDir, "codex", ".codex", "skills", "test-proc", "SKILL.md")
	data, err = os.ReadFile(procPath)
	if err != nil {
		t.Fatalf("expected proc skill file %s, got error: %v", procPath, err)
	}
	procContent := string(data)
	for _, s := range []string{"name: test-proc", "description: Test Procedure", "disable-model-invocation: true", "Procedure body."} {
		if !strings.Contains(procContent, s) {
			t.Errorf("proc skill file expected to contain %q, got:\n%s", s, procContent)
		}
	}
}

func TestEmit_Codex_MarkerSection(t *testing.T) {
	t.Run("no existing AGENTS.md", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		data, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("expected AGENTS.md to be created, got error: %v", err)
		}
		content := string(data)

		if !strings.Contains(content, MarkerBegin) {
			t.Errorf("expected AGENTS.md to contain MarkerBegin")
		}
		if !strings.Contains(content, MarkerEnd) {
			t.Errorf("expected AGENTS.md to contain MarkerEnd")
		}
		if !strings.Contains(content, ".codex/rules/architecture-memory.md") {
			t.Errorf("expected AGENTS.md to reference architecture-memory rule")
		}
		if !strings.Contains(content, ".codex/rules/coding-rules.md") {
			t.Errorf("expected AGENTS.md to reference coding-rules rule")
		}
	})

	t.Run("existing AGENTS.md without markers", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		// Write existing AGENTS.md
		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		existingContent := "# My Project Rules\n\nDo not break things.\n"
		if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
			t.Fatalf("failed to write AGENTS.md: %v", err)
		}

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		data, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		content := string(data)

		// Existing content preserved
		if !strings.Contains(content, "# My Project Rules") {
			t.Errorf("expected existing content to be preserved")
		}
		if !strings.Contains(content, "Do not break things.") {
			t.Errorf("expected existing content to be preserved")
		}
		// Marker section appended
		if !strings.Contains(content, MarkerBegin) {
			t.Errorf("expected MarkerBegin to be added")
		}
	})

	t.Run("existing AGENTS.md with markers - update", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		// Write AGENTS.md with existing markers
		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		existingContent := "# Owner Section\n\n" +
			MarkerBegin + "\nOld marker content\n" + MarkerEnd + "\n\n" +
			"# Footer\n"
		if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
			t.Fatalf("failed to write AGENTS.md: %v", err)
		}

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		data, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		content := string(data)

		// Owner sections preserved
		if !strings.Contains(content, "# Owner Section") {
			t.Errorf("expected owner section before marker to be preserved")
		}
		if !strings.Contains(content, "# Footer") {
			t.Errorf("expected footer after marker to be preserved")
		}
		// Old content replaced
		if strings.Contains(content, "Old marker content") {
			t.Errorf("expected old marker content to be replaced")
		}
		// New content present
		if !strings.Contains(content, ".codex/rules/architecture-memory.md") {
			t.Errorf("expected new marker content to reference rules")
		}
	})

	t.Run("marker content lists all rules and skills", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tempDir, "AGENTS.md"))
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		content := string(data)

		// All policies should be referenced
		if !strings.Contains(content, ".codex/rules/architecture-memory.md") {
			t.Errorf("expected architecture-memory rule reference")
		}
		if !strings.Contains(content, ".codex/rules/coding-rules.md") {
			t.Errorf("expected coding-rules rule reference")
		}
		// Skills section should exist
		if !strings.Contains(content, ".codex/skills/") {
			t.Errorf("expected skills reference")
		}
	})

	t.Run("applies_when appears in marker content", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tempDir, "AGENTS.md"))
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		content := string(data)

		// architecture-memory has applies_when -> should appear in marker
		if !strings.Contains(content, "Applies when changing architecture-sensitive code") {
			t.Errorf("expected applies_when text for architecture-memory in AGENTS.md")
		}

		// coding-rules has NO applies_when -> should only have path, no guidance
		// The line should be just the path without a " - " suffix
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "coding-rules.md") {
				if strings.Contains(line, " - Applies") {
					t.Errorf("coding-rules should not have applies_when guidance, got: %s", line)
				}
			}
		}
	})

	t.Run("no index_file skips marker management", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)

		// Remove index_file from target
		for _, target := range resolved.Entities["target"] {
			if target.ID == "codex" {
				delete(target.Raw, "index_file")
			}
		}

		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		// AGENTS.md should NOT be created when index_file is not set
		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err == nil {
			t.Errorf("expected AGENTS.md to NOT exist when index_file is not configured")
		}

		// Rules files should still be emitted
		rulesPath := filepath.Join(tempDir, ".codex", "rules", "architecture-memory.md")
		if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
			t.Errorf("expected rules files to still be emitted even without index_file")
		}
	})
}

func TestCheck_Codex(t *testing.T) {
	t.Run("no drift", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		ok, err := e.Check(resolved, buildDir)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if !ok {
			t.Errorf("expected Check to pass (no drift), but it returned false")
		}
	})

	t.Run("rules file drift", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		// Modify a rule file
		rulePath := filepath.Join(tempDir, ".codex", "rules", "architecture-memory.md")
		if err := os.WriteFile(rulePath, []byte("Modified"), 0644); err != nil {
			t.Fatalf("failed to modify rule: %v", err)
		}

		ok, err := e.Check(resolved, buildDir)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if ok {
			t.Errorf("expected Check to detect rules file drift, but it passed")
		}
	})

	t.Run("marker section drift", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		// Modify the marker section in AGENTS.md
		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		data, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("failed to read AGENTS.md: %v", err)
		}
		modified := strings.Replace(string(data), "architecture-memory", "MODIFIED", 1)
		if err := os.WriteFile(agentsPath, []byte(modified), 0644); err != nil {
			t.Fatalf("failed to write AGENTS.md: %v", err)
		}

		ok, err := e.Check(resolved, buildDir)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if ok {
			t.Errorf("expected Check to detect marker drift, but it passed")
		}
	})

	t.Run("no marker in AGENTS.md", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved := buildCodexTestManifest(tempDir)
		e := NewCodexEmitter(tempDir)

		buildDir := filepath.Join(tempDir, "build_output")

		// Deploy files but then remove markers from AGENTS.md
		if _, err := e.Emit(resolved, buildDir, true, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
			t.Fatalf("Emit (apply) failed: %v", err)
		}

		agentsPath := filepath.Join(tempDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte("# No markers here\n"), 0644); err != nil {
			t.Fatalf("failed to overwrite AGENTS.md: %v", err)
		}

		ok, err := e.Check(resolved, buildDir)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if ok {
			t.Errorf("expected Check to detect missing markers, but it passed")
		}
	})
}
