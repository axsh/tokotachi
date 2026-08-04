package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/compiler"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
	"github.com/axsh/tokotachi/pkg/resolve"
)

const (
	helpWorkspace = "Workspace root for path resolution. Relative to CWD unless absolute. Default: inferred from --project (see path expr: {workspace}). Env: TT_WORKSPACE (flag wins)."
	helpPromptsDir = "Prompts source root (manifest and memory). Relative to workspace unless absolute. Default: {workspace} + {prompts-dir} (segment: prompts). Env: TT_PROMPTS_DIR (flag wins)."
	helpProject = "Path to project.yaml. Relative to workspace if set, else CWD unless absolute. Default: {workspace} + {prompts-dir} + manifest/project.yaml. Env: (none)."
	helpBuildDir = "Build output directory for staging and digests. Relative to workspace unless absolute. Default: {workspace} + {build-dir} (from project.yaml defaults.build_dir or tmp/dist/). Env: TT_BUILD_DIR (flag wins)."
	helpResolvedManifest = "Resolved manifest output path. Relative to workspace unless absolute. Default: {workspace} + {resolved-manifest} (from project.yaml outputs.resolved_manifest or {build-dir} + manifest.resolved.yaml). Env: TT_RESOLVED_MANIFEST (flag wins)."
	helpDeployRoot = "Root directory for editor config deployment in apply mode. Relative to CWD unless absolute. Default: {deploy-root} = {workspace}. Env: TT_DEPLOY_ROOT (flag wins)."

	promptPathLongSuffix = `

Path flags resolve in order: --workspace, --prompts-dir, --project, --build-dir,
--resolved-manifest, --deploy-root. Paths compose as {workspace} + {prompts-dir} + ...
See --help for defaults and TT_* env vars.`
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage prompt manifest compilation and deployment",
}

var promptCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile prompt manifest and memory documents",
	Long:  "Compile prompt manifest and memory documents." + promptPathLongSuffix,
	RunE:  runPromptCompile,
}

var promptDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Compile and deploy prompt files to target directories",
	Long:  "Compile and deploy prompt files to target directories." + promptPathLongSuffix,
	RunE:  runPromptDeploy,
}

var promptUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for changes and update prompt files if needed",
	Long:  "Check for changes and update prompt files if needed." + promptPathLongSuffix,
	RunE:  runPromptUpdate,
}

var (
	compileProject string
	compileTarget  string
	compileDryRun  bool
	compileApply   bool

	deployProject string
	deployTarget  string
	deployForce   bool
	deployDryRun  bool
	deployMode    string

	updateProject string
	updateTarget  string
	updateForce   bool
	updateDryRun  bool

	promptWorkspace         string
	promptPromptsDir        string
	promptBuildDir          string
	promptResolvedManifest  string
	promptDeployRoot        string
	promptTags              string
	promptTagRefs           string
)

func init() {
	addPromptPathFlags(promptCompileCmd)
	addPromptTagFlags(promptCompileCmd)
	promptCompileCmd.Flags().StringVar(&compileProject, "project",
		"prompts/manifest/project.yaml", helpProject)
	promptCompileCmd.Flags().StringVar(&compileTarget, "target",
		"", "Emitter target (default from TT_TARGET or 'all')")
	promptCompileCmd.Flags().BoolVar(&compileDryRun, "dry-run",
		false, "Do not write files, print to stdout")
	promptCompileCmd.Flags().BoolVar(&compileApply, "apply",
		false, "Apply generated files to target directories")

	addPromptPathFlags(promptDeployCmd)
	addPromptTagFlags(promptDeployCmd)
	promptDeployCmd.Flags().StringVar(&deployProject, "project",
		"prompts/manifest/project.yaml", helpProject)
	promptDeployCmd.Flags().StringVar(&deployTarget, "target",
		"", "Emitter target (default from TT_TARGET or 'all')")
	promptDeployCmd.Flags().BoolVar(&deployForce, "force",
		false, "Force deploy even if no changes detected")
	promptDeployCmd.Flags().BoolVar(&deployDryRun, "dry-run",
		false, "Do not write files, print to stdout")
	promptDeployCmd.Flags().StringVar(&deployMode, "mode",
		"", "Emit mode: overwrite (default), skip, immune")

	addPromptPathFlags(promptUpdateCmd)
	addPromptTagFlags(promptUpdateCmd)
	promptUpdateCmd.Flags().StringVar(&updateProject, "project",
		"prompts/manifest/project.yaml", helpProject)
	promptUpdateCmd.Flags().StringVar(&updateTarget, "target",
		"", "Emitter target (default from TT_TARGET or 'all')")
	promptUpdateCmd.Flags().BoolVar(&updateForce, "force",
		false, "Force update even if no changes detected")
	promptUpdateCmd.Flags().BoolVar(&updateDryRun, "dry-run",
		false, "Do not write files, print to stdout")

	promptCmd.AddCommand(promptCompileCmd)
	promptCmd.AddCommand(promptDeployCmd)
	promptCmd.AddCommand(promptUpdateCmd)
}

func addPromptPathFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&promptWorkspace, "workspace", "", helpWorkspace)
	cmd.Flags().StringVar(&promptPromptsDir, "prompts-dir", "", helpPromptsDir)
	cmd.Flags().StringVar(&promptBuildDir, "build-dir", "", helpBuildDir)
	cmd.Flags().StringVar(&promptResolvedManifest, "resolved-manifest", "", helpResolvedManifest)
	cmd.Flags().StringVar(&promptDeployRoot, "deploy-root", "", helpDeployRoot)
}

func addPromptTagFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&promptTags, "tags", "",
		"Comma-separated selection tags (default: TT_TAGS or baseline)")
	cmd.Flags().StringVar(&promptTagRefs, "tag-refs", "",
		"Reference mode: include (default) or strict (default: TT_TAG_REFS or include)")
}

func buildPathOptions(cmd *cobra.Command, projectFlag string) (compiler.PathOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return compiler.PathOptions{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	f := cmd.Flags()
	return compiler.PathOptions{
		CWD:                     cwd,
		WorkspaceFlag:           promptWorkspace,
		WorkspaceFlagSet:        f.Changed("workspace"),
		PromptsDirFlag:          promptPromptsDir,
		PromptsDirFlagSet:       f.Changed("prompts-dir"),
		ProjectFlag:             projectFlag,
		ProjectFlagSet:          f.Changed("project"),
		BuildDirFlag:            promptBuildDir,
		BuildDirFlagSet:         f.Changed("build-dir"),
		ResolvedManifestFlag:    promptResolvedManifest,
		ResolvedManifestFlagSet: f.Changed("resolved-manifest"),
		DeployRootFlag:          promptDeployRoot,
		DeployRootFlagSet:       f.Changed("deploy-root"),
	}, nil
}

func resolveTargetFlag(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv(resolve.EnvKeyTarget); env != "" {
		return env
	}
	return "all"
}

// resolveTagsFlag resolves --tags / TT_TAGS / baseline.
// Prefer CLI when the flag was changed; otherwise fall back to TT_TAGS, then baseline.
func resolveTagsFlag(cmd *cobra.Command) (tags []string, warnings []string, err error) {
	raw := promptTags
	if !cmd.Flags().Changed("tags") {
		raw = os.Getenv(manifest.EnvKeyTags)
	}
	if strings.TrimSpace(raw) == "" {
		return []string{manifest.BaselineTag}, nil, nil
	}
	return manifest.NormalizeRequestedTags(raw)
}

// resolveTagRefsFlag resolves --tag-refs / TT_TAG_REFS / include.
func resolveTagRefsFlag(cmd *cobra.Command) (string, error) {
	raw := promptTagRefs
	if !cmd.Flags().Changed("tag-refs") {
		raw = os.Getenv(manifest.EnvKeyTagRefs)
	}
	return manifest.NormalizeTagRefsMode(raw)
}

func printTagWarnings(cmd *cobra.Command, warnings []string) {
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: %s\n", w)
	}
}

func runPromptCompile(cmd *cobra.Command, args []string) error {
	target := resolveTargetFlag(compileTarget)

	resolvedTarget, err := resolve.ResolveTarget(target, true)
	if err != nil {
		return err
	}

	targets, err := resolve.ResolveTargets(resolvedTarget)
	if err != nil {
		return err
	}

	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tags: %w", err)
	}
	printTagWarnings(cmd, warnings)
	tagRefs, err := resolveTagRefsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tag-refs: %w", err)
	}

	pathOpts, err := buildPathOptions(cmd, compileProject)
	if err != nil {
		return err
	}
	paths, err := compiler.ResolvePaths(pathOpts)
	if err != nil {
		return err
	}

	for _, t := range targets {
		result, err := compiler.Compile(compiler.CompileOptions{
			Paths:   paths,
			DryRun:  compileDryRun,
			Target:  t,
			Apply:   compileApply,
			Tags:    tags,
			TagRefs: tagRefs,
		})
		if err != nil {
			return fmt.Errorf("compile failed for target %s: %w", t, err)
		}
		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				fmt.Fprintln(os.Stderr, e.Error())
			}
			return fmt.Errorf("compile failed with %d validation error(s)",
				len(result.Errors))
		}
		if compileDryRun {
			fmt.Printf("=== %s: resolved manifest ===\n", t)
			fmt.Println(result.ResolvedYAML)
		} else {
			fmt.Printf("Compile succeeded for target %s.\n", t)
		}
	}
	return nil
}

