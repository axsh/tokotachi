package compiler

import (
	"fmt"
	"os"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
	"gopkg.in/yaml.v3"
)

// DeployOptions holds options for the deploy pipeline
type DeployOptions struct {
	ProjectPath string
	Paths       *PathConfig
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

	paths, err := resolvePathsFromDeployOptions(opts)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(paths.ProjectYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	rootDir := paths.Workspace

	target := opts.Target
	if target == "" {
		target = "antigravity"
	}

	currentDigest, err := ComputeSourceDigest(cfg, rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to compute digest: %w", err)
	}
	result.DigestCurrent = currentDigest

	buildDir := paths.BuildDirAbs()

	prevInfo, err := LoadDigest(DigestPath(buildDir, target))
	if err != nil {
		return nil, fmt.Errorf("failed to load previous digest: %w", err)
	}
	result.DigestPrev = prevInfo.Digest

	if !opts.Force && prevInfo.Digest == currentDigest && currentDigest != "" {
		if !CheckDrift(paths, target) {
			result.Skipped = true
			return result, nil
		}
	}

	compileResult, err := Compile(CompileOptions{
		Paths:      paths,
		DryRun:     opts.DryRun,
		Target:     target,
		Apply:      !opts.DryRun,
		EmitMode:   opts.Mode,
		EmitDryRun: opts.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("compile failed: %w", err)
	}
	result.CompileResult = compileResult
	result.EmitResult = compileResult.EmitResult

	if len(compileResult.Errors) > 0 {
		return result, nil
	}

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
func CheckDrift(paths *PathConfig, target string) bool {
	if paths == nil {
		return true
	}

	resolvedPath := paths.ResolvedManifestAbs()
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return true
	}

	var resolved manifest.ResolvedManifest
	if err := yaml.Unmarshal(data, &resolved); err != nil {
		return true
	}

	emitObj, err := newEmitterForTarget(target, paths)
	if err != nil {
		return true
	}

	buildDir := paths.BuildDirAbs()
	ok, err := emitObj.Check(&resolved, buildDir)
	if err != nil || !ok {
		return true
	}

	return false
}
