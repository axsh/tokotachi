package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const categoryTools = "External Tools"
const categoryRepo = "Repository Structure"
const categoryConfig = "Global Config (.devrc.yaml)"

// ToolChecker abstracts external command execution for testability.
type ToolChecker interface {
	// CheckTool runs "<tool> --version" and returns the first line or error.
	CheckTool(name string) (version string, err error)
}

// DefaultToolChecker runs actual commands via exec.Command.
type DefaultToolChecker struct{}

// CheckTool executes "<name> --version" and returns the first output line.
func (d *DefaultToolChecker) CheckTool(name string) (string, error) {
	out, err := exec.Command(name, "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return line, nil
}

// toolDef defines an external tool to check.
type toolDef struct {
	name     string
	required bool // false = warn-only when missing
	fixHint  string
}

// tools lists the external tools to check.
var tools = []toolDef{
	{name: "git", required: true, fixHint: "Install Git: https://git-scm.com/"},
	{name: "docker", required: true, fixHint: "Install Docker: https://docs.docker.com/get-docker/"},
	{name: "gh", required: false, fixHint: "Install GitHub CLI: https://cli.github.com/ (only required for 'tt pr')"},
}

// checkExternalTools checks git, docker, gh availability.
func checkExternalTools(checker ToolChecker) []CheckResult {
	var results []CheckResult
	for _, tool := range tools {
		version, err := checker.CheckTool(tool.name)
		if err != nil {
			status := StatusFail
			if !tool.required {
				status = StatusWarn
			}
			results = append(results, CheckResult{
				Category: categoryTools,
				Name:     tool.name,
				Status:   status,
				Message:  "not found",
				FixHint:  tool.fixHint,
			})
		} else {
			results = append(results, CheckResult{
				Category: categoryTools,
				Name:     tool.name,
				Status:   StatusPass,
				Message:  version,
			})
		}
	}
	return results
}

// dirCheck defines a directory to verify.
type dirCheck struct {
	name     string
	required bool // false = warn-only when missing
	note     string
}

// requiredDirs lists directories to check under repo root.
var requiredDirs = []dirCheck{
	{name: "features", required: true},
	{name: "work", required: false, note: "created on first 'tt up'"},
	{name: "scripts", required: false, note: "contains build and test automation"},
}

// checkRepoStructure checks required directories exist.
func checkRepoStructure(repoRoot string) []CheckResult {
	var results []CheckResult

	for _, d := range requiredDirs {
		dirPath := filepath.Join(repoRoot, d.name)
		displayName := d.name + "/"
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			status := StatusFail
			msg := "directory not found"
			if !d.required {
				status = StatusWarn
				if d.note != "" {
					msg = fmt.Sprintf("not found (%s)", d.note)
				}
			}
			results = append(results, CheckResult{
				Category: categoryRepo,
				Name:     displayName,
				Status:   status,
				Message:  msg,
				Expected: "directory exists",
				FixHint:  fmt.Sprintf("Create '%s' directory", d.name),
			})
		} else {
			results = append(results, CheckResult{
				Category: categoryRepo,
				Name:     displayName,
				Status:   StatusPass,
				Message:  "directory exists",
			})
		}
	}
	return results
}

// validEditors lists accepted editor values.
var validEditors = []string{"code", "vscode", "cursor", "ag", "antigravity", "claude"}

// validContainerModes lists accepted container mode values.
var validContainerModes = []string{"none", "devcontainer", "docker-local", "docker-ssh"}

// globalConfig mirrors resolve.GlobalConfig for YAML parsing within doctor.
type globalConfig struct {
	ProjectName          string `yaml:"project_name"`
	DefaultEditor        string `yaml:"default_editor"`
	DefaultContainerMode string `yaml:"default_container_mode"`
}

