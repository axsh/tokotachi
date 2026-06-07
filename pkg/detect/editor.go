package detect

import (
	"fmt"

	"github.com/axsh/tokotachi/pkg/resolve"
)

// Editor represents a supported editor or agent.
type Editor string

const (
	EditorVSCode Editor = "code"
	EditorCursor Editor = "cursor"
	EditorAG     Editor = "ag"
	EditorClaude Editor = "claude"
)

// EnvKeyEditor is the environment variable name for the default editor.
const EnvKeyEditor = "TT_EDITOR"

// targetToEditor maps resolved target names to Editor constants.
// This is needed because some Editor constants differ from the canonical
// target name (e.g., "antigravity" -> EditorAG="ag").
var targetToEditor = map[string]Editor{
	"antigravity": EditorAG,
	"cursor":      EditorCursor,
	"claude-code": EditorClaude,
	"codex":       Editor("codex"),
}

// ParseEditor parses a string into an Editor value.
// Accepts aliases: "vscode" -> "code", "antigravity" -> "ag".
// Uses shared target name resolution for known targets.
func ParseEditor(s string) (Editor, error) {
	if s == "" {
		return "", fmt.Errorf("editor name cannot be empty")
	}

	// Handle editor-only names not in KnownTargets
	switch s {
	case "code", "vscode":
		return EditorVSCode, nil
	}

	// Delegate to shared target resolution ("all" is not valid for editors)
	resolved, err := resolve.ResolveTarget(s, false)
	if err != nil {
		// Allow custom editor names dynamically (e.g., "vim", "emacs")
		return Editor(s), nil
	}

	// Map resolved target name to Editor constant
	if ed, ok := targetToEditor[resolved]; ok {
		return ed, nil
	}
	return Editor(resolved), nil
}

// ResolveEditor determines the editor using the following priority:
//  1. CLI flag (cliFlag)
//  2. Environment variable (envValue, from TT_EDITOR)
//  3. Feature-level config (featureConfig)
//  4. Default value ("cursor")
//  5. Default: "cursor"
func ResolveEditor(cliFlag, envValue, featureConfig, globalConfig string) (Editor, error) {
	sources := []string{cliFlag, envValue, featureConfig, globalConfig}
	for _, src := range sources {
		if src != "" {
			return ParseEditor(src)
		}
	}
	return EditorCursor, nil // default
}
