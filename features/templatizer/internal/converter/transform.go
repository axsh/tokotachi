package converter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TransformResult holds the result of a single file transformation.
type TransformResult struct {
	Path        string // original file path (relative to tempDir)
	Transformed bool   // whether the file was transformed
}

// TransformGoFiles walks tempDir and applies AST transformations to all
// Go source files (*.go) and go.mod. Files that are transformed get a
// .tmpl postfix appended to their name.
//
// Processing order:
//  1. Pass 1: Process go.mod files to discover the original module path.
//  2. Pass 2: Process *.go files using the discovered module path for import replacement.
//
// If go.mod is found, the discovered module path takes priority over oldModule
// for .go file transformations. If go.mod is not found, oldModule is used as fallback.
func TransformGoFiles(tempDir string, oldModule, newModule string) ([]TransformResult, error) {
	var results []TransformResult

	// Pass 1: Find and process go.mod files to discover the original module path.
	var discoveredModule string

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		relPath, _ := filepath.Rel(tempDir, path)

		if name != "go.mod" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read %s: %w", relPath, readErr)
		}

		transformed, origModule, changed, transformErr := TransformGoMod(content, newModule)
		if transformErr != nil {
			return fmt.Errorf("failed to transform %s: %w", relPath, transformErr)
		}

		// Store the discovered original module path.
		if origModule != "" {
			discoveredModule = origModule
		}

		if changed {
			if writeErr := os.WriteFile(path, transformed, 0o644); writeErr != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, writeErr)
			}
			// Rename to .tmpl
			tmplPath := path + ".tmpl"
			if renameErr := os.Rename(path, tmplPath); renameErr != nil {
				return fmt.Errorf("failed to rename %s to .tmpl: %w", relPath, renameErr)
			}
		}

		results = append(results, TransformResult{Path: relPath, Transformed: changed})
		return nil
	})
	if err != nil {
		return results, err
	}

	// Determine effective oldModule for .go file transformations.
	effectiveOldModule := oldModule
	if discoveredModule != "" {
		effectiveOldModule = discoveredModule
	}

	// Pass 2: Process *.go files using the effective old module path.
	err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		relPath, _ := filepath.Rel(tempDir, path)

		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read %s: %w", relPath, readErr)
		}

		transformed, changed, transformErr := TransformGoSource(content, effectiveOldModule, newModule)
		if transformErr != nil {
			return fmt.Errorf("failed to transform %s: %w", relPath, transformErr)
		}

		if changed {
			if writeErr := os.WriteFile(path, transformed, 0o644); writeErr != nil {
				return fmt.Errorf("failed to write %s: %w", relPath, writeErr)
			}
			// Rename to .tmpl
			tmplPath := path + ".tmpl"
			if renameErr := os.Rename(path, tmplPath); renameErr != nil {
				return fmt.Errorf("failed to rename %s to .tmpl: %w", relPath, renameErr)
			}
		}

		results = append(results, TransformResult{Path: relPath, Transformed: changed})

		return nil
	})

	return results, err
}
