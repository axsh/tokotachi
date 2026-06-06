package emitter

import (
	"testing"
)

func TestResolveTemplateVars(t *testing.T) {
	ctx := &TemplateContext{
		Paths: TargetPaths{
			Rules:     ".agent/rules/",
			Skills:    ".agent/skills/",
			Workflows: ".agent/workflows/",
		},
		MemBase:    "prompts/memory",
		TargetName: "antigravity",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "policy reference resolves to rules path",
			input: "Read {{policy:coding-rules}} for details.",
			want:  "Read .agent/rules/coding-rules.md for details.",
		},
		{
			name:  "policy project-instructions renames to instructions.md",
			input: "See {{policy:project-instructions}} for setup.",
			want:  "See .agent/rules/instructions.md for setup.",
		},
		{
			name:  "procedure reference resolves to workflows path",
			input: "Run {{procedure:arch-correct}} when needed.",
			want:  "Run .agent/workflows/arch-correct.md when needed.",
		},
		{
			name:  "capability reference resolves to skills path",
			input: "Use {{capability:architecture-maintainer}} skill.",
			want:  "Use .agent/skills/architecture-maintainer/SKILL.md skill.",
		},
		{
			name:  "memory index reference resolves to memory path",
			input: "Check {{memory:index}} first.",
			want:  "Check prompts/memory/index.md first.",
		},
		{
			name:  "memory inbox reference",
			input: "Write to {{memory:inbox}} if unsure.",
			want:  "Write to prompts/memory/inbox.md if unsure.",
		},
		{
			name:  "target name resolves to target name",
			input: "Run update --target \"{{target:name}}\".",
			want:  "Run update --target \"antigravity\".",
		},
		{
			name:  "target meta_dir resolves to meta directory",
			input: "Check {{target:meta_dir}} for metadata.",
			want:  "Check .agent/.meta/ for metadata.",
		},
		{
			name:  "unknown target variable is left as-is",
			input: "See {{target:unknown}} for info.",
			want:  "See {{target:unknown}} for info.",
		},
		{
			name:  "unknown kind is left as-is",
			input: "See {{unknown:foo}} for info.",
			want:  "See {{unknown:foo}} for info.",
		},
		{
			name:  "no template variables returns input unchanged",
			input: "No variables here.",
			want:  "No variables here.",
		},
		{
			name:  "multiple variables in same text",
			input: "Read {{policy:coding-rules}} and {{policy:testing-rules}} then {{memory:invariants}}.",
			want:  "Read .agent/rules/coding-rules.md and .agent/rules/testing-rules.md then prompts/memory/invariants.md.",
		},
		{
			name:  "empty input returns empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplateVars(tt.input, ctx)
			if got != tt.want {
				t.Errorf("ResolveTemplateVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveTemplateVars_CustomPaths(t *testing.T) {
	// Verify that custom target paths are respected
	ctx := &TemplateContext{
		Paths: TargetPaths{
			Rules:     "custom/rules/",
			Skills:    "custom/skills/",
			Workflows: "custom/workflows/",
		},
		MemBase: "custom/memory",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "policy with custom rules path",
			input: "{{policy:coding-rules}}",
			want:  "custom/rules/coding-rules.md",
		},
		{
			name:  "procedure with custom workflows path",
			input: "{{procedure:build-pipeline}}",
			want:  "custom/workflows/build-pipeline.md",
		},
		{
			name:  "capability with custom skills path",
			input: "{{capability:test-skill}}",
			want:  "custom/skills/test-skill/SKILL.md",
		},
		{
			name:  "memory with custom base path",
			input: "{{memory:decisions}}",
			want:  "custom/memory/decisions.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplateVars(tt.input, ctx)
			if got != tt.want {
				t.Errorf("ResolveTemplateVars() = %q, want %q", got, tt.want)
			}
		})
	}
}
