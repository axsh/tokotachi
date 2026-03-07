package editor

import (
	"os"

	"github.com/escape-dev/devctl/internal/log"
)

// LaunchOptions holds parameters for launching an editor.
type LaunchOptions struct {
	WorktreePath    string
	ContainerName   string
	NewWindow       bool
	TryDevcontainer bool
	Logger          *log.Logger
	DryRun          bool
}

// LaunchResult describes the outcome of an editor launch.
type LaunchResult struct {
	Method    string // "devcontainer", "local", "cli"
	Fallback  bool   // true if fallback was used
	EditorCmd string // the actual command executed
}

// Launcher is the interface for editor-specific launch logic.
type Launcher interface {
	Launch(opts LaunchOptions) (LaunchResult, error)
	Name() string
}

// ResolveCommand returns the editor command from environment variable,
// falling back to defaultCmd if the env var is not set.
func ResolveCommand(envKey, defaultCmd string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultCmd
}
