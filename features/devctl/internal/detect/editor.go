package detect

import "fmt"

// Editor represents a supported editor or agent.
type Editor string

const (
	EditorVSCode Editor = "code"
	EditorCursor Editor = "cursor"
	EditorAG     Editor = "ag"
	EditorClaude Editor = "claude"
)

// EnvKeyEditor is the environment variable name for the default editor.
const EnvKeyEditor = "DEVCTL_EDITOR"

// ParseEditor parses a string into an Editor value.
// Accepts aliases: "vscode" -> "code", "antigravity" -> "ag".
func ParseEditor(s string) (Editor, error) {
	switch s {
	case "code", "vscode":
		return EditorVSCode, nil
	case "cursor":
		return EditorCursor, nil
	case "ag", "antigravity":
		return EditorAG, nil
	case "claude":
		return EditorClaude, nil
	default:
		return "", fmt.Errorf("unknown editor: %q (supported: code, cursor, ag, claude)", s)
	}
}

// ResolveEditor determines the editor using the following priority:
//  1. CLI flag (cliFlag)
//  2. Environment variable (envValue, from DEVCTL_EDITOR)
//  3. Feature-level config (featureConfig)
//  4. Global config (globalConfig, from .devrc.yaml)
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
