package editor

import (
	"encoding/hex"
	"fmt"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
	"github.com/axsh/tokotachi/features/tt/internal/log"
)

// LaunchOptions holds parameters for launching an editor.
type LaunchOptions struct {
	WorktreePath    string
	ContainerName   string
	NewWindow       bool
	TryDevcontainer bool
	Logger          *log.Logger
	DryRun          bool
	CmdRunner       *cmdexec.Runner
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
// Delegates to cmdexec.ResolveCommand.
func ResolveCommand(envKey, defaultCmd string) string {
	return cmdexec.ResolveCommand(envKey, defaultCmd)
}

// DevcontainerURI builds a vscode-remote URI for attaching to a Dev Container.
// The container name must be hex-encoded as required by the VS Code / Cursor
// Dev Containers extension protocol.
func DevcontainerURI(containerName, workspaceFolder string) string {
	hexName := hex.EncodeToString([]byte(containerName))
	if workspaceFolder == "" {
		workspaceFolder = "/workspace"
	}
	return fmt.Sprintf("vscode-remote://attached-container+%s%s", hexName, workspaceFolder)
}
