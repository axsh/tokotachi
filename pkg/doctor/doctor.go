package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options holds configuration for the doctor run.
type Options struct {
	RepoRoot      string
	FeatureFilter string // empty = all features
	ToolChecker   ToolChecker
	Fix           bool
}

// Run executes all checks and returns a Report.
func Run(opts Options) (*Report, error) {
	report := &Report{}

	// 1. External tools
	report.Results = append(report.Results, checkExternalTools(opts.ToolChecker)...)

	// 2. Repository structure
	report.Results = append(report.Results, checkRepoStructure(opts.RepoRoot)...)

	// 3. Features
	var featureNames []string
	if opts.FeatureFilter != "" {
		// Validate that the feature exists
		featureDir := filepath.Join(opts.RepoRoot, "features", opts.FeatureFilter)
		info, err := os.Stat(featureDir)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("feature %q not found in features/ directory", opts.FeatureFilter)
		}
		featureNames = []string{opts.FeatureFilter}
	} else {
		var err error
		featureNames, err = discoverFeatures(opts.RepoRoot)
		if err != nil {
			// If features/ doesn't exist, we already reported it above
			featureNames = nil
		}
	}

	for _, name := range featureNames {
		report.Results = append(report.Results, checkFeature(opts.RepoRoot, name)...)
	}

	// 5. Apply fixes if requested
	if opts.Fix {
		applyFixes(opts.RepoRoot, report)
	}

	return report, nil
}

// applyFixes attempts to fix issues in the report that are auto-fixable.
func applyFixes(repoRoot string, report *Report) {
	for i, res := range report.Results {
		if res.Status != StatusWarn && res.Status != StatusFail {
			continue
		}

		var fixErr error
		var fixMsg string

		switch {
		// Directory not found (work/, scripts/)
		case res.Category == categoryRepo && (res.Status == StatusWarn || res.Status == StatusFail):
			dirName := strings.TrimSuffix(res.Name, "/")
			fixErr = fixDirectory(repoRoot, dirName)
			fixMsg = "directory created"
		}

		if fixMsg != "" {
			if fixErr == nil {
				report.Results[i].Status = StatusPass
				report.Results[i].Message = fixMsg
				report.Results[i].Fixed = true
				report.Results[i].FixHint = ""
				report.Results[i].Expected = ""
			} else {
				report.Results[i].Message = fmt.Sprintf("fix failed: %v", fixErr)
			}
		}
	}
}
