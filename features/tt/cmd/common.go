package cmd

import (
	"fmt"
	"os"

	"github.com/axsh/tokotachi/pkg/action"
	"github.com/axsh/tokotachi/internal/cmdexec"
	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/internal/log"
	"github.com/axsh/tokotachi/pkg/matrix"
	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/resolve"
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

// reservedBranchNames contains branch names that cannot be used with tt.
// These are typically default branches that must be protected from accidental
// modification or deletion.
var reservedBranchNames = []string{"main", "master"}

// validateBranchName checks if the given branch name is reserved.
// Returns an error if the branch is in the reservedBranchNames list.
// Comparison is case-sensitive (exact match only).
func validateBranchName(branch string) error {
	for _, name := range reservedBranchNames {
		if branch == name {
			return fmt.Errorf("%q is a reserved branch name and cannot be used with tt commands", branch)
		}
	}
	return nil
}

// ParseBranchFeature extracts branch and optional feature from args.
// If feature is omitted, it defaults to empty string (no container operations).
func ParseBranchFeature(args []string) (branch, feature string) {
	branch = args[0]
	if len(args) >= 2 {
		feature = args[1]
	}
	return
}

// HasFeature returns true if a feature was specified.
func (ctx *AppContext) HasFeature() bool {
	return ctx.Feature != ""
}

// InitContext builds AppContext from global flags and args.
func InitContext(args []string) (*AppContext, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("branch name is required")
	}

	branch, feature := ParseBranchFeature(args)

	if err := validateBranchName(branch); err != nil {
		return nil, err
	}

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
		Feature:     feature,
		Branch:      branch,
		EnvVars:     CollectEnvVars(),
		ShowEnvVars: flagEnv,
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

// knownEnvVars lists all TT_* environment variables.
var knownEnvVars = []envVarDef{
	{"TT_EDITOR", "cursor"},
	{"TT_CMD_CODE", "code"},
	{"TT_CMD_CURSOR", "cursor"},
	{"TT_CMD_AG", "antigravity"},
	{"TT_CMD_CLAUDE", "claude"},
	{"TT_CMD_GIT", "git"},
	{"TT_CMD_GH", "gh"},
	{"TT_LIST_WIDTH", "40"},
	{"TT_LIST_PADDING", "2"},
}

// CollectEnvVars gathers all TT_* env vars for the report.
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
