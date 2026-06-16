package emitter

import (
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// EntityIncludes holds entity-type inclusion flags for a target.
// Each flag controls whether entities of that type should be emitted.
type EntityIncludes struct {
	Policy     bool
	Capability bool
	Procedure  bool
	Subagent   bool
}

// ExtractIncludes reads includes flags from a target entity.
// Returns defaults (all true except Subagent) if the target is nil or has no includes.
// Supports both "includes" (new) and "capabilities" (legacy) keys for backward compatibility.
func ExtractIncludes(target *manifest.Entity) EntityIncludes {
	defaults := EntityIncludes{
		Policy:     true,
		Capability: true,
		Procedure:  true,
		Subagent:   false,
	}
	if target == nil {
		return defaults
	}

	// Try "includes" first, fallback to "capabilities" for backward compat
	raw, ok := target.Raw["includes"].(map[string]any)
	if !ok {
		raw, ok = target.Raw["capabilities"].(map[string]any)
		if !ok {
			return defaults
		}
	}

	result := defaults

	// New entity-type keys
	if v, ok := raw["policy"].(bool); ok {
		result.Policy = v
	}
	if v, ok := raw["capability"].(bool); ok {
		result.Capability = v
	}
	if v, ok := raw["procedure"].(bool); ok {
		result.Procedure = v
	}
	if v, ok := raw["subagent"].(bool); ok {
		result.Subagent = v
	}

	// Legacy key mapping (output-dir-based -> entity-type-based)
	if v, ok := raw["rules"].(bool); ok {
		result.Policy = v
	}
	if v, ok := raw["skills"].(bool); ok {
		result.Capability = v
	}
	if v, ok := raw["workflows"].(bool); ok {
		result.Procedure = v
	}
	if v, ok := raw["subagents"].(bool); ok {
		result.Subagent = v
	}

	return result
}
