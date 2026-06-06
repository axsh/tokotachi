package converter

import (
	"fmt"
	"strings"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

// ConvertParams holds the parameters for the template conversion pipeline.
type ConvertParams struct {
	// OldModule is the original module path in the source code (e.g. "function").
	OldModule string
	// NewModule is the new module path to replace with (e.g. "{{module_path}}/{{feature_name}}").
	NewModule string
	// OldProgram is the original feature/program name used for directory renaming (e.g. "function").
	OldProgram string
	// NewProgram is the new feature/program name for directory renaming (e.g. "my-app").
	NewProgram string
	// HintParams provides values for {{param}} placeholders in .hints files.
	HintParams map[string]string
}

// Convert executes the full template conversion pipeline on tempDir.
//
// Processing order:
//
//	Step 1: Cleanup (remove .git, go.sum, vendor, bin, .DS_Store)
//	Step 2: AST transformation (go.mod, *.go → .tmpl postfix)
//	Step 3: Directory rename (cmd/<old> → cmd/<new>)
//	Step 4: Hint file processing (*.hints → apply replacements, .tmpl postfix)
//
// Returns a ParamCollector with all template variables discovered during conversion.
// If the params indicate no conversion is needed (empty OldModule), the
// pipeline is skipped entirely.
func Convert(tempDir string, params ConvertParams) (*ParamCollector, error) {
	pc := NewParamCollector()

	// Skip if no conversion params provided.
	if params.OldModule == "" {
		return pc, nil
	}

	// Step 1: Cleanup
	if err := Clean(tempDir, DefaultExcludes); err != nil {
		return nil, fmt.Errorf("step 1 (cleanup) failed: %w", err)
	}

	// Step 2: AST transformation
	// Collect template vars from the newModule string.
	pc.AddFromString(params.NewModule)

	if _, err := TransformGoFiles(tempDir, params.OldModule, params.NewModule); err != nil {
		return nil, fmt.Errorf("step 2 (AST transform) failed: %w", err)
	}

	// Step 3: Directory rename
	if err := RenameDirectories(tempDir, params.OldProgram, params.NewProgram); err != nil {
		return nil, fmt.Errorf("step 3 (rename) failed: %w", err)
	}

	// Step 4: Hint file processing
	// Collect template vars from hint replacements before processing.
	hintVars, err := CollectHintTemplateVars(tempDir)
	if err != nil {
		return nil, fmt.Errorf("step 4 (collect hint vars) failed: %w", err)
	}
	for _, v := range hintVars {
		pc.Add(v)
	}

	if err := ProcessHints(tempDir, params.HintParams); err != nil {
		return nil, fmt.Errorf("step 4 (hints) failed: %w", err)
	}

	return pc, nil
}

// BuildConvertParams constructs ConvertParams from catalog TemplateParams.
// It extracts module_path and feature_name parameters from the template params.
// NewModule is set to template variable format (e.g. "{{module_path}}/{{feature_name}}")
// so that go.mod module line becomes a template placeholder.
func BuildConvertParams(templateParams []catalog.TemplateParam) ConvertParams {
	if len(templateParams) == 0 {
		return ConvertParams{}
	}

	params := ConvertParams{
		HintParams: make(map[string]string),
	}

	for _, tp := range templateParams {
		// Resolve old_value: explicit old_value takes priority, fallback to default.
		oldValue := tp.OldValue
		if oldValue == "" {
			oldValue = tp.Default
		}

		switch tp.Name {
		case "module_path":
			params.OldModule = oldValue
		case "feature_name":
			params.OldProgram = oldValue
			params.NewProgram = oldValue
		}
		// Populate hint params with resolved old_value.
		params.HintParams[tp.Name] = oldValue
	}

	// Strip feature_name suffix from module_path HintParams value.
	// OldModule retains the full path for go.mod/go source replacement,
	// but HintParams["module_path"] should represent only the prefix portion
	// so that {{module_path}}/{{feature_name}} reconstructs the full path.
	if featureName, ok := params.HintParams["feature_name"]; ok {
		if mp, ok := params.HintParams["module_path"]; ok {
			params.HintParams["module_path"] = stripFeatureNameSuffix(mp, featureName)
		}
	}

	// Construct template variable for module path in go.mod.
	if _, hasModulePath := params.HintParams["module_path"]; hasModulePath {
		if _, hasFeatureName := params.HintParams["feature_name"]; hasFeatureName {
			params.NewModule = "{{module_path}}/{{feature_name}}"
		} else {
			params.NewModule = "{{module_path}}"
		}
	}

	return params
}

// stripFeatureNameSuffix removes the trailing /<featureName> segment from
// modulePath if it matches. Returns the original modulePath if there is
// no match or if stripping would produce an empty string.
func stripFeatureNameSuffix(modulePath, featureName string) string {
	suffix := "/" + featureName
	if strings.HasSuffix(modulePath, suffix) {
		stripped := modulePath[:len(modulePath)-len(suffix)]
		if stripped != "" {
			return stripped
		}
	}
	return modulePath
}
