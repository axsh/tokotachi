package emitter

import (
	"strings"
	"testing"
)

func TestResolveTemplateVars(t *testing.T) {
	ctx := BuildTemplateContext("antigravity", TargetPaths{
		Rules:  ".agents/rules/",
		Skills: ".agents/skills/",
	})
	// BuildTemplateContext sets RenameProjectInstructionsTo for antigravity.
	ctx.Paths.Rules = ".agents/rules/"
	ctx.Paths.Skills = ".agents/skills/"

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
			name:  "procedure reference resolves to skills path when workflows empty",
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
		{
			name:  "backslash escape leaves literal mustache",
			input: "Docs: \\{{policy:coding-rules}}.",
			want:  "Docs: {{policy:coding-rules}}.",
		},
		{
			name:  "backslash escape on unknown kind still strips slash",
			input: "\\{{unknown:foo}}",
			want:  "{{unknown:foo}}",
		},
		{
			name:  "escaped and live refs in same text",
			input: "Esc \\{{policy:coding-rules}} live {{procedure:arch-correct}}.",
			want:  "Esc {{policy:coding-rules}} live .agents/skills/arch-correct/SKILL.md.",
		},
		{
			name:  "double backslash yields literal slash plus resolved path",
			input: "\\\\{{policy:coding-rules}}",
			want:  "\\.agents/rules/coding-rules.md",
		},
		{
			name:  "triple backslash yields slash plus literal mustache",
			input: "\\\\\\{{policy:coding-rules}}",
			want:  "\\{{policy:coding-rules}}",
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
	ctx := BuildTemplateContext("antigravity", TargetPaths{
		Rules:     "custom/rules/",
		Skills:    "custom/skills/",
		Workflows: "custom/workflows/",
	})

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

func TestResolveTemplateVars_CursorPolicyExt(t *testing.T) {
	ctx := BuildTemplateContext("cursor", TargetPaths{
		Rules:  ".cursor/rules/",
		Skills: ".cursor/skills/",
	})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "policy uses mdc",
			input: "{{policy:coding-rules}}",
			want:  ".cursor/rules/coding-rules.mdc",
		},
		{
			name:  "project-instructions keeps id with mdc",
			input: "{{policy:project-instructions}}",
			want:  ".cursor/rules/project-instructions.mdc",
		},
		{
			name:  "procedure to skills",
			input: "{{procedure:test-generator}}",
			want:  ".cursor/skills/test-generator/SKILL.md",
		},
		{
			name:  "capability to skills",
			input: "{{capability:portable-file-references}}",
			want:  ".cursor/skills/portable-file-references/SKILL.md",
		},
		{
			name:  "backslash escape leaves literal mustache",
			input: "\\{{policy:coding-rules}}",
			want:  "{{policy:coding-rules}}",
		},
		{
			name:  "double backslash yields slash plus mdc path",
			input: "\\\\{{policy:coding-rules}}",
			want:  "\\.cursor/rules/coding-rules.mdc",
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

func TestResolveTemplateVars_CodexAndClaude(t *testing.T) {
	codex := BuildTemplateContext("codex", TargetPaths{
		Rules:  ".codex/rules/",
		Skills: ".codex/skills/",
	})
	claude := BuildTemplateContext("claude-code", TargetPaths{
		Rules:  ".claude/rules/",
		Skills: ".claude/skills/",
	})

	if got := ResolveTemplateVars("{{policy:coding-rules}}", codex); got != ".codex/rules/coding-rules.md" {
		t.Errorf("codex policy = %q", got)
	}
	if got := ResolveTemplateVars("{{policy:project-instructions}}", codex); got != ".codex/rules/project-instructions.md" {
		t.Errorf("codex project-instructions = %q", got)
	}
	if got := ResolveTemplateVars("{{procedure:test-generator}}", codex); got != ".codex/skills/test-generator/SKILL.md" {
		t.Errorf("codex procedure = %q", got)
	}
	if got := ResolveTemplateVars("{{policy:coding-rules}}", claude); got != ".claude/rules/coding-rules.md" {
		t.Errorf("claude policy = %q", got)
	}
	if got := ResolveTemplateVars("{{policy:project-instructions}}", claude); got != ".claude/rules/project-instructions.md" {
		t.Errorf("claude project-instructions = %q", got)
	}
}

func TestResolveTemplateVars_AntigravityPolicyRename(t *testing.T) {
	ctx := BuildTemplateContext("antigravity", TargetPaths{
		Rules:     ".agent/rules/",
		Skills:    ".agent/skills/",
		Workflows: ".agent/workflows/",
	})
	if got := ResolveTemplateVars("{{policy:project-instructions}}", ctx); got != ".agent/rules/instructions.md" {
		t.Errorf("got %q", got)
	}
	if got := ResolveTemplateVars("{{procedure:test-generator}}", ctx); got != ".agent/workflows/test-generator.md" {
		t.Errorf("got %q", got)
	}
}

func TestTemplateRefManual_ContainsRequiredSections(t *testing.T) {
	manual := TemplateRefManual()
	required := []string{
		"{{kind:id}}",
		"policy",
		"procedure",
		"capability",
		"target",
		"emit",
		".mdc",
		"instructions.md",
		"unknown",
		"memory",
		"Escape",
		"\\{{kind:id}}",
	}
	for _, s := range required {
		if !strings.Contains(manual, s) {
			t.Errorf("TemplateRefManual missing %q", s)
		}
	}
}
