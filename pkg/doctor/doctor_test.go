package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_AllPass(t *testing.T) {
	root := t.TempDir()

	// Create minimal valid structure
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features", "myfeature", ".devcontainer"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "work"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"),
		[]byte("project_name: test\ndefault_editor: cursor\ndefault_container_mode: docker-local\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "features", "myfeature", ".devcontainer", "devcontainer.json"),
		[]byte(`{"name":"myfeature"}`), 0o644))

	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}

	report, err := Run(Options{
		RepoRoot:    root,
		ToolChecker: checker,
	})
	require.NoError(t, err)
	assert.False(t, report.HasFailures(), "expected no failures in valid setup")

	s := report.Summary()
	assert.Equal(t, 0, s.Failed)
}

func TestRun_WithFailure(t *testing.T) {
	root := t.TempDir()

	// Create structure with invalid .devrc.yaml
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features", "broken"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "work"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"),
		[]byte("key: [unterminated\n\t: bad:\nindent"), 0o644))
	// feature.yaml missing is now WARN, but .devrc.yaml parse error is FAIL

	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}

	report, err := Run(Options{
		RepoRoot:    root,
		ToolChecker: checker,
	})
	require.NoError(t, err)
	assert.True(t, report.HasFailures(), "expected failures with invalid config")
}

func TestRun_FeatureFilter(t *testing.T) {
	root := t.TempDir()

	// Create two features
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features", "alpha"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features", "beta"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "work"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"),
		[]byte("project_name: test\ndefault_editor: cursor\ndefault_container_mode: docker-local\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "features", "alpha", "feature.yaml"),
		[]byte("name: alpha\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "features", "beta", "feature.yaml"),
		[]byte("name: beta\n"), 0o644))

	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}

	report, err := Run(Options{
		RepoRoot:      root,
		FeatureFilter: "alpha",
		ToolChecker:   checker,
	})
	require.NoError(t, err)

	// Only alpha feature should be checked
	alphaFound := false
	betaFound := false
	for _, r := range report.Results {
		if r.Category == featureCategoryPrefix("alpha") {
			alphaFound = true
		}
		if r.Category == featureCategoryPrefix("beta") {
			betaFound = true
		}
	}
	assert.True(t, alphaFound, "alpha feature should be checked")
	assert.False(t, betaFound, "beta feature should not be checked")
}

func TestRun_FeatureFilterNotFound(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "work"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"),
		[]byte("project_name: test\ndefault_editor: cursor\ndefault_container_mode: docker-local\n"), 0o644))

	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}

	_, err := Run(Options{
		RepoRoot:      root,
		FeatureFilter: "nonexistent",
		ToolChecker:   checker,
	})
	assert.Error(t, err, "should error when specified feature does not exist")
}

func TestRun_WithFix(t *testing.T) {
	root := t.TempDir()

	// Create minimal structure without .devrc.yaml
	require.NoError(t, os.MkdirAll(filepath.Join(root, "features", "myfeature", ".devcontainer"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0o755))
	// No work/ dir, no .devrc.yaml
	require.NoError(t, os.WriteFile(filepath.Join(root, "features", "myfeature", ".devcontainer", "devcontainer.json"),
		[]byte(`{"name":"myfeature"}`), 0o644))

	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}

	report, err := Run(Options{
		RepoRoot:    root,
		ToolChecker: checker,
		Fix:         true,
	})
	require.NoError(t, err)
	assert.False(t, report.HasFailures(), "expected no failures after fix")

	// Check that .devrc.yaml was created
	_, err = os.Stat(filepath.Join(root, ".devrc.yaml"))
	assert.NoError(t, err, ".devrc.yaml should have been created")

	// Check that work/ was created
	_, err = os.Stat(filepath.Join(root, "work"))
	assert.NoError(t, err, "work/ directory should have been created")

	// Check that Fixed items exist in report
	fixedCount := 0
	for _, r := range report.Results {
		if r.Fixed {
			fixedCount++
		}
	}
	assert.GreaterOrEqual(t, fixedCount, 2, "at least 2 items should be fixed (.devrc.yaml and work/)")
}

func TestFixGlobalConfig(t *testing.T) {
	root := t.TempDir()

	err := fixGlobalConfig(root)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(root, ".devrc.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "default_editor: cursor")
	assert.Contains(t, content, "default_container_mode: docker-local")
}

func TestFixDirectory(t *testing.T) {
	root := t.TempDir()

	err := fixDirectory(root, "work")
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(root, "work"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// mockToolChecker is defined in checks_test.go, reuse it here via same package.
// This test file is in the same package so it has access to unexported symbols.
// The mock type is already defined in checks_test.go so we use a build constraint-free approach:
// We call the mock directly since both files are in package doctor and compiled together.
var _ = fmt.Sprintf // avoid unused import
