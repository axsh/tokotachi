package compiler

import (
	"fmt"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
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
// compile -> emit -> apply
func Deploy(opts DeployOptions) (*DeployResult, error) {
	result := &DeployResult{}

	// 1. Load config
	_, err := LoadConfig(opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Resolve target early
	target := opts.Target
	if target == "" {
		target = "antigravity"
	}

	// 3. Compile
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
	result.Skipped = false

	return result, nil
}

