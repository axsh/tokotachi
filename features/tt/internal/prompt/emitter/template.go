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
}

// TargetPaths holds the target-specific output paths.
// All paths must end with a trailing slash.
type TargetPaths struct {
	Rules     string // e.g., ".agents/rules/"
	Skills    string // e.g., ".agents/skills/"
	Workflows string // e.g., ".agents/workflows/"
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
		return ensureTrailingSlash(ctx.Paths.Workflows) + id + ".md"
	case "capability":
		return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
	case "target":
		return resolveTargetVar(id, ctx)
	default:
		return ""
	}
}

// resolvePolicyPath resolves a policy ID to a file path.
// project-instructions is renamed to instructions.md by convention.
func resolvePolicyPath(id string, ctx *TemplateContext) string {
	filename := id + ".md"
	if id == "project-instructions" {
		filename = "instructions.md"
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
