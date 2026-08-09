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
	Paths                     TargetPaths
	MemBase                   string // e.g., "prompts/memory"
	TargetName                string // e.g., "antigravity"
	PolicyExt                 string // ".md" or ".mdc"; empty defaults to ".md"
	RenameProjectInstructions bool   // true => project-instructions -> instructions{ext}
}

// TargetPaths holds the target-specific output paths.
// All paths must end with a trailing slash.
type TargetPaths struct {
	Rules     string // e.g., ".agents/rules/"
	Skills    string // e.g., ".agents/skills/"
	Workflows string // e.g., ".agents/workflows/"
}

// NewTemplateContext builds a TemplateContext with target-specific naming defaults.
func NewTemplateContext(targetName string, paths TargetPaths) *TemplateContext {
	ctx := &TemplateContext{
		Paths:      normalizeTargetPaths(paths),
		MemBase:    "prompts/memory",
		TargetName: targetName,
		PolicyExt:  ".md",
	}
	switch targetName {
	case "cursor":
		ctx.PolicyExt = ".mdc"
	case "antigravity":
		ctx.RenameProjectInstructions = true
	}
	return ctx
}

// ResolveTemplateVars replaces all {{kind:id}} occurrences in text
// with the resolved target-specific path.
// Unknown kind or id patterns are left as-is.
func ResolveTemplateVars(text string, ctx *TemplateContext) string {
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

// resolvePolicyPath resolves a policy ID to a file path.
// When RenameProjectInstructions is set, project-instructions becomes instructions{ext}.
func resolvePolicyPath(id string, ctx *TemplateContext) string {
	ext := ctx.PolicyExt
	if ext == "" {
		ext = ".md"
	}
	filename := id + ext
	if id == "project-instructions" && ctx.RenameProjectInstructions {
		filename = "instructions" + ext
	}
	return ensureTrailingSlash(ctx.Paths.Rules) + filename
}

// ensureTrailingSlash adds a trailing slash if not already present.
func ensureTrailingSlash(s string) string {
	if s == "" {
		return s
	}
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

// resolveTargetVar resolves a target-scoped template variable.
// Supported IDs: name, meta_dir, rules, skills, workflows.
func resolveTargetVar(id string, ctx *TemplateContext) string {
	switch id {
	case "name":
		return ctx.TargetName
	case "meta_dir":
		return resolve.MetaDir(ctx.TargetName)
	case "rules":
		return ensureTrailingSlash(ctx.Paths.Rules)
	case "skills":
		return ensureTrailingSlash(ctx.Paths.Skills)
	case "workflows":
		// Procedure home directory: prefer Workflows, else Skills.
		if ctx.Paths.Workflows != "" {
			return ensureTrailingSlash(ctx.Paths.Workflows)
		}
		return ensureTrailingSlash(ctx.Paths.Skills)
	default:
		return ""
	}
}
