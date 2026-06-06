package emitter

import (
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// EmitMode controls how files are written during emit.
type EmitMode string

const (
	// EmitModeOverwrite always writes files, overwriting existing ones (default).
	EmitModeOverwrite EmitMode = "overwrite"
	// EmitModeImmune writes files and removes orphan files from target directories.
	EmitModeImmune EmitMode = "immune"
	// EmitModeSkip writes files only if they do not already exist.
	EmitModeSkip EmitMode = "skip"
)

// ValidEmitModes returns true if the given mode is a recognized emit mode.
func ValidEmitModes(mode EmitMode) bool {
	switch mode {
	case EmitModeOverwrite, EmitModeImmune, EmitModeSkip:
		return true
	}
	return false
}

// EmitOptions holds options passed to Emit.
type EmitOptions struct {
	Mode   EmitMode
	DryRun bool
}

// Emitter defines the interface for emitting resolved manifests to agent-specific config files.
type Emitter interface {
	// Emit generates target-specific files into buildDir or project paths.
	Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) error
	// Check verifies if generated files in project paths match the resolved manifest.
	Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error)
}
