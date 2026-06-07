package converter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// HintFile represents the YAML structure of a .hints file.
type HintFile struct {
	Replacements []HintReplacement `yaml:"replacements"`
}

// HintReplacement represents a single replacement rule in a hints file.
type HintReplacement struct {
	Match       string `yaml:"match"`
	ReplaceWith string `yaml:"replace_with"`
}

// ProcessHints finds all *.hints files in tempDir, applies the replacement
// rules to their corresponding target files, renames the target files with
// a .tmpl postfix, and removes the .hints files.
//
// The params map provides values for {{param}} placeholders in replace_with fields.
func ProcessHints(tempDir string, params map[string]string) error {
	// Collect all hints files first to avoid modifying the tree while walking.
	var hintsFiles []string
	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".hints") {
			hintsFiles = append(hintsFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory for hints: %w", err)
	}

	for _, hintsPath := range hintsFiles {
		if err := processOneHint(hintsPath, params); err != nil {
			return err
		}
	}

	return nil
}

// processOneHint processes a single .hints file.
func processOneHint(hintsPath string, params map[string]string) error {
	// Read and parse the hints file.
	hintsData, err := os.ReadFile(hintsPath)
	if err != nil {
		return fmt.Errorf("failed to read hints file %s: %w", hintsPath, err)
	}

	var hints HintFile
	if err := yaml.Unmarshal(hintsData, &hints); err != nil {
		return fmt.Errorf("failed to parse hints file %s: %w", hintsPath, err)
	}

	// Determine the target file path (remove .hints suffix).
	targetPath := strings.TrimSuffix(hintsPath, ".hints")

	// Read the target file.
	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read target file %s: %w", targetPath, err)
	}

	// Sort replacements by match length (longest first) to prevent
	// shorter matches from corrupting longer ones.
	sortedRepls := make([]HintReplacement, len(hints.Replacements))
	copy(sortedRepls, hints.Replacements)
	sortByMatchLength(sortedRepls)

	// Apply replacements.
	content := string(targetData)
	for _, repl := range sortedRepls {
		// Expand {{param}} placeholders in replace_with.
		expandedReplacement := expandPlaceholders(repl.ReplaceWith, params)
		content = strings.ReplaceAll(content, repl.Match, expandedReplacement)
	}

	// Write the transformed content back.
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write target file %s: %w", targetPath, err)
	}

	// Rename the target file to .tmpl.
	tmplPath := targetPath + ".tmpl"
	if err := os.Rename(targetPath, tmplPath); err != nil {
		return fmt.Errorf("failed to rename %s to .tmpl: %w", targetPath, err)
	}

	// Remove the hints file.
	if err := os.Remove(hintsPath); err != nil {
		return fmt.Errorf("failed to remove hints file %s: %w", hintsPath, err)
	}

	return nil
}

// expandPlaceholders replaces {{param}} placeholders in s with values from params.
func expandPlaceholders(s string, params map[string]string) string {
	result := s
	for key, value := range params {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// sortByMatchLength sorts replacements by match string length in descending order.
// This ensures longer matches are applied first, preventing shorter matches from
// corrupting parts of longer match strings.
func sortByMatchLength(repls []HintReplacement) {
	sort.Slice(repls, func(i, j int) bool {
		return len(repls[i].Match) > len(repls[j].Match)
	})
}

// CollectHintTemplateVars scans all .hints files in tempDir and extracts
// template variable names from replace_with fields. This should be called
// before ProcessHints since ProcessHints removes .hints files.
func CollectHintTemplateVars(tempDir string) ([]string, error) {
	var allVars []string
	seen := make(map[string]bool)

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".hints") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read hints file %s: %w", path, readErr)
		}

		var hints HintFile
		if parseErr := yaml.Unmarshal(data, &hints); parseErr != nil {
			return fmt.Errorf("failed to parse hints file %s: %w", path, parseErr)
		}

		for _, repl := range hints.Replacements {
			for _, v := range ExtractTemplateVars(repl.ReplaceWith) {
				if !seen[v] {
					seen[v] = true
					allVars = append(allVars, v)
				}
			}
		}

		return nil
	})

	sort.Strings(allVars)
	return allVars, err
}
