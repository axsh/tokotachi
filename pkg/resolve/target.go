package resolve

import (
	"fmt"
	"strings"
)

// EnvKeyTarget is the environment variable name for the default target.
const EnvKeyTarget = "TT_TARGET"

// KnownTargets is the list of canonical target names (sorted alphabetically).
var KnownTargets = []string{"antigravity", "claude-code", "codex", "cursor"}

// TargetAliases maps alias names to canonical target names.
var TargetAliases = map[string]string{
	"ag":     "antigravity",
	"agy":    "antigravity",
	"claude": "claude-code",
}

// AllTarget is the special value representing all targets.
const AllTarget = "all"

// targetMetaDirs maps canonical target names to their metadata directories.
var targetMetaDirs = map[string]string{
	"antigravity": ".agent/.meta/antigravity/",
	"cursor":      ".cursor/.meta/",
	"claude-code": ".claude/.meta/",
	"codex":       ".codex/.meta/codex/",
}

// ResolveTarget resolves a target name using prefix matching.
// It first checks aliases, then exact matches, then prefix matches.
// Returns an error if the input is ambiguous (multiple matches) or unknown.
// allowAll controls whether "all" is a valid target name.
func ResolveTarget(input string, allowAll bool) (string, error) {
	if input == "" {
		return "", fmt.Errorf("target name cannot be empty")
	}

	// 1. Check aliases for exact match
	if canonical, ok := TargetAliases[input]; ok {
		return canonical, nil
	}

	// 2. Build full candidate list (always includes "all" for matching)
	candidates := make([]string, 0, len(KnownTargets)+1)
	candidates = append(candidates, KnownTargets...)
	candidates = append(candidates, AllTarget)

	// 3. Check for exact match
	for _, c := range candidates {
		if c == input {
			if c == AllTarget && !allowAll {
				return "", fmt.Errorf("target %q is not allowed in this context", AllTarget)
			}
			return c, nil
		}
	}

	// 4. Prefix matching
	var matches []string
	for _, c := range candidates {
		if strings.HasPrefix(c, input) {
			matches = append(matches, c)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("unknown target %q", input)
	case 1:
		result := matches[0]
		if result == AllTarget && !allowAll {
			return "", fmt.Errorf("target %q is not allowed in this context", AllTarget)
		}
		return result, nil
	default:
		return "", fmt.Errorf("ambiguous target %q: matches %v", input, matches)
	}
}

// ResolveTargets resolves a target name and returns the list of concrete
// target names. When input resolves to "all", it returns all known targets.
func ResolveTargets(input string) ([]string, error) {
	resolved, err := ResolveTarget(input, true)
	if err != nil {
		return nil, err
	}

	if resolved == AllTarget {
		result := make([]string, len(KnownTargets))
		copy(result, KnownTargets)
		return result, nil
	}

	return []string{resolved}, nil
}

// MetaDir returns the metadata directory path for a given target.
// Returns empty string for unknown targets.
func MetaDir(target string) string {
	return targetMetaDirs[target]
}
