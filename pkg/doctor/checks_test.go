package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockToolChecker is a test double for ToolChecker.
type mockToolChecker struct {
	results map[string]toolResult
}

type toolResult struct {
	version string
	err     error
}

func (m *mockToolChecker) CheckTool(name string) (string, error) {
	r, ok := m.results[name]
	if !ok {
		return "", fmt.Errorf("%s: not found", name)
	}
	return r.version, r.err
}

func TestCheckExternalTools_AllPresent(t *testing.T) {
	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}
	results := checkExternalTools(checker)
	for _, r := range results {
		assert.Equal(t, StatusPass, r.Status, "tool %s should pass", r.Name)
	}
}

func TestCheckExternalTools_GitMissing(t *testing.T) {
	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {err: fmt.Errorf("not found")},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {version: "gh version 2.40.0"},
		},
	}
	results := checkExternalTools(checker)
	var gitResult CheckResult
	for _, r := range results {
		if r.Name == "git" {
			gitResult = r
		}
	}
	assert.Equal(t, StatusFail, gitResult.Status)
	assert.NotEmpty(t, gitResult.FixHint)
}

func TestCheckExternalTools_GhMissing(t *testing.T) {
	checker := &mockToolChecker{
		results: map[string]toolResult{
			"git":    {version: "git version 2.43.0"},
			"docker": {version: "Docker version 24.0.7"},
			"gh":     {err: fmt.Errorf("not found")},
		},
	}
	results := checkExternalTools(checker)
	var ghResult CheckResult
	for _, r := range results {
		if r.Name == "gh" {
			ghResult = r
		}
	}
	// gh is only a warning, not a failure
	assert.Equal(t, StatusWarn, ghResult.Status)
}

func TestCheckRepoStructure(t *testing.T) {
	tests := []struct {
		name     string
		dirs     []string
		wantFail []string
		wantWarn []string
	}{
		{
			name:     "all directories present",
			dirs:     []string{"features", "work", "scripts"},
			wantFail: nil,
			wantWarn: nil,
		},
		{
			name:     "features missing",
			dirs:     []string{"work", "scripts"},
			wantFail: []string{"features/"},
		},
		{
			name:     "work missing is warn",
			dirs:     []string{"features", "scripts"},
			wantWarn: []string{"work/"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, d := range tt.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(root, d), 0o755))
			}
			results := checkRepoStructure(root)
			for _, r := range results {
				if contains(tt.wantFail, r.Name) {
					assert.Equal(t, StatusFail, r.Status, "%s should fail", r.Name)
				} else if contains(tt.wantWarn, r.Name) {
					assert.Equal(t, StatusWarn, r.Status, "%s should warn", r.Name)
				}
			}
		})
	}
}