func runPromptDeploy(cmd *cobra.Command, args []string) error {
	target := resolveTargetFlag(deployTarget)

	resolvedTarget, err := resolve.ResolveTarget(target, true)
	if err != nil {
		return err
	}

	targets, err := resolve.ResolveTargets(resolvedTarget)
	if err != nil {
		return err
	}

	mode := emitter.EmitMode(deployMode)
	if mode != "" && !emitter.ValidEmitModes(mode) {
		return fmt.Errorf("invalid mode %q: must be overwrite, skip, or immune", deployMode)
	}

	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tags: %w", err)
	}
	printTagWarnings(cmd, warnings)
	tagRefs, err := resolveTagRefsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tag-refs: %w", err)
	}

	pathOpts, err := buildPathOptions(cmd, deployProject)
	if err != nil {
		return err
	}
	paths, err := compiler.ResolvePaths(pathOpts)
	if err != nil {
		return err
	}

	type deployEntry struct {
		target string
		result *compiler.DeployResult
	}
	var entries []deployEntry

	for _, t := range targets {
		result, err := compiler.Deploy(compiler.DeployOptions{
			Paths:   paths,
			Target:  t,
			Force:   deployForce,
			DryRun:  deployDryRun,
			Mode:    mode,
			Tags:    tags,
			TagRefs: tagRefs,
		})
		if err != nil {
			return fmt.Errorf("deploy failed for target %s: %w", t, err)
		}
		entries = append(entries, deployEntry{target: t, result: result})
		if result.Skipped {
			fmt.Printf("No changes detected for target %s. Skipping deploy.\n", t)
		} else if deployDryRun {
			fmt.Printf("Deploy dry-run completed for target %s.\n", t)
		} else {
			fmt.Printf("Deploy succeeded for target %s.\n", t)
		}
	}

	if mode == emitter.EmitModeImmune && !deployDryRun {
		mergedEmitted := make(map[string]bool)
		var allTargetDirs []string
		for _, e := range entries {
			if e.result.EmitResult != nil {
				for k, v := range e.result.EmitResult.EmittedFiles {
					mergedEmitted[k] = v
				}
				allTargetDirs = append(allTargetDirs, e.result.EmitResult.TargetDirs...)
			}
		}
		uniqueDirs := deduplicateDirs(allTargetDirs)
		if len(uniqueDirs) > 0 {
			if _, err := emitter.CleanOrphanFiles(uniqueDirs, mergedEmitted, false); err != nil {
				return fmt.Errorf("immune orphan cleanup failed: %w", err)
			}
		}
	}

	return nil
}

func deduplicateDirs(dirs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, d := range dirs {
		clean := filepath.Clean(d)
		if !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}
	return result
}

func runPromptUpdate(cmd *cobra.Command, args []string) error {
	target := resolveTargetFlag(updateTarget)

	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tags: %w", err)
	}
	printTagWarnings(cmd, warnings)
	tagRefs, err := resolveTagRefsFlag(cmd)
	if err != nil {
		return fmt.Errorf("invalid --tag-refs: %w", err)
	}

	pathOpts, err := buildPathOptions(cmd, updateProject)
	if err != nil {
		return err
	}
	paths, err := compiler.ResolvePaths(pathOpts)
	if err != nil {
		return err
	}

	result, err := compiler.Update(compiler.UpdateOptions{
		Paths:   paths,
		Target:  target,
		Force:   updateForce,
		DryRun:  updateDryRun,
		Tags:    tags,
		TagRefs: tagRefs,
	})
	if err != nil {
		return err
	}

	for t, tr := range result.TargetResults {
		if tr.Skipped {
			fmt.Printf("No changes detected for target %s. Skipping update.\n", t)
		} else if updateDryRun {
			fmt.Printf("Update dry-run completed for target %s.\n", t)
		} else {
			fmt.Printf("Update succeeded for target %s.\n", t)
		}
	}
	return nil
}

// Exported for tests.
func PromptPathHelpConstants() []string {
	return []string{
		helpWorkspace,
		helpPromptsDir,
		helpProject,
		helpBuildDir,
		helpResolvedManifest,
		helpDeployRoot,
	}
}

func PromptPathHelpContainsPathExpr(s string) bool {
	return strings.Contains(s, "{workspace} + {prompts-dir}")
}