// checkGlobalConfig checks .devrc.yaml existence and validity.
func checkGlobalConfig(repoRoot string) []CheckResult {
	var results []CheckResult
	path := filepath.Join(repoRoot, ".devrc.yaml")

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     ".devrc.yaml",
			Status:   StatusWarn,
			Message:  "file not found (defaults will be used)",
			FixHint:  "Create .devrc.yaml with: project_name, default_editor, default_container_mode",
		})
		return results
	}
	if err != nil {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     ".devrc.yaml",
			Status:   StatusFail,
			Message:  fmt.Sprintf("read error: %v", err),
		})
		return results
	}

	var cfg globalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     ".devrc.yaml",
			Status:   StatusFail,
			Message:  fmt.Sprintf("YAML parse error: %v", err),
			Expected: "valid YAML",
			FixHint:  "Fix YAML syntax in .devrc.yaml",
		})
		return results
	}

	results = append(results, CheckResult{
		Category: categoryConfig,
		Name:     ".devrc.yaml",
		Status:   StatusPass,
		Message:  "file exists and is valid YAML",
	})

	// project_name
	if cfg.ProjectName == "" {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "project_name",
			Status:   StatusWarn,
			Message:  "empty (default 'tt' will be used)",
			FixHint:  "Set project_name in .devrc.yaml",
		})
	} else {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "project_name",
			Status:   StatusPass,
			Message:  fmt.Sprintf("%q", cfg.ProjectName),
		})
	}

	// default_editor
	if cfg.DefaultEditor == "" {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_editor",
			Status:   StatusPass,
			Message:  "empty (default 'cursor' will be used)",
		})
	} else if !slices.Contains(validEditors, cfg.DefaultEditor) {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_editor",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("unknown value %q", cfg.DefaultEditor),
			Expected: fmt.Sprintf("one of: %s", strings.Join(validEditors, ", ")),
			FixHint:  "Set default_editor to a supported value",
		})
	} else {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_editor",
			Status:   StatusPass,
			Message:  fmt.Sprintf("%q", cfg.DefaultEditor),
		})
	}

	// default_container_mode
	if cfg.DefaultContainerMode == "" {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_container_mode",
			Status:   StatusPass,
			Message:  "empty (default 'docker-local' will be used)",
		})
	} else if !slices.Contains(validContainerModes, cfg.DefaultContainerMode) {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_container_mode",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("unknown value %q", cfg.DefaultContainerMode),
			Expected: fmt.Sprintf("one of: %s", strings.Join(validContainerModes, ", ")),
			FixHint:  "Set default_container_mode to a supported value",
		})
	} else {
		results = append(results, CheckResult{
			Category: categoryConfig,
			Name:     "default_container_mode",
			Status:   StatusPass,
			Message:  fmt.Sprintf("%q", cfg.DefaultContainerMode),
		})
	}

	return results
}

// featureCategoryPrefix returns the category name for a feature.
func featureCategoryPrefix(name string) string {
	return fmt.Sprintf("Feature: %s", name)
}

// checkFeature checks a single feature directory.
func checkFeature(repoRoot, featureName string) []CheckResult {
	category := featureCategoryPrefix(featureName)
	featureDir := filepath.Join(repoRoot, "features", featureName)
	var results []CheckResult

	// .devcontainer/devcontainer.json
	dcPath := filepath.Join(featureDir, ".devcontainer", "devcontainer.json")
	dcData, err := os.ReadFile(dcPath)
	if errors.Is(err, os.ErrNotExist) {
		results = append(results, CheckResult{
			Category: category,
			Name:     "devcontainer.json",
			Status:   StatusWarn,
			Message:  "not found (optional)",
			FixHint:  fmt.Sprintf("Create features/%s/.devcontainer/devcontainer.json for container support", featureName),
		})
	} else if err != nil {
		results = append(results, CheckResult{
			Category: category,
			Name:     "devcontainer.json",
			Status:   StatusFail,
			Message:  fmt.Sprintf("read error: %v", err),
		})
	} else {
		var parsed map[string]any
		if err := json.Unmarshal(dcData, &parsed); err != nil {
			results = append(results, CheckResult{
				Category: category,
				Name:     "devcontainer.json",
				Status:   StatusFail,
				Message:  fmt.Sprintf("JSON parse error: %v", err),
				Expected: "valid JSON",
				FixHint:  "Fix JSON syntax in devcontainer.json",
			})
		} else {
			results = append(results, CheckResult{
				Category: category,
				Name:     "devcontainer.json",
				Status:   StatusPass,
				Message:  "exists and valid",
			})
		}
	}

	return results
}

// fixGlobalConfig creates .devrc.yaml with default settings.
func fixGlobalConfig(repoRoot string) error {
	path := filepath.Join(repoRoot, ".devrc.yaml")
	content := "project_name: \"\"\ndefault_editor: cursor\ndefault_container_mode: docker-local\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// fixDirectory creates a directory if it doesn't exist.
func fixDirectory(repoRoot, dirName string) error {
	return os.MkdirAll(filepath.Join(repoRoot, dirName), 0o755)
}

// discoverFeatures lists all subdirectories under features/.
func discoverFeatures(repoRoot string) ([]string, error) {
	featuresDir := filepath.Join(repoRoot, "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read features directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
