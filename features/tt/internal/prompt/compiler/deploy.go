package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
	"gopkg.in/yaml.v3"
)

// DeployOptions holds options for the deploy pipeline
type DeployOptions struct {
	ProjectPath string
	Target      string // default: "antigravity"
	Force       bool
	DryRun      bool
	Mode        emitter.EmitMode
}

// DeployResult holds the output of the deploy pipeline
type DeployResult struct {
	Skipped       bool     // true if digest matched (no changes)
	DigestCurrent string   // current computed digest
	DigestPrev    string   // previous stored digest
	CompileResult *CompileResult
	Warnings      []string // untracked file warnings
	EmitResult    *emitter.EmitResult // emitted files info for coordinated cleanup
}

// Deploy executes the full deploy pipeline:
// digest check -> compile -> emit -> apply -> save digest
func Deploy(opts DeployOptions) (*DeployResult, error) {
	result := &DeployResult{}

	// 1. Load config
	cfg, err := LoadConfig(opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Resolve project root
	rootDir, err := ResolveProjectRoot(opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}

	// 3. Resolve target early (needed for digest path)
	target := opts.Target
	if target == "" {
		target = "antigravity"
	}

	// 4. Compute current digest
	currentDigest, err := ComputeSourceDigest(cfg, rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to compute digest: %w", err)
	}
	result.DigestCurrent = currentDigest

	// 5. Resolve build dir
	buildDir := filepath.Clean(filepath.Join(rootDir, cfg.Defaults.BuildDir))

	// 6. Load previous digest
	prevInfo, err := LoadDigest(DigestPath(buildDir, target))
	if err != nil {
		return nil, fmt.Errorf("failed to load previous digest: %w", err)
	}
	result.DigestPrev = prevInfo.Digest

	// 7. Check if changes detected
	if !opts.Force && prevInfo.Digest == currentDigest && currentDigest != "" {
		if !CheckDrift(rootDir, opts.ProjectPath, target) {
			result.Skipped = true
			return result, nil
		}
	}

	// 8. Compile
	compileResult, err := Compile(CompileOptions{
		ProjectPath: opts.ProjectPath,
		DryRun:      opts.DryRun,
		Target:      target,
		Apply:       !opts.DryRun,
		EmitMode:    opts.Mode,
		EmitDryRun:  opts.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("compile failed: %w", err)
	}
	result.CompileResult = compileResult
	result.EmitResult = compileResult.EmitResult

	// 9. If validation errors, return without saving digest
	if len(compileResult.Errors) > 0 {
		return result, nil
	}

	// 10. Save digest (only when not dry-run and no errors)
	// Recompute digest after compile because compile may generate files
	// into source directories, changing the effective digest.
	if !opts.DryRun {
		postDigest, err := ComputeSourceDigest(cfg, rootDir)
		if err != nil {
			return nil, fmt.Errorf("failed to recompute digest after compile: %w", err)
		}
		result.DigestCurrent = postDigest

		newInfo := &DigestInfo{
			Digest: postDigest,
			Target: target,
		}
		if err := SaveDigest(DigestPath(buildDir, target), newInfo); err != nil {
			return nil, fmt.Errorf("failed to save digest: %w", err)
		}
	}

	return result, nil
}

// CheckDrift verifies if target files have drifted from the resolved manifest.
// Returns true if there is drift (or if check fails), false if target is fully consistent.
func CheckDrift(rootDir, projectPath, target string) bool {
	cfg, err := LoadConfig(projectPath)
	if err != nil {
		return true // assume drift if config can't be loaded
	}

	resolvedPath := filepath.Join(rootDir, cfg.Outputs.ResolvedManifest)
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return true // assume drift if resolved manifest is missing
	}

	var resolved manifest.ResolvedManifest
	if err := yaml.Unmarshal(data, &resolved); err != nil {
		return true // assume drift if resolved manifest is invalid
	}

	var emitObj emitter.Emitter
	switch target {
	case "antigravity":
		emitObj = emitter.NewAntigravityEmitter(rootDir)
	case "cursor":
		emitObj = emitter.NewCursorEmitter(rootDir)
	case "claude-code":
		emitObj = emitter.NewClaudeCodeEmitter(rootDir)
	case "codex":
		emitObj = emitter.NewCodexEmitter(rootDir)
	default:
		return true
	}

	buildDir := filepath.Clean(filepath.Join(rootDir, cfg.Defaults.BuildDir))
	ok, err := emitObj.Check(&resolved, buildDir)
	if err != nil || !ok {
		return true // drift detected or check failed
	}

	return false // no drift
}
