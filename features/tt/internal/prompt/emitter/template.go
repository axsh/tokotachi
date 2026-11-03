package emitter

import (
	"regexp"
	"strings"

	"github.com/axsh/tokotachi/pkg/resolve"
)

// templateVarRegex matches {{kind:id}} patterns where kind is a word
// and id is a word optionally containing hyphens.
var templateVarRegex = regexp.MustCompile(`\{\{(\w+):([\w][\w-]*)\}\}`)

// TemplateContext holds the information needed to resolve template variables.
type TemplateContext struct {
	Paths      TargetPaths
	MemBase    string // e.g., "prompts/memory"
	TargetName string // e.g., "antigravity"
	// PolicyExt is the file extension for policy outputs, including the dot.
	// Cursor uses ".mdc"; other targets use ".md". Empty means ".md".
	PolicyExt string
	// RenameProjectInstructionsTo, when non-empty, replaces the filename for
	// id "project-instructions" (Antigravity: "instructions.md").
	RenameProjectInstructionsTo string
}

// TargetPaths holds the target-specific output paths.
// All paths must end with a trailing slash.
type TargetPaths struct {
	Rules     string // e.g., ".agents/rules/"
	Skills    string // e.g., ".agents/skills/"
	Workflows string // e.g., ".agents/workflows/"
}

// BuildTemplateContext creates a TemplateContext for the given target ID and paths.
func BuildTemplateContext(targetID string, paths TargetPaths) *TemplateContext {
	ctx := &TemplateContext{
		Paths:      paths,
		MemBase:    resolveMemoryBase(),
		TargetName: targetID,
		PolicyExt:  ".md",
	}
	switch targetID {
	case "cursor":
		ctx.PolicyExt = ".mdc"
	case "antigravity":
		ctx.RenameProjectInstructionsTo = "instructions.md"
	}
	return ctx
}

// ResolveTemplateVars replaces all {{kind:id}} occurrences in text
// with the resolved target-specific path.
// Unknown kind or id patterns are left as-is.
func ResolveTemplateVars(text string, ctx *TemplateContext) string {
	if ctx == nil {
		return text
	}
	return templateVarRegex.ReplaceAllStringFunc(text, func(match string) string {
		subs := templateVarRegex.FindStringSubmatch(match)
		if len(subs) != 3 {
			return match
		}
		kind := subs[1]
		id := subs[2]
		resolved := resolveRef(kind, id, ctx)
		if resolved == "" {
			return match
		}
		return resolved
	})
}

// resolveRef resolves a single template reference to a target-specific path.
// Returns empty string if the kind is unknown.
func resolveRef(kind, id string, ctx *TemplateContext) string {
	switch kind {
	case "policy":
		return resolvePolicyPath(id, ctx)
	case "procedure":
		if ctx.Paths.Workflows != "" {
			return ensureTrailingSlash(ctx.Paths.Workflows) + id + ".md"
		}
		return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
	case "capability":
		return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
	case "target":
		return resolveTargetVar(id, ctx)
	default:
		return ""
	}
}

// resolvePolicyPath resolves a policy ID to a file path using target naming rules.
func resolvePolicyPath(id string, ctx *TemplateContext) string {
	ext := ctx.PolicyExt
	if ext == "" {
		ext = ".md"
	}
	filename := id + ext
	if id == "project-instructions" && ctx.RenameProjectInstructionsTo != "" {
		filename = ctx.RenameProjectInstructionsTo
	}
	return ensureTrailingSlash(ctx.Paths.Rules) + filename
}

// ensureTrailingSlash adds a trailing slash if not already present.
func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

// resolveTargetVar resolves a target-scoped template variable.
// Supported IDs: "name" (target name), "meta_dir" (metadata directory).
func resolveTargetVar(id string, ctx *TemplateContext) string {
	switch id {
	case "name":
		return ctx.TargetName
	case "meta_dir":
		return resolve.MetaDir(ctx.TargetName)
	default:
		return ""
	}
}

// TemplateRefManual returns the {{kind:id}} template reference manual text.
func TemplateRefManual() string {
	return `Tokotachi prompt template references
====================================

Format
------
  {{kind:id}}

Manifest body text may use these target-independent references. They are
resolved to agent-specific paths at emit time (tt prompt deploy / update),
not at compile time.

Supported kinds
---------------
  policy      Rules path for the policy id
  procedure   Workflows/{id}.md when the target has workflows; otherwise skills/{id}/SKILL.md
  capability  skills/{id}/SKILL.md
  target      Special variables: name, meta_dir

Unsupported / not resolved
--------------------------
  Unknown kinds are left unchanged (e.g. {{unknown:foo}}).
  skip, bundle, guard, worker have no deploy path and are not resolved.
  memory kind was removed and is not reintroduced.

Per-target examples
-------------------
  Cursor:
    {{policy:coding-rules}}           -> .cursor/rules/coding-rules.mdc
    {{policy:project-instructions}}   -> .cursor/rules/project-instructions.mdc
    {{procedure:test-generator}}      -> .cursor/skills/test-generator/SKILL.md
    {{capability:portable-file-references}} -> .cursor/skills/portable-file-references/SKILL.md

  Claude Code:
    {{policy:coding-rules}}           -> .claude/rules/coding-rules.md
    {{policy:project-instructions}}   -> .claude/rules/project-instructions.md
    {{procedure:test-generator}}      -> .claude/skills/test-generator/SKILL.md

  Codex:
    {{policy:coding-rules}}           -> .codex/rules/coding-rules.md
    {{policy:project-instructions}}   -> .codex/rules/project-instructions.md
    {{procedure:test-generator}}      -> .codex/skills/test-generator/SKILL.md

  Antigravity:
    {{policy:coding-rules}}           -> .agent/rules/coding-rules.md
    {{policy:project-instructions}}   -> .agent/rules/instructions.md
    {{procedure:test-generator}}      -> .agent/workflows/test-generator.md
    {{capability:portable-file-references}} -> .agent/skills/portable-file-references/SKILL.md

Base paths (rules / skills / workflows) follow each target YAML paths.* values.
`
}