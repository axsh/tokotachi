package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/compiler"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
	"github.com/axsh/tokotachi/pkg/resolve"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage prompt manifest compilation and deployment",
}

// --- compile ---

var promptCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile prompt manifest and memory documents",
	RunE:  runPromptCompile,
}

var (
	compileProject string
	compileTarget  string
	compileDryRun  bool
	compileApply   bool
)

// --- deploy ---

var promptDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Compile and deploy prompt files to target directories",
	RunE:  runPromptDeploy,
}

var (
	deployProject string
	deployTarget  string
	deployForce   bool
	deployDryRun  bool
	deployMode    string
)

// --- update ---

var promptUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for changes and update prompt files if needed",
	RunE:  runPromptUpdate,
}

var (
	updateProject string
	updateTarget  string
	updateForce   bool
	updateDryRun  bool
)

func init() {
	// compile flags
	promptCompileCmd.Flags().StringVar(&compileProject, "project",
		"prompts/manifest/project.yaml", "Path to project.yaml")
	promptCompileCmd.Flags().StringVar(&compileTarget, "target",
		"", "Emitter target (default from TT_TARGET or 'all')")
	promptCompileCmd.Flags().BoolVar(&compileDryRun, "dry-run",
		false, "Do not write files, print to stdout")
	promptCompileCmd.Flags().BoolVar(&compileApply, "apply",
		false, "Apply generated files to target directories")

	// deploy flags
	promptDeployCmd.Flags().StringVar(&deployProject, "project",
		"prompts/manifest/project.yaml", "Path to project.yaml")
	promptDeployCmd.Flags().StringVar(&deployTarget, "target",
		"", "Emitter target (default from TT_TARGET or 'all')")
	promptDeployCmd.Flags().BoolVar(&deployForce, "force",
		false, "Force deploy even if no changes detected")
	promptDeployCmd.Flags().BoolVar(&deployDryRun, "dry-run",
		false, "Do not write files, print to stdout")
	promptDeployCmd.Flags().StringVar(&deployMode, "mode",
		"", "Emit mode: overwrite (default), skip, immune")

	// update flags
	promptUpdateCmd.Flags().StringVar(&updateProject, "project",
		"prompts/manifest/project.yaml", "Path to project.yaml")
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

// resolveTargetFlag resolves the target from flag -> env -> default("all").
func resolveTargetFlag(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv(resolve.EnvKeyTarget); env != "" {
		return env
	}
	return "all"
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

	for _, t := range targets {
		result, err := compiler.Compile(compiler.CompileOptions{
			ProjectPath: compileProject,
			DryRun:      compileDryRun,
			Target:      t,
			Apply:       compileApply,
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

	// Validate mode flag
	mode := emitter.EmitMode(deployMode)
	if mode != "" && !emitter.ValidEmitModes(mode) {
		return fmt.Errorf("invalid mode %q: must be overwrite, skip, or immune", deployMode)
	}

	// Collect deploy results for immune mode coordination
	type deployEntry struct {
		target string
		result *compiler.DeployResult
	}
	var entries []deployEntry

	for _, t := range targets {
		result, err := compiler.Deploy(compiler.DeployOptions{
			ProjectPath: deployProject,
			Target:      t,
			Force:       deployForce,
			DryRun:      deployDryRun,
			Mode:        mode,
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

	// Immune mode: coordinated orphan cleanup across all deployed targets
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

// deduplicateDirs removes duplicate directory paths from the list.
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

	result, err := compiler.Update(compiler.UpdateOptions{
		ProjectPath: updateProject,
		Target:      target,
		Force:       updateForce,
		DryRun:      updateDryRun,
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
