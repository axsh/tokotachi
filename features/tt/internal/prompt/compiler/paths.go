package compiler

import (
	"fmt"
	"os"
	"path/filepath"
)

// Path expression reference (help / manual):
//
//	| prompts source root     | {workspace} + {prompts-dir}                              |
//	| project.yaml (default)  | {workspace} + {prompts-dir} + manifest/project.yaml    |
//	| schemas directory       | {workspace} + {prompts-dir} + manifest/schemas           |
//	| update git watch (man.) | {workspace} + {prompts-dir} + manifest/                |
//	| update git watch (mem.) | {workspace} + {prompts-dir} + memory/                  |
//	| build output root       | {workspace} + {build-dir}                              |
//	| resolved manifest (fb)  | {workspace} + {build-dir} + manifest.resolved.yaml       |
//	| digest file             | {workspace} + {build-dir} + .compile-digest[-{target}] |
//	| deploy target (apply)   | {deploy-root} + .cursor/rules/ (etc.)                    |

// PathOptions holds raw path inputs from CLI and environment.
type PathOptions struct {
	CWD string

	WorkspaceFlag        string
	WorkspaceFlagSet     bool
	PromptsDirFlag       string
	PromptsDirFlagSet    bool
	ProjectFlag          string
	ProjectFlagSet       bool
	BuildDirFlag         string
	BuildDirFlagSet      bool
	ResolvedManifestFlag string
	ResolvedManifestFlagSet bool
	DeployRootFlag       string
	DeployRootFlagSet    bool
}

// PathConfig holds fully resolved paths used by compile/deploy/update.
type PathConfig struct {
	Workspace        string // absolute workspace root
	ProjectYAML      string // absolute path to project.yaml
	PromptsDir       string // relative to workspace (e.g. "prompts")
	BuildDir         string // relative to workspace (e.g. "tmp/dist/")
	ResolvedManifest string // relative to workspace
	DeployRoot       string // absolute deploy root for apply mode
	SchemasDir       string // absolute: workspace + prompts-dir + manifest/schemas
}

// GitWatchPaths returns workspace-relative paths for prompt update git monitoring.
func (pc PathConfig) GitWatchPaths() []string {
	return []string{
		filepath.ToSlash(filepath.Join(pc.PromptsDir, "manifest")),
		filepath.ToSlash(filepath.Join(pc.PromptsDir, "memory")),
	}
}

// BuildDirAbs returns the absolute build output directory.
func (pc PathConfig) BuildDirAbs() string {
	return filepath.Clean(filepath.Join(pc.Workspace, pc.BuildDir))
}

// ResolvedManifestAbs returns the absolute resolved manifest output path.
func (pc PathConfig) ResolvedManifestAbs() string {
	return filepath.Clean(filepath.Join(pc.Workspace, pc.ResolvedManifest))
}

// PromptsDirAbs returns the absolute prompts source root.
func (pc PathConfig) PromptsDirAbs() string {
	return filepath.Clean(filepath.Join(pc.Workspace, pc.PromptsDir))
}

// ResolvePaths merges CLI, env, project.yaml, and built-in defaults.
// Priority: CLI flag (Changed) > env > project.yaml > built-in default.
func ResolvePaths(opts PathOptions) (*PathConfig, error) {
	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	projectRel := mergePathValue(
		opts.ProjectFlag, opts.ProjectFlagSet,
		os.Getenv("TT_PROJECT"),
		"prompts/manifest/project.yaml",
	)

	workspaceRaw := mergePathValue(
		opts.WorkspaceFlag, opts.WorkspaceFlagSet,
		os.Getenv("TT_WORKSPACE"),
		"",
	)

	promptsDir := mergePathValue(
		opts.PromptsDirFlag, opts.PromptsDirFlagSet,
		os.Getenv("TT_PROMPTS_DIR"),
		"prompts",
	)

	if !opts.ProjectFlagSet && os.Getenv("TT_PROJECT") == "" {
		projectRel = filepath.ToSlash(filepath.Join(promptsDir, "manifest", "project.yaml"))
	}

	var workspace string
	var projectYAML string

	if workspaceRaw != "" {
		workspace = resolvePath(cwd, workspaceRaw)
		candidate := resolveProjectCandidate(cwd, workspace, projectRel)
		var err error
		projectYAML, err = filepath.Abs(candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project.yaml path: %w", err)
		}
	} else {
		candidate := resolveProjectCandidate(cwd, "", projectRel)
		var err error
		projectYAML, err = filepath.Abs(candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project.yaml path: %w", err)
		}
		inferred, err := ResolveProjectRoot(projectYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project root: %w", err)
		}
		workspace = inferred
	}

	cfg, err := LoadConfig(projectYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	yamlBuild := cfg.Defaults.BuildDir
	if yamlBuild == "" {
		yamlBuild = "tmp/dist/"
	}

	buildDir := mergePathValue(
		opts.BuildDirFlag, opts.BuildDirFlagSet,
		os.Getenv("TT_BUILD_DIR"),
		yamlBuild,
	)

	yamlResolved := cfg.Outputs.ResolvedManifest
	resolved := mergePathValue(
		opts.ResolvedManifestFlag, opts.ResolvedManifestFlagSet,
		os.Getenv("TT_RESOLVED_MANIFEST"),
		yamlResolved,
	)
	if resolved == "" {
		resolved = filepath.ToSlash(filepath.Join(buildDir, "manifest.resolved.yaml"))
	}

	deployRaw := mergePathValue(
		opts.DeployRootFlag, opts.DeployRootFlagSet,
		os.Getenv("TT_DEPLOY_ROOT"),
		"",
	)
	deployRoot := workspace
	if deployRaw != "" {
		deployRoot = resolvePath(cwd, deployRaw)
	}

	schemasDir := filepath.Clean(filepath.Join(workspace, promptsDir, "manifest", "schemas"))

	return &PathConfig{
		Workspace:        workspace,
		ProjectYAML:      projectYAML,
		PromptsDir:       filepath.ToSlash(promptsDir),
		BuildDir:         filepath.ToSlash(buildDir),
		ResolvedManifest: filepath.ToSlash(resolved),
		DeployRoot:       deployRoot,
		SchemasDir:       schemasDir,
	}, nil
}

func mergePathValue(flag string, flagSet bool, env, fallback string) string {
	if flagSet {
		return flag
	}
	if env != "" {
		return env
	}
	return fallback
}

func resolvePath(anchor, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(anchor, p))
}

func resolveProjectCandidate(cwd, workspace, rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	if workspace != "" {
		return filepath.Clean(filepath.Join(workspace, rel))
	}
	return filepath.Clean(filepath.Join(cwd, rel))
}

func resolvePathsFromOptions(opts CompileOptions) (*PathConfig, error) {
	if opts.Paths != nil {
		return opts.Paths, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	return ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    opts.ProjectPath,
		ProjectFlagSet: true,
	})
}

func resolvePathsFromDeployOptions(opts DeployOptions) (*PathConfig, error) {
	if opts.Paths != nil {
		return opts.Paths, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	return ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    opts.ProjectPath,
		ProjectFlagSet: true,
	})
}

func resolvePathsFromUpdateOptions(opts UpdateOptions) (*PathConfig, error) {
	if opts.Paths != nil {
		return opts.Paths, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	return ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    opts.ProjectPath,
		ProjectFlagSet: true,
	})
}
