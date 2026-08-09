package emitter

import (
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// ExtractTargetPaths merges defaults with optional overrides from target.Raw["paths"].
func ExtractTargetPaths(target *manifest.Entity, defaults TargetPaths) TargetPaths {
	tp := defaults
	if target == nil {
		return normalizeTargetPaths(tp)
	}
	paths, ok := target.Raw["paths"].(map[string]any)
	if !ok {
		return normalizeTargetPaths(tp)
	}
	if r, ok := paths["rules"].(string); ok {
		tp.Rules = r
	}
	if s, ok := paths["skills"].(string); ok {
		tp.Skills = s
	}
	if w, ok := paths["workflows"].(string); ok {
		tp.Workflows = w
	}
	return normalizeTargetPaths(tp)
}

func normalizeTargetPaths(tp TargetPaths) TargetPaths {
	if tp.Rules != "" {
		tp.Rules = ensureTrailingSlash(tp.Rules)
	}
	if tp.Skills != "" {
		tp.Skills = ensureTrailingSlash(tp.Skills)
	}
	if tp.Workflows != "" {
		tp.Workflows = ensureTrailingSlash(tp.Workflows)
	}
	return tp
}
