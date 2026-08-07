package emitter

import (
	"testing"
)

func TestResolveTemplateVars(t *testing.T) {
	ctx := &TemplateContext{
		Paths: TargetPaths{
			Rules:  ".agents/rules/",
			Skills: ".agents/skills/",
		},
		MemBase:                   "prompts/memory",
		TargetName:                "antigravity",
		PolicyExt:                 ".md",
		RenameProjectInstructions: true,
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "policy reference resolves to rules path",
			input: "Read {{policy:coding-rules}} for details.",
			want:  "Read .agents/rules/coding-rules.md for details.",
		},
		{
			name:  "policy project-instructions renames to instructions.md",
			input: "See {{policy:project-instructions}} for setup.",
			want:  "See .agents/rules/instructions.md for setup.",
		},
		{
			name:  "procedure reference resolves to skills path",
			input: "Run {{procedure:arch-correct}} when needed.",
			want:  "Run .agents/skills/arch-correct/SKILL.md when needed.",
		},
		{
			name:  "capability reference resolves to skills path",
			input: "Use {{capability:architecture-maintainer}} skill.",
			want:  "Use .agents/skills/architecture-maintainer/SKILL.md skill.",
		},
		{
			name:  "target name resolves to target name",
			input: "Run update --target \"{{target:name}}\".",
			want:  "Run update --target \"antigravity\".",
		},
		{
			name:  "target meta_dir resolves to meta directory",
			input: "Check {{target:meta_dir}} for metadata.",
			want:  "Check .agent/.meta/antigravity/ for metadata.",
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
			input: "Read {{policy:coding-rules}} and {{policy:testing-rules}}.",
			want:  "Read .agents/rules/coding-rules.md and .agents/rules/testing-rules.md.",
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

func TestResolveTemplateVars_PerTargetPolicyNaming(t *testing.T) {
	tests := []struct {
		name string
		ctx  *TemplateContext
		in   string
		want string
	}{
		{
			name: "claude policy testing-rules",
			ctx: NewTemplateContext("claude-code", TargetPaths{
				Rules:  ".claude/rules/",
				Skills: ".claude/skills/",
			}),
			in:   "{{policy:testing-rules}}",
			want: ".claude/rules/testing-rules.md",
		},
		{
			name: "codex policy testing-rules",
			ctx: NewTemplateContext("codex", TargetPaths{
				Rules:  ".codex/rules/",
				Skills: ".codex/skills/",
			}),
			in:   "{{policy:testing-rules}}",
			want: ".codex/rules/testing-rules.md",
		},
		{
			name: "cursor policy testing-rules uses mdc",
			ctx: NewTemplateContext("cursor", TargetPaths{
				Rules:  ".cursor/rules/",
				Skills: ".cursor/skills/",
			}),
			in:   "{{policy:testing-rules}}",
			want: ".cursor/rules/testing-rules.mdc",
		},
		{
			name: "cursor project-instructions keeps id with mdc",
			ctx: NewTemplateContext("cursor", TargetPaths{
				Rules:  ".cursor/rules/",
				Skills: ".cursor/skills/",
			}),
			in:   "{{policy:project-instructions}}",
			want: ".cursor/rules/project-instructions.mdc",
		},
		{
			name: "claude project-instructions keeps id",
			ctx: NewTemplateContext("claude-code", TargetPaths{
				Rules:  ".claude/rules/",
				Skills: ".claude/skills/",
			}),
			in:   "{{policy:project-instructions}}",
			want: ".claude/rules/project-instructions.md",
		},
		{
			name: "antigravity project-instructions renames",
			ctx: NewTemplateContext("antigravity", TargetPaths{
				Rules:     ".agent/rules/",
				Skills:    ".agent/skills/",
				Workflows: ".agent/workflows/",
			}),
			in:   "{{policy:project-instructions}}",
			want: ".agent/rules/instructions.md",
		},
		{
			name: "claude procedure falls back to skills",
			ctx: NewTemplateContext("claude-code", TargetPaths{
				Rules:  ".claude/rules/",
				Skills: ".claude/skills/",
			}),
			in:   "{{procedure:build-pipeline}}",
			want: ".claude/skills/build-pipeline/SKILL.md",
		},
		{
			name: "antigravity procedure uses workflows",
			ctx: NewTemplateContext("antigravity", TargetPaths{
				Rules:     ".agent/rules/",
				Skills:    ".agent/skills/",
				Workflows: ".agent/workflows/",
			}),
			in:   "{{procedure:build-pipeline}}",
			want: ".agent/workflows/build-pipeline.md",
		},
		{
			name: "antigravity target workflows dir",
			ctx: NewTemplateContext("antigravity", TargetPaths{
				Rules:     ".agent/rules/",
				Skills:    ".agent/skills/",
				Workflows: ".agent/workflows/",
			}),
			in:   "{{target:workflows}}",
			want: ".agent/workflows/",
		},
		{
			name: "claude target workflows falls back to skills",
			ctx: NewTemplateContext("claude-code", TargetPaths{
				Rules:  ".claude/rules/",
				Skills: ".claude/skills/",
			}),
			in:   "{{target:workflows}}",
			want: ".claude/skills/",
		},
		{
			name: "codex target rules",
			ctx: NewTemplateContext("codex", TargetPaths{
				Rules:  ".codex/rules/",
				Skills: ".codex/skills/",
			}),
			in:   "{{target:rules}}",
			want: ".codex/rules/",
		},
		{
			name: "cursor target skills",
			ctx: NewTemplateContext("cursor", TargetPaths{
				Rules:  ".cursor/rules/",
				Skills: ".cursor/skills/",
			}),
			in:   "{{target:skills}}",
			want: ".cursor/skills/",
		},
		{
			name: "unknown kind left as-is",
			ctx: NewTemplateContext("claude-code", TargetPaths{
				Rules:  ".claude/rules/",
				Skills: ".claude/skills/",
			}),
			in:   "{{unknown:foo}}",
			want: "{{unknown:foo}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplateVars(tt.in, tt.ctx)
			if got != tt.want {
				t.Errorf("ResolveTemplateVars() = %q, want %q", got, tt.want)
			}
		})
	}
}
