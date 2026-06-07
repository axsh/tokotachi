package converter

import (
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

// templateVarRe matches {{var_name}} template variable placeholders.
var templateVarRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ParamCollector accumulates template variable names found during conversion.
type ParamCollector struct {
	params map[string]bool
}

// NewParamCollector creates a new ParamCollector.
func NewParamCollector() *ParamCollector {
	return &ParamCollector{params: make(map[string]bool)}
}

// Add records a discovered template variable name.
func (pc *ParamCollector) Add(name string) {
	pc.params[name] = true
}

// AddFromString extracts all {{xxx}} template variables from s and adds them.
func (pc *ParamCollector) AddFromString(s string) {
	for _, name := range ExtractTemplateVars(s) {
		pc.Add(name)
	}
}

// Names returns sorted list of all discovered param names.
func (pc *ParamCollector) Names() []string {
	if len(pc.params) == 0 {
		return nil
	}
	names := make([]string, 0, len(pc.params))
	for name := range pc.params {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ExtractTemplateVars extracts {{xxx}} variable names from a string.
// Returns sorted, deduplicated list.
func ExtractTemplateVars(s string) []string {
	matches := templateVarRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

// MergeParams merges discovered params (from ParamCollector) with
// the original scaffold.yaml template_params.
//   - Params in both: use scaffold.yaml definition (has description, default, etc.)
//   - Params discovered but not in scaffold.yaml: auto-add with name only, required=true
//   - Params in scaffold.yaml but not discovered: keep as-is
func MergeParams(defined []catalog.TemplateParam, discovered []string) []catalog.TemplateParam {
	// Build lookup map from defined params.
	definedMap := make(map[string]catalog.TemplateParam, len(defined))
	for _, p := range defined {
		definedMap[p.Name] = p
	}

	// Build set of discovered params.
	discoveredSet := make(map[string]bool, len(discovered))
	for _, name := range discovered {
		discoveredSet[name] = true
	}

	// Merge: start with discovered params (preserving defined metadata).
	merged := make(map[string]catalog.TemplateParam)
	for _, name := range discovered {
		if p, ok := definedMap[name]; ok {
			merged[name] = p
		} else {
			// Auto-add undiscovered param with warning.
			fmt.Fprintf(os.Stderr, "  [WARN] Template variable {{%s}} found during conversion but not defined in scaffold.yaml template_params. Auto-adding.\n", name)
			merged[name] = catalog.TemplateParam{
				Name:      name,
				Required:  true,
				ValueSpec: catalog.DefaultValueSpec(),
			}
		}
	}

	// Add defined params that were not discovered (keep as-is).
	for _, p := range defined {
		if _, exists := merged[p.Name]; !exists {
			merged[p.Name] = p
		}
	}

	// Sort by name and return.
	result := make([]catalog.TemplateParam, 0, len(merged))
	for _, p := range merged {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}
