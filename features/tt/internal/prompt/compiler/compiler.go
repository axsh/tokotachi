package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/emitter"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/memory"
)

// CompileOptions holds options for the compile pipeline
type CompileOptions struct {
	ProjectPath string
	DryRun      bool
	Target      string
	Apply       bool
	EmitMode    emitter.EmitMode
	EmitDryRun  bool
}

// CompileResult holds the output of the compile pipeline
type CompileResult struct {
	ResolvedYAML string // generated resolved manifest content
	Resolved     *manifest.ResolvedManifest
	Errors       []manifest.ValidationError
	EmitResult   *emitter.EmitResult // emitted files info for coordinated cleanup
}

// Compile executes the full parse -> validate -> resolve -> generate pipeline
func Compile(opts CompileOptions) (*CompileResult, error) {
	result := &CompileResult{}

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

	// 3. Parse all entities
	entities, parseErrors := manifest.ParseAllEntities(cfg, rootDir)
	result.Errors = append(result.Errors, parseErrors...)

	// 4. Parse all arch docs
	archPattern := cfg.Sources["memory_docs"]
	var memDocs []*manifest.MemoryDoc
	if archPattern != "" {
		var fmErrors []manifest.ValidationError
		memDocs, fmErrors = memory.ParseAllMemoryDocs(rootDir, archPattern)
		result.Errors = append(result.Errors, fmErrors...)
	}

	// 5. Schema validation
	schemasDir := filepath.Join(rootDir, "prompts", "manifest", "schemas")
	validator, err := manifest.NewValidator(schemasDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}
	for _, e := range entities {
		schemaErrors := validator.ValidateSchema(e)
		result.Errors = append(result.Errors, schemaErrors...)
	}

	// 6. ID uniqueness
	idErrors := manifest.ValidateIDUniqueness(entities, memDocs)
	result.Errors = append(result.Errors, idErrors...)

	// 7. Reference integrity
	refErrors := manifest.ValidateReferences(entities, memDocs, rootDir)
	result.Errors = append(result.Errors, refErrors...)

	// 8. If validation errors exist, return without generating
	if len(result.Errors) > 0 {
		return result, nil
	}

	// Clean build directory (unless dry-run)
	if !opts.DryRun {
		buildDir := filepath.Clean(filepath.Join(rootDir, cfg.Defaults.BuildDir))
		if err := os.RemoveAll(buildDir); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to clean build dir %s: %w", buildDir, err)
		}
		if err := os.MkdirAll(buildDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create build dir %s: %w", buildDir, err)
		}
	}

	// 9. Resolve
	resolved, err := manifest.Resolve(cfg, entities, memDocs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve manifest: %w", err)
	}
	result.Resolved = resolved

	// 10. Marshal resolved manifest
	resolvedYAML, err := manifest.MarshalResolvedManifest(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resolved manifest: %w", err)
	}
	result.ResolvedYAML = resolvedYAML

	// 11. Write files (unless DryRun)
	if !opts.DryRun {
		// Write resolved manifest
		resolvedPath := filepath.Join(rootDir, cfg.Outputs.ResolvedManifest)
		if err := writeFile(resolvedPath, resolvedYAML); err != nil {
			return nil, fmt.Errorf("failed to write resolved manifest: %w", err)
		}
	}

	// 13. Call emitter if Target is specified
	if opts.Target != "" {
		var emitObj emitter.Emitter
		switch opts.Target {
		case "antigravity":
			emitObj = emitter.NewAntigravityEmitter(rootDir)
		case "cursor":
			emitObj = emitter.NewCursorEmitter(rootDir)
		case "claude-code":
			emitObj = emitter.NewClaudeCodeEmitter(rootDir)
		case "codex":
			emitObj = emitter.NewCodexEmitter(rootDir)
		default:
			return nil, fmt.Errorf("unknown emitter target: %s", opts.Target)
		}
		apply := opts.Apply && !opts.DryRun
		buildDir := filepath.Clean(filepath.Join(rootDir, cfg.Defaults.BuildDir))
		emitOpts := emitter.EmitOptions{
			Mode:   opts.EmitMode,
			DryRun: opts.EmitDryRun,
		}
		if emitOpts.Mode == "" {
			emitOpts.Mode = emitter.EmitModeOverwrite
		}
		emitResult, err := emitObj.Emit(resolved, buildDir, apply, emitOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to emit target %s: %w", opts.Target, err)
		}
		result.EmitResult = emitResult
	}

	return result, nil
}

// writeFile writes content to a file, creating parent directories if needed
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
