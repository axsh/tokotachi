package converter

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// goModModuleRe matches the "module <path>" line in go.mod files.
var goModModuleRe = regexp.MustCompile(`(?m)^module\s+(.+)$`)

// TransformGoMod transforms the module directive in a go.mod file.
// It replaces the module path with newModule unconditionally.
// Returns the transformed content, the original module path, whether a change was made, and any error.
func TransformGoMod(content []byte, newModule string) ([]byte, string, bool, error) {
	src := string(content)

	match := goModModuleRe.FindStringSubmatch(src)
	if match == nil {
		return content, "", false, nil
	}

	currentModule := strings.TrimSpace(match[1])
	if currentModule == newModule {
		return content, currentModule, false, nil
	}

	result := goModModuleRe.ReplaceAllStringFunc(src, func(line string) string {
		return "module " + newModule
	})

	return []byte(result), currentModule, true, nil
}

// TransformGoSource transforms import paths in a Go source file.
// It replaces import paths that match oldModule (exact or prefix) with newModule.
// Returns the transformed content, whether a change was made, and any error.
func TransformGoSource(content []byte, oldModule, newModule string) ([]byte, bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "source.go", content, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse Go source: %w", err)
	}

	changed := false

	ast.Inspect(f, func(n ast.Node) bool {
		importSpec, ok := n.(*ast.ImportSpec)
		if !ok {
			return true
		}

		// Get the import path value (unquoted).
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			return true
		}

		// Check for exact match or prefix match (with "/" separator).
		if importPath == oldModule {
			importSpec.Path.Value = strconv.Quote(newModule)
			changed = true
		} else if strings.HasPrefix(importPath, oldModule+"/") {
			newPath := newModule + importPath[len(oldModule):]
			importSpec.Path.Value = strconv.Quote(newPath)
			changed = true
		}

		return true
	})

	if !changed {
		return content, false, nil
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, f); err != nil {
		return nil, false, fmt.Errorf("failed to format Go source: %w", err)
	}

	return []byte(buf.String()), true, nil
}