func TestCheckGlobalConfig(t *testing.T) {
	t.Run("file not found is warn", func(t *testing.T) {
		root := t.TempDir()
		results := checkGlobalConfig(root)
		// First result should be about file existence
		assert.Equal(t, StatusWarn, results[0].Status)
	})

	t.Run("invalid yaml is fail", func(t *testing.T) {
		root := t.TempDir()
		err := os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte("key: [unterminated\n\t: bad:\nindent"), 0o644)
		require.NoError(t, err)
		results := checkGlobalConfig(root)
		hasFail := false
		for _, r := range results {
			if r.Status == StatusFail {
				hasFail = true
			}
		}
		assert.True(t, hasFail, "invalid YAML should produce a fail result")
	})

	t.Run("valid yaml with all fields", func(t *testing.T) {
		root := t.TempDir()
		content := `project_name: myproject
default_editor: cursor
default_container_mode: docker-local
`
		err := os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0o644)
		require.NoError(t, err)
		results := checkGlobalConfig(root)
		for _, r := range results {
			assert.Equal(t, StatusPass, r.Status, "check %s should pass, got %v: %s", r.Name, r.Status, r.Message)
		}
	})

	t.Run("empty project_name is warn", func(t *testing.T) {
		root := t.TempDir()
		content := `default_editor: cursor
default_container_mode: docker-local
`
		err := os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0o644)
		require.NoError(t, err)
		results := checkGlobalConfig(root)
		var projectResult CheckResult
		for _, r := range results {
			if r.Name == "project_name" {
				projectResult = r
			}
		}
		assert.Equal(t, StatusWarn, projectResult.Status)
	})

	t.Run("invalid editor value is warn", func(t *testing.T) {
		root := t.TempDir()
		content := `project_name: myproject
default_editor: emacs
default_container_mode: docker-local
`
		err := os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0o644)
		require.NoError(t, err)
		results := checkGlobalConfig(root)
		var editorResult CheckResult
		for _, r := range results {
			if r.Name == "default_editor" {
				editorResult = r
			}
		}
		assert.Equal(t, StatusWarn, editorResult.Status)
	})

	t.Run("invalid container_mode is warn", func(t *testing.T) {
		root := t.TempDir()
		content := `project_name: myproject
default_editor: cursor
default_container_mode: kubernetes
`
		err := os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0o644)
		require.NoError(t, err)
		results := checkGlobalConfig(root)
		var modeResult CheckResult
		for _, r := range results {
			if r.Name == "default_container_mode" {
				modeResult = r
			}
		}
		assert.Equal(t, StatusWarn, modeResult.Status)
	})
}

func TestCheckFeature(t *testing.T) {
	t.Run("valid feature with all files", func(t *testing.T) {
		root := t.TempDir()
		featureDir := filepath.Join(root, "features", "myfeature")
		devcontainerDir := filepath.Join(featureDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0o755))

		dcJSON, _ := json.Marshal(map[string]string{"name": "myfeature"})
		require.NoError(t, os.WriteFile(
			filepath.Join(devcontainerDir, "devcontainer.json"),
			dcJSON, 0o644))

		results := checkFeature(root, "myfeature")
		for _, r := range results {
			assert.Equal(t, StatusPass, r.Status, "check %s should pass", r.Name)
		}
	})

	t.Run("devcontainer.json missing is warn", func(t *testing.T) {
		root := t.TempDir()
		featureDir := filepath.Join(root, "features", "nodc")
		require.NoError(t, os.MkdirAll(featureDir, 0o755))

		results := checkFeature(root, "nodc")
		var dcResult CheckResult
		for _, r := range results {
			if r.Name == "devcontainer.json" {
				dcResult = r
			}
		}
		assert.Equal(t, StatusWarn, dcResult.Status)
	})

	t.Run("invalid devcontainer.json is fail", func(t *testing.T) {
		root := t.TempDir()
		featureDir := filepath.Join(root, "features", "baddc")
		devcontainerDir := filepath.Join(featureDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0o755))

		require.NoError(t, os.WriteFile(
			filepath.Join(devcontainerDir, "devcontainer.json"),
			[]byte("{invalid json}"), 0o644))

		results := checkFeature(root, "baddc")
		var dcResult CheckResult
		for _, r := range results {
			if r.Name == "devcontainer.json" {
				dcResult = r
			}
		}
		assert.Equal(t, StatusFail, dcResult.Status)
	})

}

func TestDiscoverFeatures(t *testing.T) {
	root := t.TempDir()
	featuresDir := filepath.Join(root, "features")
	require.NoError(t, os.MkdirAll(filepath.Join(featuresDir, "alpha"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(featuresDir, "beta"), 0o755))
	// Create a file (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(featuresDir, "readme.md"), []byte("# Features"), 0o644))

	features, err := discoverFeatures(root)
	require.NoError(t, err)
	assert.Len(t, features, 2)
	assert.Contains(t, features, "alpha")
	assert.Contains(t, features, "beta")
}

// contains checks if a string slice contains a value.
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
