package cmd

import (
	"fmt"
	"os"

	"github.com/escape-dev/devctl/internal/action"
	"github.com/escape-dev/devctl/internal/cmdexec"
	"github.com/escape-dev/devctl/internal/detect"
	"github.com/escape-dev/devctl/internal/log"
	"github.com/escape-dev/devctl/internal/matrix"
	"github.com/escape-dev/devctl/internal/report"
	"github.com/escape-dev/devctl/internal/resolve"
)

// AppContext holds shared state for all subcommands.
type AppContext struct {
	Logger       *log.Logger
	CmdRunner    *cmdexec.Runner
	Recorder     *cmdexec.Recorder
	Report       *report.Report
	ActionRunner *action.Runner
	RepoRoot     string
	Feature      string
	Branch       string
	DryRun       bool
	Verbose      bool
	ReportFile   string
}

// ParseFeatureBranch extracts feature and optional branch from args.
// If branch is omitted, defaults to feature name.
func ParseFeatureBranch(args []string) (feature, branch string) {
	feature = args[0]
	if len(args) >= 2 {
		branch = args[1]
	} else {
		branch = feature
	}
	return
}

// InitContext builds AppContext from global flags and args.
func InitContext(args []string) (*AppContext, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("feature name is required")
	}

	feature, branch := ParseFeatureBranch(args)

	logger := log.New(os.Stderr, flagVerbose)
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: flagDryRun, Recorder: rec}

	repoRoot, err := os.Getwd()
	if err != nil {
		repoRoot = "."
	}

	ctx := &AppContext{
		Logger:     logger,
		CmdRunner:  runner,
		Recorder:   rec,
		RepoRoot:   repoRoot,
		Feature:    feature,
		Branch:     branch,
		DryRun:     flagDryRun,
		Verbose:    flagVerbose,
		ReportFile: flagReport,
		ActionRunner: &action.Runner{
			Logger:    logger,
			DryRun:    flagDryRun,
			CmdRunner: runner,
		},
	}

	ctx.Report = &report.Report{
		Feature: feature,
		Branch:  branch,
		EnvVars: CollectEnvVars(),
	}

	return ctx, nil
}

// ResolveEnvironment loads config, resolves editor, detects OS.
func (ctx *AppContext) ResolveEnvironment(editorFlag string) (detect.OS, detect.Editor, matrix.ContainerMode, error) {
	currentOS := detect.CurrentOS()
	ctx.Logger.Debug("OS=%s", currentOS)
	ctx.Report.OS = string(currentOS)

	globalCfg, err := resolve.LoadGlobalConfig(ctx.RepoRoot)
	if err != nil {
		ctx.Logger.Warn("Failed to load .devrc.yaml: %v", err)
	}
	featureCfg, err := resolve.LoadFeatureConfig(ctx.RepoRoot, ctx.Feature)
	if err != nil {
		ctx.Logger.Warn("Failed to load feature.yaml: %v", err)
	}

	ed, err := detect.ResolveEditor(
		editorFlag,
		os.Getenv(detect.EnvKeyEditor),
		featureCfg.Dev.EditorDefault,
		globalCfg.DefaultEditor,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("editor resolution failed: %w", err)
	}
	ctx.Logger.Debug("Editor=%s", ed)
	ctx.Report.Editor = string(ed)

	containerMode := matrix.ContainerMode(globalCfg.DefaultContainerMode)
	ctx.Report.ContainerMode = string(containerMode)

	return currentOS, ed, containerMode, nil
}

// envVarDef holds an env var key and its default value.
type envVarDef struct {
	key      string
	fallback string
}

// knownEnvVars lists all DEVCTL_* environment variables.
var knownEnvVars = []envVarDef{
	{"DEVCTL_EDITOR", "cursor"},
	{"DEVCTL_CMD_CODE", "code"},
	{"DEVCTL_CMD_CURSOR", "cursor"},
	{"DEVCTL_CMD_AG", "antigravity"},
	{"DEVCTL_CMD_CLAUDE", "claude"},
	{"DEVCTL_CMD_GIT", "git"},
	{"DEVCTL_CMD_GH", "gh"},
}

// CollectEnvVars gathers all DEVCTL_* env vars for the report.
func CollectEnvVars() []report.EnvVar {
	vars := make([]report.EnvVar, 0, len(knownEnvVars))
	for _, d := range knownEnvVars {
		val := os.Getenv(d.key)
		vars = append(vars, report.EnvVar{
			Name:    d.key,
			Value:   val,
			Default: d.fallback,
			WasSet:  val != "",
		})
	}
	return vars
}
